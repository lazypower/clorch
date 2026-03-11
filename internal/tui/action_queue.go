package tui

import (
	"fmt"
	"strings"

	"github.com/lazypower/clorch/internal/state"
)

func renderActionQueue(queue []state.ActionItem, focusedAction string) string {
	if len(queue) == 0 {
		return agentIdleStyle.Render("  No pending actions")
	}
	var lines []string
	for _, item := range queue {
		letter := actionLetterStyle.Render(item.Letter + ")")
		name := item.Agent.ProjectName
		if item.Agent.DisplayName != "" {
			name = item.Agent.DisplayName
		}

		var statusLabel string
		switch item.Agent.Status {
		case state.StatusWaitingPermission:
			statusLabel = agentWaitingStyle.Render(item.Agent.LastTool)
		case state.StatusWaitingAnswer:
			statusLabel = agentWaitingStyle.Render("question")
		case state.StatusError:
			statusLabel = agentErrorStyle.Render("error")
		}

		header := fmt.Sprintf("  %s %s: %s", letter, name, statusLabel)
		summary := ""
		if item.Agent.ToolRequestSummary != nil {
			s := *item.Agent.ToolRequestSummary
			if len(s) > 60 {
				s = s[:57] + "..."
			}
			summary = "\n     " + actionSummaryStyle.Render(s)
		}
		focused := ""
		if item.Letter == focusedAction {
			focused = " ◀"
		}
		lines = append(lines, header+focused+summary)
	}
	return strings.Join(lines, "\n")
}
