package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// markerSubstrings are used to identify clorch hook entries in settings.json.
// We check for both the install path marker and the script filenames.
var markerSubstrings = []string{"clorch/hooks/", "/event_handler.sh", "/notify_handler.sh"}

var eventHandlerEvents = []string{
	"SessionStart", "PreToolUse", "PostToolUse", "PostToolUseFailure",
	"Stop", "SessionEnd", "PermissionRequest", "UserPromptSubmit",
	"SubagentStart", "SubagentStop", "PreCompact", "TeammateIdle", "TaskCompleted",
}

type TemplateData struct {
	StateDir string
	HooksDir string
	Version  string
	Event    string
}

func Install(stateDir, hooksDir, settingsPath, version string, dryRun bool) error {
	hooksDir = expandHome(hooksDir)
	settingsPath = expandHome(settingsPath)

	if !dryRun {
		if err := os.MkdirAll(hooksDir, 0755); err != nil {
			return fmt.Errorf("creating hooks dir: %w", err)
		}
		if err := os.MkdirAll(stateDir, 0755); err != nil {
			return fmt.Errorf("creating state dir: %w", err)
		}
	}

	data := TemplateData{StateDir: stateDir, HooksDir: hooksDir, Version: version}

	if !dryRun {
		if err := renderScript(eventHandlerTmpl, filepath.Join(hooksDir, "event_handler.sh"), data); err != nil {
			return fmt.Errorf("rendering event_handler.sh: %w", err)
		}
		if err := renderScript(notifyHandlerTmpl, filepath.Join(hooksDir, "notify_handler.sh"), data); err != nil {
			return fmt.Errorf("rendering notify_handler.sh: %w", err)
		}
	}

	settings, err := readSettings(settingsPath)
	if err != nil {
		return err
	}

	if !dryRun {
		if err := backupSettings(settingsPath); err != nil {
			return err
		}
	}

	hooks := buildHookEntries(hooksDir)
	existingHooks, _ := settings["hooks"].(map[string]interface{})
	if existingHooks == nil {
		existingHooks = make(map[string]interface{})
	}
	for event, entry := range hooks {
		mergeEventHooks(existingHooks, event, entry)
	}
	settings["hooks"] = existingHooks

	if dryRun {
		out, _ := json.MarshalIndent(settings, "", "  ")
		fmt.Println(string(out))
		return nil
	}
	return writeSettings(settingsPath, settings)
}

func Uninstall(hooksDir, settingsPath string) error {
	hooksDir = expandHome(hooksDir)
	settingsPath = expandHome(settingsPath)

	settings, err := readSettings(settingsPath)
	if err != nil {
		return err
	}
	if err := backupSettings(settingsPath); err != nil {
		return err
	}

	existingHooks, _ := settings["hooks"].(map[string]interface{})
	if existingHooks != nil {
		for event, val := range existingHooks {
			groups := toInterfaceSlice(val)
			if groups == nil {
				continue
			}
			var filtered []interface{}
			for _, g := range groups {
				group, ok := g.(map[string]interface{})
				if !ok {
					filtered = append(filtered, g)
					continue
				}
				if !groupContainsMarker(group) {
					filtered = append(filtered, g)
				}
			}
			if len(filtered) == 0 {
				delete(existingHooks, event)
			} else {
				existingHooks[event] = filtered
			}
		}
		if len(existingHooks) == 0 {
			delete(settings, "hooks")
		}
	}

	if err := writeSettings(settingsPath, settings); err != nil {
		return err
	}
	os.Remove(filepath.Join(hooksDir, "event_handler.sh"))
	os.Remove(filepath.Join(hooksDir, "notify_handler.sh"))
	os.RemoveAll(hooksDir)
	return nil
}

func renderScript(tmplStr, path string, data TemplateData) error {
	t, err := template.New("hook").Parse(tmplStr)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := t.Execute(f, data); err != nil {
		return err
	}
	return os.Chmod(path, 0755)
}

func buildHookEntries(hooksDir string) map[string]map[string]interface{} {
	entries := make(map[string]map[string]interface{})
	eventScript := filepath.Join(hooksDir, "event_handler.sh")
	for _, event := range eventHandlerEvents {
		entries[event] = map[string]interface{}{
			"matcher": "",
			"hooks": []map[string]interface{}{
				{"type": "command", "command": fmt.Sprintf("CLORCH_EVENT=%s %s", event, eventScript), "async": true},
			},
		}
	}
	entries["Notification"] = map[string]interface{}{
		"matcher": "",
		"hooks": []map[string]interface{}{
			{"type": "command", "command": filepath.Join(hooksDir, "notify_handler.sh"), "async": true},
		},
	}
	return entries
}

func mergeEventHooks(existing map[string]interface{}, event string, newEntry map[string]interface{}) {
	val, exists := existing[event]
	if !exists {
		existing[event] = []interface{}{newEntry}
		return
	}

	// Normalize to []interface{} — handles both typed and untyped slices after JSON round-trip
	groups := toInterfaceSlice(val)
	if groups == nil {
		existing[event] = []interface{}{newEntry}
		return
	}

	for i, g := range groups {
		group, ok := g.(map[string]interface{})
		if !ok {
			continue
		}
		if groupContainsMarker(group) {
			groups[i] = newEntry
			existing[event] = groups
			return
		}
	}
	existing[event] = append(groups, newEntry)
}

// toInterfaceSlice converts any slice type to []interface{}.
func toInterfaceSlice(v interface{}) []interface{} {
	if s, ok := v.([]interface{}); ok {
		return s
	}
	// Handle []map[string]interface{} from buildHookEntries
	if s, ok := v.([]map[string]interface{}); ok {
		out := make([]interface{}, len(s))
		for i, m := range s {
			out[i] = m
		}
		return out
	}
	return nil
}

func groupContainsMarker(group map[string]interface{}) bool {
	hooksVal, exists := group["hooks"]
	if !exists {
		return false
	}
	hooks := toInterfaceSlice(hooksVal)
	if hooks == nil {
		return false
	}
	for _, h := range hooks {
		hook, ok := h.(map[string]interface{})
		if !ok {
			continue
		}
		cmd, _ := hook["command"].(string)
		for _, marker := range markerSubstrings {
			if strings.Contains(cmd, marker) {
				return true
			}
		}
	}
	return false
}

func readSettings(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]interface{}), nil
		}
		return nil, err
	}
	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parsing settings: %w", err)
	}
	return settings, nil
}

func writeSettings(path string, settings map[string]interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func backupSettings(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.WriteFile(fmt.Sprintf("%s.bak.%d", path, time.Now().Unix()), data, 0644)
}

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
