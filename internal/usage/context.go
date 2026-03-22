package usage

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ModelContextCapacity returns the context window size for a given model string.
// Uses prefix matching after stripping the "claude-" prefix and date suffix.
func ModelContextCapacity(model string) int64 {
	key := resolveModelKey(model)
	if strings.HasPrefix(key, "opus") {
		return 1_000_000
	}
	if strings.HasPrefix(key, "sonnet") {
		return 200_000
	}
	if strings.HasPrefix(key, "haiku") {
		return 200_000
	}
	return 200_000
}

// ContextWindowPct computes how full the context window is based on the last
// API call's input tokens and the model's capacity.
func ContextWindowPct(lastInput int64, capacity int64) float64 {
	if capacity <= 0 || lastInput <= 0 {
		return 0
	}
	pct := float64(lastInput) / float64(capacity) * 100
	if pct > 100 {
		return 100
	}
	return pct
}

// ContextPctColor returns a Nord-palette color based on context fill percentage.
// <60% green, 60-80% yellow, 80%+ red.
func ContextPctColor(pct float64) lipgloss.Color {
	if pct >= 80 {
		return lipgloss.Color("#bf616a") // nord red
	}
	if pct >= 60 {
		return lipgloss.Color("#ebcb8b") // nord yellow
	}
	return lipgloss.Color("#a3be8c") // nord green
}

// RenderContextGauge renders a bar gauge like [████░░░░] 67%
func RenderContextGauge(pct float64, width int) string {
	if width < 4 {
		width = 4
	}
	filled := int(pct / 100 * float64(width))
	if filled > width {
		filled = width
	}
	empty := width - filled

	color := ContextPctColor(pct)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	styled := lipgloss.NewStyle().Foreground(color).Render("[" + bar + "]")
	label := lipgloss.NewStyle().Foreground(color).Render(fmt.Sprintf(" %d%%", int(pct)))
	return styled + label
}
