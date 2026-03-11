package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEmptyEngine_ReturnsAsk(t *testing.T) {
	e := &Engine{}
	got := e.Evaluate("bash", "echo hello")
	if got != Ask {
		t.Fatalf("expected Ask, got %d", got)
	}
}

func TestYOLO_NoRules_ReturnsApprove(t *testing.T) {
	e := &Engine{config: Config{YOLO: true}}
	got := e.Evaluate("bash", "rm -rf /")
	if got != Approve {
		t.Fatalf("expected Approve, got %d", got)
	}
}

func TestToolMatching_ExactMatch(t *testing.T) {
	e := &Engine{config: Config{
		Rules: []Rule{
			{Tools: []string{"bash"}, Action: "approve"},
		},
	}}
	got := e.Evaluate("bash", "ls")
	if got != Approve {
		t.Fatalf("expected Approve, got %d", got)
	}
}

func TestToolMatching_NoMatch(t *testing.T) {
	e := &Engine{config: Config{
		Rules: []Rule{
			{Tools: []string{"bash"}, Action: "approve"},
		},
	}}
	got := e.Evaluate("write", "something")
	if got != Ask {
		t.Fatalf("expected Ask (no matching tool), got %d", got)
	}
}

func TestPatternMatching_ContainsMatch(t *testing.T) {
	e := &Engine{config: Config{
		Rules: []Rule{
			{Tools: []string{"bash"}, Pattern: "git push", Action: "deny"},
		},
	}}
	got := e.Evaluate("bash", "running git push --force")
	if got != Deny {
		t.Fatalf("expected Deny, got %d", got)
	}
}

func TestPatternMatching_NoMatch(t *testing.T) {
	e := &Engine{config: Config{
		Rules: []Rule{
			{Tools: []string{"bash"}, Pattern: "git push", Action: "deny"},
		},
	}}
	got := e.Evaluate("bash", "git status")
	if got != Ask {
		t.Fatalf("expected Ask (pattern not matched), got %d", got)
	}
}

func TestFirstMatchWins(t *testing.T) {
	e := &Engine{config: Config{
		Rules: []Rule{
			{Tools: []string{"bash"}, Action: "deny"},
			{Tools: []string{"bash"}, Action: "approve"},
		},
	}}
	got := e.Evaluate("bash", "ls")
	if got != Deny {
		t.Fatalf("expected Deny (first rule wins), got %d", got)
	}
}

func TestFirstMatchWins_PatternNarrowBeforeBroad(t *testing.T) {
	e := &Engine{config: Config{
		Rules: []Rule{
			{Tools: []string{"bash"}, Pattern: "rm", Action: "deny"},
			{Tools: []string{"bash"}, Action: "approve"},
		},
	}}

	got := e.Evaluate("bash", "rm -rf /")
	if got != Deny {
		t.Fatalf("expected Deny for rm command, got %d", got)
	}

	got = e.Evaluate("bash", "ls -la")
	if got != Approve {
		t.Fatalf("expected Approve for ls command, got %d", got)
	}
}

func TestDenyRule(t *testing.T) {
	e := &Engine{config: Config{
		Rules: []Rule{
			{Tools: []string{"bash"}, Pattern: "DROP TABLE", Action: "deny"},
		},
	}}
	got := e.Evaluate("bash", "DROP TABLE users")
	if got != Deny {
		t.Fatalf("expected Deny, got %d", got)
	}
}

func TestEmptyToolsList_MatchesEverything(t *testing.T) {
	e := &Engine{config: Config{
		Rules: []Rule{
			{Tools: []string{}, Action: "approve"},
		},
	}}

	for _, tool := range []string{"bash", "write", "read", "anything"} {
		got := e.Evaluate(tool, "some summary")
		if got != Approve {
			t.Fatalf("expected Approve for tool %q with empty tools list, got %d", tool, got)
		}
	}
}

func TestSetYOLO_IsYOLO(t *testing.T) {
	e := &Engine{}
	if e.IsYOLO() {
		t.Fatal("expected IsYOLO false initially")
	}

	e.SetYOLO(true)
	if !e.IsYOLO() {
		t.Fatal("expected IsYOLO true after SetYOLO(true)")
	}

	e.SetYOLO(false)
	if e.IsYOLO() {
		t.Fatal("expected IsYOLO false after SetYOLO(false)")
	}
}

func TestNewEngine_NonexistentFile_ReturnsEmptyEngine(t *testing.T) {
	e, err := NewEngine("/nonexistent/path/to/rules.yaml")
	if err != nil {
		t.Fatalf("unexpected error for missing file: %v", err)
	}
	got := e.Evaluate("bash", "ls")
	if got != Ask {
		t.Fatalf("expected Ask from empty engine, got %d", got)
	}
}

func TestNewEngine_LoadFromYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yaml")

	content := `yolo: false
rules:
  - tools: ["bash"]
    pattern: "rm"
    action: deny
  - tools: ["bash"]
    action: approve
  - tools: ["write"]
    action: approve
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp rules file: %v", err)
	}

	e, err := NewEngine(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		name     string
		tool     string
		summary  string
		expected Action
	}{
		{"deny bash rm", "bash", "rm -rf /tmp/stuff", Deny},
		{"approve bash non-rm", "bash", "ls -la", Approve},
		{"approve write", "write", "writing a file", Approve},
		{"ask unknown tool", "unknown", "something", Ask},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := e.Evaluate(tt.tool, tt.summary)
			if got != tt.expected {
				t.Errorf("Evaluate(%q, %q) = %d, want %d", tt.tool, tt.summary, got, tt.expected)
			}
		})
	}
}

func TestNewEngine_YOLOFromYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yaml")

	content := `yolo: true
rules: []
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp rules file: %v", err)
	}

	e, err := NewEngine(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !e.IsYOLO() {
		t.Fatal("expected IsYOLO true from YAML")
	}

	got := e.Evaluate("anything", "whatever")
	if got != Approve {
		t.Fatalf("expected Approve in YOLO mode, got %d", got)
	}
}

func TestNewEngine_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")

	if err := os.WriteFile(path, []byte("{{{{not yaml"), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err := NewEngine(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestToolMatches_Helpers(t *testing.T) {
	if !toolMatches(nil, "anything") {
		t.Fatal("nil tools should match everything")
	}
	if !toolMatches([]string{}, "anything") {
		t.Fatal("empty tools should match everything")
	}
	if !toolMatches([]string{"a", "b"}, "b") {
		t.Fatal("should match 'b' in list")
	}
	if toolMatches([]string{"a", "b"}, "c") {
		t.Fatal("should not match 'c'")
	}
}

func TestEvaluate_UnknownAction_FallsThrough(t *testing.T) {
	e := &Engine{config: Config{
		Rules: []Rule{
			{Tools: []string{"bash"}, Action: "unknown_action"},
		},
	}}
	got := e.Evaluate("bash", "ls")
	if got != Ask {
		t.Fatalf("expected Ask for unrecognized action string, got %d", got)
	}
}
