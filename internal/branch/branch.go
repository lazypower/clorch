package branch

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lazypower/clorch/internal/state"
)

// Max directory size (in bytes) we'll copy without git. Prevents accidentally
// cloning $HOME or other massive directories. 500MB.
const maxCopySize = 500 * 1024 * 1024

// Result holds the outcome of a branch operation.
type Result struct {
	WorkDir string
	Err     error
}

// Branch creates a new working directory (git worktree or directory copy) and
// spawns a new Claude Code session in a tmux window. Clorch never holds the
// PID — tmux owns the process, and clorch discovers it via hooks.
func Branch(agent state.AgentState, targetDir string) Result {
	if agent.TmuxSession == "" {
		return Result{Err: fmt.Errorf("agent has no tmux session")}
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(targetDir), 0755); err != nil {
		return Result{Err: fmt.Errorf("create parent dir: %w", err)}
	}

	// Guard against branching from dangerous directories
	if isDangerousDir(agent.CWD) {
		return Result{Err: fmt.Errorf("refusing to branch from %s — too broad, use a project directory", agent.CWD)}
	}

	// Create the working directory
	if isGitRepo(agent.CWD) {
		if err := createWorktree(agent.CWD, targetDir); err != nil {
			return Result{Err: fmt.Errorf("git worktree: %w", err)}
		}
	} else {
		if err := safeCopyDir(agent.CWD, targetDir); err != nil {
			return Result{Err: fmt.Errorf("copy dir: %w", err)}
		}
	}

	// Spawn claude in a new tmux window — tmux owns the process
	if err := spawnInTmux(agent.TmuxSession, targetDir); err != nil {
		return Result{Err: fmt.Errorf("tmux spawn: %w", err)}
	}

	return Result{WorkDir: targetDir}
}

// DefaultTargetDir returns the default branch directory for an agent.
// Uses a random suffix so multiple branches from the same session don't collide.
func DefaultTargetDir(agent state.AgentState) string {
	return filepath.Join(agent.CWD, ".clorch", "branches", shortID())
}

func isGitRepo(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	// .git can be a file (worktree) or directory
	return info != nil
}

func createWorktree(repoDir, targetDir string) error {
	// Find the git toplevel in case CWD is a subdirectory
	out, err := exec.Command("git", "-C", repoDir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return fmt.Errorf("find git root: %w", err)
	}
	gitRoot := strings.TrimSpace(string(out))

	branchName := fmt.Sprintf("clorch-branch-%s", shortID())
	cmd := exec.Command("git", "-C", gitRoot, "worktree", "add", targetDir, "-b", branchName)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// safeCopyDir copies a directory but refuses if it's suspiciously large.
func safeCopyDir(src, dst string) error {
	size, err := dirSize(src)
	if err != nil {
		return fmt.Errorf("check dir size: %w", err)
	}
	if size > maxCopySize {
		return fmt.Errorf("directory too large to copy (%d MB) — use a git repo for branching", size/(1024*1024))
	}
	cmd := exec.Command("cp", "-r", src, dst)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func dirSize(path string) (int64, error) {
	var total int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if !info.IsDir() {
			total += info.Size()
		}
		// Bail early if already over the limit
		if total > maxCopySize {
			return fmt.Errorf("exceeded limit")
		}
		return nil
	})
	if err != nil && total <= maxCopySize {
		return total, err
	}
	return total, nil
}

func isDangerousDir(dir string) bool {
	home, _ := os.UserHomeDir()
	dangerous := []string{"/", "/tmp", "/var", home}
	for _, d := range dangerous {
		if d != "" && dir == d {
			return true
		}
	}
	return false
}

func spawnInTmux(tmuxSession, workDir string) error {
	// tmux new-window runs the command inside tmux — tmux owns the process.
	// When clorch exits, this window keeps running.
	// Trailing colon tells tmux "this session, auto-assign window index"
	cmd := exec.Command("tmux", "new-window", "-t", tmuxSession+":",
		"-c", workDir,
		"claude")
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func shortID() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = chars[r.Intn(len(chars))]
	}
	return string(b)
}
