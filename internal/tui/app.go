package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lazypower/clorch/internal/notify"
	"github.com/lazypower/clorch/internal/rules"
	"github.com/lazypower/clorch/internal/state"
	"github.com/lazypower/clorch/internal/tmux"
	"github.com/lazypower/clorch/internal/usage"
)

type Model struct {
	agents       []state.AgentState
	summary      state.StatusSummary
	actionQueue  []state.ActionItem
	usageSummary usage.UsageSummary

	selectedIdx   int
	focusedAction string
	showDetail    bool
	showHelp      bool
	yoloEnabled   bool
	soundEnabled  bool

	stateManager *state.Manager
	rules        *rules.Engine
	notifier     *notify.Notifier
	navigator    *tmux.Navigator

	width, height int
}

func NewModel(
	stateManager *state.Manager,
	rulesEngine *rules.Engine,
	notifier *notify.Notifier,
	navigator *tmux.Navigator,
) Model {
	return Model{
		stateManager: stateManager,
		rules:        rulesEngine,
		notifier:     notifier,
		navigator:    navigator,
		soundEnabled: true,
		yoloEnabled:  rulesEngine.IsYOLO(),
	}
}

type ApprovalResultMsg struct {
	SessionID string
	Action    string
	Err       error
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case state.StateUpdateMsg:
		m.agents = msg.Agents
		m.summary = msg.Summary
		m.actionQueue = msg.Queue

		for _, a := range m.agents {
			m.notifier.OnTransition(a.SessionID, a.Status, a.ProjectName)
		}

		var cmds []tea.Cmd
		for _, item := range m.actionQueue {
			if item.Agent.Status != state.StatusWaitingPermission {
				continue
			}
			summary := ""
			if item.Agent.ToolRequestSummary != nil {
				summary = *item.Agent.ToolRequestSummary
			}
			if m.rules.Evaluate(item.Agent.LastTool, summary) == rules.Approve {
				agent := item.Agent
				cmds = append(cmds, func() tea.Msg {
					if !confirmStillWaiting(m.stateManager.StateDir(), agent.SessionID) {
						return nil
					}
					err := m.navigator.Approve(agent)
					return ApprovalResultMsg{SessionID: agent.SessionID, Action: "approved", Err: err}
				})
			}
		}
		if m.selectedIdx >= len(m.agents) {
			m.selectedIdx = maxInt(0, len(m.agents)-1)
		}
		return m, tea.Batch(cmds...)

	case usage.UsageUpdateMsg:
		m.usageSummary = msg.Summary

	case ApprovalResultMsg:
		// Could log to event log

	case tea.KeyMsg:
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Up):
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
		case key.Matches(msg, keys.Down):
			if m.selectedIdx < len(m.agents)-1 {
				m.selectedIdx++
			}
		case key.Matches(msg, keys.Jump):
			if m.selectedIdx < len(m.agents) {
				m.navigator.JumpToAgent(m.agents[m.selectedIdx])
			}
		case key.Matches(msg, keys.Approve):
			return m, m.approveAction()
		case key.Matches(msg, keys.Deny):
			return m, m.denyAction()
		case key.Matches(msg, keys.ApproveAll):
			return m, m.approveAllActions()
		case key.Matches(msg, keys.YOLO):
			m.yoloEnabled = !m.yoloEnabled
			m.rules.SetYOLO(m.yoloEnabled)
		case key.Matches(msg, keys.Sound):
			m.soundEnabled = !m.soundEnabled
			m.notifier.SetSound(m.soundEnabled)
		case key.Matches(msg, keys.Detail):
			m.showDetail = !m.showDetail
		case key.Matches(msg, keys.Help):
			m.showHelp = true
		default:
			if s := msg.String(); len(s) == 1 && s[0] >= 'a' && s[0] <= 'z' {
				m.focusedAction = s
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}
	if m.showHelp {
		return m.renderHelp()
	}

	header := renderHeader(m.summary, m.usageSummary, m.yoloEnabled, m.width)
	leftWidth := m.width * 60 / 100
	rightWidth := m.width - leftWidth - 3

	leftPanel := sectionTitleStyle.Render("AGENTS") + "\n" + renderAgentTable(m.agents, m.selectedIdx, leftWidth)

	rightPanel := sectionTitleStyle.Render("ACTIONS") + "\n" + renderActionQueue(m.actionQueue, m.focusedAction)
	if m.showDetail && m.selectedIdx < len(m.agents) {
		rightPanel = renderAgentDetail(m.agents[m.selectedIdx])
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(leftWidth).Render(leftPanel),
		lipgloss.NewStyle().Foreground(nordDimmed).Render(" │ "),
		lipgloss.NewStyle().Width(rightWidth).Render(rightPanel),
	)

	footer := renderFooter(m.yoloEnabled, m.soundEnabled)

	return strings.Join([]string{
		header,
		strings.Repeat("─", m.width),
		body,
		strings.Repeat("─", m.width),
		footer,
	}, "\n")
}

