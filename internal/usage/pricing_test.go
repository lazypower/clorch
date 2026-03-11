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
		{"claude-opus-4-6-20260301", "opus-4-6"},
		{"claude-sonnet-4-6-20260301", "sonnet-4-6"},
		{"claude-haiku-4-5-20251001", "haiku-4-5"},
		{"claude-opus-4-5-20260301", "opus-4-5"},
		{"opus-4-6", "opus-4-6"},
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
	// 1M input * $15/M + 100K output * $75/M = $15 + $7.50 = $22.50
	if !approxEqual(cost, 22.50, 0.01) {
		t.Errorf("opus cost = %.2f, want 22.50", cost)
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
	// Input: (500K + 200K) * $15/M = $10.50
	// Output: 100K * $75/M = $7.50
	// Cache reads: 300K * $15/M * 0.1 = $0.45
	// Total = $18.45
	if !approxEqual(cost, 18.45, 0.01) {
		t.Errorf("cached cost = %.2f, want 18.45", cost)
	}
}

func TestCalculateCostUnknownModel(t *testing.T) {
	tokens := TokenUsage{InputTokens: 1_000_000}
	// Unknown model defaults to opus pricing
	cost := CalculateCost(tokens, "claude-unknown-99")
	expected := CalculateCost(tokens, "claude-opus-4-6-20260301")
	if !approxEqual(cost, expected, 0.01) {
		t.Errorf("unknown model cost = %.2f, want %.2f (opus default)", cost, expected)
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
