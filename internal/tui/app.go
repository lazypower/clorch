package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lazypower/clorch/internal/branch"
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

	injectMode  bool
	injectInput textinput.Model

	branchMode    bool
	branchStep    int // 0 = path, 1 = label
	branchPath    string
	branchInput   textinput.Model

	labelMode  bool
	labelInput textinput.Model

	renameMode  bool
	renameInput textinput.Model

	version   string
	hooksStale bool

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
	version string,
	hooksStale bool,
) Model {
	ti := textinput.New()
	ti.Placeholder = "type message to inject..."
	ti.CharLimit = 500

	bi := textinput.New()
	bi.Placeholder = "path for branch working directory..."
	bi.CharLimit = 500

	li := textinput.New()
	li.Placeholder = "label (optional, Enter to skip)..."
	li.CharLimit = 100

	ri := textinput.New()
	ri.Placeholder = "new window name..."
	ri.CharLimit = 100

	return Model{
		injectInput:  ti,
		branchInput:  bi,
		labelInput:   li,
		renameInput:  ri,
		version:      version,
		hooksStale:   hooksStale,
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

type BranchResultMsg struct {
	Result branch.Result
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

		// Alert on stuck loops and compaction thresholds
		for _, a := range m.agents {
			if a.StuckLoop {
				m.notifier.OnTransition(a.SessionID, "STUCK_LOOP", a.ProjectName)
			}
			if a.CompactCount == 3 || a.CompactCount == 5 {
				m.notifier.OnTransition(a.SessionID, "COMPACT_WARN", a.ProjectName)
			}
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

	case BranchResultMsg:
		// Branch completed — new session will appear via hook discovery
		// Could log result or show error

	case tea.KeyMsg:
		if m.injectMode {
			switch msg.Type {
			case tea.KeyEsc:
				m.injectMode = false
				m.injectInput.SetValue("")
				m.injectInput.Blur()
				return m, nil
			case tea.KeyEnter:
				value := m.injectInput.Value()
				m.injectMode = false
				m.injectInput.SetValue("")
				m.injectInput.Blur()
				if value != "" && m.selectedIdx < len(m.agents) {
					agent := m.agents[m.selectedIdx]
					if agent.TmuxSession != "" {
						return m, func() tea.Msg {
							tmux.SendLiteral(agent.TmuxSession, agent.TmuxWindowIndex, agent.TmuxPane, value)
							tmux.SendKeys(agent.TmuxSession, agent.TmuxWindowIndex, agent.TmuxPane, "Enter")
							return nil
						}
					}
				}
				return m, nil
			default:
				var cmd tea.Cmd
				m.injectInput, cmd = m.injectInput.Update(msg)
				return m, cmd
			}
		}
		if m.branchMode {
			switch msg.Type {
			case tea.KeyEsc:
				m.branchMode = false
				m.branchStep = 0
				m.branchInput.SetValue("")
				m.branchInput.Blur()
				return m, nil
			case tea.KeyEnter:
				if m.branchStep == 0 {
					// Step 0: path confirmed, move to label
					m.branchPath = m.branchInput.Value()
					if m.branchPath == "" {
						m.branchMode = false
						return m, nil
					}
					m.branchStep = 1
					m.branchInput.SetValue("")
					m.branchInput.Placeholder = "label (optional, Enter to skip)..."
					return m, nil
				}
				// Step 1: label confirmed (may be empty), execute branch
				label := m.branchInput.Value()
				m.branchMode = false
				m.branchStep = 0
				m.branchInput.SetValue("")
				m.branchInput.Placeholder = "path for branch working directory..."
				m.branchInput.Blur()
				if m.selectedIdx < len(m.agents) {
					agent := m.agents[m.selectedIdx]
					targetDir := m.branchPath
					return m, func() tea.Msg {
						return BranchResultMsg{Result: branch.Branch(agent, targetDir, label)}
					}
				}
				return m, nil
			default:
				var cmd tea.Cmd
				m.branchInput, cmd = m.branchInput.Update(msg)
				return m, cmd
			}
		}
		if m.labelMode {
			switch msg.Type {
			case tea.KeyEsc:
				m.labelMode = false
				m.labelInput.SetValue("")
				m.labelInput.Blur()
				return m, nil
			case tea.KeyEnter:
				label := m.labelInput.Value()
				m.labelMode = false
				m.labelInput.SetValue("")
				m.labelInput.Blur()
				if m.selectedIdx < len(m.agents) {
					agent := m.agents[m.selectedIdx]
					return m, func() tea.Msg {
						setLabel(m.stateManager.StateDir(), agent.SessionID, label)
						return nil
					}
				}
				return m, nil
			default:
				var cmd tea.Cmd
				m.labelInput, cmd = m.labelInput.Update(msg)
				return m, cmd
			}
		}
		if m.renameMode {
			switch msg.Type {
			case tea.KeyEsc:
				m.renameMode = false
				m.renameInput.SetValue("")
				m.renameInput.Blur()
				return m, nil
			case tea.KeyEnter:
				newName := m.renameInput.Value()
				m.renameMode = false
				m.renameInput.SetValue("")
				m.renameInput.Blur()
				if newName != "" && m.selectedIdx < len(m.agents) {
					agent := m.agents[m.selectedIdx]
					if agent.TmuxSession != "" && agent.TmuxWindowIndex != "" {
						return m, func() tea.Msg {
							tmux.RenameWindow(agent.TmuxSession, agent.TmuxWindowIndex, newName)
							return nil
						}
					}
				}
				return m, nil
			default:
				var cmd tea.Cmd
				m.renameInput, cmd = m.renameInput.Update(msg)
				return m, cmd
			}
		}
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
		case key.Matches(msg, keys.Inject):
			m.injectMode = true
			m.injectInput.SetValue("")
			m.injectInput.Focus()
			return m, m.injectInput.Cursor.BlinkCmd()
		case key.Matches(msg, keys.Branch):
			if m.selectedIdx < len(m.agents) {
				m.branchMode = true
				m.branchInput.SetValue(branch.DefaultTargetDir(m.agents[m.selectedIdx]))
				m.branchInput.Focus()
				return m, m.branchInput.Cursor.BlinkCmd()
			}
		case key.Matches(msg, keys.Label):
			if m.selectedIdx < len(m.agents) {
				m.labelMode = true
				m.labelInput.SetValue("")
				m.labelInput.Focus()
				return m, m.labelInput.Cursor.BlinkCmd()
			}
		case key.Matches(msg, keys.Rename):
			if m.selectedIdx < len(m.agents) {
				agent := m.agents[m.selectedIdx]
				m.renameMode = true
				m.renameInput.SetValue(agent.TmuxWindow)
				m.renameInput.Focus()
				return m, m.renameInput.Cursor.BlinkCmd()
			}
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

	header := renderHeader(m.summary, m.usageSummary, m.yoloEnabled, m.version, m.width)
	leftWidth := m.width * 60 / 100
	rightWidth := m.width - leftWidth - 3

	sessionCosts := m.usageSummary.PerSession
	if sessionCosts == nil {
		sessionCosts = make(map[string]usage.SessionCost)
	}
	leftPanel := sectionTitleStyle.Render("AGENTS") + "\n" + renderAgentTable(m.agents, m.selectedIdx, leftWidth, sessionCosts)

	rightPanel := sectionTitleStyle.Render("ACTIONS") + "\n" + renderActionQueue(m.actionQueue, m.focusedAction)
	if m.showDetail && m.selectedIdx < len(m.agents) {
		agent := m.agents[m.selectedIdx]
		events := state.ReadEvents(m.stateManager.StateDir(), agent.SessionID, 200)
		// Panel height: total height minus header(1), two separators(2), footer(1)
		panelHeight := m.height - 4
		if panelHeight < 10 {
			panelHeight = 10
		}
		rightPanel = renderAgentDetail(agent, events, sessionCosts[agent.SessionID], panelHeight)
	}

	// Body height: total height minus header(1) + top separator(1) + bottom separator(1) + footer(1)
	bodyHeight := m.height - 4
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(leftWidth).Height(bodyHeight).Render(leftPanel),
		lipgloss.NewStyle().Foreground(nordDimmed).Height(bodyHeight).Render(" │ "),
		lipgloss.NewStyle().Width(rightWidth).Height(bodyHeight).Render(rightPanel),
	)

	var footer string
	if m.injectMode && m.selectedIdx < len(m.agents) {
		agentName := m.agents[m.selectedIdx].ProjectName
		if agentName == "" {
			agentName = m.agents[m.selectedIdx].SessionID
		}
		footer = footerStyle.Render("Inject to " + agentName + ": " + m.injectInput.View())
	} else if m.labelMode && m.selectedIdx < len(m.agents) {
		agentName := m.agents[m.selectedIdx].ProjectName
		if agentName == "" {
			agentName = m.agents[m.selectedIdx].SessionID
		}
		footer = footerStyle.Render("Label " + agentName + ": " + m.labelInput.View())
	} else if m.renameMode && m.selectedIdx < len(m.agents) {
		agentName := m.agents[m.selectedIdx].ProjectName
		if agentName == "" {
			agentName = m.agents[m.selectedIdx].SessionID
		}
		footer = footerStyle.Render("Rename " + agentName + " window: " + m.renameInput.View())
	} else if m.branchMode && m.selectedIdx < len(m.agents) {
		agentName := m.agents[m.selectedIdx].ProjectName
		if agentName == "" {
			agentName = m.agents[m.selectedIdx].SessionID
		}
		footer = footerStyle.Render("Branch " + agentName + " to: " + m.branchInput.View())
	} else {
		footer = renderFooter(m.yoloEnabled, m.soundEnabled, m.hooksStale)
	}

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
	if item == nil {
		return nil
	}
	agent := item.Agent
	return func() tea.Msg {
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
	for _, item := range m.actionQueue {
		agent := item.Agent
		cmds = append(cmds, func() tea.Msg {
			err := m.navigator.Approve(agent)
			return ApprovalResultMsg{SessionID: agent.SessionID, Action: "approved", Err: err}
		})
	}
	return tea.Batch(cmds...)
}

func (m Model) findFocusedAction() *state.ActionItem {
	if len(m.actionQueue) == 0 {
		return nil
	}
	if m.focusedAction != "" {
		for i := range m.actionQueue {
			if m.actionQueue[i].Letter == m.focusedAction {
				return &m.actionQueue[i]
			}
		}
	}
	// Fall back to first action if focused letter not found or empty
	return &m.actionQueue[0]
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
    i             Inject prompt to selected agent
    b             Branch session (git worktree + new tmux window)
    l             Set/change label for selected agent
    W             Rename tmux window for selected agent

  Settings
    !             Toggle YOLO mode (auto-approve)
    s             Toggle sound notifications

  General
    ?             Toggle this help
    q, Ctrl+C     Quit

  Press any key to close this help.
`)
}

func setLabel(stateDir, sessionID, label string) {
	path := filepath.Join(stateDir, sessionID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return
	}
	if label == "" {
		delete(raw, "branch_label")
	} else {
		raw["branch_label"] = label
	}
	out, err := json.Marshal(raw)
	if err != nil {
		return
	}
	tmpFile := path + ".tmp"
	if err := os.WriteFile(tmpFile, out, 0644); err != nil {
		return
	}
	os.Rename(tmpFile, path)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
