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
	tokens, model := parser.Poll()

	if tokens.InputTokens != 3000 {
		t.Errorf("input tokens = %d, want 3000", tokens.InputTokens)
	}
	if tokens.OutputTokens != 500 {
		t.Errorf("output tokens = %d, want 500", tokens.OutputTokens)
	}
	if tokens.CacheCreationTokens != 100 {
		t.Errorf("cache creation = %d, want 100", tokens.CacheCreationTokens)
	}
	if tokens.CacheReadTokens != 500 {
		t.Errorf("cache read = %d, want 500", tokens.CacheReadTokens)
	}
	if model != "claude-opus-4-6-20260301" {
		t.Errorf("model = %q, want claude-opus-4-6-20260301", model)
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
	tokens, model := parser.Poll()

	if tokens.InputTokens != 0 || tokens.OutputTokens != 0 {
		t.Errorf("expected zero tokens for non-assistant records, got in=%d out=%d",
			tokens.InputTokens, tokens.OutputTokens)
	}
	if model != "" {
		t.Errorf("expected empty model, got %q", model)
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
	tokens1, _ := parser.Poll()
	if tokens1.InputTokens != 1000 {
		t.Fatalf("first poll: input = %d, want 1000", tokens1.InputTokens)
	}

	// Append more data
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	line2 := `{"type":"assistant","sessionId":"s1","message":{"role":"assistant","model":"claude-opus-4-6-20260301","content":[],"usage":{"input_tokens":2000,"output_tokens":200,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}` + "\n"
	f.WriteString(line2)
	f.Close()

	tokens2, _ := parser.Poll()
	// Second poll should only get new data
	if tokens2.InputTokens != 2000 {
		t.Errorf("second poll: input = %d, want 2000 (incremental)", tokens2.InputTokens)
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
	tokens, _ := parser.Poll() // should re-read everything
	if tokens.InputTokens != 1000 {
		t.Errorf("after reset: input = %d, want 1000", tokens.InputTokens)
	}
}

func TestParseEmptyDir(t *testing.T) {
	dir := t.TempDir()
	parser := NewParser(dir)
	tokens, model := parser.Poll()
	if tokens.InputTokens != 0 {
		t.Errorf("expected 0 tokens for empty dir")
	}
	if model != "" {
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
	tokens, _ := parser.Poll()
	if tokens.InputTokens != 0 {
		t.Errorf("expected 0 tokens for malformed json, got %d", tokens.InputTokens)
	}
}
