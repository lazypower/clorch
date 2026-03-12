package usage

// TokenUsage holds aggregated token counts.
type TokenUsage struct {
	InputTokens         int64
	OutputTokens        int64
	CacheCreationTokens int64
	CacheReadTokens     int64
}

type ModelPrice struct {
	Input  float64
	Output float64
}

// SessionCost holds computed cost for a single session.
type SessionCost struct {
	Tokens TokenUsage
	Cost   float64
}

// UsageSummary is the aggregate sent to the TUI.
type UsageSummary struct {
	Tokens     TokenUsage
	Cost       float64
	BurnRate   float64
	PerSession map[string]SessionCost
}
