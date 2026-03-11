package notify

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// NativeNotify sends an OS-native notification. Best-effort.
func NativeNotify(title, message string) {
	if runtime.GOOS == "darwin" {
		title = strings.ReplaceAll(title, `"`, `\"`)
		message = strings.ReplaceAll(message, `"`, `\"`)
		script := fmt.Sprintf(`display notification "%s" with title "%s"`, message, title)
		exec.Command("osascript", "-e", script).Start()
	}
}

// Notifier manages notification state and deduplication.
type Notifier struct {
	soundEnabled bool
	lastStatus   map[string]string
}

func NewNotifier() *Notifier {
	return &Notifier{
		soundEnabled: true,
		lastStatus:   make(map[string]string),
	}
}

func (n *Notifier) SetSound(enabled bool)  { n.soundEnabled = enabled }
func (n *Notifier) SoundEnabled() bool      { return n.soundEnabled }

// OnTransition fires notifications on status transitions. Returns true if fired.
func (n *Notifier) OnTransition(sessionID, newStatus, projectName string) bool {
	old := n.lastStatus[sessionID]
	n.lastStatus[sessionID] = newStatus
	if old == newStatus {
		return false
	}

	switch newStatus {
	case "WAITING_PERMISSION":
		Bell()
		NativeNotify("clorch — "+projectName, "Permission requested")
		if n.soundEnabled {
			PlaySound(SoundPermission)
		}
		return true
	case "WAITING_ANSWER":
		Bell()
		NativeNotify("clorch — "+projectName, "Question waiting")
		if n.soundEnabled {
			PlaySound(SoundQuestion)
		}
		return true
	case "ERROR":
		Bell()
		NativeNotify("clorch — "+projectName, "Error occurred")
		if n.soundEnabled {
			PlaySound(SoundError)
		}
		return true
	}
	return false
}
