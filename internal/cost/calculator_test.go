package cost

import (
	"testing"

	"github.com/tejgokani/shipcheck/internal/session"
)

func TestComputeCost(t *testing.T) {
	tests := []struct {
		name    string
		usage   session.TokenUsage
		modelID string
		want    float64
	}{
		{
			name: "claude-sonnet-4-6 basic",
			usage: session.TokenUsage{
				InputTokens:  1_000_000,
				OutputTokens: 1_000_000,
				CacheReads:   0,
			},
			modelID: "claude-sonnet-4-6",
			want:    3.00 + 15.00, // 3.00 input + 15.00 output
		},
		{
			name: "claude-sonnet-4-6 with cache reads",
			usage: session.TokenUsage{
				InputTokens:  0,
				OutputTokens: 0,
				CacheReads:   1_000_000,
			},
			modelID: "claude-sonnet-4-6",
			want:    0.30, // cache read rate
		},
		{
			name: "unknown model falls back to sonnet",
			usage: session.TokenUsage{
				InputTokens:  1_000_000,
				OutputTokens: 0,
			},
			modelID: "gpt-99-ultra",
			want:    3.00, // sonnet input rate
		},
		{
			name: "gpt-4o",
			usage: session.TokenUsage{
				InputTokens:  500_000,
				OutputTokens: 200_000,
			},
			modelID: "gpt-4o",
			want:    2.50*0.5 + 10.00*0.2,
		},
		{
			name: "zero usage",
			usage: session.TokenUsage{},
			modelID: "claude-opus-4-6",
			want:    0,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeCost(tc.usage, tc.modelID)
			const epsilon = 0.0001
			if diff := got - tc.want; diff < -epsilon || diff > epsilon {
				t.Errorf("ComputeCost(%v, %q) = %.6f, want %.6f", tc.usage, tc.modelID, got, tc.want)
			}
		})
	}
}

func TestCacheSavings(t *testing.T) {
	tests := []struct {
		name       string
		cacheReads int64
		modelID    string
		want       float64
	}{
		{
			name:       "sonnet 1M cache reads",
			cacheReads: 1_000_000,
			modelID:    "claude-sonnet-4-6",
			// savings = (3.00 - 0.30) = 2.70
			want: 2.70,
		},
		{
			name:       "zero cache reads",
			cacheReads: 0,
			modelID:    "claude-sonnet-4-6",
			want:       0,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := CacheSavings(tc.cacheReads, tc.modelID)
			const epsilon = 0.0001
			if diff := got - tc.want; diff < -epsilon || diff > epsilon {
				t.Errorf("CacheSavings(%d, %q) = %.6f, want %.6f", tc.cacheReads, tc.modelID, got, tc.want)
			}
		})
	}
}
