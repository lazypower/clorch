package terminal

import (
	"os/exec"
	"strings"
)

type AppleTerminalBackend struct{}

func (b *AppleTerminalBackend) GetTTYMap() (map[string]string, error) {
	script := `tell application "Terminal"
	set output to ""
	repeat with w in windows
		repeat with t in tabs of w
			set output to output & (tty of t) & "|||" & (id of w) & ":" & (index of t) & linefeed
		end repeat
	end repeat
	return output
end tell`
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return nil, err
	}
	return parseAppleTTYMap(string(out)), nil
}

func (b *AppleTerminalBackend) ActivateTab(id string) error {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		return nil
	}
	script := `tell application "Terminal"
	activate
	set index of window id ` + parts[0] + ` to 1
	set selected of tab ` + parts[1] + ` of window id ` + parts[0] + ` to true
end tell`
	return exec.Command("osascript", "-e", script).Run()
}

func (b *AppleTerminalBackend) BringToFront() error {
	return exec.Command("osascript", "-e", `tell application "Terminal" to activate`).Run()
}

func (b *AppleTerminalBackend) CanResolveTabs() bool { return true }

func parseAppleTTYMap(output string) map[string]string {
	m := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		parts := strings.SplitN(line, "|||", 2)
		if len(parts) == 2 {
			m[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return m
}
