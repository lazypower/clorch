package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/lazypower/clorch/internal/state"
)

var sparkChars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

func renderAgentTable(agents []state.AgentState, selectedIdx int, width int) string {
	if len(agents) == 0 {
		return agentIdleStyle.Render("  No active agents")
	}
	var lines []string
	for i, a := range agents {
		lines = append(lines, renderAgentRow(a, i == selectedIdx))
	}
	return strings.Join(lines, "\n")
}

func renderAgentRow(a state.AgentState, selected bool) string {
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
	if a.DisplayName != "" {
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

	subagents := ""
	if a.SubagentCount > 0 {
		subagents = fmt.Sprintf(" [%d▸]", a.SubagentCount)
	}

	line1 := fmt.Sprintf("  %s %s  %s  %s%s  %s  %s",
		indicator, name, statusText, spark, subagents, branch, agoStyled)
	line2 := fmt.Sprintf("    %s  %s",
		agentIdleStyle.Render(a.CWD), agentIdleStyle.Render(fmt.Sprintf("%d tools", a.ToolCount)))

	result := line1 + "\n" + line2
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
