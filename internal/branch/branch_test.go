package branch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lazypower/clorch/internal/state"
)

func TestDefaultTargetDir(t *testing.T) {
	agent := state.AgentState{
		SessionID: "abcdef1234567890",
		CWD:       "/home/user/myproject",
	}
	got := DefaultTargetDir(agent)
	if !strings.HasPrefix(got, "/home/user/myproject/.clorch/branches/") {
		t.Errorf("DefaultTargetDir = %q, want prefix /home/user/myproject/.clorch/branches/", got)
	}
	suffix := filepath.Base(got)
	if len(suffix) != 6 {
		t.Errorf("expected 6-char suffix, got %q (%d chars)", suffix, len(suffix))
	}
}

func TestDefaultTargetDir_Unique(t *testing.T) {
	agent := state.AgentState{
		SessionID: "abc",
		CWD:       "/tmp/proj",
	}
	a := DefaultTargetDir(agent)
	b := DefaultTargetDir(agent)
	if a == b {
		t.Error("expected unique paths for repeated calls")
	}
}

func TestIsGitRepo(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, ".git"), 0755)
	if !isGitRepo(dir) {
		t.Error("expected git repo with .git directory")
	}

	noGit := t.TempDir()
	if isGitRepo(noGit) {
		t.Error("expected non-git directory")
	}
}

func TestIsGitRepo_WorktreeFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: /some/path"), 0644)
	if !isGitRepo(dir) {
		t.Error("expected git repo with .git file (worktree)")
	}
}

func TestBranch_NoTmuxSession(t *testing.T) {
	agent := state.AgentState{
		SessionID:   "test123",
		CWD:         "/tmp",
		TmuxSession: "",
	}
	result := Branch(agent, "/tmp/branch-target")
	if result.Err == nil {
		t.Error("expected error for agent with no tmux session")
	}
	if !strings.Contains(result.Err.Error(), "no tmux session") {
		t.Errorf("unexpected error: %v", result.Err)
	}
}

func TestSafeCopyDir(t *testing.T) {
	src := t.TempDir()
	os.WriteFile(filepath.Join(src, "test.txt"), []byte("hello"), 0644)
	os.Mkdir(filepath.Join(src, "subdir"), 0755)
	os.WriteFile(filepath.Join(src, "subdir", "nested.txt"), []byte("world"), 0644)

	dst := filepath.Join(t.TempDir(), "copy")
	if err := safeCopyDir(src, dst); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dst, "test.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Errorf("got %q, want %q", string(data), "hello")
	}

	data, err = os.ReadFile(filepath.Join(dst, "subdir", "nested.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "world" {
		t.Errorf("got %q, want %q", string(data), "world")
	}
}

func TestDirSize(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("world!"), 0644)

	size, err := dirSize(dir)
	if err != nil {
		t.Fatal(err)
	}
	if size != 11 {
		t.Errorf("expected size 11, got %d", size)
	}
}

func TestShortID(t *testing.T) {
	id := shortID()
	if len(id) != 6 {
		t.Errorf("expected 6 chars, got %d", len(id))
	}
	// Should be unique
	id2 := shortID()
	if id == id2 {
		t.Error("expected unique IDs")
	}
}
