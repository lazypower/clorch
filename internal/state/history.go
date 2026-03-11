package state

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HistoryResolver resolves session display names from Claude Code history.
type HistoryResolver struct {
	historyPath string
	projectsDir string
	names       map[string]string
	historyMod  time.Time
}

func NewHistoryResolver(historyPath, projectsDir string) *HistoryResolver {
	return &HistoryResolver{
		historyPath: historyPath,
		projectsDir: projectsDir,
		names:       make(map[string]string),
	}
}

func (r *HistoryResolver) Resolve(sessionID string) string {
	r.refresh()
	return r.names[sessionID]
}

func (r *HistoryResolver) EnrichAgents(agents []AgentState) {
	r.refresh()
	for i := range agents {
		if name, ok := r.names[agents[i].SessionID]; ok {
			agents[i].DisplayName = name
		}
	}
}

func (r *HistoryResolver) refresh() {
	r.loadHistory()
	r.loadCustomTitles()
}

type historyRecord struct {
	SessionID string `json:"sessionId"`
	Display   string `json:"display"`
}

func (r *HistoryResolver) loadHistory() {
	info, err := os.Stat(r.historyPath)
	if err != nil {
		return
	}
	if !info.ModTime().After(r.historyMod) {
		return
	}
	r.historyMod = info.ModTime()

	f, err := os.Open(r.historyPath)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 256*1024)
	for scanner.Scan() {
		var rec historyRecord
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			continue
		}
		if rec.SessionID != "" && rec.Display != "" {
			if _, exists := r.names[rec.SessionID]; !exists {
				r.names[rec.SessionID] = rec.Display
			}
		}
	}
}

type customTitleRecord struct {
	Type        string `json:"type"`
	CustomTitle string `json:"customTitle"`
}

func (r *HistoryResolver) loadCustomTitles() {
	dirs, err := filepath.Glob(filepath.Join(r.projectsDir, "*"))
	if err != nil {
		return
	}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
				continue
			}
			sessionID := strings.TrimSuffix(entry.Name(), ".jsonl")
			path := filepath.Join(dir, entry.Name())
			if title := findCustomTitle(path); title != "" {
				r.names[sessionID] = title
			}
		}
	}
}

func findCustomTitle(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var lastTitle string
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "custom-title") {
			continue
		}
		var rec customTitleRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue
		}
		if rec.Type == "custom-title" && rec.CustomTitle != "" {
			lastTitle = rec.CustomTitle
		}
	}
	return lastTitle
}
