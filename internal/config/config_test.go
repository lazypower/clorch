package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// Clear env vars that would override
	os.Unsetenv("CLORCH_STATE_DIR")
	os.Unsetenv("CLORCH_SESSION")
	os.Unsetenv("CLORCH_POLL_MS")
	os.Unsetenv("CLORCH_RULES")
	os.Unsetenv("CLORCH_TERMINAL")

	cfg := Load()
	if cfg.StateDir != DefaultStateDir {
		t.Errorf("StateDir = %q, want %q", cfg.StateDir, DefaultStateDir)
	}
	if cfg.Session != DefaultSession {
		t.Errorf("Session = %q, want %q", cfg.Session, DefaultSession)
	}
	if cfg.PollMS != DefaultPollMS {
		t.Errorf("PollMS = %d, want %d", cfg.PollMS, DefaultPollMS)
	}
	if cfg.Terminal != "" {
		t.Errorf("Terminal = %q, want empty", cfg.Terminal)
	}
}

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("CLORCH_STATE_DIR", "/custom/state")
	os.Setenv("CLORCH_SESSION", "my-session")
	os.Setenv("CLORCH_POLL_MS", "1000")
	os.Setenv("CLORCH_TERMINAL", "iterm")
	defer func() {
		os.Unsetenv("CLORCH_STATE_DIR")
		os.Unsetenv("CLORCH_SESSION")
		os.Unsetenv("CLORCH_POLL_MS")
		os.Unsetenv("CLORCH_TERMINAL")
	}()

	cfg := Load()
	if cfg.StateDir != "/custom/state" {
		t.Errorf("StateDir = %q, want /custom/state", cfg.StateDir)
	}
	if cfg.Session != "my-session" {
		t.Errorf("Session = %q, want my-session", cfg.Session)
	}
	if cfg.PollMS != 1000 {
		t.Errorf("PollMS = %d, want 1000", cfg.PollMS)
	}
	if cfg.Terminal != "iterm" {
		t.Errorf("Terminal = %q, want iterm", cfg.Terminal)
	}
}

func TestPollMSInvalidFallback(t *testing.T) {
	os.Setenv("CLORCH_POLL_MS", "notanumber")
	defer os.Unsetenv("CLORCH_POLL_MS")

	cfg := Load()
	if cfg.PollMS != DefaultPollMS {
		t.Errorf("PollMS = %d, want %d (fallback)", cfg.PollMS, DefaultPollMS)
	}
}

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()
	got := expandHome("~/test/path")
	if !strings.HasPrefix(got, home) {
		t.Errorf("expandHome(~/test/path) = %q, want prefix %q", got, home)
	}
	if !strings.HasSuffix(got, "test/path") {
		t.Errorf("expandHome(~/test/path) = %q, want suffix test/path", got)
	}

	// Absolute path should pass through
	got = expandHome("/absolute/path")
	if got != "/absolute/path" {
		t.Errorf("expandHome(/absolute/path) = %q", got)
	}
}

func TestSettingsPath(t *testing.T) {
	os.Unsetenv("CLORCH_STATE_DIR")
	cfg := Load()
	p := cfg.SettingsPath()
	if !strings.Contains(p, ".claude/settings.json") {
		t.Errorf("SettingsPath() = %q, want to contain .claude/settings.json", p)
	}
}

func TestProjectsDir(t *testing.T) {
	cfg := Load()
	p := cfg.ProjectsDir()
	if !strings.Contains(p, ".claude/projects") {
		t.Errorf("ProjectsDir() = %q, want to contain .claude/projects", p)
	}
}
