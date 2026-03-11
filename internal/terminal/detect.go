package terminal

import "os"

func Detect() Backend {
	if forced := os.Getenv("CLORCH_TERMINAL"); forced != "" {
		return backendForName(forced)
	}
	switch os.Getenv("TERM_PROGRAM") {
	case "iTerm.app", "iTerm2":
		return &ITermBackend{}
	case "ghostty":
		return &GhosttyBackend{}
	case "Apple_Terminal":
		return &AppleTerminalBackend{}
	}
	return nil
}

func backendForName(name string) Backend {
	switch name {
	case "iterm":
		return &ITermBackend{}
	case "ghostty":
		return &GhosttyBackend{}
	case "apple_terminal":
		return &AppleTerminalBackend{}
	default:
		return nil
	}
}
