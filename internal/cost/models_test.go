package cost

import "testing"

func TestPricingFor_KnownModels(t *testing.T) {
	tests := []struct {
		modelID    string
		wantInput  float64
		wantOutput float64
	}{
		{"claude-opus-4-6", 15.00, 75.00},
		{"claude-sonnet-4-6", 3.00, 15.00},
		{"claude-haiku-4-5", 0.80, 4.00},
		{"gpt-4o", 2.50, 10.00},
		{"gpt-4o-mini", 0.15, 0.60},
		{"o3", 10.00, 40.00},
		{"gemini-2.5-pro", 1.25, 10.00},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.modelID, func(t *testing.T) {
			p := PricingFor(tc.modelID)
			if p.Input != tc.wantInput {
				t.Errorf("Input: got %v, want %v", p.Input, tc.wantInput)
			}
			if p.Output != tc.wantOutput {
				t.Errorf("Output: got %v, want %v", p.Output, tc.wantOutput)
			}
		})
	}
}

func TestPricingFor_UnknownModelFallback(t *testing.T) {
	p := PricingFor("unknown-model-xyz")
	sonnet := ModelPricing[defaultModel]
	if p.Input != sonnet.Input || p.Output != sonnet.Output {
		t.Errorf("unknown model should fall back to %s pricing; got input=%.2f output=%.2f", defaultModel, p.Input, p.Output)
	}
}
