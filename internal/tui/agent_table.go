package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/lazypower/clorch/internal/state"
)

var sparkChars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// treeEntry holds an agent and its tree-drawing prefix for display.
type treeEntry struct {
	agent  state.AgentState
	idx    int    // original index in the flat agents slice
	prefix string // tree characters like "├── " or "└── "
}

// buildTree reorders agents into lineage groups with tree prefixes.
// Roots are sorted by their original attention-sort order. Children nest
// under their parent in the same relative order.
func buildTree(agents []state.AgentState) []treeEntry {
	// Build parent→children map
	childMap := make(map[string][]int) // parent session ID → child indices
	isChild := make(map[int]bool)
	idxBySession := make(map[string]int)
	for i, a := range agents {
		idxBySession[a.SessionID] = i
	}
	for i, a := range agents {
		if a.BranchedFrom != "" {
			if _, parentExists := idxBySession[a.BranchedFrom]; parentExists {
				childMap[a.BranchedFrom] = append(childMap[a.BranchedFrom], i)
				isChild[i] = true
			}
		}
	}

	var entries []treeEntry
	var walk func(idx int, indent string, last bool, isRoot bool)
	walk = func(idx int, indent string, last bool, isRoot bool) {
		prefix := ""
		if !isRoot {
			if last {
				prefix = indent + "└── "
			} else {
				prefix = indent + "├── "
			}
		}
		entries = append(entries, treeEntry{agent: agents[idx], idx: idx, prefix: prefix})

		children := childMap[agents[idx].SessionID]
		childIndent := indent
		if !isRoot {
			if last {
				childIndent = indent + "    "
			} else {
				childIndent = indent + "│   "
			}
		}
		for ci, childIdx := range children {
			walk(childIdx, childIndent, ci == len(children)-1, false)
		}
	}

	// Walk roots in their original order (attention-sorted)
	for i := range agents {
		if !isChild[i] {
			walk(i, "", false, true)
		}
	}
	return entries
}

func renderAgentTable(agents []state.AgentState, selectedIdx int, width int) string {
	if len(agents) == 0 {
		return agentIdleStyle.Render("  No active agents")
	}
	tree := buildTree(agents)
	var lines []string
	for _, entry := range tree {
		lines = append(lines, renderAgentRow(entry.agent, entry.idx == selectedIdx, entry.prefix))
	}
	return strings.Join(lines, "\n")
}

func renderAgentRow(a state.AgentState, selected bool, treePrefix string) string {
	var indicator, statusText string
	switch a.Status {
	case state.StatusWorking:
		indicator = agentWorkingStyle.Render("●")
		statusText = agentWorkingStyle.Render("working")
	case state.StatusIdle:
		indicator = agentIdleStyle.Render("●")
		statusText = agentIdleStyle.Render("idle")
	case state.StatusWaitingPermission:
		indicator = agentWaitingStyle.Render("◉")
		statusText = agentWaitingStyle.Render("WAITING")
	case state.StatusWaitingAnswer:
		indicator = agentWaitingStyle.Render("◉")
		statusText = agentWaitingStyle.Render("QUESTION")
	case state.StatusError:
		indicator = agentErrorStyle.Render("✕")
		statusText = agentErrorStyle.Render("ERROR")
	default:
		indicator = agentIdleStyle.Render("○")
		statusText = agentIdleStyle.Render(a.Status)
	}

	name := a.ProjectName
	if a.BranchLabel != "" {
		name = a.BranchLabel
	} else if a.DisplayName != "" {
		name = a.DisplayName
	} else if a.TmuxWindow != "" {
		name = a.TmuxWindow
	}

	spark := renderSparkline(a.ActivityHistory)

	ago := formatDuration(a.StaleDuration)
	agoStyled := agentIdleStyle.Render(ago + " ago")
	if a.StaleDuration > 120*time.Second {
		agoStyled = staleCritStyle.Render(ago + " ago")
	} else if a.StaleDuration > 30*time.Second {
		agoStyled = staleWarnStyle.Render(ago + " ago")
	}

	branch := ""
	if a.GitBranch != "" {
		branch = agentIdleStyle.Render(a.GitBranch)
	}

	stuckIndicator := ""
	if a.StuckLoop {
		stuckIndicator = " " + stuckLoopStyle.Render("⚠ loop")
	}

	subagents := ""
	if a.SubagentCount > 0 {
		subagents = fmt.Sprintf(" [%d▸]", a.SubagentCount)
	}

	treePfx := ""
	detailIndent := "    "
	if treePrefix != "" {
		treePfx = agentIdleStyle.Render(treePrefix)
		// Align detail line under the agent name
		detailIndent = strings.Repeat(" ", len(treePrefix)+4)
	}

	line1 := fmt.Sprintf("  %s%s %s  %s%s  %s%s  %s  %s",
		treePfx, indicator, name, statusText, stuckIndicator, spark, subagents, branch, agoStyled)
	line2 := fmt.Sprintf("%s%s  %s",
		detailIndent, agentIdleStyle.Render(a.CWD), agentIdleStyle.Render(fmt.Sprintf("%d tools", a.ToolCount)))

	result := line1 + "\n" + line2

	// Idle fade: dim agents inactive for > 5 minutes
	if a.StaleDuration > 5*time.Minute && a.Status == state.StatusIdle {
		result = idleFadeStyle.Render(result)
	}

	if selected {
		result = agentSelectedStyle.Render(result)
	}
	return result
}

func renderSparkline(history []int) string {
	if len(history) == 0 {
		return ""
	}
	maxVal := 1
	for _, v := range history {
		if v > maxVal {
			maxVal = v
		}
	}
	var chars []rune
	for _, v := range history {
		idx := v * (len(sparkChars) - 1) / maxVal
		if idx >= len(sparkChars) {
			idx = len(sparkChars) - 1
		}
		chars = append(chars, sparkChars[idx])
	}
	return sparkStyle.Render(string(chars))
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return "0s"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}
