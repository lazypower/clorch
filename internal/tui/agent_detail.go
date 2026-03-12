package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/lazypower/clorch/internal/state"
	"github.com/lazypower/clorch/internal/usage"
)

func renderAgentDetail(a state.AgentState, events []state.TimelineEvent, sessionCost usage.SessionCost, panelHeight int) string {
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
	meta = append(meta, fmt.Sprintf("  Subagents: %d", a.SubagentCount))
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

	// Calculate how many timeline lines we can fit
	// title(1) + meta lines + timeline header(2: blank + "TIMELINE") + hint(1)
	usedLines := 1 + len(meta) + 2
	availableForTimeline := panelHeight - usedLines - 1
	if availableForTimeline < 3 {
		availableForTimeline = 3
	}

	var lines []string
	lines = append(lines, meta...)

	if len(events) > 0 {
		lines = append(lines, "")
		lines = append(lines, sectionTitleStyle.Render("TIMELINE"))

		// Show most recent events that fit, newest at top
		showCount := minInt(len(events), availableForTimeline)
		start := len(events) - showCount
		for i := len(events) - 1; i >= start; i-- {
			ev := events[i]
			ts := formatEventTime(ev.Time)
			eventStyled := formatEventType(ev.Event)
			summary := ev.Summary
			if len(summary) > 55 {
				summary = summary[:52] + "..."
			}
			lines = append(lines, fmt.Sprintf("  %s  %s  %s", ts, eventStyled, agentIdleStyle.Render(summary)))
		}
		if start > 0 {
			lines = append(lines, agentIdleStyle.Render(fmt.Sprintf("  ▲ %d older events", start)))
		}
	}

	return title + "\n" + strings.Join(lines, "\n")
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
