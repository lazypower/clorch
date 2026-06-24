package usage

import (
	"math"
	"testing"
)

func approxEqual(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}

func TestResolveModelKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"claude-opus-4-8-20260601", "opus-4-8"},
		{"claude-opus-4-6-20260301", "opus-4-6"},
		{"claude-sonnet-4-6-20260301", "sonnet-4-6"},
		{"claude-haiku-4-5-20251001", "haiku-4-5"},
		{"claude-3-5-haiku-20241022", "haiku-3-5"},
		{"claude-opus-4-5-20260301", "opus-4-5"},
		{"opus-4-6", "opus-4-6"},
		{" CLAUDE-SONNET-4-5-20250929 ", "sonnet-4-5"},
		{"", ""},
	}
	for _, tt := range tests {
		got := resolveModelKey(tt.input)
		if got != tt.want {
			t.Errorf("resolveModelKey(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCalculateCostOpus(t *testing.T) {
	tokens := TokenUsage{
		InputTokens:  1_000_000,
		OutputTokens: 100_000,
	}
	cost := CalculateCost(tokens, "claude-opus-4-6-20260301")
	// 1M input * $5/M + 100K output * $25/M = $5 + $2.50 = $7.50
	if !approxEqual(cost, 7.50, 0.01) {
		t.Errorf("opus cost = %.2f, want 7.50", cost)
	}
}

func TestCalculateCostSonnet(t *testing.T) {
	tokens := TokenUsage{
		InputTokens:  1_000_000,
		OutputTokens: 100_000,
	}
	cost := CalculateCost(tokens, "claude-sonnet-4-6-20260301")
	// 1M * $3/M + 100K * $15/M = $3 + $1.50 = $4.50
	if !approxEqual(cost, 4.50, 0.01) {
		t.Errorf("sonnet cost = %.2f, want 4.50", cost)
	}
}

func TestCalculateCostWithCache(t *testing.T) {
	tokens := TokenUsage{
		InputTokens:         500_000,
		OutputTokens:        100_000,
		CacheCreationTokens: 200_000,
		CacheReadTokens:     300_000,
	}
	cost := CalculateCost(tokens, "claude-opus-4-6-20260301")
	// Input: 500K * $5/M = $2.50
	// 5m cache writes: 200K * $6.25/M = $1.25
	// Cache reads: 300K * $0.50/M = $0.15
	// Output: 100K * $25/M = $2.50
	// Total = $6.40
	if !approxEqual(cost, 6.40, 0.01) {
		t.Errorf("cached cost = %.2f, want 6.40", cost)
	}
}

func TestCalculateCostHaiku45(t *testing.T) {
	tokens := TokenUsage{
		InputTokens:  1_000_000,
		OutputTokens: 100_000,
	}
	cost := CalculateCost(tokens, "claude-haiku-4-5-20251001")
	// 1M input * $1/M + 100K output * $5/M = $1 + $0.50 = $1.50
	if !approxEqual(cost, 1.50, 0.01) {
		t.Errorf("haiku cost = %.2f, want 1.50", cost)
	}
}

func TestCalculateCostUnknownModel(t *testing.T) {
	tokens := TokenUsage{InputTokens: 1_000_000}
	// Unknown model defaults to current Opus pricing.
	cost := CalculateCost(tokens, "claude-unknown-99")
	expected := CalculateCost(tokens, "claude-opus-4-8-20260601")
	if !approxEqual(cost, expected, 0.01) {
		t.Errorf("unknown model cost = %.2f, want %.2f (current opus default)", cost, expected)
	}
}

func TestCalculateCostZeroTokens(t *testing.T) {
	cost := CalculateCost(TokenUsage{}, "claude-opus-4-6-20260301")
	if cost != 0 {
		t.Errorf("zero tokens cost = %.2f, want 0", cost)
	}
}

func TestIsAllDigits(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"20260301", true},
		{"12345678", true},
		{"", false},
		{"abc", false},
		{"123abc", false},
	}
	for _, tt := range tests {
		if got := isAllDigits(tt.input); got != tt.want {
			t.Errorf("isAllDigits(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
