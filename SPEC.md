# clorch-go — Claude Code Session Orchestrator

**Status:** Ready
**Created:** 2026-03-09
**Language:** Go
**License:** MIT
**Prior Art:** [androsovm/clorch](https://github.com/androsovm/clorch) (Python/Textual)

---

## 1. Problem Statement

Running multiple Claude Code sessions in parallel is increasingly common — split a spec into sub-tasks, launch 5-10 agents across tmux windows, and let them work. The bottleneck isn't the agents. It's the human. You're alt-tabbing between terminals, hunting for permission prompts buried in scroll-back, and losing track of which agent is idle vs. blocked.

The Python clorch solved this with a Textual TUI. It works. But it carries Python as a runtime dependency, pulls in a TUI framework with its own event loop, and requires `pip install` in a world where the rest of the toolchain (`claude`, `tmux`, `jq`) is native binaries. A Go rewrite eliminates the runtime dependency, ships as a single static binary, and opens the door to tighter integration with the agentic runtime platform.

## 2. Goals

1. **Single static binary.** `go install` or download. No Python, no pip, no virtualenv.
2. **Hook-based state ingestion.** Read Claude Code hook events via the same file-based protocol — shell scripts write JSON to a state directory, the dashboard reads it. No terminal scraping, no API calls.
3. **TUI dashboard.** Bubble Tea-based interface showing all active sessions, their status, pending permissions, and git context. Real-time updates via filesystem polling.
4. **Permission management.** Approve/deny tool requests from the dashboard. Batch approve. YOLO mode with configurable deny rules.
5. **tmux integration.** Jump to any agent's tmux pane with a keystroke. tmux status-bar widget for at-a-glance counts.
6. **Multi-terminal support.** Navigate to agent sessions in iTerm2, Ghostty, and Apple Terminal via native APIs (osascript/AppleScript).
7. **Notifications.** macOS native notifications, terminal bell, system sounds. Configurable per-event.
8. **Usage tracking.** Parse Claude Code session logs for token usage, calculate cost by model, show burn rate.
9. **Rules engine.** YAML-based auto-approve/deny rules. First-match-wins. Deny rules override YOLO.
10. **Hook installer.** `clorch init` installs hooks into `~/.claude/settings.json` non-destructively (backup + merge).

## 3. Non-Goals

- Launching or managing Claude Code processes. Clorch observes — it doesn't spawn agents. You start agents yourself in tmux; clorch discovers them via hooks.
- Web UI or remote access. Terminal only. If you want remote, run clorch inside tmux and SSH in.
- Multi-user. Single operator, single machine.
- Custom TUI theming. Ship one theme (Nord-derived). If someone wants to change colors, they can fork.
- Windows support. macOS and Linux only.

## 4. Architecture

### 4.1 System Overview

```
┌──────────────────────────────────────────────────────────────┐
│  clorch (single binary)                                      │
│                                                              │
│  ┌───────────┐  ┌──────────────┐  ┌───────────────────────┐  │
│  │  TUI      │  │  State       │  │  Hook Installer       │  │
│  │  (Bubble  │←→│  Manager     │  │  (init / uninstall)   │  │
│  │   Tea)    │  │              │  └───────────────────────┘  │
│  └─────┬─────┘  └──────┬───────┘                             │
│        │               │                                     │
│        │        ┌──────┴───────┐                             │
│        │        │  State Dir   │  /tmp/clorch/state/*.json   │
│        │        │  (filesystem │←── Hook scripts write here  │
│        │        │   polling)   │                             │
│        │        └──────────────┘                             │
│        │                                                     │
│  ┌─────┴─────────────────────────────────────────────────┐   │
│  │  Subsystems                                           │   │
│  │                                                       │   │
│  │  ┌─────────────┐ ┌────────────┐ ┌──────────────────┐  │   │
│  │  │ Rules       │ │ Notifier   │ │ Usage Tracker    │  │   │
│  │  │ Engine      │ │            │ │                  │  │   │
│  │  │ (YAML)      │ │ - bell     │ │ - JSONL parser   │  │   │
│  │  │ - approve   │ │ - sound    │ │ - cost calc      │  │   │
│  │  │ - deny      │ │ - macOS    │ │ - burn rate      │  │   │
│  │  │ - yolo      │ │   notify   │ │                  │  │   │
│  │  └─────────────┘ └────────────┘ └──────────────────┘  │   │
│  │                                                       │   │
│  │  ┌─────────────┐ ┌────────────────────────────────┐   │   │
│  │  │ tmux        │ │ Terminal Backends              │   │   │
│  │  │             │ │                                │   │   │
│  │  │ - navigate  │ │ - iTerm2    (osascript)        │   │   │
│  │  │ - send-keys │ │ - Ghostty   (osascript)        │   │   │
│  │  │ - widget    │ │ - Apple Terminal (osascript)    │   │   │
│  │  │ - session   │ │ - detect (TERM_PROGRAM)        │   │   │
│  │  └─────────────┘ └────────────────────────────────┘   │   │
│  └───────────────────────────────────────────────────────┘   │
│                                                              │
└──────────────────────────────────────────────────────────────┘

External data flow:

  Claude Code hooks (event_handler.sh, notify_handler.sh)
    → atomic JSON write to /tmp/clorch/state/<session_id>.json
      → clorch watches state dir via fsnotify (fallback: 500ms poll)
        → TUI updates, notifications fire, rules evaluate
```

### 4.2 State Directory Protocol

The state directory is the sole communication channel between Claude Code and clorch. Hook scripts are the writers; clorch is the reader.

**State file:** `/tmp/clorch/state/<session_id>.json`

```json
{
  "session_id": "abc123",
  "status": "WORKING",
  "cwd": "/home/user/myproject",
  "project_name": "myproject",
  "model": "claude-opus-4-6",
  "last_event": "PreToolUse",
  "last_event_time": "2026-03-09T10:05:30Z",
  "last_tool": "Bash",
  "notification_message": null,
  "tool_request_summary": null,
  "started_at": "2026-03-09T10:00:00Z",
  "tool_count": 47,
  "error_count": 1,
  "subagent_count": 2,
  "compact_count": 0,
  "last_compact_time": "",
  "task_completed_count": 0,
  "activity_history": [0, 0, 2, 5, 3, 1, 0, 4, 2, 1],
  "pid": 54321,
  "git_branch": "feat/new-thing",
  "git_dirty_count": 3,
  "tmux_window": "agent-0",
  "tmux_pane": "0",
  "tmux_session": "claude",
  "tmux_window_index": "3",
  "term_program": "iTerm.app"
}
```

**Full field reference:**

| Field | Type | Source | Description |
|---|---|---|---|
| `session_id` | string | stdin JSON | Claude Code session UUID |
| `status` | string | derived | `IDLE`, `WORKING`, `WAITING_PERMISSION`, `WAITING_ANSWER`, `ERROR` |
| `cwd` | string | stdin JSON | Agent's working directory |
| `project_name` | string | derived | `basename(cwd)` |
| `model` | string | stdin JSON | Model ID from `SessionStart` event |
| `last_event` | string | hook | Name of the most recent hook event |
| `last_event_time` | string | hook | ISO 8601 UTC timestamp of last event |
| `last_tool` | string | stdin JSON | Last tool name seen in `PreToolUse`/`PermissionRequest` |
| `notification_message` | string? | stdin JSON | Message from `Notification` event, cleared on next tool use |
| `tool_request_summary` | string? | derived | Human-readable summary of pending tool request (see §4.3) |
| `started_at` | string | hook | When this session's state file was first created |
| `tool_count` | int | derived | Running count of `PreToolUse` events |
| `error_count` | int | derived | Running count of `PostToolUseFailure` events |
| `subagent_count` | int | derived | Net active subagents (`SubagentStart` increments, `SubagentStop` decrements) |
| `compact_count` | int | derived | Number of `PreCompact` events (context compactions) |
| `last_compact_time` | string | hook | Timestamp of last compaction |
| `task_completed_count` | int | derived | Number of `TaskCompleted` events |
| `activity_history` | int[10] | derived | Sliding window of tool counts per interval (sparkline data) |
| `pid` | int | `$PPID` | Claude Code process PID (parent of hook script) |
| `git_branch` | string | derived | Current git branch in `cwd` |
| `git_dirty_count` | int | derived | `git status --porcelain | wc -l` in `cwd` |
| `tmux_window` | string | derived | tmux window name (matched via TTY) |
| `tmux_pane` | string | derived | tmux pane index within window |
| `tmux_session` | string | derived | tmux session name |
| `tmux_window_index` | string | derived | tmux window index number |
| `term_program` | string | `$TERM_PROGRAM` | Terminal emulator identifier |

**Statuses:** `IDLE`, `WORKING`, `WAITING_PERMISSION`, `WAITING_ANSWER`, `ERROR` (uppercase, matching upstream)

**Atomicity:** Hook scripts write to a temp file (`mktemp` in state dir) and `mv` into place. No partial reads.

**Cleanup:** clorch removes state files via three-pass maintenance:
1. **Dead process removal:** If `pid` is set, check `kill(pid, 0)`. If `ProcessLookupError` → remove file.
2. **Time-based removal:** If no `pid`, remove if `last_event_time` is older than 1 hour.
3. **PID deduplication:** When multiple state files share the same PID (session restart), keep the one with highest `tool_count`, remove the rest.

**Stale permission reset:** If a state file has `WAITING_PERMISSION` status but the PID is dead, reset to `IDLE` and clear `tool_request_summary`. This handles the case where Claude Code doesn't fire a `Stop` event after permission denial.

### 4.3 Claude Code Hook Contract

Clorch hooks integrate with Claude Code's native hook system. Full documentation: [Claude Code Hooks](https://docs.anthropic.com/en/docs/claude-code/hooks).

**Calling convention:** Claude Code invokes hook commands as shell processes. Input is JSON on **stdin**. Exit code determines behavior (0 = success, 2 = block, other = non-blocking error). Stdout is parsed as JSON on exit 0.

**Common stdin fields (all events):**

```json
{
  "session_id": "abc123",
  "transcript_path": "/Users/.../.claude/projects/<hash>/<session>.jsonl",
  "cwd": "/Users/.../my-project",
  "permission_mode": "default",
  "hook_event_name": "PreToolUse"
}
```

**Event-specific additional fields:**

| Event | Additional stdin fields |
|---|---|
| `SessionStart` | `source` ("startup"\|"resume"\|"clear"\|"compact"), `model` |
| `PreToolUse` | `tool_name`, `tool_input` (object), `tool_use_id` |
| `PostToolUse` | `tool_name`, `tool_input`, `tool_use_id`, `tool_response` |
| `PostToolUseFailure` | `tool_name`, `tool_input`, `tool_use_id`, `error` (string), `is_interrupt` (bool) |
| `PermissionRequest` | `tool_name`, `tool_input`, `tool_use_id`, `permission_suggestions` |
| `Notification` | `message`, `title` (optional), `notification_type` ("permission_prompt"\|"idle_prompt"\|"auth_success"\|"elicitation_dialog") |
| `Stop` | `stop_hook_active`, `last_assistant_message` |
| `SubagentStart` | `agent_id`, `agent_type` |
| `SubagentStop` | `agent_id`, `agent_type`, `agent_transcript_path` |
| `TaskCompleted` | `task_id`, `task_subject` |
| `PreCompact` | `trigger` ("manual"\|"auto") |
| `SessionEnd` | `reason` |
| `UserPromptSubmit` | `prompt` |

**Event type passing:** The hook command itself doesn't receive the event type as an argument. Upstream solves this by prefixing the command with an env var: `CLORCH_EVENT=PreToolUse /path/to/event_handler.sh`. The Go templates should generate commands in this format.

### 4.4 Hook Scripts

Two bash scripts, generated from Go `text/template` templates embedded in the binary. Regenerated on every `clorch init`.

**Template variables available to `.sh.tmpl` files:**

```go
type HookTemplateData struct {
    StateDir    string // e.g. "/tmp/clorch/state"
    HooksDir    string // e.g. "~/.local/share/clorch/hooks"
    Version     string // clorch version string
    Event       string // event name for CLORCH_EVENT prefix
}
```

**`event_handler.sh.tmpl`** — Registered for: `SessionStart`, `PreToolUse`, `PostToolUse`, `PostToolUseFailure`, `Stop`, `SessionEnd`, `PermissionRequest`, `UserPromptSubmit`, `SubagentStart`, `SubagentStop`, `PreCompact`, `TeammateIdle`, `TaskCompleted`.

Event handler logic per event type:

| Event | State mutation |
|---|---|
| `SessionStart` | Create new state file. Set status=`IDLE`, extract `cwd`, `model`, `project_name`. Detect tmux pane, git context. |
| `PreToolUse` | Set status=`WORKING`, `last_tool`=tool_name, increment `tool_count`, shift `activity_history`. Clear `notification_message` and `tool_request_summary`. |
| `PostToolUse` | Set status=`WORKING`. Clear `notification_message` and `tool_request_summary`. |
| `PostToolUseFailure` | Set status=`ERROR`. Increment `error_count`. |
| `PermissionRequest` | Set status=`WAITING_PERMISSION`, `last_tool`=tool_name. Build `tool_request_summary` from `tool_input` (see below). |
| `Stop` | Set status=`IDLE` **unless** current status is `WAITING_ANSWER` (preserve it — Stop fires before the user answers). |
| `SessionEnd` | **Delete** the state file. |
| `UserPromptSubmit` | Set status=`WORKING`. Clear `notification_message` and `tool_request_summary`. |
| `SubagentStart` | Increment `subagent_count`. |
| `SubagentStop` | Decrement `subagent_count` (floor 0). |
| `PreCompact` | Increment `compact_count`, set `last_compact_time`. Fire notification. |
| `TaskCompleted` | Increment `task_completed_count`. |
| `TeammateIdle` | Update `last_event`/`last_event_time` only. |

**Tool request summary construction** (for `PermissionRequest`): Build a human-readable summary from `tool_input` based on tool type:

| Tool | Summary format |
|---|---|
| `Bash` | `$ <command>` (truncated to 300 chars) |
| `Edit` | `<file_path>` + first 3 lines of old/new with `-`/`+` prefixes |
| `Write` | `<file_path> (N lines)` + first 3 lines of content |
| `Read` | `<file_path>` |
| `WebFetch` | `<url>` |
| `Grep` | `<pattern> in <path>` |
| `Glob` | `<pattern> in <path>` |
| `Task` (Agent) | `[<subagent_type>] <description>` |
| Other | JSON string of `tool_input` truncated to 300 chars |

All summaries capped at 500 chars.

**tmux pane detection:** The hook script finds the Claude Code process's TTY via `ps -p $PPID -o tty=`, then matches it against `tmux list-panes -a -F '#{pane_tty}|||#{window_name}|||#{pane_index}|||#{session_name}|||#{window_index}'`. This handles the case where a tmux server is running but the agent is in a native terminal tab (no match = no tmux fields set).

**VS Code detection:** When `TERM_PROGRAM` is unset, check for `VSCODE_PID` or `VSCODE_IPC_HOOK_CLI` env vars, or grep the parent process command for `.vscode/extensions/`.

**`notify_handler.sh.tmpl`** — Registered for `Notification` events only.

Determines status from the `message` field via keyword matching:
- Contains "permission" (case-insensitive) → `WAITING_PERMISSION`
- Contains "question", "input", "answer", or "elicitation" → `WAITING_ANSWER`
- Otherwise → no status change, just update `notification_message`

Also fires terminal bell (`\a`) and macOS native notification via `osascript`.

**All hooks are registered with `"async": true`** so they don't block Claude Code's execution. Exit code doesn't matter for async hooks.

Scripts are installed to `~/.local/share/clorch/hooks/` and referenced by absolute path in `~/.claude/settings.json`.

### 4.4 Approval Mechanism

When the TUI detects a `waiting_permission` state, it can send approval/denial by writing keystrokes into the agent's tmux pane:

```
tmux send-keys -t <pane> "y" Enter    # approve
tmux send-keys -t <pane> "n" Enter    # deny
```

This is the same mechanism the Python clorch uses. It works because Claude Code's permission prompt reads from stdin.

## 5. Package Layout

```
clorch/
├── cmd/
│   └── clorch/
│       └── main.go              # entry point
├── internal/
│   ├── cli/
│   │   └── cli.go               # cobra command tree
│   ├── config/
│   │   └── config.go            # paths, env vars, defaults
│   ├── hooks/
│   │   ├── installer.go         # install/uninstall hooks
│   │   ├── templates.go         # go:embed for .sh.tmpl templates
│   │   ├── event_handler.sh.tmpl  # hook script template
│   │   └── notify_handler.sh.tmpl # hook script template
│   ├── state/
│   │   ├── models.go            # AgentState, ActionItem, StatusSummary
│   │   ├── manager.go           # scan state dir, enrich, dedup, cleanup
│   │   ├── watcher.go           # fsnotify watcher, fallback to polling
│   │   └── history.go           # resolve session names from history.jsonl
│   ├── rules/
│   │   └── rules.go             # YAML rule loading, first-match evaluation
│   ├── notify/
│   │   ├── bell.go              # terminal bell + tmux bell
│   │   ├── sound.go             # macOS afplay system sounds
│   │   └── macos.go             # osascript native notifications
│   ├── tmux/
│   │   ├── navigator.go         # jump to agent pane, cycle attention
│   │   ├── session.go           # session/window/pane management, send-keys
│   │   └── statusbar.go         # tmux status-right widget output
│   ├── terminal/
│   │   ├── backend.go           # TerminalBackend interface
│   │   ├── detect.go            # auto-detect from TERM_PROGRAM
│   │   ├── iterm.go             # iTerm2 via osascript
│   │   ├── ghostty.go           # Ghostty via osascript
│   │   └── apple.go             # Apple Terminal via osascript
│   ├── usage/
│   │   ├── models.go            # TokenUsage, SessionUsage, UsageSummary
│   │   ├── parser.go            # incremental JSONL parser with byte offset
│   │   ├── pricing.go           # per-model cost calculation
│   │   └── tracker.go           # rolling window burn rate
│   └── tui/
│       ├── app.go               # root Bubble Tea model
│       ├── keys.go              # key bindings
│       ├── styles.go            # lipgloss styles, Nord palette
│       ├── agent_table.go       # agent list view
│       ├── action_queue.go      # pending permissions/questions
│       ├── agent_detail.go      # expanded single-agent view
│       ├── header.go            # title bar with session counts
│       ├── footer.go            # context bar with keybinding hints
│       ├── event_log.go         # scrollable event history
│       └── settings.go          # settings overlay (sound, yolo toggle)
├── go.mod
├── go.sum
└── Makefile
```

## 6. Dependencies

| Dependency | Purpose |
|---|---|
| `github.com/charmbracelet/bubbletea` | TUI framework |
| `github.com/charmbracelet/lipgloss` | TUI styling |
| `github.com/charmbracelet/bubbles` | TUI components (table, viewport, help) |
| `github.com/spf13/cobra` | CLI command parsing |
| `gopkg.in/yaml.v3` | Rules file parsing |
| `github.com/fsnotify/fsnotify` | Filesystem event watching for state dir |
| stdlib `os`, `os/exec`, `encoding/json`, `text/template`, `path/filepath`, `time` | Everything else |

Seven dependencies. The full `go.sum` should be auditable in one sitting.

## 7. CLI Interface

```
clorch                  # launch TUI dashboard (default)
clorch init             # install hooks into ~/.claude/settings.json
clorch init --dry-run   # preview hook changes without writing
clorch uninstall        # remove hooks from settings
clorch status           # one-line summary for scripting (e.g., "3 working, 1 waiting")
clorch list             # table view of all agents (non-interactive)
clorch tmux-widget      # output for tmux status-right integration
clorch version          # print version and exit
```

### 7.1 Environment Variables

| Variable | Default | Purpose |
|---|---|---|
| `CLORCH_STATE_DIR` | `/tmp/clorch/state` | State file directory |
| `CLORCH_SESSION` | `claude` | tmux session name to manage |
| `CLORCH_TERMINAL` | auto-detect | Force terminal backend: `iterm`, `ghostty`, `apple_terminal` |
| `CLORCH_POLL_MS` | `500` | Fallback poll interval when fsnotify unavailable |
| `CLORCH_RULES` | `~/.config/clorch/rules.yaml` | Path to rules file |

## 8. TUI Design

### 8.1 Layout

```
┌─────────────────────────────────────────────────────────────┐
│  CLORCH  ▪ 4 agents  ▪ 2 working  1 idle  1 waiting  │ $23 │
├────────────────────────────────────┬────────────────────────┤
│  AGENTS                            │  ACTIONS               │
│                                    │                        │
│  ● agent-0  working   feat/auth    │  a) agent-2: Bash      │
│    /home/user/backend  2s ago      │     rm -rf node_mod... │
│                                    │                        │
│  ● agent-1  idle      main         │  b) agent-3: question  │
│    /home/user/frontend  45s ago    │     Which test frmwk?  │
│                                    │                        │
│  ◉ agent-2  WAITING   feat/api     │                        │
│    /home/user/backend  8s ago      │                        │
│                                    │                        │
│  ◉ agent-3  QUESTION  main         │                        │
│    /home/user/docs  12s ago        │                        │
│                                    │                        │
├────────────────────────────────────┴────────────────────────┤
│  j/k:navigate  →:jump  y/n:approve  Y:all  !:yolo  ?:help  │
└─────────────────────────────────────────────────────────────┘
```

### 8.2 Key Bindings

| Key | Action |
|---|---|
| `j` / `k` | Move selection up/down in agent list |
| `Enter` / `→` | Jump to selected agent's tmux pane |
| `a`-`z` | Focus the corresponding action item |
| `y` | Approve focused permission |
| `n` | Deny focused permission |
| `Y` | Approve all pending permissions |
| `!` | Toggle YOLO mode |
| `s` | Toggle sound notifications |
| `d` | Toggle agent detail panel |
| `?` | Help overlay |
| `q` | Quit |

### 8.3 Staleness Indicators

Agents that haven't updated recently get visual warnings:
- **> 30s idle:** Yellow indicator
- **> 120s idle:** Red indicator

This catches stuck agents or sessions where the hook failed to fire.

## 9. Rules Engine

`~/.config/clorch/rules.yaml`:

```yaml
yolo: false

rules:
  - tools: [Read, Glob, Grep]
    action: approve

  - tools: [Bash]
    pattern: "rm -rf"
    action: deny

  - tools: [Bash]
    pattern: "git push --force"
    action: deny

  - tools: [Edit, Write]
    action: approve
```

**Evaluation logic:**
1. Walk rules top-to-bottom. First match wins.
2. If no rule matches and YOLO is on → approve.
3. If no rule matches and YOLO is off → wait for human.
4. Deny rules always require manual review, even in YOLO mode.

**Implementation:** `rules.Evaluate(toolName string, summary string) Action` returns `Approve`, `Deny`, or `Ask`.

Pattern matching uses `strings.Contains` on the tool request summary. No regex — keep it simple, keep it fast.

## 10. Usage Tracking

Parses Claude Code's JSONL session transcripts for token usage data.

### 10.1 Session Transcript Format

Transcripts live at `~/.claude/projects/<project-hash>/<session-uuid>.jsonl`. Each line is a JSON object. The format is **not formally documented by Anthropic** — this is derived from observation and the upstream parser.

**Assistant message record (the only type we parse for usage):**

```json
{
  "type": "assistant",
  "parentUuid": "...",
  "isSidechain": false,
  "cwd": "/Users/.../project",
  "sessionId": "abc123",
  "version": "1.0.33",
  "gitBranch": "main",
  "timestamp": "2026-03-09T10:05:30.123Z",
  "message": {
    "id": "msg_...",
    "type": "message",
    "role": "assistant",
    "model": "claude-opus-4-6-20260301",
    "content": [...],
    "usage": {
      "input_tokens": 12500,
      "output_tokens": 3200,
      "cache_creation_input_tokens": 0,
      "cache_read_input_tokens": 8000
    }
  }
}
```

**Parsing strategy:**
- Fast pre-filter: skip lines that don't contain `"assistant"` (string match before JSON parse).
- Navigate to `.message.role == "assistant"` → extract `.message.usage` and `.message.model`.
- Track four token counters: `input_tokens`, `output_tokens`, `cache_creation_input_tokens`, `cache_read_input_tokens`.

**Other record types** (ignored by usage parser):
- `type: "user"` — user messages
- `type: "file-history-snapshot"` — file state snapshots
- `type: "custom-title"` — session rename via `/rename` command (used by history resolver, see §10.4)

> **AMBIGUITY FLAG:** The transcript JSONL schema is undocumented. Field names, nesting, and record types may change across Claude Code versions. The parser should be defensive — skip records that don't match expected structure rather than erroring.

### 10.2 Parser Implementation

- **File discovery:** Scan `~/.claude/projects/*/` for `*.jsonl` files modified today (compare `st_mtime` against midnight local time).
- **Incremental reads:** Track byte offset per file path. On each poll, `Seek` to stored offset, read new lines only. Return new offset = `file.Tell()` after reading.
- **Full rescan:** Every 60 seconds, reset all offsets and re-discover files. Catches rotated/new files.
- **Token aggregation:** Sum across all files into `TokenUsage{InputTokens, OutputTokens, CacheCreationTokens, CacheReadTokens}`.

### 10.3 Cost Model

```go
var Pricing = map[string]ModelPrice{
    "opus-4-6":   {Input: 15.0, Output: 75.0},  // per 1M tokens
    "opus-4-5":   {Input: 15.0, Output: 75.0},
    "sonnet-4-6": {Input: 3.0,  Output: 15.0},
    "haiku-4-5":  {Input: 0.80, Output: 4.0},
}
```

Model name resolved from the full model ID (e.g. `claude-opus-4-6-20260301`) via prefix matching — strip the date suffix, match against known keys.

### 10.4 Burn Rate

Rolling 10-minute window. Calculate tokens consumed in the window, extrapolate to $/hour. Displayed in the TUI header.

### 10.5 History Resolution

Session display names are resolved from two sources (highest priority first):

1. **Custom title:** Scan transcript `<session_id>.jsonl` for records with `{"type": "custom-title", "customTitle": "..."}`. These are written when the user runs `/rename` in Claude Code.

2. **History file:** `~/.claude/history.jsonl` contains one JSON object per line:
   ```json
   {"sessionId": "abc123", "display": "fix the auth bug", ...}
   ```
   The `display` field is typically the first user prompt. Use the first occurrence per session ID.

Both sources are mtime-cached — only re-read when the file changes on disk.

## 11. Hook Installation

### 11.1 settings.json Hook Schema

Claude Code hooks are defined in `~/.claude/settings.json` under the `hooks` key. Each event maps to an array of matcher groups:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "CLORCH_EVENT=PreToolUse ~/.local/share/clorch/hooks/event_handler.sh",
            "async": true
          }
        ]
      }
    ],
    "Notification": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "~/.local/share/clorch/hooks/notify_handler.sh",
            "async": true
          }
        ]
      }
    ]
  }
}
```

**Matcher group structure:**
- `matcher`: regex string to filter (empty string = match all). For `PreToolUse`/`PostToolUse`, matches against tool name. For `Notification`, matches against `notification_type`.
- `hooks`: array of hook handler objects.

**Hook handler fields:**
- `type`: `"command"` (the only type clorch uses)
- `command`: shell command string. Receives JSON on stdin.
- `async`: `true` — run in background, don't block Claude Code.
- `timeout`: optional, defaults to 600s for command type.

Clorch registers one matcher group per event, with `matcher: ""` (match all) and `async: true`.

**Events clorch registers:**

| Event | Script | Command prefix |
|---|---|---|
| `SessionStart` | `event_handler.sh` | `CLORCH_EVENT=SessionStart` |
| `PreToolUse` | `event_handler.sh` | `CLORCH_EVENT=PreToolUse` |
| `PostToolUse` | `event_handler.sh` | `CLORCH_EVENT=PostToolUse` |
| `PostToolUseFailure` | `event_handler.sh` | `CLORCH_EVENT=PostToolUseFailure` |
| `Stop` | `event_handler.sh` | `CLORCH_EVENT=Stop` |
| `SessionEnd` | `event_handler.sh` | `CLORCH_EVENT=SessionEnd` |
| `PermissionRequest` | `event_handler.sh` | `CLORCH_EVENT=PermissionRequest` |
| `UserPromptSubmit` | `event_handler.sh` | `CLORCH_EVENT=UserPromptSubmit` |
| `SubagentStart` | `event_handler.sh` | `CLORCH_EVENT=SubagentStart` |
| `SubagentStop` | `event_handler.sh` | `CLORCH_EVENT=SubagentStop` |
| `PreCompact` | `event_handler.sh` | `CLORCH_EVENT=PreCompact` |
| `TeammateIdle` | `event_handler.sh` | `CLORCH_EVENT=TeammateIdle` |
| `TaskCompleted` | `event_handler.sh` | `CLORCH_EVENT=TaskCompleted` |
| `Notification` | `notify_handler.sh` | (no prefix needed — dedicated script) |

### 11.2 Install Flow

`clorch init`:

1. Read `~/.claude/settings.json`. If it doesn't exist, create `{}`.
2. Create timestamped backup: `settings.json.bak.<unix_timestamp>`.
3. Render hook script templates and write to `~/.local/share/clorch/hooks/`. Set `chmod +x`.
4. Build the hooks structure (14 event entries, all `async: true`).
5. Merge into existing settings:
   - For each event key, if it doesn't exist → add the full entry.
   - If it exists, scan the matcher group list for any entry where a hook command contains `clorch/hooks/` (the **marker substring**). If found → replace in-place. If not → append.
6. Write updated settings JSON with `indent=2`.

`clorch uninstall`:

1. For each event key in `hooks`, filter out matcher groups where any hook command contains the `clorch/hooks/` marker substring.
2. If an event's matcher group list becomes empty, delete the event key.
3. If the entire `hooks` object becomes empty, delete it.
4. Delete hook scripts from `~/.local/share/clorch/hooks/`.
5. Backup before modifying.

`clorch init --dry-run`: Print the JSON diff that would be applied, write nothing.

## 12. tmux Integration

### 12.1 Session Discovery

Clorch expects agents to run in tmux windows within a configurable session (default: `claude`). The hook scripts detect the tmux pane ID by matching the agent's TTY against `tmux list-panes -a -F '#{pane_tty} #{pane_id}'`.

### 12.2 Navigation

`navigator.JumpToAgent(agent AgentState)`:

1. If terminal backend supports native tabs (iTerm2) and agent has a mapped tab → activate tab.
2. Otherwise → `tmux select-window -t <window>` + `tmux select-pane -t <pane>`.

`navigator.JumpToNextAttention()`: Cycle through agents with `waiting_permission` or `waiting_answer` status.

### 12.3 Status Bar Widget

`clorch tmux-widget` outputs a compact string for `status-right`:

```
#[fg=#a3be8c]●3 #[fg=#ebcb8b]◉1 #[fg=#bf616a]✕0
```

(3 working, 1 waiting, 0 errored — using tmux color escapes with Nord palette)

## 13. TUI Architecture

### 13.1 Bubble Tea Model

The root `Model` struct owns all TUI state:

```go
type Model struct {
    // State
    agents       []state.AgentState
    summary      state.StatusSummary
    actionQueue  []state.ActionItem
    usage        usage.UsageSummary

    // UI state
    selectedIdx  int          // cursor position in agent list
    focusedAction string      // letter of focused action item ("a"-"z" or "")
    showDetail   bool         // agent detail panel visible
    showHelp     bool         // help overlay visible
    yoloEnabled  bool         // YOLO mode active
    soundEnabled bool         // sound notifications active

    // Subsystems (not owned — references)
    stateManager *state.Manager
    watcher      *state.Watcher
    rules        *rules.Engine
    notifier     *notify.Notifier
    navigator    *tmux.Navigator
    usageTracker *usage.Tracker

    // Dimensions
    width, height int
}
```

### 13.2 Message Types

```go
// StateUpdateMsg is sent by the watcher when state files change.
type StateUpdateMsg struct {
    Agents  []state.AgentState
    Summary state.StatusSummary
    Queue   []state.ActionItem
}

