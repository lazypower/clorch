package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up         key.Binding
	Down       key.Binding
	Jump       key.Binding
	Approve    key.Binding
	Deny       key.Binding
	ApproveAll key.Binding
	YOLO       key.Binding
	Sound      key.Binding
	Detail     key.Binding
	Inject     key.Binding
	Branch       key.Binding
	Help       key.Binding
	Quit       key.Binding
}

var keys = keyMap{
	Up:         key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "up")),
	Down:       key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "down")),
	Jump:       key.NewBinding(key.WithKeys("enter", "right"), key.WithHelp("→/enter", "jump to agent")),
	Approve:    key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "approve")),
	Deny:       key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "deny")),
	ApproveAll: key.NewBinding(key.WithKeys("Y"), key.WithHelp("Y", "approve all")),
	YOLO:       key.NewBinding(key.WithKeys("!"), key.WithHelp("!", "toggle YOLO")),
	Sound:      key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "toggle sound")),
	Detail:     key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "detail panel")),
	Inject:     key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "inject prompt")),
	Branch:     key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "branch session")),
	Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}
