package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/lazypower/clorch/internal/state"
	"github.com/lazypower/clorch/internal/usage"
)

// recentActivityCount bounds the inline timeline in the detail panel. The full
// log lives in the scrollable history view (h) — keeping the panel preview
// fixed stops a long-running session from dominating the panel and shifting the
// rest of the detail content as new events stream in.
const recentActivityCount = 5

func renderAgentDetail(a state.AgentState, events []state.TimelineEvent, sessionCost usage.SessionCost) string {
	title := sectionTitleStyle.Render("DETAIL")
	name := a.ProjectName
	if a.BranchLabel != "" {
		name = a.BranchLabel
	} else if a.DisplayName != "" {
		name = a.DisplayName
	}

	var meta []string
	meta = append(meta, fmt.Sprintf("  Name:      %s", name))
	meta = append(meta, fmt.Sprintf("  Session:   %s", a.SessionID))
	meta = append(meta, fmt.Sprintf("  Status:    %s", a.Status))
	meta = append(meta, fmt.Sprintf("  Model:     %s", a.Model))
	meta = append(meta, fmt.Sprintf("  CWD:       %s", a.CWD))
	meta = append(meta, fmt.Sprintf("  PID:       %d", a.PID))
	meta = append(meta, "")
	meta = append(meta, fmt.Sprintf("  Tools:     %d", a.ToolCount))
	meta = append(meta, fmt.Sprintf("  Errors:    %d", a.ErrorCount))
	meta = append(meta, fmt.Sprintf("  Subagents: %d (%d running)", a.SubagentCount, a.RunningSubagentCount()))
	meta = append(meta, fmt.Sprintf("  Compacts:  %d", a.CompactCount))
	meta = append(meta, fmt.Sprintf("  Tasks:     %d", a.TaskCompletedCount))
	if sessionCost.Cost > 0 {
		meta = append(meta, costStyle.Render(fmt.Sprintf("  Cost:      $%.2f", sessionCost.Cost)))
		tok := sessionCost.Tokens
		meta = append(meta, agentIdleStyle.Render(fmt.Sprintf("  In/Out:    %dk / %dk tokens", (tok.InputTokens+tok.CacheCreationTokens)/1000, tok.OutputTokens/1000)))
		if tok.CacheReadTokens > 0 {
			meta = append(meta, agentIdleStyle.Render(fmt.Sprintf("  Cached:    %dk tokens", tok.CacheReadTokens/1000)))
		}
	}
	if sessionCost.Model != "" && sessionCost.Tokens.LastInput > 0 {
		cap := usage.ModelContextCapacity(sessionCost.Model)
		pct := usage.ContextWindowPct(sessionCost.Tokens.LastInput, cap)
		gauge := usage.RenderContextGauge(pct, 16)
		compactSuffix := ""
		if a.CompactCount > 0 {
			compactSuffix = fmt.Sprintf(" (%dc)", a.CompactCount)
		}
		meta = append(meta, fmt.Sprintf("  Context:  %s%s", gauge, compactSuffix))
	}
	meta = append(meta, "")
	if a.GitBranch != "" {
		meta = append(meta, fmt.Sprintf("  Branch:    %s", a.GitBranch))
		meta = append(meta, fmt.Sprintf("  Dirty:     %d files", a.GitDirtyCount))
	}
	if a.TmuxSession != "" {
		meta = append(meta, fmt.Sprintf("  tmux:      %s:%s.%s", a.TmuxSession, a.TmuxWindowIndex, a.TmuxPane))
	}
	if a.BranchedFrom != "" {
		meta = append(meta, fmt.Sprintf("  Parent:    %s", a.BranchedFrom[:minInt(12, len(a.BranchedFrom))]))
	}
	meta = append(meta, fmt.Sprintf("  Last tool:  %s", a.LastTool))
	meta = append(meta, fmt.Sprintf("  Started:    %s", a.StartedAt))
	if a.ToolRequestSummary != nil {
		meta = append(meta, agentWaitingStyle.Render("  Pending: "+*a.ToolRequestSummary))
	}
	if len(a.FilesModified) > 0 {
		meta = append(meta, "")
		meta = append(meta, sectionTitleStyle.Render(fmt.Sprintf("FILES (%d)", len(a.FilesModified))))
		// Show most recent files (last modified = end of list), capped
		maxFiles := 8
		start := 0
		if len(a.FilesModified) > maxFiles {
			start = len(a.FilesModified) - maxFiles
		}
		for _, fp := range a.FilesModified[start:] {
			// Shorten paths relative to CWD
			display := fp
			if a.CWD != "" && len(fp) > len(a.CWD)+1 {
				if fp[:len(a.CWD)] == a.CWD {
					display = fp[len(a.CWD)+1:]
				}
			}
			meta = append(meta, agentIdleStyle.Render("  "+display))
		}
		if start > 0 {
			meta = append(meta, agentIdleStyle.Render(fmt.Sprintf("  … %d more", start)))
		}
	}

	if len(a.Subagents) > 0 {
		// Only render bubbles for running subagents — a long session can spawn
		// hundreds, and one row each (the dead ones never go away) buries the
		// rest of the detail. Finished subagents collapse into a count, and the
		// running list is sorted newest-first for a stable order (the map
		// itself iterates randomly) and capped.
		var running []state.SubAgent
		for _, sub := range a.Subagents {
			if sub.Status == "running" {
				running = append(running, sub)
			}
		}
		sort.Slice(running, func(i, j int) bool {
			return running[i].StartedAt > running[j].StartedAt
		})
		finished := len(a.Subagents) - len(running)

		header := fmt.Sprintf("SUBAGENTS (%d running", len(running))
		if finished > 0 {
			header += fmt.Sprintf(", %d done", finished)
		}
		header += ")"
		meta = append(meta, "")
		meta = append(meta, sectionTitleStyle.Render(header))

		const maxSubagentRows = 10
		shown := running
		if len(shown) > maxSubagentRows {
			shown = shown[:maxSubagentRows]
		}
		for _, sub := range shown {
			statusIcon := agentWorkingStyle.Render("●")
			meta = append(meta, fmt.Sprintf("  %s %s  %s", statusIcon, sub.AgentType, agentIdleStyle.Render(sub.AgentID[:minInt(12, len(sub.AgentID))])))
		}
		if len(running) > maxSubagentRows {
			meta = append(meta, agentIdleStyle.Render(fmt.Sprintf("  … %d more running", len(running)-maxSubagentRows)))
		}
	}

	var lines []string
	lines = append(lines, meta...)

	if len(events) > 0 {
		lines = append(lines, "")
		lines = append(lines, sectionTitleStyle.Render("RECENT ACTIVITY"))

		// Show the most recent events, newest at top. Bounded by
		// recentActivityCount — the full log is in the history view (h).
		showCount := minInt(len(events), recentActivityCount)
		start := len(events) - showCount
		for i := len(events) - 1; i >= start; i-- {
			lines = append(lines, formatEventLine(events[i], 55))
		}
		if start > 0 {
			lines = append(lines, agentIdleStyle.Render(fmt.Sprintf("  ▲ %d older — press h for full history", start)))
		}
	}

	return title + "\n" + strings.Join(lines, "\n")
}

