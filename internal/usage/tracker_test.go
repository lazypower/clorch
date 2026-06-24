package usage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTrackerAggregatesPerSessionCosts(t *testing.T) {
	dir := t.TempDir()
	projDir := filepath.Join(dir, "proj1")
	if err := os.MkdirAll(projDir, 0755); err != nil {
		t.Fatal(err)
	}

	opus := `{"type":"assistant","sessionId":"s1","message":{"role":"assistant","model":"claude-opus-4-6-20260301","content":[],"usage":{"input_tokens":1000000,"output_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}` + "\n"
	haiku := `{"type":"assistant","sessionId":"s2","message":{"role":"assistant","model":"claude-haiku-4-5-20251001","content":[],"usage":{"input_tokens":1000000,"output_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}` + "\n"
	if err := os.WriteFile(filepath.Join(projDir, "s1.jsonl"), []byte(opus), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "s2.jsonl"), []byte(haiku), 0644); err != nil {
		t.Fatal(err)
	}

	tracker := NewTracker(NewParser(dir), nil)
	tracker.poll()

	if len(tracker.history) != 1 {
		t.Fatalf("history entries = %d, want 1", len(tracker.history))
	}
	if !approxEqual(tracker.history[0].cost, 6.00, 0.01) {
		t.Fatalf("cost = %.2f, want 6.00", tracker.history[0].cost)
	}
}

func TestTrackerCostStaysVisibleAcrossIdlePolls(t *testing.T) {
	dir := t.TempDir()
	projDir := filepath.Join(dir, "proj1")
	if err := os.MkdirAll(projDir, 0755); err != nil {
		t.Fatal(err)
	}

	line := `{"type":"assistant","sessionId":"s1","message":{"role":"assistant","model":"claude-sonnet-4-6-20260301","content":[],"usage":{"input_tokens":1000000,"output_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}` + "\n"
	if err := os.WriteFile(filepath.Join(projDir, "s1.jsonl"), []byte(line), 0644); err != nil {
		t.Fatal(err)
	}

	tracker := NewTracker(NewParser(dir), nil)
	tracker.poll()
	tracker.poll()

	if len(tracker.history) != 2 {
		t.Fatalf("history entries = %d, want 2", len(tracker.history))
	}
	if !approxEqual(tracker.history[1].cost, tracker.history[0].cost, 0.01) {
		t.Fatalf("idle poll cost = %.2f, want %.2f", tracker.history[1].cost, tracker.history[0].cost)
	}
}
