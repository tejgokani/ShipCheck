package session

import (
	"testing"
	"time"
)

const cursorSessionJSONL = `{"model":"gpt-4o","timestamp":"2026-05-01T10:00:00Z","cost":0.05,"usage":{"input_tokens":8000,"output_tokens":1500},"file":"src/components/Dashboard.tsx"}
{"model":"gpt-4o","timestamp":"2026-05-01T10:05:00Z","cost":0.03,"usage":{"input_tokens":5000,"output_tokens":1000},"file":"src/utils/db.ts"}
{"model":"gpt-4o","timestamp":"2026-05-01T10:10:00Z","cost":0.01,"usage":{"input_tokens":2000,"output_tokens":400},"file":"src/components/Dashboard.tsx"}
`

func TestParseCursorFile(t *testing.T) {
	since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		content       string
		wantNil       bool
		wantTokensIn  int64
		wantTokensOut int64
		wantFiles     int
	}{
		{
			name:          "valid cursor session",
			content:       cursorSessionJSONL,
			wantNil:       false,
			wantTokensIn:  8000 + 5000 + 2000,
			wantTokensOut: 1500 + 1000 + 400,
			wantFiles:     2,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			path := writeTempJSONL(t, tc.content)
			sess, err := parseCursorFile(path, since)
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
		})
	}
}

func TestParseCursorFile_SinceFiltering(t *testing.T) {
	since := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	path := writeTempJSONL(t, cursorSessionJSONL)
	sess, err := parseCursorFile(path, since)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess != nil {
		t.Errorf("expected nil session when all entries before since")
	}
}

func TestParseCursorFile_DashboardTouchedTwice(t *testing.T) {
	since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	path := writeTempJSONL(t, cursorSessionJSONL)
	sess, err := parseCursorFile(path, since)
	if err != nil || sess == nil {
		t.Fatalf("setup: err=%v, sess=%v", err, sess)
	}
	fileMap := map[string]*SessionFile{}
	for i := range sess.Files {
		fileMap[sess.Files[i].Path] = &sess.Files[i]
	}
	dash, ok := fileMap["src/components/Dashboard.tsx"]
	if !ok {
		t.Fatal("expected Dashboard.tsx in files")
	}
	if dash.TouchCount != 2 {
		t.Errorf("Dashboard.tsx TouchCount: got %d, want 2", dash.TouchCount)
	}
}

func TestCursorParser_ToolName(t *testing.T) {
	p := NewCursorParser()
	if p.ToolName() != "cursor" {
		t.Errorf("ToolName: got %q, want %q", p.ToolName(), "cursor")
	}
}
