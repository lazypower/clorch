package terminal

import "os/exec"

type GhosttyBackend struct{}

func (b *GhosttyBackend) GetTTYMap() (map[string]string, error) { return nil, nil }
func (b *GhosttyBackend) ActivateTab(id string) error           { return nil }
func (b *GhosttyBackend) BringToFront() error {
	return exec.Command("osascript", "-e", `tell application "Ghostty" to activate`).Run()
}
func (b *GhosttyBackend) CanResolveTabs() bool { return false }
