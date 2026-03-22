# Feature Proposals

Collected ideas for making clorch better than the original. Organized by origin and rough effort.

---

## Operator-Facing (Chuck's Big Bangers)

### 1. Stuck Detector

Agents get stuck in loops — re-reading the same file, retrying a command, oscillating between plan updates. Clorch should detect this automatically.

**Heuristic:** Same tool + same arguments > 3 times in 30 seconds.

**Dashboard indicator:**
```
Session #8 ⚠ possible loop

tool: read_file
path: config.go
repeated 5 times
```

**Operator actions:** `[interrupt]` `[inject hint]` `[restart session]`

**Why:** Saves babysitting, surfaces problems early, makes the dashboard feel smart.

**Implementation:** Maintain a small rolling buffer of recent tool calls per agent. Compare tool name + key arguments. Flag when threshold hit. ~150 lines of Go + TUI indicator.

---

### 2. Hotkey Prompt Injection

Nudge a running agent without opening its terminal.

**Hotkey:** `i` → inject message

**UI:**
```
Inject message to session #12:

> check the failing test output
```

Clorch sends the message directly to the agent's input stream via tmux `send-keys`.

**Why:** Quick steering, avoids tmux tab switching, makes clorch a real control console.

**Implementation:** Text input component (bubbles `textinput`) + `tmux send-keys` with the typed message + Enter. Most of the work is UI. ~200 lines.

---

### 3. Resource Telemetry Per Session

Lightweight stats per agent:

```
Session #3
CPU: 22%
RAM: 1.3GB
tokens used: 24k
runtime: 18m
```

**Why:** Spot runaway sessions, compare models, monitor cost.

**Implementation:** Read `/proc/<pid>/stat` (Linux) or `ps -p <pid> -o %cpu,rss` (macOS). Token counts already tracked. Runtime from `started_at`. ~100 lines for collection, fold into agent detail panel.

---

### Bonus: Idle Session Fade

Sessions idle > N minutes get visually dimmed in the agent list.

```
Session #6 (idle 9m)     ← dimmed/faded text
```

Makes the UI way easier to scan when running many agents.

**Implementation:** Already have `StaleDuration` on `AgentState`. Just add a lipgloss style conditional in `renderAgentRow()`. ~50 lines.

---

## Infrastructure / Polish (Claude's Proposals)

### 4. Event Log Panel

Scrollable, timestamped log of clorch decisions: auto-approvals, rule matches, notification fires, state transitions. Right now clorch is a black box — you can't tell *why* it approved something or *when* an agent transitioned.

**Spec called for:** `event_log.go` (never built).

**Implementation:** Ring buffer of log entries, viewport component, toggle with `e` key. ~200 lines.

---

### 5. Settings Persistence

YOLO and sound toggles reset every session. Write to `~/.config/clorch/settings.yaml` or merge into `rules.yaml`.

**Implementation:** ~80 lines. Read on startup, write on toggle.

---

### 6. Agent Grouping / Filtering

Once you're running 8+ agents, the flat list breaks down. Group by project, filter by status, or let users tag agents.

**Implementation:** ~150 lines. Filter state in model, render group headers.

---

### 7. `clorch approve --all` CLI Command

Scriptable approval from outside the TUI. Pairs with `clorch status` for automation.

**Implementation:** ~50 lines. Read state dir, find WAITING_PERMISSION, send-keys to each.

---

### 8. Rule Hit Counters / Dry-Run

`clorch rules test "Bash" "rm -rf /tmp"` — tells you which rule matched and what action. Add hit counters visible in the TUI.

**Implementation:** ~100 lines. Counter map in rules engine, CLI subcommand.

---

### 9. Compact Detection Alerting ✅ (v0.6.0)

Compaction is a silent productivity killer. Clorch tracks `compact_count` but doesn't surface it prominently. Flashing indicator or dedicated notification when an agent compacts.

**Implementation:** ~30 lines. Style change in agent row when `compact_count` increases.

---

## The Big One: Session Forking (Agent Branching)

`git branch` for running agent sessions.

**Problem:** If an agent is midway through something and you want to try a different direction, you start a new session, re-paste context, and hope it behaves similarly.

**Solution:** Fork the state of a running session.

**UI:**
```
Session #14

[f] Fork session

→ Session #14 (original)
→ Session #27 (fork of #14)
```

The fork starts with the same working directory, conversation history, and plan state. Then you inject a different instruction.

**Real-world example:**
```
Session 12 → rewrite function
  Fork → Session 13 → rewrite using channels
  Fork → Session 14 → rewrite using worker pool
  Fork → Session 15 → keep original approach
```

Compare outputs, keep the best result.

**Why this is the differentiator:** Most dashboards help you *observe* agents. Forking helps you *think with them*. It turns clorch into a parallel idea exploration tool instead of linear agent interaction. Agents are probabilistic planners — forking lets you explore multiple outcomes.

**Critical constraint: observation must never control lifecycle.**

If clorch crashes, every agent must keep running. Clorch is htop, not systemd. This means clorch cannot own forked processes directly — it must delegate to tmux and walk away.

**Revised fork flow:**
```
user presses "f"
→ clorch creates git worktree (or dir copy if not a git repo)
→ clorch runs: tmux new-window -t <session> "claude --resume <id>"
→ tmux owns the process (clorch does NOT hold the PID)
→ claude emits hooks to /tmp as usual
→ clorch discovers the fork like any other session
```

**Environment strategy:**
- Git repo → `git worktree add .clorch/forks/<session-id> -b clorch-<session-id>` (fast, shared object store, minimal disk)
- No git → `cp -r` fallback (heavier but works everywhere)
- Worktrees are the primary path; most Claude Code use is inside git repos

