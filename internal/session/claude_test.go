package session

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// testdataPath returns the absolute path to a file in testdata/sessions/.
func testdataPath(name string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "sessions", name)
}

func TestParseClaudeFile(t *testing.T) {
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
			name:          "valid session",
			fixture:       testdataPath("claude_session.jsonl"),
			wantNil:       false,
			wantTokensIn:  21000 + 10500 + 10810,
			wantTokensOut: 4100 + 2050 + 2054,
			wantFiles:     2, // middleware.ts and routes.go
			wantCost:      0.42 + 0.21 + 0.21,
		},
		{
			name:          "malformed lines are skipped",
			fixture:       testdataPath("claude_malformed.jsonl"),
			wantNil:       false,
			wantTokensIn:  5000 + 2500,
			wantTokensOut: 1000 + 500,
			wantFiles:     1, // main.go
			wantCost:      0.10 + 0.05,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			sess, err := parseClaudeFile(tc.fixture, since)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantNil {
				if sess != nil {
					t.Fatal("expected nil session, got non-nil")
				}
				return
			}
			if sess == nil {
				t.Fatal("expected non-nil session, got nil")
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

func TestParseClaudeFile_SinceFiltering(t *testing.T) {
	// Set since to after all entries — should return nil session.
	since := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	sess, err := parseClaudeFile(testdataPath("claude_session.jsonl"), since)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess != nil {
		t.Errorf("expected nil session when all entries are before since, got session with %d tokens", sess.Usage.InputTokens)
	}
}

func TestParseClaudeFile_FileTouchCounts(t *testing.T) {
	since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	sess, err := parseClaudeFile(testdataPath("claude_session.jsonl"), since)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess == nil {
		t.Fatal("expected non-nil session")
	}

	fileMap := map[string]*SessionFile{}
	for i := range sess.Files {
		fileMap[sess.Files[i].Path] = &sess.Files[i]
	}

	// middleware.ts should be touched twice.
	mw, ok := fileMap["src/auth/middleware.ts"]
	if !ok {
		t.Fatal("expected src/auth/middleware.ts in files")
	}
	if mw.TouchCount != 2 {
		t.Errorf("middleware.ts TouchCount: got %d, want 2", mw.TouchCount)
	}

	// routes.go should be touched once.
	rt, ok := fileMap["src/api/routes.go"]
	if !ok {
		t.Fatal("expected src/api/routes.go in files")
	}
	if rt.TouchCount != 1 {
		t.Errorf("routes.go TouchCount: got %d, want 1", rt.TouchCount)
	}
}

func TestParseClaudeFile_MissingFile(t *testing.T) {
	since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := parseClaudeFile(testdataPath("does_not_exist.jsonl"), since)
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestClaudeParser_ToolName(t *testing.T) {
	p := NewClaudeParser()
	if p.ToolName() != "claude-code" {
		t.Errorf("ToolName: got %q, want %q", p.ToolName(), "claude-code")
	}
}
