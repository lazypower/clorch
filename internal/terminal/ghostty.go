package terminal

import (
	"os/exec"
	"strings"
	"time"
)

type GhosttyBackend struct{}

func (b *GhosttyBackend) GetTTYMap() (map[string]string, error) { return nil, nil }
func (b *GhosttyBackend) ActivateTab(id string) error           { return nil }
func (b *GhosttyBackend) BringToFront() error {
	return exec.Command("osascript", "-e", `tell application "Ghostty" to activate`).Run()
}
func (b *GhosttyBackend) CanResolveTabs() bool { return false }

// OpenTab opens a new Ghostty tab and runs a command using clipboard-based paste.
// This avoids keystroke encoding issues by using the system clipboard.
func (b *GhosttyBackend) OpenTab(command string) error {
	// Save current clipboard
	savedClip, _ := exec.Command("pbpaste").Output()

	// Set clipboard to command
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(command)
	if err := cmd.Run(); err != nil {
		return err
	}

	// Activate Ghostty, open new tab, paste, enter
	script := `
tell application "Ghostty" to activate
delay 0.3
tell application "System Events"
	keystroke "t" using command down
	delay 0.3
	keystroke "v" using command down
	delay 0.1
	key code 36
end tell`
	if err := exec.Command("osascript", "-e", script).Run(); err != nil {
		// Restore clipboard before returning error
		restoreClipboard(savedClip)
		return err
	}

	// Restore clipboard after a brief delay
	time.AfterFunc(500*time.Millisecond, func() {
		restoreClipboard(savedClip)
	})

	return nil
}

func restoreClipboard(data []byte) {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(string(data))
	cmd.Run()
}
