package config

import (
	"os"
	"path/filepath"
)

const (
	DefaultStateDir  = "/tmp/clorch/state"
	DefaultSession   = "claude"
	DefaultPollMS    = 500
	DefaultRulesPath = "~/.config/clorch/rules.yaml"
	defaultHooksDir  = "~/.local/share/clorch/hooks"
	settingsPath     = "~/.claude/settings.json"
	historyPath      = "~/.claude/history.jsonl"
	projectsDir      = "~/.claude/projects"
)

type Config struct {
	StateDir string
	Session  string
	PollMS   int
	RulesPath string
	HooksDir  string
	Terminal  string
}

func Load() *Config {
	return &Config{
		StateDir:  envOr("CLORCH_STATE_DIR", DefaultStateDir),
		Session:   envOr("CLORCH_SESSION", DefaultSession),
		PollMS:    envOrInt("CLORCH_POLL_MS", DefaultPollMS),
		RulesPath: expandHome(envOr("CLORCH_RULES", DefaultRulesPath)),
		HooksDir:  expandHome(defaultHooksDir),
		Terminal:  os.Getenv("CLORCH_TERMINAL"),
	}
}

func (c *Config) SettingsPath() string { return expandHome(settingsPath) }
func (c *Config) HistoryPath() string  { return expandHome(historyPath) }
func (c *Config) ProjectsDir() string  { return expandHome(projectsDir) }

func expandHome(path string) string {
	if len(path) > 1 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n := 0
	for _, c := range v {
		if c < '0' || c > '9' {
			return fallback
		}
		n = n*10 + int(c-'0')
	}
	return n
}
