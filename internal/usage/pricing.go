package usage

import "strings"

var pricing = map[string]ModelPrice{
	"opus-4-6":   {Input: 15.0, Output: 75.0},
	"opus-4-5":   {Input: 15.0, Output: 75.0},
	"sonnet-4-6": {Input: 3.0, Output: 15.0},
	"haiku-4-5":  {Input: 0.80, Output: 4.0},
}

func CalculateCost(tokens TokenUsage, model string) float64 {
	key := resolveModelKey(model)
	price, ok := pricing[key]
	if !ok {
		price = pricing["opus-4-6"]
	}
	inputCost := float64(tokens.InputTokens+tokens.CacheCreationTokens) * price.Input / 1_000_000
	outputCost := float64(tokens.OutputTokens) * price.Output / 1_000_000
	cacheCost := float64(tokens.CacheReadTokens) * price.Input * 0.1 / 1_000_000
	return inputCost + outputCost + cacheCost
}

func resolveModelKey(model string) string {
	model = strings.TrimPrefix(model, "claude-")
	parts := strings.Split(model, "-")
	for i := len(parts) - 1; i >= 0; i-- {
		if len(parts[i]) >= 8 && isAllDigits(parts[i]) {
			parts = parts[:i]
			break
		}
	}
	return strings.Join(parts, "-")
}

func isAllDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}
