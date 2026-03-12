package usage

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// UsageUpdateMsg is sent to the TUI with updated usage data.
type UsageUpdateMsg struct {
	Summary UsageSummary
}

type Tracker struct {
	parser    *Parser
	program   *tea.Program
	cancel    context.CancelFunc
	history   []usageSnapshot
	resetTick int
}

type usageSnapshot struct {
	time time.Time
	cost float64
}

func NewTracker(parser *Parser, program *tea.Program) *Tracker {
	return &Tracker{parser: parser, program: program}
}

func (t *Tracker) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel
	go t.loop(ctx)
}

func (t *Tracker) Stop() {
	if t.cancel != nil {
		t.cancel()
	}
}

func (t *Tracker) loop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	t.poll()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.resetTick++
			if t.resetTick >= 6 {
				t.parser.Reset()
				t.resetTick = 0
			}
			t.poll()
		}
	}
}

func (t *Tracker) poll() {
	pr := t.parser.Poll()
	cost := CalculateCost(pr.Total, pr.Model)
	now := time.Now()

	t.history = append(t.history, usageSnapshot{time: now, cost: cost})
	cutoff := now.Add(-10 * time.Minute)
	trimIdx := 0
	for i, s := range t.history {
		if s.time.After(cutoff) {
			trimIdx = i
			break
		}
	}
	t.history = t.history[trimIdx:]

	perSession := make(map[string]SessionCost)
	for sid, st := range pr.PerSession {
		perSession[sid] = SessionCost{
			Tokens: st.Tokens,
			Cost:   CalculateCost(st.Tokens, st.Model),
		}
	}

	if t.program != nil {
		t.program.Send(UsageUpdateMsg{
			Summary: UsageSummary{
				Tokens:     pr.Total,
				Cost:       cost,
				BurnRate:   t.burnRate(),
				PerSession: perSession,
			},
		})
	}
}

func (t *Tracker) burnRate() float64 {
	if len(t.history) < 2 {
		return 0
	}
	first := t.history[0]
	last := t.history[len(t.history)-1]
	hours := last.time.Sub(first.time).Hours()
	if hours <= 0 {
		return 0
	}
	return (last.cost - first.cost) / hours
}