// formatEventLine renders a single timeline event row, truncating the summary
// to maxSummary characters. Shared by the detail preview and history view.
func formatEventLine(ev state.TimelineEvent, maxSummary int) string {
	summary := ev.Summary
	if len(summary) > maxSummary {
		summary = summary[:maxSummary-3] + "..."
	}
	return fmt.Sprintf("  %s  %s  %s", formatEventTime(ev.Time), formatEventType(ev.Event), agentIdleStyle.Render(summary))
}

// renderHistory draws the full, scrollable session history as a standalone
// view. Events are listed newest-first; scroll reveals older entries. height is
// the total terminal height; chrome (title, blank, footer) is reserved here.
func renderHistory(a state.AgentState, events []state.TimelineEvent, scroll, width, height int) string {
	name := a.ProjectName
	if a.BranchLabel != "" {
		name = a.BranchLabel
	} else if a.DisplayName != "" {
		name = a.DisplayName
	}

	title := sectionTitleStyle.Render("HISTORY") + agentIdleStyle.Render("  "+name)

	if len(events) == 0 {
		hint := footerStyle.Render("esc/h:close")
		return strings.Join([]string{title, "", agentIdleStyle.Render("  no events recorded"), "", hint}, "\n")
	}

	// Reserve: title(1) + blank(1) + blank(1) + footer(1).
	visible := height - 4
	if visible < 1 {
		visible = 1
	}

	maxScroll := maxInt(0, len(events)-visible)
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}

	// Newest-first: index 0 of the display = last event. Trim summary to fit
	// width, leaving room for the timestamp + type columns.
	summaryWidth := width - 22
	if summaryWidth < 20 {
		summaryWidth = 20
	}

	var lines []string
	for row := 0; row < visible; row++ {
		idx := len(events) - 1 - (scroll + row)
		if idx < 0 {
			break
		}
		lines = append(lines, formatEventLine(events[idx], summaryWidth))
	}

	first := scroll + 1
	last := scroll + len(lines)
	hint := footerStyle.Render(fmt.Sprintf("j/k:scroll  g/G:top/bottom  esc/h:close   %d–%d of %d", first, last, len(events)))

	return strings.Join([]string{title, "", strings.Join(lines, "\n"), "", hint}, "\n")
}

func formatEventTime(raw string) string {
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return agentIdleStyle.Render(raw)
	}
	return agentIdleStyle.Render(t.Local().Format("15:04:05"))
}

func formatEventType(event string) string {
	switch event {
	case "PreToolUse":
		return agentWorkingStyle.Render("TOOL")
	case "PostToolUseFailure":
		return agentErrorStyle.Render("FAIL")
	case "PermissionRequest":
		return agentWaitingStyle.Render("PERM")
	case "SessionStart":
		return titleStyle.Render("START")
	case "Stop":
		return agentIdleStyle.Render("STOP")
	case "UserPromptSubmit":
		return titleStyle.Render("PROMPT")
	case "SubagentStart":
		return agentWorkingStyle.Render("SUB+")
	case "SubagentStop":
		return agentWorkingStyle.Render("SUB-")
	case "PreCompact":
		return costStyle.Render("COMPACT")
	case "TaskCompleted":
		return agentWorkingStyle.Render("TASK")
	default:
		return agentIdleStyle.Render(event)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
