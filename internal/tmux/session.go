package tmux

import (
	"os/exec"
	"strings"
)

func IsAvailable() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

func SendKeys(session, windowIndex, pane, keys string) error {
	target := session + ":" + windowIndex + "." + pane
	return exec.Command("tmux", "send-keys", "-t", target, keys, "Enter").Run()
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
