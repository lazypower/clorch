# clorch — Claude Code Session Orchestrator

## Quick Reference

- **Language:** Go
- **Build:** `devbox run -- go build -o bin/clorch ./cmd/clorch` or `make build`
- **Test:** `devbox run -- go test ./...`
- **Lint:** `devbox run -- go vet ./...`
- **Module:** `github.com/lazypower/clorch`

## Architecture

Hook-based observer pattern. Claude Code hooks write JSON state files to `/tmp/clorch/state/`. The TUI reads them via fsnotify. No API calls, no terminal scraping.

```
hooks (bash) → state dir (JSON files) → fsnotify watcher → Bubble Tea TUI
```

## Package Map

| Package | Role |
|---|---|
| `cmd/clorch` | Entry point |
| `internal/cli` | Cobra command tree |
| `internal/config` | Paths, env vars, defaults |
| `internal/hooks` | Hook installer + embedded bash templates |
| `internal/state` | Models, manager (scan/cleanup), watcher (fsnotify), history resolver |
| `internal/rules` | YAML rules engine (first-match-wins) |
| `internal/notify` | Bell, macOS sound, native notifications |
| `internal/tmux` | send-keys, navigation, status widget |
| `internal/terminal` | iTerm2/Ghostty/Apple Terminal backends via osascript |
| `internal/usage` | JSONL transcript parser, cost calculator, burn rate tracker |
| `internal/tui` | Bubble Tea model, Nord-themed views |

## Key Conventions

- State files are the sole IPC between hooks and the TUI
- All hook scripts are `async: true` — they never block Claude Code
- Approval mechanism: `tmux send-keys` into the agent's pane
- Version injected via `-ldflags "-X main.version=$(VERSION)"`
- Hook scripts are Go `text/template` rendered at `clorch init` time
