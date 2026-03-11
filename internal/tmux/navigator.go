package tmux

import "github.com/lazypower/clorch/internal/state"

type Navigator struct {
	attentionIdx int
}

func NewNavigator() *Navigator {
	return &Navigator{}
}

func (n *Navigator) JumpToAgent(agent state.AgentState) error {
	if agent.TmuxSession == "" || agent.TmuxWindowIndex == "" {
		return nil
	}
	return SelectPane(agent.TmuxSession, agent.TmuxWindowIndex, agent.TmuxPane)
}

func (n *Navigator) Approve(agent state.AgentState) error {
	if agent.TmuxSession == "" {
		return nil
	}
	return SendKeys(agent.TmuxSession, agent.TmuxWindowIndex, agent.TmuxPane, "y")
}

func (n *Navigator) Deny(agent state.AgentState) error {
	if agent.TmuxSession == "" {
		return nil
	}
	return SendKeys(agent.TmuxSession, agent.TmuxWindowIndex, agent.TmuxPane, "n")
}

func (n *Navigator) JumpToNextAttention(agents []state.AgentState) error {
	var attention []state.AgentState
	for _, a := range agents {
		if a.Status == state.StatusWaitingPermission || a.Status == state.StatusWaitingAnswer {
			attention = append(attention, a)
		}
	}
	if len(attention) == 0 {
		return nil
	}
	if n.attentionIdx >= len(attention) {
		n.attentionIdx = 0
	}
	agent := attention[n.attentionIdx]
	n.attentionIdx++
	return n.JumpToAgent(agent)
}
