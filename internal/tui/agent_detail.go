package tui

import (
	"fmt"
	"strings"

	"github.com/lazypower/clorch/internal/state"
)

func renderAgentDetail(a state.AgentState) string {
	title := sectionTitleStyle.Render("DETAIL")
	name := a.ProjectName
	if a.DisplayName != "" {
		name = a.DisplayName
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("  Name:      %s", name))
	lines = append(lines, fmt.Sprintf("  Session:   %s", a.SessionID))
	lines = append(lines, fmt.Sprintf("  Status:    %s", a.Status))
	lines = append(lines, fmt.Sprintf("  Model:     %s", a.Model))
	lines = append(lines, fmt.Sprintf("  CWD:       %s", a.CWD))
	lines = append(lines, fmt.Sprintf("  PID:       %d", a.PID))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  Tools:     %d", a.ToolCount))
	lines = append(lines, fmt.Sprintf("  Errors:    %d", a.ErrorCount))
	lines = append(lines, fmt.Sprintf("  Subagents: %d", a.SubagentCount))
	lines = append(lines, fmt.Sprintf("  Compacts:  %d", a.CompactCount))
	lines = append(lines, fmt.Sprintf("  Tasks:     %d", a.TaskCompletedCount))
	lines = append(lines, "")
	if a.GitBranch != "" {
		lines = append(lines, fmt.Sprintf("  Branch:    %s", a.GitBranch))
		lines = append(lines, fmt.Sprintf("  Dirty:     %d files", a.GitDirtyCount))
	}
	if a.TmuxSession != "" {
		lines = append(lines, fmt.Sprintf("  tmux:      %s:%s.%s", a.TmuxSession, a.TmuxWindowIndex, a.TmuxPane))
	}
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  Last event: %s", a.LastEvent))
	lines = append(lines, fmt.Sprintf("  Last tool:  %s", a.LastTool))
	lines = append(lines, fmt.Sprintf("  Started:    %s", a.StartedAt))
	if a.NotificationMessage != nil {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("  Notification: %s", *a.NotificationMessage))
	}
	if a.ToolRequestSummary != nil {
		lines = append(lines, "")
		lines = append(lines, "  Pending request:")
		lines = append(lines, "  "+*a.ToolRequestSummary)
	}
	return title + "\n" + strings.Join(lines, "\n")
}