// UsageUpdateMsg is sent by the usage tracker on each poll cycle.
type UsageUpdateMsg struct {
    Summary usage.UsageSummary
}

// ApprovalResultMsg is sent after an approve/deny keystroke is dispatched.
type ApprovalResultMsg struct {
    SessionID string
    Action    string // "approved" or "denied"
    Err       error
}
```

### 13.3 Watcher → TUI Integration

The `state.Watcher` runs in its own goroutine. When fsnotify (or poll fallback) detects a change in the state directory:

1. Watcher calls `stateManager.Scan()` to read all state files.
2. Watcher diffs against its previous snapshot. If changed:
3. Watcher calls `program.Send(StateUpdateMsg{...})` to inject a message into the Bubble Tea event loop.

The `tea.Program` reference is passed to the watcher at startup. `program.Send()` is goroutine-safe.

The usage tracker follows the same pattern — runs in a goroutine, sends `UsageUpdateMsg` via `program.Send()` every 10 seconds.

### 13.4 Concurrency Model

```
main goroutine
  └── tea.Program.Run()  ← owns the terminal, processes Msgs

watcher goroutine
  └── fsnotify event loop (or ticker for poll fallback)
      └── on change → stateManager.Scan() → program.Send(StateUpdateMsg)

usage goroutine
  └── ticker (10s interval)
      └── usageTracker.Poll() → program.Send(UsageUpdateMsg)

