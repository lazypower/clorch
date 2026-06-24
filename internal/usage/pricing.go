package usage

import "strings"

var pricing = map[string]ModelPrice{
	"fable-5":  claudePrice(10.0, 50.0),
	"mythos-5": claudePrice(10.0, 50.0),

	"opus-4-8": claudePrice(5.0, 25.0),
	"opus-4-7": claudePrice(5.0, 25.0),
	"opus-4-6": claudePrice(5.0, 25.0),
	"opus-4-5": claudePrice(5.0, 25.0),
	"opus-4-1": claudePrice(15.0, 75.0),
	"opus-4":   claudePrice(15.0, 75.0),

	"sonnet-4-6": claudePrice(3.0, 15.0),
	"sonnet-4-5": claudePrice(3.0, 15.0),
	"sonnet-4":   claudePrice(3.0, 15.0),

	"haiku-4-5": claudePrice(1.0, 5.0),
	"haiku-3-5": claudePrice(0.80, 4.0),
}

func CalculateCost(tokens TokenUsage, model string) float64 {
	price := priceForModel(model)
	inputCost := float64(tokens.InputTokens) * price.Input / 1_000_000
	cacheWriteCost := float64(tokens.CacheCreationTokens) * price.CacheWrite / 1_000_000
	cacheReadCost := float64(tokens.CacheReadTokens) * price.CacheRead / 1_000_000
	outputCost := float64(tokens.OutputTokens) * price.Output / 1_000_000
	return inputCost + cacheWriteCost + cacheReadCost + outputCost
}

func claudePrice(input, output float64) ModelPrice {
	return ModelPrice{
		Input:      input,
		CacheWrite: input * 1.25,
		CacheRead:  input * 0.1,
		Output:     output,
	}
}

func priceForModel(model string) ModelPrice {
	key := resolveModelKey(model)
	if price, ok := pricing[key]; ok {
		return price
	}
	switch {
	case strings.HasPrefix(key, "fable-"), strings.HasPrefix(key, "mythos-"):
		return pricing["fable-5"]
	case strings.HasPrefix(key, "opus-"):
		return pricing["opus-4-8"]
	case strings.HasPrefix(key, "sonnet-"):
		return pricing["sonnet-4-6"]
	case strings.HasPrefix(key, "haiku-"):
		return pricing["haiku-4-5"]
	default:
		return pricing["opus-4-8"]
	}
}

func resolveModelKey(model string) string {
	model = strings.ToLower(strings.TrimSpace(model))
	model = strings.TrimPrefix(model, "claude-")
	parts := strings.Split(model, "-")
	for i := len(parts) - 1; i >= 0; i-- {
		if len(parts[i]) >= 8 && isAllDigits(parts[i]) {
			parts = parts[:i]
			break
		}
	}
	if len(parts) >= 3 && isClaude3LegacyFamily(parts[len(parts)-1]) {
		family := parts[len(parts)-1]
		version := strings.Join(parts[:len(parts)-1], "-")
		return family + "-" + version
	}
	return strings.Join(parts, "-")
}

func isClaude3LegacyFamily(s string) bool {
	return s == "opus" || s == "sonnet" || s == "haiku"
}

func isAllDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}
