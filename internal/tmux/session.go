package tmux

import (
	"fmt"
	"os/exec"
	"strings"
)

func IsAvailable() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

// IsSafeTarget validates a tmux target component (session name, window index, etc.).
// Rejects empty strings and strings containing characters that could cause injection.
func IsSafeTarget(value string) bool {
	if value == "" {
		return false
	}
	for _, c := range value {
		switch c {
		case ':', '"', '\'', ';', '&', '|', '$', '`', '\\', '\n', '\r':
			return false
		}
	}
	return true
}

// SendKeys sends a keystroke sequence to a tmux pane.
// Each argument after the target is a separate key to send.
func SendKeys(session, windowIndex, pane string, keys ...string) error {
	if !IsSafeTarget(session) || !IsSafeTarget(windowIndex) {
		return fmt.Errorf("invalid tmux target: session=%q window=%q", session, windowIndex)
	}
	target := session + ":" + windowIndex + "." + pane
	args := append([]string{"send-keys", "-t", target}, keys...)
	return exec.Command("tmux", args...).Run()
}

// SendLiteral sends literal text to a tmux pane (using -l flag so spaces and
// special characters are not interpreted as key names).
func SendLiteral(session, windowIndex, pane string, text string) error {
	if !IsSafeTarget(session) || !IsSafeTarget(windowIndex) {
		return fmt.Errorf("invalid tmux target: session=%q window=%q", session, windowIndex)
	}
	target := session + ":" + windowIndex + "." + pane
	return exec.Command("tmux", "send-keys", "-t", target, "-l", text).Run()
}

func SelectPane(session, windowIndex, pane string) error {
	if !IsSafeTarget(session) || !IsSafeTarget(windowIndex) {
		return fmt.Errorf("invalid tmux target: session=%q window=%q", session, windowIndex)
	}
	target := session + ":" + windowIndex
	if err := exec.Command("tmux", "select-window", "-t", target).Run(); err != nil {
		return err
	}
	return exec.Command("tmux", "select-pane", "-t", target+"."+pane).Run()
}

// RenameWindow renames a tmux window.
func RenameWindow(session, windowTarget, newName string) error {
	if !IsSafeTarget(session) || !IsSafeTarget(windowTarget) {
		return fmt.Errorf("invalid tmux target: session=%q window=%q", session, windowTarget)
	}
	target := session + ":" + windowTarget
	return exec.Command("tmux", "rename-window", "-t", target, newName).Run()
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
