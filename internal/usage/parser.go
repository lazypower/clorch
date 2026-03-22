package usage

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Parser struct {
	projectsDir string
	offsets     map[string]int64
}

func NewParser(projectsDir string) *Parser {
	return &Parser{
		projectsDir: projectsDir,
		offsets:     make(map[string]int64),
	}
}

type transcriptRecord struct {
	Type    string `json:"type"`
	Message *struct {
		Role  string `json:"role"`
		Model string `json:"model"`
		Usage *struct {
			InputTokens              int64 `json:"input_tokens"`
			OutputTokens             int64 `json:"output_tokens"`
			CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
			CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

// SessionTokens holds per-session token usage and model.
type SessionTokens struct {
	Tokens TokenUsage
	Model  string
}

// PollResult contains both aggregate and per-session usage data.
type PollResult struct {
	Total      TokenUsage
	Model      string
	PerSession map[string]SessionTokens
}

func (p *Parser) Poll() PollResult {
	result := PollResult{PerSession: make(map[string]SessionTokens)}

	for _, path := range p.discoverFiles() {
		tokens, model := p.parseFile(path)
		result.Total.InputTokens += tokens.InputTokens
		result.Total.OutputTokens += tokens.OutputTokens
		result.Total.CacheCreationTokens += tokens.CacheCreationTokens
		result.Total.CacheReadTokens += tokens.CacheReadTokens
		if model != "" {
			result.Model = model
		}

		base := filepath.Base(path)
		sessionID := strings.TrimSuffix(base, ".jsonl")
		if sessionID != base {
			result.PerSession[sessionID] = SessionTokens{Tokens: tokens, Model: model}
		}
	}
	return result
}

func (p *Parser) Reset() {
	p.offsets = make(map[string]int64)
}

func (p *Parser) discoverFiles() []string {
	today := time.Now().Truncate(24 * time.Hour)
	var files []string
	dirs, _ := filepath.Glob(filepath.Join(p.projectsDir, "*"))
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
				continue
			}
			info, err := entry.Info()
			if err != nil || info.ModTime().Before(today) {
				continue
			}
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}
	return files
}

func (p *Parser) parseFile(path string) (TokenUsage, string) {
	var tokens TokenUsage
	var model string

	f, err := os.Open(path)
	if err != nil {
		return tokens, model
	}
	defer f.Close()

	if offset := p.offsets[path]; offset > 0 {
		if _, err := f.Seek(offset, 0); err != nil {
			return tokens, model
		}
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, `"assistant"`) {
			continue
		}
		var rec transcriptRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue
		}
		if rec.Type != "assistant" || rec.Message == nil || rec.Message.Role != "assistant" || rec.Message.Usage == nil {
			continue
		}
		tokens.InputTokens += rec.Message.Usage.InputTokens
		tokens.OutputTokens += rec.Message.Usage.OutputTokens
		tokens.CacheCreationTokens += rec.Message.Usage.CacheCreationInputTokens
		tokens.CacheReadTokens += rec.Message.Usage.CacheReadInputTokens
		// LastInput overwrites per message — represents the most recent API call's context fill
		tokens.LastInput = rec.Message.Usage.InputTokens + rec.Message.Usage.CacheCreationInputTokens + rec.Message.Usage.CacheReadInputTokens
		if rec.Message.Model != "" {
			model = rec.Message.Model
		}
	}

	newOffset, _ := f.Seek(0, 1)
	p.offsets[path] = newOffset
	return tokens, model
}