cleanup goroutine
  └── ticker (60s interval)
      └── stateManager.CleanupStale()
```

**Shutdown sequence:**
1. User presses `q` → `Update()` returns `tea.Quit`.
2. `tea.Program.Run()` returns.
3. Main calls `watcher.Stop()` (closes fsnotify watcher, goroutine exits).
4. Main calls `usageTracker.Stop()` (stops ticker, goroutine exits).
5. Main calls cleanup goroutine cancel via context.
6. Process exits.

All goroutines accept a `context.Context` for cancellation. No shared mutable state between goroutines — all communication is via `program.Send()`.

### 13.5 Action Queue Lifecycle

1. Hook script sets state to `WAITING_PERMISSION` with `tool_request_summary`.
2. Next watcher scan picks up the change → `StateUpdateMsg` includes the agent in the action queue.
3. TUI renders the action item with a letter key (`a`-`z`).
4. **User approves from clorch:** TUI sends `tmux send-keys`, then the next hook event (`PreToolUse` or `UserPromptSubmit`) updates state to `WORKING`, clearing the action.
5. **User approves from terminal directly:** Same outcome — next hook event clears the state. Clorch doesn't need to know *how* the permission was resolved.
6. **User denies:** Claude Code doesn't fire a `Stop` event after denial. The `WAITING_PERMISSION` state becomes stale. The cleanup routine detects the dead PID and resets to `IDLE` (see §4.2 stale permission reset).
7. **Safety guard:** Before sending `tmux send-keys`, re-read the state file to confirm the agent is still in `WAITING_PERMISSION`. Prevents misfire if the state changed between the TUI render and the keypress.

Action queue sort order: `WAITING_PERMISSION` first (actionable), then `WAITING_ANSWER`, then `ERROR`. Within the same tier, agents with tmux panes sort before agents without (ensures approve/deny works).

### 13.6 Auto-Approve Flow (YOLO + Rules)

When a `StateUpdateMsg` arrives with new `WAITING_PERMISSION` agents:

1. For each new permission agent, call `rules.Evaluate(toolName, summary)`.
2. If result is `Approve` → immediately dispatch `tmux send-keys "y" Enter`. Log to event log.
3. If result is `Deny` → add to action queue with visual indicator that it was rule-blocked. Do NOT auto-deny (let the human review).
4. If result is `Ask` → add to action queue normally.

## 14. Notifications

Three notification channels, independently toggleable:

| Channel | Trigger | macOS | Linux |
|---|---|---|---|
| Terminal bell | Any attention event | `\a` to stdout | `\a` to stdout |
| System sound | Permission, question, error | `afplay <sound>` | `paplay <sound>` (if available, else skip) |
| Native notification | Permission, question, error | `osascript -e 'display notification ...'` | `notify-send` (if available, else skip) |

**macOS sound mapping** (system sounds):
- Permission request → `/System/Library/Sounds/Sosumi.aiff`
- Question → `/System/Library/Sounds/Ping.aiff`
- Error → `/System/Library/Sounds/Basso.aiff`

**Linux sound mapping:** Use XDG sound theme if available, otherwise degrade gracefully. Sound and native notification are best-effort on Linux — check for `paplay`/`notify-send` on PATH at startup, disable the channel silently if missing.

**Notification deduplication:** Only fire notifications on state *transitions* (idle→waiting, not waiting→waiting). The watcher tracks previous status per agent to detect transitions.

## 15. Terminal Backends

Interface:

```go
type TerminalBackend interface {
    // GetTTYMap returns a map of TTY device → tab/window identifier
    GetTTYMap() (map[string]string, error)
    // ActivateTab brings a specific tab to the foreground
    ActivateTab(id string) error
    // BringToFront raises the terminal application
    BringToFront() error
    // CanResolveTabs reports whether this backend can map agents to native tabs
    CanResolveTabs() bool
}
```

Detection: read `TERM_PROGRAM` env var, fall back to `CLORCH_TERMINAL`. Map to backend:
- `iTerm.app` / `iTerm2` → iTerm backend (full AppleScript support)
- `ghostty` → Ghostty backend (limited AppleScript, needs Accessibility)
- `Apple_Terminal` → Apple Terminal backend
- Anything else → nil backend (tmux-only navigation)

## 16. Build & Release

```makefile
VERSION := $(shell git describe --tags --always --dirty)

