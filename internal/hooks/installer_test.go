package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallFreshSettings(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	hooksDir := filepath.Join(dir, "hooks")
	stateDir := filepath.Join(dir, "state")

	err := Install(stateDir, hooksDir, settingsPath, "test", false)
	if err != nil {
		t.Fatal(err)
	}

	// Verify settings written
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatal(err)
	}
	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("expected hooks key in settings")
	}

	// Check all event types are registered
	expectedEvents := append(eventHandlerEvents, "Notification")
	for _, event := range expectedEvents {
		if _, ok := hooks[event]; !ok {
			t.Errorf("missing hook for event %s", event)
		}
	}

	// Verify scripts exist and are executable
	for _, name := range []string{"event_handler.sh", "notify_handler.sh"} {
		path := filepath.Join(hooksDir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("script %s not found: %v", name, err)
			continue
		}
		if info.Mode()&0111 == 0 {
			t.Errorf("script %s is not executable", name)
		}
	}
}

func TestInstallPreservesExistingHooks(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	hooksDir := filepath.Join(dir, "hooks")
	stateDir := filepath.Join(dir, "state")

	// Write existing settings with a custom hook
	existing := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "/usr/local/bin/my-custom-hook",
						},
					},
				},
			},
		},
		"other_setting": true,
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	os.WriteFile(settingsPath, data, 0644)

	err := Install(stateDir, hooksDir, settingsPath, "test", false)
	if err != nil {
		t.Fatal(err)
	}

	// Read back
	data, _ = os.ReadFile(settingsPath)
	var settings map[string]interface{}
	json.Unmarshal(data, &settings)

	// other_setting should be preserved
	if settings["other_setting"] != true {
		t.Error("existing settings key was lost")
	}

	// PreToolUse should have both the custom hook and clorch hook
	hooks := settings["hooks"].(map[string]interface{})
	preToolUse := hooks["PreToolUse"].([]interface{})
	if len(preToolUse) != 2 {
		t.Errorf("expected 2 PreToolUse entries (custom + clorch), got %d", len(preToolUse))
	}
}

func TestInstallIdempotent(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	hooksDir := filepath.Join(dir, "hooks")
	stateDir := filepath.Join(dir, "state")

	// Install twice
	Install(stateDir, hooksDir, settingsPath, "v1", false)
	Install(stateDir, hooksDir, settingsPath, "v2", false)

	data, _ := os.ReadFile(settingsPath)
	var settings map[string]interface{}
	json.Unmarshal(data, &settings)

	hooks := settings["hooks"].(map[string]interface{})
	preToolUse := hooks["PreToolUse"].([]interface{})
	if len(preToolUse) != 1 {
		t.Errorf("expected 1 PreToolUse entry after double install, got %d", len(preToolUse))
	}
}

func TestInstallDryRun(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	hooksDir := filepath.Join(dir, "hooks")
	stateDir := filepath.Join(dir, "state")

	err := Install(stateDir, hooksDir, settingsPath, "test", true)
	if err != nil {
		t.Fatal(err)
	}

	// Settings file should NOT be written
	if _, err := os.Stat(settingsPath); !os.IsNotExist(err) {
		t.Error("dry-run should not write settings file")
	}
	// Hook scripts should NOT be written
	if _, err := os.Stat(filepath.Join(hooksDir, "event_handler.sh")); !os.IsNotExist(err) {
		t.Error("dry-run should not write hook scripts")
	}
}

func TestInstallCreatesBackup(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	hooksDir := filepath.Join(dir, "hooks")
	stateDir := filepath.Join(dir, "state")

	os.WriteFile(settingsPath, []byte(`{"existing": true}`), 0644)

	Install(stateDir, hooksDir, settingsPath, "test", false)

	// Should have a .bak file
	matches, _ := filepath.Glob(settingsPath + ".bak.*")
	if len(matches) == 0 {
		t.Error("expected backup file to be created")
	}
}

func TestUninstall(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	hooksDir := filepath.Join(dir, "hooks")
	stateDir := filepath.Join(dir, "state")

	// Install first
	Install(stateDir, hooksDir, settingsPath, "test", false)

	// Then uninstall
	err := Uninstall(hooksDir, settingsPath)
	if err != nil {
		t.Fatal(err)
	}

	// Settings should not have hooks
	data, _ := os.ReadFile(settingsPath)
	var settings map[string]interface{}
	json.Unmarshal(data, &settings)
	if _, ok := settings["hooks"]; ok {
		t.Error("hooks should be removed after uninstall")
	}

	// Scripts should be gone
	if _, err := os.Stat(filepath.Join(hooksDir, "event_handler.sh")); !os.IsNotExist(err) {
		t.Error("event_handler.sh should be deleted")
	}
}

func TestUninstallPreservesOtherHooks(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	hooksDir := filepath.Join(dir, "hooks")
	stateDir := filepath.Join(dir, "state")

	// Install clorch hooks
	Install(stateDir, hooksDir, settingsPath, "test", false)

	// Add a custom hook to PreToolUse
	data, _ := os.ReadFile(settingsPath)
	var settings map[string]interface{}
	json.Unmarshal(data, &settings)

	hooks := settings["hooks"].(map[string]interface{})
	preToolUse := hooks["PreToolUse"].([]interface{})
	preToolUse = append(preToolUse, map[string]interface{}{
		"matcher": "",
		"hooks": []interface{}{
			map[string]interface{}{
				"type":    "command",
				"command": "/usr/local/bin/custom-hook",
			},
		},
	})
	hooks["PreToolUse"] = preToolUse
	data, _ = json.MarshalIndent(settings, "", "  ")
	os.WriteFile(settingsPath, data, 0644)

	// Uninstall
	Uninstall(hooksDir, settingsPath)

	// Custom hook should survive
	data, _ = os.ReadFile(settingsPath)
	json.Unmarshal(data, &settings)
	hooks = settings["hooks"].(map[string]interface{})
	preToolUse = hooks["PreToolUse"].([]interface{})
	if len(preToolUse) != 1 {
		t.Errorf("expected 1 remaining hook, got %d", len(preToolUse))
	}
}

func TestGroupContainsMarker(t *testing.T) {
	tests := []struct {
		name  string
		group map[string]interface{}
		want  bool
	}{
		{
			"clorch hook",
			map[string]interface{}{
				"hooks": []interface{}{
					map[string]interface{}{
						"command": "CLORCH_EVENT=PreToolUse /home/user/.local/share/clorch/hooks/event_handler.sh",
					},
				},
			},
			true,
		},
		{
			"other hook",
			map[string]interface{}{
				"hooks": []interface{}{
					map[string]interface{}{
						"command": "/usr/local/bin/other-tool",
					},
				},
			},
			false,
		},
		{
			"no hooks key",
			map[string]interface{}{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := groupContainsMarker(tt.group); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
