package usage

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Parser struct {
	projectsDir string
	offsets     map[string]int64
	files       map[string]SessionTokens
}

func NewParser(projectsDir string) *Parser {
	return &Parser{
		projectsDir: projectsDir,
		offsets:     make(map[string]int64),
		files:       make(map[string]SessionTokens),
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

	files := p.discoverFiles()
	active := make(map[string]bool)
	for _, path := range files {
		active[path] = true
		p.parseFile(path)
	}
	for path := range p.files {
		if !active[path] {
			delete(p.files, path)
			delete(p.offsets, path)
		}
	}
	for _, path := range files {
		st, ok := p.files[path]
		if !ok {
			continue
		}
		result.Total.InputTokens += st.Tokens.InputTokens
		result.Total.OutputTokens += st.Tokens.OutputTokens
		result.Total.CacheCreationTokens += st.Tokens.CacheCreationTokens
		result.Total.CacheReadTokens += st.Tokens.CacheReadTokens
		if st.Tokens.LastInput > 0 {
			result.Total.LastInput = st.Tokens.LastInput
		}
		if st.Model != "" {
			result.Model = st.Model
		}

		base := filepath.Base(path)
		sessionID := strings.TrimSuffix(base, ".jsonl")
		if sessionID != base {
			result.PerSession[sessionID] = st
		}
	}
	return result
}

func (p *Parser) Reset() {
	p.offsets = make(map[string]int64)
	p.files = make(map[string]SessionTokens)
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
	sort.Strings(files)
	return files
}

func (p *Parser) parseFile(path string) {
	var tokens TokenUsage
	var model string

	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	if info, err := f.Stat(); err == nil && info.Size() < p.offsets[path] {
		p.offsets[path] = 0
		delete(p.files, path)
	}

	if offset := p.offsets[path]; offset > 0 {
		if _, err := f.Seek(offset, 0); err != nil {
			return
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

	st := p.files[path]
	st.Tokens.InputTokens += tokens.InputTokens
	st.Tokens.OutputTokens += tokens.OutputTokens
	st.Tokens.CacheCreationTokens += tokens.CacheCreationTokens
	st.Tokens.CacheReadTokens += tokens.CacheReadTokens
	if tokens.LastInput > 0 {
		st.Tokens.LastInput = tokens.LastInput
	}
	if model != "" {
		st.Model = model
	}
	p.files[path] = st
}
