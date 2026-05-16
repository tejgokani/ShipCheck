package session

import (
	"testing"
	"time"
)

func TestParseCodexFile(t *testing.T) {
	since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		fixture       string
		wantNil       bool
		wantTokensIn  int64
		wantTokensOut int64
		wantFiles     int
		wantCost      float64
	}{
		{
			name:    "valid codex session — mixed prompt_tokens and input_tokens fields",
			fixture: testdataPath("codex_session.jsonl"),
			wantNil: false,
			// First entry uses prompt_tokens/completion_tokens; second uses input_tokens/output_tokens.
			wantTokensIn:  15000 + 7500,
			wantTokensOut: 3000 + 1500,
			wantFiles:     2,
			wantCost:      0.50 + 0.25,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			sess, err := parseCodexFile(tc.fixture, since)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantNil {
				if sess != nil {
					t.Fatal("expected nil session")
				}
				return
			}
			if sess == nil {
				t.Fatal("expected non-nil session")
			}
			if sess.Usage.InputTokens != tc.wantTokensIn {
				t.Errorf("InputTokens: got %d, want %d", sess.Usage.InputTokens, tc.wantTokensIn)
			}
			if sess.Usage.OutputTokens != tc.wantTokensOut {
				t.Errorf("OutputTokens: got %d, want %d", sess.Usage.OutputTokens, tc.wantTokensOut)
			}
			if len(sess.Files) != tc.wantFiles {
				t.Errorf("Files count: got %d, want %d", len(sess.Files), tc.wantFiles)
			}
			const epsilon = 0.001
			if diff := sess.TotalCost - tc.wantCost; diff < -epsilon || diff > epsilon {
				t.Errorf("TotalCost: got %.4f, want %.4f", sess.TotalCost, tc.wantCost)
			}
		})
	}
}

func TestParseCodexFile_SinceFiltering(t *testing.T) {
	since := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	sess, err := parseCodexFile(testdataPath("codex_session.jsonl"), since)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess != nil {
		t.Errorf("expected nil session when all entries before since")
	}
}

func TestCodexParser_ToolName(t *testing.T) {
	p := NewCodexParser()
	if p.ToolName() != "codex" {
		t.Errorf("ToolName: got %q, want %q", p.ToolName(), "codex")
	}
}
