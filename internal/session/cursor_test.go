package session

import (
	"testing"
	"time"
)


func TestParseCursorFile(t *testing.T) {
	since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		fixture       string
		wantNil       bool
		wantTokensIn  int64
		wantTokensOut int64
		wantFiles     int
	}{
		{
			name:          "valid cursor session",
			fixture:       testdataPath("cursor_session.jsonl"),
			wantNil:       false,
			wantTokensIn:  8000 + 5000 + 2000,
			wantTokensOut: 1500 + 1000 + 400,
			wantFiles:     2, // Dashboard.tsx and db.ts
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			sess, err := parseCursorFile(tc.fixture, since)
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
	sess, err := parseCursorFile(testdataPath("cursor_session.jsonl"), since)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess != nil {
		t.Errorf("expected nil session when all entries before since")
	}
}

func TestParseCursorFile_DashboardTouchedTwice(t *testing.T) {
	since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	sess, err := parseCursorFile(testdataPath("cursor_session.jsonl"), since)
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
