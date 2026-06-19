package usage

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestModelContextCapacity(t *testing.T) {
	tests := []struct {
		model string
		want  int64
	}{
		{"claude-opus-4-6-20260301", 1_000_000},
		{"claude-opus-4-5-20250101", 1_000_000},
		{"claude-sonnet-4-6-20260301", 200_000},
		{"claude-haiku-4-5-20251001", 200_000},
		{"unknown-model", 200_000},
		{"", 200_000},
	}
	for _, tt := range tests {
		got := ModelContextCapacity(tt.model)
		if got != tt.want {
			t.Errorf("ModelContextCapacity(%q) = %d, want %d", tt.model, got, tt.want)
		}
	}
}

func TestContextWindowPct(t *testing.T) {
	tests := []struct {
		lastInput int64
		capacity  int64
		want      float64
	}{
		{100_000, 200_000, 50},
		{200_000, 200_000, 100},
		{250_000, 200_000, 100}, // capped at 100
		{0, 200_000, 0},
		{100_000, 0, 0},
	}
	for _, tt := range tests {
		got := ContextWindowPct(tt.lastInput, tt.capacity)
		if got != tt.want {
			t.Errorf("ContextWindowPct(%d, %d) = %f, want %f", tt.lastInput, tt.capacity, got, tt.want)
		}
	}
}

func TestContextPctColor(t *testing.T) {
	// Compare the Dark variant, which carries the original Nord hue.
	darkHue := func(pct float64) string {
		c, ok := ContextPctColor(pct).(lipgloss.AdaptiveColor)
		if !ok {
			t.Fatalf("%v%% color is not an AdaptiveColor", pct)
		}
		return c.Dark
	}
	if green := darkHue(0); green != "#a3be8c" {
		t.Errorf("0%% color = %s, want green", green)
	}
	if green59 := darkHue(59.9); green59 != "#a3be8c" {
		t.Errorf("59.9%% color = %s, want green", green59)
	}
	if yellow := darkHue(60); yellow != "#ebcb8b" {
		t.Errorf("60%% color = %s, want yellow", yellow)
	}
	if yellow79 := darkHue(79.9); yellow79 != "#ebcb8b" {
		t.Errorf("79.9%% color = %s, want yellow", yellow79)
	}
	if red := darkHue(80); red != "#bf616a" {
		t.Errorf("80%% color = %s, want red", red)
	}
	if red100 := darkHue(100); red100 != "#bf616a" {
		t.Errorf("100%% color = %s, want red", red100)
	}
}

func TestRenderContextGauge(t *testing.T) {
	// Smoke test — just verify it doesn't panic and returns non-empty
	result := RenderContextGauge(50, 8)
	if result == "" {
		t.Error("RenderContextGauge returned empty string")
	}
	result100 := RenderContextGauge(100, 8)
	if result100 == "" {
		t.Error("RenderContextGauge(100) returned empty string")
	}
	result0 := RenderContextGauge(0, 8)
	if result0 == "" {
		t.Error("RenderContextGauge(0) returned empty string")
	}
}
