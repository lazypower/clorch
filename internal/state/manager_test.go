package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeState(t *testing.T, dir string, agent AgentState) {
	t.Helper()
	data, err := json.Marshal(agent)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, agent.SessionID+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestScanEmpty(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)
	agents, summary, queue := mgr.Scan()
	if len(agents) != 0 {
		t.Errorf("expected 0 agents, got %d", len(agents))
	}
	if summary.Total != 0 {
		t.Errorf("expected total 0, got %d", summary.Total)
	}
	if len(queue) != 0 {
		t.Errorf("expected 0 queue items, got %d", len(queue))
	}
}

func TestScanMultipleAgents(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)

	writeState(t, dir, AgentState{
		SessionID:     "s1",
		Status:        StatusWorking,
		ProjectName:   "proj1",
		LastEventTime: "2026-03-10T10:00:00Z",
		ToolCount:     5,
	})
	writeState(t, dir, AgentState{
		SessionID:     "s2",
		Status:        StatusIdle,
		ProjectName:   "proj2",
		LastEventTime: "2026-03-10T09:00:00Z",
	})

	agents, summary, _ := mgr.Scan()
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}
	if summary.Working != 1 {
		t.Errorf("expected 1 working, got %d", summary.Working)
	}
	if summary.Idle != 1 {
		t.Errorf("expected 1 idle, got %d", summary.Idle)
	}
}

func TestScanSortOrder(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)

	writeState(t, dir, AgentState{
		SessionID:     "idle1",
		Status:        StatusIdle,
		LastEventTime: "2026-03-10T10:00:00Z",
	})
	writeState(t, dir, AgentState{
		SessionID:     "waiting1",
		Status:        StatusWaitingPermission,
		LastEventTime: "2026-03-10T09:00:00Z",
	})
	writeState(t, dir, AgentState{
		SessionID:     "working1",
		Status:        StatusWorking,
		LastEventTime: "2026-03-10T10:00:00Z",
	})

	agents, _, _ := mgr.Scan()
	if len(agents) != 3 {
		t.Fatalf("expected 3 agents, got %d", len(agents))
	}
	// Waiting should sort first
	if agents[0].SessionID != "waiting1" {
		t.Errorf("expected waiting agent first, got %s", agents[0].SessionID)
	}
	// Working before idle
	if agents[1].Status != StatusWorking {
		t.Errorf("expected working second, got %s", agents[1].Status)
	}
}

func TestActionQueue(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)

	summary := "$ rm -rf /"
	writeState(t, dir, AgentState{
		SessionID:          "w1",
		Status:             StatusWaitingPermission,
		ProjectName:        "proj1",
		LastTool:           "Bash",
		ToolRequestSummary: &summary,
		TmuxPane:           "0",
		LastEventTime:      "2026-03-10T10:00:00Z",
	})
	writeState(t, dir, AgentState{
		SessionID:     "w2",
		Status:        StatusWaitingAnswer,
		ProjectName:   "proj2",
		LastEventTime: "2026-03-10T10:00:00Z",
	})
	writeState(t, dir, AgentState{
		SessionID:     "ok",
		Status:        StatusWorking,
		LastEventTime: "2026-03-10T10:00:00Z",
	})

	_, _, queue := mgr.Scan()
	if len(queue) != 2 {
		t.Fatalf("expected 2 queue items, got %d", len(queue))
	}
	// Permission first
	if queue[0].Agent.Status != StatusWaitingPermission {
		t.Errorf("expected permission first, got %s", queue[0].Agent.Status)
	}
	if queue[0].Letter != "a" {
		t.Errorf("expected letter 'a', got '%s'", queue[0].Letter)
	}
	if queue[1].Letter != "b" {
		t.Errorf("expected letter 'b', got '%s'", queue[1].Letter)
	}
}

func TestScanSkipsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)

	os.WriteFile(filepath.Join(dir, "bad.json"), []byte("not json"), 0644)
	writeState(t, dir, AgentState{
		SessionID:     "good",
		Status:        StatusIdle,
		LastEventTime: "2026-03-10T10:00:00Z",
	})

	agents, _, _ := mgr.Scan()
	if len(agents) != 1 {
		t.Errorf("expected 1 agent (skip bad json), got %d", len(agents))
	}
}

func TestScanSkipsNonJSON(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)

	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hello"), 0644)
	agents, _, _ := mgr.Scan()
	if len(agents) != 0 {
		t.Errorf("expected 0 agents, got %d", len(agents))
	}
}

func TestEnsureStateDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "state")
	mgr := NewManager(dir)
	if err := mgr.EnsureStateDir(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}
}

func TestStateDir(t *testing.T) {
	mgr := NewManager("/tmp/test")
	if mgr.StateDir() != "/tmp/test" {
		t.Errorf("expected /tmp/test, got %s", mgr.StateDir())
	}
}

func TestComputeSummary(t *testing.T) {
	agents := []AgentState{
		{Status: StatusWorking},
		{Status: StatusWorking},
		{Status: StatusIdle},
		{Status: StatusWaitingPermission},
		{Status: StatusWaitingAnswer},
		{Status: StatusError},
	}
	s := ComputeSummary(agents)
	if s.Total != 6 {
		t.Errorf("total: got %d, want 6", s.Total)
	}
	if s.Working != 2 {
		t.Errorf("working: got %d, want 2", s.Working)
	}
	if s.Idle != 1 {
		t.Errorf("idle: got %d, want 1", s.Idle)
	}
	if s.Waiting != 2 {
		t.Errorf("waiting: got %d, want 2", s.Waiting)
	}
	if s.Error != 1 {
		t.Errorf("error: got %d, want 1", s.Error)
	}
}

// --- Stuck detector tests ---

func TestDetectStuckLoop_NoTools(t *testing.T) {
	if detectStuckLoop(nil, time.Now()) {
		t.Error("expected no stuck loop with nil tools")
	}
	if detectStuckLoop([]RecentToolCall{}, time.Now()) {
		t.Error("expected no stuck loop with empty tools")
	}
}

func TestDetectStuckLoop_TooFewCalls(t *testing.T) {
	now := time.Now()
	tools := []RecentToolCall{
		{Tool: "Read", ArgsHash: "abc123", Time: now.Add(-5 * time.Second).Format(time.RFC3339)},
		{Tool: "Read", ArgsHash: "abc123", Time: now.Add(-3 * time.Second).Format(time.RFC3339)},
	}
	if detectStuckLoop(tools, now) {
		t.Error("expected no stuck loop with only 2 calls")
	}
}

func TestDetectStuckLoop_Detected(t *testing.T) {
	now := time.Now()
	tools := []RecentToolCall{
		{Tool: "Read", ArgsHash: "abc123", Time: now.Add(-20 * time.Second).Format(time.RFC3339)},
		{Tool: "Read", ArgsHash: "abc123", Time: now.Add(-10 * time.Second).Format(time.RFC3339)},
		{Tool: "Read", ArgsHash: "abc123", Time: now.Add(-5 * time.Second).Format(time.RFC3339)},
	}
	if !detectStuckLoop(tools, now) {
		t.Error("expected stuck loop detected")
	}
}

func TestDetectStuckLoop_DifferentTools(t *testing.T) {
	now := time.Now()
	tools := []RecentToolCall{
		{Tool: "Read", ArgsHash: "abc123", Time: now.Add(-20 * time.Second).Format(time.RFC3339)},
		{Tool: "Write", ArgsHash: "def456", Time: now.Add(-10 * time.Second).Format(time.RFC3339)},
		{Tool: "Bash", ArgsHash: "ghi789", Time: now.Add(-5 * time.Second).Format(time.RFC3339)},
	}
	if detectStuckLoop(tools, now) {
		t.Error("expected no stuck loop with different tools")
	}
}

func TestDetectStuckLoop_SameToolDifferentArgs(t *testing.T) {
	now := time.Now()
	tools := []RecentToolCall{
		{Tool: "Read", ArgsHash: "aaa", Time: now.Add(-20 * time.Second).Format(time.RFC3339)},
		{Tool: "Read", ArgsHash: "bbb", Time: now.Add(-10 * time.Second).Format(time.RFC3339)},
		{Tool: "Read", ArgsHash: "ccc", Time: now.Add(-5 * time.Second).Format(time.RFC3339)},
	}
	if detectStuckLoop(tools, now) {
		t.Error("expected no stuck loop with different args hashes")
	}
}

func TestDetectStuckLoop_OldCallsIgnored(t *testing.T) {
	now := time.Now()
	tools := []RecentToolCall{
		{Tool: "Read", ArgsHash: "abc123", Time: now.Add(-60 * time.Second).Format(time.RFC3339)},
		{Tool: "Read", ArgsHash: "abc123", Time: now.Add(-50 * time.Second).Format(time.RFC3339)},
		{Tool: "Read", ArgsHash: "abc123", Time: now.Add(-40 * time.Second).Format(time.RFC3339)},
	}
	if detectStuckLoop(tools, now) {
		t.Error("expected no stuck loop — all calls older than 30s")
	}
}

