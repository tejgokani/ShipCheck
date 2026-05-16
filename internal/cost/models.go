// Package cost provides token-to-dollar cost calculation for agentic sessions.
package cost

// ModelPrice holds the pricing per million tokens for a specific model.
type ModelPrice struct {
	Input     float64 // cost per million input tokens (USD)
	Output    float64 // cost per million output tokens (USD)
	CacheRead float64 // cost per million cache-read tokens (USD)
}

// ModelPricing maps model IDs to their per-million-token pricing.
// Keep this table updated on each release — pricing changes frequently.
var ModelPricing = map[string]ModelPrice{
	"claude-opus-4-6":   {Input: 15.00, Output: 75.00, CacheRead: 1.50},
	"claude-sonnet-4-6": {Input: 3.00, Output: 15.00, CacheRead: 0.30},
	"claude-haiku-4-5":  {Input: 0.80, Output: 4.00, CacheRead: 0.08},
	"gpt-4o":            {Input: 2.50, Output: 10.00, CacheRead: 1.25},
	"gpt-4o-mini":       {Input: 0.15, Output: 0.60, CacheRead: 0.075},
	"o3":                {Input: 10.00, Output: 40.00, CacheRead: 2.50},
	"gemini-2.5-pro":    {Input: 1.25, Output: 10.00, CacheRead: 0.31},
}

// defaultModel is used when the model ID is unknown or missing.
const defaultModel = "claude-sonnet-4-6"

// PricingFor returns the ModelPrice for the given model ID.
// If the model is unknown, it returns the default (claude-sonnet-4-6) pricing.
func PricingFor(modelID string) ModelPrice {
	if p, ok := ModelPricing[modelID]; ok {
		return p
	}
	return ModelPricing[defaultModel]
}