**Spawn target:** Same tmux session, new window. Keeps everything in one operational context for visibility.

**What clorch reproduces:** Conversation history + filesystem state + environment. NOT internal Claude runtime state. If those three match, the agent behaves similarly enough.

**Session identity:** Fork discovered via normal hook pipeline. The `session_id` from Claude's hooks is the canonical identifier — no need for clorch to track fork parentage internally (though a `forked_from` field in state would enable tree views later).

**Open questions:**
- Does `claude --resume <session_id>` work reliably? Need to test experimentally.
- If resume doesn't work, fall back to transcript replay (messier but functional).
- Should forks be visually linked in the TUI? (tree view vs flat list with fork badge — start with badge, evolve to tree)

~200-300 lines. No architecture change — clorch triggers, tmux owns, hooks discover.

---

### Bonus 2: Attention Sort Mode

Sort agent list by urgency instead of linearly:

```
1. waiting approval     ← needs you NOW
2. stuck (loop detect)  ← needs you SOON
3. finished task        ← review ready
4. idle                 ← nothing happening
5. active               ← leave it alone
```

Dashboard becomes an operator queue, not just a list.

**Implementation:** Already have status priority in `manager.go` sort logic. Just refine the ordering and make it the default. ~30 lines.

---

## Instrument-Grade Features (The Category Shift)

These move clorch from **agent monitor** to **agent development instrument**.

**Litmus test:** If it answers "what is the agent doing right now?" or "why did the agent do that?" → it belongs in clorch. If it answers "what does the agent remember?" → it belongs in Continuity.

### 10. Session Timeline (Execution Debugger)

Clorch shows state but not decision history. Add a timeline view per session.

```
Session #12
----------------------------------
10:42  prompt received
10:42  plan generated
10:43  tool: read_file → config.go
10:43  diff created → main.go
10:44  bash: go test
10:44  tool result: passed
```

**Why:** Debug agent behavior. See *why* something happened. Compare agent runs. Think `git log` for agent execution.

**Data source:** The JSONL transcript already has everything — tool calls, results, timestamps. Parser exists in `internal/usage/parser.go`, just needs to extract event-level records instead of just token counts.

**Implementation:** Extend parser to extract tool events, new `timeline.go` TUI component with viewport scroll. ~250 lines.

---

### 11. Artifact Awareness

Agents generate real work artifacts (code, diffs, files, plans, shell output) but they're buried in terminal scrollback. Clorch could detect and track them.

```
Artifacts (session #8)

  plan.md           created 10:42
  main.go           diff +47/-12
  migration.sql     created 10:44
  test_results.txt  created 10:45
```

**Why:** Inspect output without switching terminals. Track long-running sessions. Surface important results automatically.

**Data source:** `PostToolUse` events contain tool responses. Write/Edit tools have file paths. Bash has stdout. Hook already captures `last_tool` — extend to capture artifact paths.

**Implementation:** Track file paths from Write/Edit/Read tool events in state file. New `artifacts` field on AgentState. Detail panel or dedicated view. ~200 lines (hook changes + TUI).

---

### 12. Session Comparison

Compare two agent runs side by side.

```
Compare Session 14 vs Session 15
---------------------------------
                  #14          #15
Plan approach     channels     worker pool
Files touched     3            5
Commands run      12           8
Errors            0            2
Duration          4m           7m
Model             opus-4-6     sonnet-4-6
Cost              $1.20        $0.18
```

**Why:** Tune prompts. Debug agent regressions. Compare models. Answer "why did Sonnet succeed but Haiku fail?" without reading both transcripts.

**Data source:** JSONL transcripts + state files. Diff tool call sequences, file paths touched, error counts, timing.

**Implementation:** CLI subcommand `clorch compare <session1> <session2>` + optional TUI overlay. ~300 lines.

---

## Prioritization

## Scope Boundary

**Clorch is a local operator tool.** Single machine, single operator, process-level control. No distributed agents, no multi-machine coordination, no networking layer. That's a different problem for a different time.

---

| Feature | Effort | Impact | Category | Priority |
|---------|--------|--------|----------|----------|
| Stuck detector | ~150 LOC | Very high | Operator | **P0** |
| Session forking | ~300 LOC | Transformative | Operator | **P0** |
| Idle session fade | ~50 LOC | High | Operator | **P0** |
| Hotkey prompt injection | ~200 LOC | Very high | Operator | **P0** |
| Attention sort mode | ~30 LOC | High | Operator | **P0** |
| Session timeline | ~250 LOC | Very high | Instrument | **P1** |
| Compact alerting | ~30 LOC | Medium | Operator | **P1** |
| Resource telemetry | ~100 LOC | High | Operator | **P1** |
| Artifact awareness | ~200 LOC | High | Instrument | **P2** |
| Session comparison | ~300 LOC | High | Instrument | **P2** |
| Event log panel | ~200 LOC | High | Polish | **P2** |
| Settings persistence | ~80 LOC | Medium | Polish | **P2** |
| `approve --all` CLI | ~50 LOC | Medium | Polish | **P2** |
| Rule hit counters | ~100 LOC | Medium | Polish | **P3** |
| Agent grouping | ~150 LOC | Medium | Polish | **P3** |

### P0 defines the operator experience

These five features together turn clorch into an **attention router**:

1. **Stuck detector** — tells you *which* agent needs help
2. **Session forking** — lets you *branch* agent exploration
3. **Idle fade** — dims noise so you *see* what matters
4. **Prompt injection** — lets you *act* without leaving the dashboard
5. **Attention sort** — puts what needs you *first*

That's the full loop: see → understand → act → explore.