func TestDetectStuckLoop_MixedTimestamps(t *testing.T) {
	now := time.Now()
	tools := []RecentToolCall{
		{Tool: "Read", ArgsHash: "abc123", Time: now.Add(-60 * time.Second).Format(time.RFC3339)}, // old, ignored
		{Tool: "Read", ArgsHash: "abc123", Time: now.Add(-20 * time.Second).Format(time.RFC3339)},
		{Tool: "Write", ArgsHash: "xyz", Time: now.Add(-15 * time.Second).Format(time.RFC3339)},   // different tool
		{Tool: "Read", ArgsHash: "abc123", Time: now.Add(-10 * time.Second).Format(time.RFC3339)},
		{Tool: "Read", ArgsHash: "abc123", Time: now.Add(-5 * time.Second).Format(time.RFC3339)},
	}
	if !detectStuckLoop(tools, now) {
		t.Error("expected stuck loop — 3 matching calls within 30s window")
	}
}

func TestDetectStuckLoop_InvalidTimestamp(t *testing.T) {
	now := time.Now()
	tools := []RecentToolCall{
		{Tool: "Read", ArgsHash: "abc123", Time: "not-a-timestamp"},
		{Tool: "Read", ArgsHash: "abc123", Time: "also-bad"},
		{Tool: "Read", ArgsHash: "abc123", Time: "nope"},
	}
	if detectStuckLoop(tools, now) {
		t.Error("expected no stuck loop with invalid timestamps")
	}
}

// --- Attention sort tests ---

func TestAttentionSort_RecencyWithinTier(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)

	now := time.Now()
	writeState(t, dir, AgentState{
		SessionID:     "old-worker",
		Status:        StatusWorking,
		LastEventTime: now.Add(-60 * time.Second).Format(time.RFC3339),
	})
	writeState(t, dir, AgentState{
		SessionID:     "new-worker",
		Status:        StatusWorking,
		LastEventTime: now.Add(-5 * time.Second).Format(time.RFC3339),
	})

	agents, _, _ := mgr.Scan()
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}
	// More recently active agent should sort first within same tier
	if agents[0].SessionID != "new-worker" {
		t.Errorf("expected new-worker first, got %s", agents[0].SessionID)
	}
}

func TestAttentionSort_UrgencyBeatsRecency(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)

	now := time.Now()
	// Recent but just working
	writeState(t, dir, AgentState{
		SessionID:     "active",
		Status:        StatusWorking,
		LastEventTime: now.Add(-1 * time.Second).Format(time.RFC3339),
	})
	// Old but waiting permission — should still sort first
	writeState(t, dir, AgentState{
		SessionID:     "waiting",
		Status:        StatusWaitingPermission,
		LastEventTime: now.Add(-120 * time.Second).Format(time.RFC3339),
	})

	agents, _, _ := mgr.Scan()
	if agents[0].SessionID != "waiting" {
		t.Errorf("expected waiting agent first regardless of recency, got %s", agents[0].SessionID)
	}
}

func TestStuckLoopDetectedInScan(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)

	now := time.Now()
	writeState(t, dir, AgentState{
		SessionID:     "stuck",
		Status:        StatusWorking,
		LastEventTime: now.Format(time.RFC3339),
		RecentTools: []RecentToolCall{
			{Tool: "Read", ArgsHash: "aaa", Time: now.Add(-15 * time.Second).Format(time.RFC3339)},
			{Tool: "Read", ArgsHash: "aaa", Time: now.Add(-10 * time.Second).Format(time.RFC3339)},
			{Tool: "Read", ArgsHash: "aaa", Time: now.Add(-5 * time.Second).Format(time.RFC3339)},
		},
	})
	writeState(t, dir, AgentState{
		SessionID:     "healthy",
		Status:        StatusWorking,
		LastEventTime: now.Format(time.RFC3339),
	})

	agents, _, _ := mgr.Scan()
	var stuck, healthy *AgentState
	for i := range agents {
		if agents[i].SessionID == "stuck" {
			stuck = &agents[i]
		}
		if agents[i].SessionID == "healthy" {
			healthy = &agents[i]
		}
	}
	if stuck == nil || !stuck.StuckLoop {
		t.Error("expected stuck agent to have StuckLoop=true")
	}
	if healthy == nil || healthy.StuckLoop {
		t.Error("expected healthy agent to have StuckLoop=false")
	}
}
