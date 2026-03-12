package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

// Manager reads and manages agent state files from the state directory.
type Manager struct {
	stateDir string
}

func NewManager(stateDir string) *Manager {
	return &Manager{stateDir: stateDir}
}

// StateDir returns the state directory path.
func (m *Manager) StateDir() string { return m.stateDir }

// Scan reads all state files and returns enriched agent states, summary, and action queue.
func (m *Manager) Scan() ([]AgentState, StatusSummary, []ActionItem) {
	entries, err := os.ReadDir(m.stateDir)
	if err != nil {
		return nil, StatusSummary{}, nil
	}

	now := time.Now()
	var agents []AgentState

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(m.stateDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var agent AgentState
		if err := json.Unmarshal(data, &agent); err != nil {
			continue
		}
		if agent.LastEventTime != "" {
			if t, err := time.Parse(time.RFC3339, agent.LastEventTime); err == nil {
				agent.StaleDuration = now.Sub(t)
			}
		}
		agent.StuckLoop = detectStuckLoop(agent.RecentTools, now)
		agents = append(agents, agent)
	}

	// Attention sort: urgency first, then recency within tier
	sort.Slice(agents, func(i, j int) bool {
		pi := statusPriority(agents[i].Status)
		pj := statusPriority(agents[j].Status)
		if pi != pj {
			return pi < pj
		}
		// Within same priority: most recently active first
		return agents[i].StaleDuration < agents[j].StaleDuration
	})

	summary := ComputeSummary(agents)
	queue := buildActionQueue(agents)
	return agents, summary, queue
}

// CleanupStale removes state files for dead processes, old timestamps, and PID duplicates.
func (m *Manager) CleanupStale() {
	entries, err := os.ReadDir(m.stateDir)
	if err != nil {
		return
	}

	type stateFile struct {
		path  string
		agent AgentState
	}
	var files []stateFile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(m.stateDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var agent AgentState
		if err := json.Unmarshal(data, &agent); err != nil {
			continue
		}
		files = append(files, stateFile{path: path, agent: agent})
	}

	removeWithEvents := func(path string) {
		os.Remove(path)
		os.Remove(strings.TrimSuffix(path, ".json") + ".events")
	}

	// Pass 1: Dead process removal
	for _, f := range files {
		if f.agent.PID > 0 && !processAlive(f.agent.PID) {
			removeWithEvents(f.path)
		}
	}

	// Pass 2: Time-based removal (no PID, older than 1 hour)
	cutoff := time.Now().Add(-1 * time.Hour)
	for _, f := range files {
		if f.agent.PID == 0 && f.agent.LastEventTime != "" {
			if t, err := time.Parse(time.RFC3339, f.agent.LastEventTime); err == nil {
				if t.Before(cutoff) {
					removeWithEvents(f.path)
				}
			}
		}
	}

	// Pass 3: PID deduplication
	pidMap := make(map[int]stateFile)
	for _, f := range files {
		if f.agent.PID <= 0 {
			continue
		}
		if existing, ok := pidMap[f.agent.PID]; ok {
			if f.agent.ToolCount > existing.agent.ToolCount {
				removeWithEvents(existing.path)
				pidMap[f.agent.PID] = f
			} else {
				removeWithEvents(f.path)
			}
		} else {
			pidMap[f.agent.PID] = f
		}
	}
}

// EnsureStateDir creates the state directory if it doesn't exist.
func (m *Manager) EnsureStateDir() error {
	return os.MkdirAll(m.stateDir, 0755)
}

func statusPriority(status string) int {
	switch status {
	case StatusWaitingPermission:
		return 0
	case StatusWaitingAnswer:
		return 1
	case StatusError:
		return 2
	case StatusWorking:
		return 3
	case StatusIdle:
		return 4
	default:
		return 5
	}
}

func buildActionQueue(agents []AgentState) []ActionItem {
	var items []ActionItem
	letter := 'a'
	for _, a := range agents {
		if a.Status != StatusWaitingPermission && a.Status != StatusWaitingAnswer && a.Status != StatusError {
			continue
		}
		if letter > 'z' {
			break
		}
		items = append(items, ActionItem{Agent: a, Letter: string(letter)})
		letter++
	}

	sort.SliceStable(items, func(i, j int) bool {
		pi := actionPriority(items[i].Agent.Status)
		pj := actionPriority(items[j].Agent.Status)
		if pi != pj {
			return pi < pj
		}
		ti := items[i].Agent.TmuxPane != ""
		tj := items[j].Agent.TmuxPane != ""
		return ti && !tj
	})

	for i := range items {
		items[i].Letter = fmt.Sprintf("%c", 'a'+i)
	}
	return items
}

func actionPriority(status string) int {
	switch status {
	case StatusWaitingPermission:
		return 0
	case StatusWaitingAnswer:
		return 1
	case StatusError:
		return 2
	default:
		return 3
	}
}

func processAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}

// detectStuckLoop returns true if 3+ recent tool calls with the same tool+args_hash
// occurred within the last 30 seconds.
func detectStuckLoop(recentTools []RecentToolCall, now time.Time) bool {
	if len(recentTools) < 3 {
		return false
	}
	cutoff := now.Add(-30 * time.Second)
	type key struct {
		tool     string
		argsHash string
	}
	counts := make(map[key]int)
	for _, rt := range recentTools {
		t, err := time.Parse(time.RFC3339, rt.Time)
		if err != nil {
			continue
		}
		if t.Before(cutoff) {
			continue
		}
		k := key{tool: rt.Tool, argsHash: rt.ArgsHash}
		counts[k]++
		if counts[k] >= 3 {
			return true
		}
	}
	return false
}
