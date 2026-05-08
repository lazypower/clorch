package tmux

import (
	"fmt"
	"os"
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

// CurrentSession returns the tmux session name for the current terminal.
// Returns ("", nil) if not running inside tmux.
func CurrentSession() (string, error) {
	if os.Getenv("TMUX") == "" {
		return "", nil
	}
	out, err := exec.Command("tmux", "display-message", "-p", "#{session_name}").Output()
	if err != nil {
		return "", fmt.Errorf("get tmux session: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// ListWindowNames returns the names of all windows in a tmux session.
func ListWindowNames(session string) ([]string, error) {
	if !IsSafeTarget(session) {
		return nil, fmt.Errorf("invalid tmux session: %q", session)
	}
	out, err := exec.Command("tmux", "list-windows", "-t", session, "-F", "#{window_name}").Output()
	if err != nil {
		return nil, fmt.Errorf("list windows: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, nil
	}
	return lines, nil
}

// SpawnWindow creates a new tmux window with 2 panes (vertical split).
// Pane 0 (top) runs the given command, pane 1 (bottom) is a shell.
// Both panes are cd'd to workDir.
func SpawnWindow(session, workDir, windowName, command string) error {
	if !IsSafeTarget(session) || !IsSafeTarget(windowName) {
		return fmt.Errorf("invalid tmux target: session=%q window=%q", session, windowName)
	}

	// Create window with pane 0 running the command
	cmd := exec.Command("tmux", "new-window", "-t", session+":",
		"-c", workDir, "-n", windowName, command)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("new-window: %w", err)
	}

	// Split to create pane 1 (shell) below
	target := session + ":" + windowName
	split := exec.Command("tmux", "split-window", "-t", target, "-c", workDir)
	split.Stderr = os.Stderr
	if err := split.Run(); err != nil {
		return fmt.Errorf("split-window: %w", err)
	}

	// Focus back on pane 0 (the command pane)
	return exec.Command("tmux", "select-pane", "-t", target+".0").Run()
}
