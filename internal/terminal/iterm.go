package terminal

import (
	"os/exec"
	"strings"
)

type ITermBackend struct{}

func (b *ITermBackend) GetTTYMap() (map[string]string, error) {
	script := `tell application "iTerm2"
	set output to ""
	repeat with w in windows
		repeat with t in tabs of w
			repeat with s in sessions of t
				set output to output & (tty of s) & "|||" & (id of w) & ":" & (index of t) & linefeed
			end repeat
		end repeat
	end repeat
	return output
end tell`
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return nil, err
	}
	return parseTTYMap(string(out)), nil
}

func (b *ITermBackend) ActivateTab(id string) error {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		return nil
	}
	script := `tell application "iTerm2"
	activate
	repeat with w in windows
		if (id of w) as text = "` + parts[0] + `" then
			set index of w to 1
			set current tab of w to tab ` + parts[1] + ` of w
		end if
	end repeat
end tell`
	return exec.Command("osascript", "-e", script).Run()
}

func (b *ITermBackend) BringToFront() error {
	return exec.Command("osascript", "-e", `tell application "iTerm2" to activate`).Run()
}

func (b *ITermBackend) CanResolveTabs() bool { return true }

func parseTTYMap(output string) map[string]string {
	m := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		parts := strings.SplitN(line, "|||", 2)
		if len(parts) == 2 {
			m[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return m
}