func (m Model) approveAction() tea.Cmd {
	item := m.findFocusedAction()
	if item == nil || item.Agent.Status != state.StatusWaitingPermission {
		return nil
	}
	agent := item.Agent
	stateDir := m.stateManager.StateDir()
	return func() tea.Msg {
		if !confirmStillWaiting(stateDir, agent.SessionID) {
			return nil
		}
		err := m.navigator.Approve(agent)
		return ApprovalResultMsg{SessionID: agent.SessionID, Action: "approved", Err: err}
	}
}

func (m Model) denyAction() tea.Cmd {
	item := m.findFocusedAction()
	if item == nil || item.Agent.Status != state.StatusWaitingPermission {
		return nil
	}
	agent := item.Agent
	return func() tea.Msg {
		err := m.navigator.Deny(agent)
		return ApprovalResultMsg{SessionID: agent.SessionID, Action: "denied", Err: err}
	}
}

func (m Model) approveAllActions() tea.Cmd {
	var cmds []tea.Cmd
	stateDir := m.stateManager.StateDir()
	for _, item := range m.actionQueue {
		if item.Agent.Status != state.StatusWaitingPermission {
			continue
		}
		agent := item.Agent
		cmds = append(cmds, func() tea.Msg {
			if !confirmStillWaiting(stateDir, agent.SessionID) {
				return nil
			}
			err := m.navigator.Approve(agent)
			return ApprovalResultMsg{SessionID: agent.SessionID, Action: "approved", Err: err}
		})
	}
	return tea.Batch(cmds...)
}

func (m Model) findFocusedAction() *state.ActionItem {
	if m.focusedAction == "" {
		if len(m.actionQueue) > 0 {
			return &m.actionQueue[0]
		}
		return nil
	}
	for i := range m.actionQueue {
		if m.actionQueue[i].Letter == m.focusedAction {
			return &m.actionQueue[i]
		}
	}
	return nil
}

func confirmStillWaiting(stateDir, sessionID string) bool {
	data, err := os.ReadFile(filepath.Join(stateDir, sessionID+".json"))
	if err != nil {
		return false
	}
	var agent state.AgentState
	if err := json.Unmarshal(data, &agent); err != nil {
		return false
	}
	return agent.Status == state.StatusWaitingPermission
}

func (m Model) renderHelp() string {
	return lipgloss.NewStyle().Foreground(nordFg).Padding(1, 2).Render(`
  CLORCH — Claude Code Session Orchestrator

  Navigation
    j/k, ↑/↓     Move selection in agent list
    Enter, →      Jump to selected agent's tmux pane
    d             Toggle agent detail panel

  Actions
    a-z           Focus action item by letter
    y             Approve focused permission
    n             Deny focused permission
    Y             Approve ALL pending permissions

  Settings
    !             Toggle YOLO mode (auto-approve)
    s             Toggle sound notifications

  General
    ?             Toggle this help
    q, Ctrl+C     Quit

  Press any key to close this help.
`)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
