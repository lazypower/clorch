package tmux

import (
	"strings"
	"testing"

	"github.com/lazypower/clorch/internal/state"
)

func TestWidget(t *testing.T) {
	summary := state.StatusSummary{
		Working: 3,
		Waiting: 1,
		Error:   0,
	}
	got := Widget(summary)

	if !strings.Contains(got, "●3") {
		t.Errorf("widget missing working count: %q", got)
	}
	if !strings.Contains(got, "◉1") {
		t.Errorf("widget missing waiting count: %q", got)
	}
	if !strings.Contains(got, "✕0") {
		t.Errorf("widget missing error count: %q", got)
	}
	// Should contain tmux color escapes
	if !strings.Contains(got, "#[fg=") {
		t.Errorf("widget missing tmux color escapes: %q", got)
	}
}

func TestWidgetZeroes(t *testing.T) {
	got := Widget(state.StatusSummary{})
	if !strings.Contains(got, "●0") {
		t.Errorf("zero widget missing working: %q", got)
	}
}
