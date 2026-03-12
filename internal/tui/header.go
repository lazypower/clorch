package tui

import (
	"fmt"

	"github.com/lazypower/clorch/internal/state"
	"github.com/lazypower/clorch/internal/usage"
)

func renderHeader(summary state.StatusSummary, usageSummary usage.UsageSummary, yolo bool, version string, width int) string {
	title := titleStyle.Render("CLORCH " + version)
	counts := headerStyle.Render(fmt.Sprintf("▪ %d agents  ▪ %s  %s  %s  %s",
		summary.Total,
		agentWorkingStyle.Render(fmt.Sprintf("%d running", summary.Working)),
		agentIdleStyle.Render(fmt.Sprintf("%d idle", summary.Idle)),
		agentWaitingStyle.Render(fmt.Sprintf("%d waiting", summary.Waiting)),
		agentErrorStyle.Render(fmt.Sprintf("%d errors", summary.Error)),
	))
	cost := costStyle.Render(fmt.Sprintf("$%.2f", usageSummary.Cost))
	if usageSummary.BurnRate > 0 {
		cost += costStyle.Render(fmt.Sprintf(" ($%.0f/hr)", usageSummary.BurnRate))
	}
	yoloIndicator := ""
	if yolo {
		yoloIndicator = " " + yoloActiveStyle.Render("YOLO")
	}
	return title + "  " + counts + "  │ " + cost + yoloIndicator
}
