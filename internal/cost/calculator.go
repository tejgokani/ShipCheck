package cost

import (
	"github.com/tejgokani/shipcheck/internal/session"
)

// ComputeCost calculates the dollar cost for a given token usage and model.
// Falls back to claude-sonnet-4-6 pricing when the model is unknown.
func ComputeCost(usage session.TokenUsage, modelID string) float64 {
	p := PricingFor(modelID)
	return tokensToDollars(usage.InputTokens, p.Input) +
		tokensToDollars(usage.OutputTokens, p.Output) +
		tokensToDollars(usage.CacheReads, p.CacheRead)
}

// tokensToDollars converts a token count to dollars using the given per-million rate.
func tokensToDollars(tokens int64, perMillion float64) float64 {
	return float64(tokens) / 1_000_000 * perMillion
}

// CacheSavings estimates how much was saved by cache hits relative to input pricing.
// Cache reads are charged at a lower rate; this returns the difference.
func CacheSavings(cacheReads int64, modelID string) float64 {
	p := PricingFor(modelID)
	// Savings = cost if cache hits were billed as inputs minus actual cache-read cost.
	fullCost := tokensToDollars(cacheReads, p.Input)
	cacheCost := tokensToDollars(cacheReads, p.CacheRead)
	savings := fullCost - cacheCost
	if savings < 0 {
		return 0
	}
	return savings
}
