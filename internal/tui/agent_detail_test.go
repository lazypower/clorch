package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/lazypower/clorch/internal/state"
)

func TestHistoryVisibleRows(t *testing.T) {
	cases := []struct {
		height int
		want   int
	}{
		{height: 40, want: 36}, // 40 - 4 chrome
		{height: 5, want: 1},
		{height: 4, want: 1}, // clamps to at least 1
		{height: 0, want: 1},
		{height: -10, want: 1},
	}
	for _, c := range cases {
		if got := historyVisibleRows(c.height); got != c.want {
			t.Errorf("historyVisibleRows(%d) = %d, want %d", c.height, got, c.want)
		}
	}
}

func TestHistoryMaxScrollFor(t *testing.T) {
	cases := []struct {
		name   string
		total  int
		height int
		want   int
	}{
		{name: "fits entirely", total: 10, height: 40, want: 0},   // 36 visible >= 10
		{name: "exactly fits", total: 36, height: 40, want: 0},    // visible == total
		{name: "overflows by one", total: 37, height: 40, want: 1},
		{name: "long log", total: 200, height: 40, want: 164},     // 200 - 36
		{name: "empty", total: 0, height: 40, want: 0},
		{name: "never negative", total: 3, height: 4, want: 2},    // visible clamps to 1
	}
	for _, c := range cases {
		if got := historyMaxScrollFor(c.total, c.height); got != c.want {
			t.Errorf("%s: historyMaxScrollFor(%d, %d) = %d, want %d", c.name, c.total, c.height, got, c.want)
		}
	}
}

func makeEvents(n int) []state.TimelineEvent {
	events := make([]state.TimelineEvent, n)
	for i := range events {
		events[i] = state.TimelineEvent{
			Time:    "2026-06-20T00:00:00Z",
			Event:   "PreToolUse",
			Summary: fmt.Sprintf("event-%d", i),
		}
	}
	return events
}

func TestRenderHistory_Empty(t *testing.T) {
	out := renderHistory(state.AgentState{ProjectName: "proj"}, nil, 0, 80, 40)
	if !strings.Contains(out, "no events") {
		t.Errorf("expected empty-state message, got:\n%s", out)
	}
}

func TestRenderHistory_CounterAtTop(t *testing.T) {
	// 100 events, height 40 → 36 visible rows. At scroll 0 (newest-first),
	// the counter should read "1–36 of 100".
	out := renderHistory(state.AgentState{ProjectName: "proj"}, makeEvents(100), 0, 80, 40)
	if !strings.Contains(out, "1–36 of 100") {
		t.Errorf("expected counter '1–36 of 100', got:\n%s", out)
	}
	// Newest event (index 99) shows first; oldest (0) must be scrolled past.
	if !strings.Contains(out, "event-99") {
		t.Error("expected newest event-99 at top")
	}
	if strings.Contains(out, "event-0\n") || strings.Contains(out, "event-0 ") {
		t.Error("oldest event-0 should not be visible at scroll 0")
	}
}

func TestRenderHistory_ClampsOverscroll(t *testing.T) {
	// Scroll well past the end; render must clamp to the bottom window and
	// surface the oldest event rather than blanking out.
	events := makeEvents(100)
	out := renderHistory(state.AgentState{ProjectName: "proj"}, events, 9999, 80, 40)
	// maxScroll = 100 - 36 = 64 → window is events 0..35 (oldest first shown).
	if !strings.Contains(out, fmt.Sprintf("65–100 of %d", 100)) {
		t.Errorf("expected clamped counter '65–100 of 100', got:\n%s", out)
	}
	if !strings.Contains(out, "event-0") {
		t.Error("expected oldest event-0 visible at the bottom of the log")
	}
}

func TestRenderHistory_ShortLogNoScroll(t *testing.T) {
	// Fewer events than visible rows: counter covers all of them.
	out := renderHistory(state.AgentState{ProjectName: "proj"}, makeEvents(5), 0, 80, 40)
	if !strings.Contains(out, "1–5 of 5") {
		t.Errorf("expected counter '1–5 of 5', got:\n%s", out)
	}
}
