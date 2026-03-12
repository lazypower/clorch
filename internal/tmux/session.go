package tmux

import (
	"os/exec"
	"strings"
)

func IsAvailable() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

// SendKeys sends a keystroke sequence to a tmux pane.
// Each argument after the target is a separate key to send.
func SendKeys(session, windowIndex, pane string, keys ...string) error {
	target := session + ":" + windowIndex + "." + pane
	args := append([]string{"send-keys", "-t", target}, keys...)
	return exec.Command("tmux", args...).Run()
}

// SendLiteral sends literal text to a tmux pane (using -l flag so spaces and
// special characters are not interpreted as key names).
func SendLiteral(session, windowIndex, pane string, text string) error {
	target := session + ":" + windowIndex + "." + pane
	return exec.Command("tmux", "send-keys", "-t", target, "-l", text).Run()
}

func SelectPane(session, windowIndex, pane string) error {
	target := session + ":" + windowIndex
	if err := exec.Command("tmux", "select-window", "-t", target).Run(); err != nil {
		return err
	}
	return exec.Command("tmux", "select-pane", "-t", target+"."+pane).Run()
}

func ListPanes() ([]string, error) {
	out, err := exec.Command("tmux", "list-panes", "-a", "-F",
		"#{pane_tty}|||#{window_name}|||#{pane_index}|||#{session_name}|||#{window_index}").Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, nil
	}
	return lines, nil
}
