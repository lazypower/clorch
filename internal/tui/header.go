package tui

import (
	"fmt"

	"github.com/lazypower/clorch/internal/state"
	"github.com/lazypower/clorch/internal/usage"
)

func renderHeader(summary state.StatusSummary, usageSummary usage.UsageSummary, yolo bool, width int) string {
	title := titleStyle.Render("CLORCH")
	counts := headerStyle.Render(fmt.Sprintf("▪ %d agents  ▪ %s  %s  %s",
		summary.Total,
		agentWorkingStyle.Render(fmt.Sprintf("%d working", summary.Working)),
		agentIdleStyle.Render(fmt.Sprintf("%d idle", summary.Idle)),
		agentWaitingStyle.Render(fmt.Sprintf("%d waiting", summary.Waiting)),
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
