package state

import (
	"context"
	"reflect"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
)

// StateUpdateMsg is sent to the TUI when state files change.
type StateUpdateMsg struct {
	Agents  []AgentState
	Summary StatusSummary
	Queue   []ActionItem
}

// Watcher monitors the state directory and sends updates to a tea.Program.
type Watcher struct {
	manager  *Manager
	program  *tea.Program
	pollMS   int
	cancel   context.CancelFunc
	lastSnap []AgentState
}

func NewWatcher(manager *Manager, program *tea.Program, pollMS int) *Watcher {
	return &Watcher{
		manager: manager,
		program: program,
		pollMS:  pollMS,
	}
}

func (w *Watcher) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		go w.pollLoop(ctx)
		return
	}
	if err := fsw.Add(w.manager.stateDir); err != nil {
		fsw.Close()
		go w.pollLoop(ctx)
		return
	}
	go w.fsnotifyLoop(ctx, fsw)
}

func (w *Watcher) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
}

func (w *Watcher) fsnotifyLoop(ctx context.Context, fsw *fsnotify.Watcher) {
	defer fsw.Close()
	debounce := time.NewTimer(100 * time.Millisecond)
	debounce.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-fsw.Events:
			if !ok {
				return
			}
			debounce.Reset(100 * time.Millisecond)
		case <-fsw.Errors:
		case <-debounce.C:
			w.scanAndSend()
		}
	}
}

func (w *Watcher) pollLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(w.pollMS) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.scanAndSend()
		}
	}
}

func (w *Watcher) scanAndSend() {
	agents, summary, queue := w.manager.Scan()
	if !w.changed(agents) {
		return
	}
	w.lastSnap = agents
	if w.program != nil {
		w.program.Send(StateUpdateMsg{Agents: agents, Summary: summary, Queue: queue})
	}
}

func (w *Watcher) changed(agents []AgentState) bool {
	if len(agents) != len(w.lastSnap) {
		return true
	}
	return !reflect.DeepEqual(agents, w.lastSnap)
}