build:
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o bin/clorch ./cmd/clorch

install:
	go install -ldflags "-s -w -X main.version=$(VERSION)" ./cmd/clorch
```

Release artifacts: `clorch-darwin-arm64`, `clorch-darwin-amd64`, `clorch-linux-amd64`, `clorch-linux-arm64`. Distributed via GitHub releases and `go install`.

## 17. Testing Strategy

| Layer | Approach |
|---|---|
| State parsing | Unit tests: feed JSON files into `state.Manager`, assert `AgentState` output |
| Rules engine | Unit tests: load YAML, evaluate tool+summary combos, assert actions |
| Hook installer | Unit tests: mock filesystem, verify merge logic preserves existing hooks |
| Usage parser | Unit tests: fixture JSONL files, verify token counts and cost calculations |
| tmux commands | Integration tests: verify command strings without executing (mock `exec.Command`) |
| TUI | Manual testing. Bubble Tea's `teatest` package for critical flows if warranted |

## 18. Migration from Python clorch

The Go version is a clean rewrite, not a port. However, it maintains protocol compatibility:

- **Same state directory** (`/tmp/clorch/state/`) and JSON schema
- **Same hook scripts** (bash, not Python)
- **Same approval mechanism** (tmux `send-keys`)
- **Same rules file format** (`~/.config/clorch/rules.yaml`)

Users can switch between Python and Go clorch without reinstalling hooks. Both read the same state files.

## 19. Ambiguity Flags

Items that need clarification or may break across Claude Code versions:

1. **Transcript JSONL schema is undocumented.** The `usage` field nesting (`.message.usage.input_tokens`), record types (`"assistant"`, `"user"`, `"custom-title"`, `"file-history-snapshot"`), and field names are derived from observation. A Claude Code update could change this structure silently. The parser must be defensive — skip unparseable lines, don't crash on missing fields.

2. **`history.jsonl` format is undocumented.** The `sessionId` and `display` fields are inferred from upstream's parser. The file lives at `~/.claude/history.jsonl`. Unknown whether this is a stable interface or internal implementation detail.

3. **Notification message keyword matching is fragile.** The notify handler determines `WAITING_PERMISSION` vs `WAITING_ANSWER` by checking if the `message` field contains "permission", "question", "input", "answer", or "elicitation". If Claude Code changes notification wording, status detection breaks. The `notification_type` field (`"permission_prompt"`, `"idle_prompt"`, `"elicitation_dialog"`) would be more reliable — upstream doesn't use it yet but we should prefer it when present and fall back to keyword matching.

4. **`Stop` event after permission denial.** Claude Code does **not** fire a `Stop` event when the user denies a permission. The session goes to an "Interrupted" state with no hook. This means `WAITING_PERMISSION` can persist in the state file after the user has already acted. The stale permission reset (§4.2) handles this, but only for dead PIDs. For live sessions where the user denied from the terminal, the state will correct itself on the next `UserPromptSubmit` or `PreToolUse` event.

6. **AppleScript for Ghostty.** Ghostty's osascript support requires the Accessibility permission to be granted. The spec doesn't detail what AppleScript commands are needed or how to handle the permission prompt. Upstream uses `osascript` but the exact scripts are in terminal backend files not fully analyzed here.

## 20. References

- [Claude Code Hooks Documentation](https://docs.anthropic.com/en/docs/claude-code/hooks) — canonical reference for hook events, stdin schema, exit codes, settings.json format
- [androsovm/clorch](https://github.com/androsovm/clorch) — upstream Python implementation, source of truth for state protocol and hook scripts
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — Go TUI framework (Elm architecture)
- [fsnotify](https://github.com/fsnotify/fsnotify) — cross-platform filesystem notification library (kqueue on macOS, inotify on Linux)

## 21. Resolved Design Decisions

1. **State dir location.** `/tmp/clorch/state`. It's ephemeral state that should die on reboot. No reason to complicate this with XDG.
2. **Hook transport.** `fsnotify` for filesystem event watching instead of polling. Adds one dependency but gets sub-100ms response time on both macOS (kqueue) and Linux (inotify). Fall back to 500ms polling if fsnotify init fails.
3. **tmux scope.** Single tmux session. One session with many windows is the workflow. Multi-session adds complexity for no real gain.
4. **Hook scripts.** Go `text/template` generated at install time. Templates allow embedding the state dir path, version metadata, and any future per-install configuration directly into the scripts. `clorch init` regenerates them on every run so upgrades are automatic.
