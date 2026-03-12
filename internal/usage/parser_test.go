package usage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTranscript(t *testing.T) {
	dir := t.TempDir()
	projDir := filepath.Join(dir, "proj1")
	os.MkdirAll(projDir, 0755)

	transcript := `{"type":"user","sessionId":"s1","message":{"role":"user","content":"hello"}}
{"type":"assistant","sessionId":"s1","message":{"role":"assistant","model":"claude-opus-4-6-20260301","content":[{"type":"text","text":"hi"}],"usage":{"input_tokens":1000,"output_tokens":200,"cache_creation_input_tokens":0,"cache_read_input_tokens":500}}}
{"type":"assistant","sessionId":"s1","message":{"role":"assistant","model":"claude-opus-4-6-20260301","content":[{"type":"text","text":"done"}],"usage":{"input_tokens":2000,"output_tokens":300,"cache_creation_input_tokens":100,"cache_read_input_tokens":0}}}
`
	os.WriteFile(filepath.Join(projDir, "s1.jsonl"), []byte(transcript), 0644)

	parser := NewParser(dir)
	pr := parser.Poll()

	if pr.Total.InputTokens != 3000 {
		t.Errorf("input tokens = %d, want 3000", pr.Total.InputTokens)
	}
	if pr.Total.OutputTokens != 500 {
		t.Errorf("output tokens = %d, want 500", pr.Total.OutputTokens)
	}
	if pr.Total.CacheCreationTokens != 100 {
		t.Errorf("cache creation = %d, want 100", pr.Total.CacheCreationTokens)
	}
	if pr.Total.CacheReadTokens != 500 {
		t.Errorf("cache read = %d, want 500", pr.Total.CacheReadTokens)
	}
	if pr.Model != "claude-opus-4-6-20260301" {
		t.Errorf("model = %q, want claude-opus-4-6-20260301", pr.Model)
	}
}

func TestParsePerSession(t *testing.T) {
	dir := t.TempDir()
	projDir := filepath.Join(dir, "proj1")
	os.MkdirAll(projDir, 0755)

	t1 := `{"type":"assistant","sessionId":"s1","message":{"role":"assistant","model":"claude-opus-4-6-20260301","content":[],"usage":{"input_tokens":1000,"output_tokens":100,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}` + "\n"
	t2 := `{"type":"assistant","sessionId":"s2","message":{"role":"assistant","model":"claude-sonnet-4-6-20260301","content":[],"usage":{"input_tokens":500,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}` + "\n"

	os.WriteFile(filepath.Join(projDir, "sess-aaa.jsonl"), []byte(t1), 0644)
	os.WriteFile(filepath.Join(projDir, "sess-bbb.jsonl"), []byte(t2), 0644)

	parser := NewParser(dir)
	pr := parser.Poll()

	if len(pr.PerSession) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(pr.PerSession))
	}
	if pr.PerSession["sess-aaa"].Tokens.InputTokens != 1000 {
		t.Errorf("sess-aaa input = %d, want 1000", pr.PerSession["sess-aaa"].Tokens.InputTokens)
	}
	if pr.PerSession["sess-bbb"].Tokens.InputTokens != 500 {
		t.Errorf("sess-bbb input = %d, want 500", pr.PerSession["sess-bbb"].Tokens.InputTokens)
	}
	if pr.PerSession["sess-bbb"].Model != "claude-sonnet-4-6-20260301" {
		t.Errorf("sess-bbb model = %q", pr.PerSession["sess-bbb"].Model)
	}
}

func TestParseSkipsNonAssistant(t *testing.T) {
	dir := t.TempDir()
	projDir := filepath.Join(dir, "proj1")
	os.MkdirAll(projDir, 0755)

	transcript := `{"type":"user","sessionId":"s1","message":{"role":"user","content":"hello"}}
{"type":"file-history-snapshot","sessionId":"s1"}
{"type":"custom-title","customTitle":"my session"}
`
	os.WriteFile(filepath.Join(projDir, "s1.jsonl"), []byte(transcript), 0644)

	parser := NewParser(dir)
	pr := parser.Poll()

	if pr.Total.InputTokens != 0 || pr.Total.OutputTokens != 0 {
		t.Errorf("expected zero tokens for non-assistant records, got in=%d out=%d",
			pr.Total.InputTokens, pr.Total.OutputTokens)
	}
	if pr.Model != "" {
		t.Errorf("expected empty model, got %q", pr.Model)
	}
}

func TestParseIncrementalReads(t *testing.T) {
	dir := t.TempDir()
	projDir := filepath.Join(dir, "proj1")
	os.MkdirAll(projDir, 0755)

	line1 := `{"type":"assistant","sessionId":"s1","message":{"role":"assistant","model":"claude-opus-4-6-20260301","content":[],"usage":{"input_tokens":1000,"output_tokens":100,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}` + "\n"
	path := filepath.Join(projDir, "s1.jsonl")
	os.WriteFile(path, []byte(line1), 0644)

	parser := NewParser(dir)
	pr1 := parser.Poll()
	if pr1.Total.InputTokens != 1000 {
		t.Fatalf("first poll: input = %d, want 1000", pr1.Total.InputTokens)
	}

	// Append more data
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	line2 := `{"type":"assistant","sessionId":"s1","message":{"role":"assistant","model":"claude-opus-4-6-20260301","content":[],"usage":{"input_tokens":2000,"output_tokens":200,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}` + "\n"
	f.WriteString(line2)
	f.Close()

	pr2 := parser.Poll()
	// Second poll should only get new data
	if pr2.Total.InputTokens != 2000 {
		t.Errorf("second poll: input = %d, want 2000 (incremental)", pr2.Total.InputTokens)
	}
}

func TestParseReset(t *testing.T) {
	dir := t.TempDir()
	projDir := filepath.Join(dir, "proj1")
	os.MkdirAll(projDir, 0755)

	line := `{"type":"assistant","sessionId":"s1","message":{"role":"assistant","model":"claude-opus-4-6-20260301","content":[],"usage":{"input_tokens":1000,"output_tokens":100,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}` + "\n"
	os.WriteFile(filepath.Join(projDir, "s1.jsonl"), []byte(line), 0644)

	parser := NewParser(dir)
	parser.Poll() // first read

	parser.Reset()
	pr := parser.Poll() // should re-read everything
	if pr.Total.InputTokens != 1000 {
		t.Errorf("after reset: input = %d, want 1000", pr.Total.InputTokens)
	}
}

func TestParseEmptyDir(t *testing.T) {
	dir := t.TempDir()
	parser := NewParser(dir)
	pr := parser.Poll()
	if pr.Total.InputTokens != 0 {
		t.Errorf("expected 0 tokens for empty dir")
	}
	if pr.Model != "" {
		t.Errorf("expected empty model for empty dir")
	}
}

func TestParseMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	projDir := filepath.Join(dir, "proj1")
	os.MkdirAll(projDir, 0755)

	transcript := "not json\n{broken\n"
	os.WriteFile(filepath.Join(projDir, "s1.jsonl"), []byte(transcript), 0644)

	parser := NewParser(dir)
	pr := parser.Poll()
	if pr.Total.InputTokens != 0 {
		t.Errorf("expected 0 tokens for malformed json, got %d", pr.Total.InputTokens)
	}
}
