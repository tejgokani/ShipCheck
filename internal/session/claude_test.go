package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// claudeSessionJSONL is an inline fixture representing a valid Claude Code session.
const claudeSessionJSONL = `{"type":"usage","cost":0.42,"tokensIn":21000,"tokensOut":4100,"cacheReads":0,"cacheWrites":0,"timestamp":"2026-05-01T10:00:00Z","file":"src/auth/middleware.ts","model":"claude-sonnet-4-6"}
{"type":"usage","cost":0.21,"tokensIn":10500,"tokensOut":2050,"cacheReads":0,"cacheWrites":0,"timestamp":"2026-05-01T10:05:00Z","file":"src/api/routes.go","model":"claude-sonnet-4-6"}
{"type":"usage","cost":0.21,"tokensIn":10810,"tokensOut":2054,"cacheReads":0,"cacheWrites":0,"timestamp":"2026-05-01T10:10:00Z","file":"src/auth/middleware.ts","model":"claude-sonnet-4-6"}
`

// claudeMalformedJSONL has some invalid lines that should be skipped gracefully.
const claudeMalformedJSONL = `{"type":"usage","cost":0.10,"tokensIn":5000,"tokensOut":1000,"cacheReads":0,"cacheWrites":0,"timestamp":"2026-05-01T09:00:00Z","file":"main.go","model":"claude-sonnet-4-6"}
this is not valid json
{"broken":
{"type":"usage","cost":0.05,"tokensIn":2500,"tokensOut":500,"cacheReads":0,"cacheWrites":0,"timestamp":"2026-05-01T09:05:00Z","file":"main.go","model":"claude-sonnet-4-6"}
`

func writeTempJSONL(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writeTempJSONL: %v", err)
	}
	return path
}

func TestParseClaudeFile(t *testing.T) {
	since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		content       string
		wantNil       bool
		wantTokensIn  int64
		wantTokensOut int64
		wantFiles     int
		wantCost      float64
	}{
		{
			name:          "valid session",
			content:       claudeSessionJSONL,
			wantNil:       false,
			wantTokensIn:  21000 + 10500 + 10810,
			wantTokensOut: 4100 + 2050 + 2054,
			wantFiles:     2,
			wantCost:      0.42 + 0.21 + 0.21,
		},
		{
			name:          "malformed lines are skipped",
			content:       claudeMalformedJSONL,
			wantNil:       false,
			wantTokensIn:  5000 + 2500,
			wantTokensOut: 1000 + 500,
			wantFiles:     1,
			wantCost:      0.10 + 0.05,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			path := writeTempJSONL(t, tc.content)
			sess, err := parseClaudeFile(path, since)
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
	path := writeTempJSONL(t, claudeSessionJSONL)
	sess, err := parseClaudeFile(path, since)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess != nil {
		t.Errorf("expected nil session when all entries are before since, got session with %d tokens", sess.Usage.InputTokens)
	}
}

func TestParseClaudeFile_FileTouchCounts(t *testing.T) {
	since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	path := writeTempJSONL(t, claudeSessionJSONL)
	sess, err := parseClaudeFile(path, since)
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

	mw, ok := fileMap["src/auth/middleware.ts"]
	if !ok {
		t.Fatal("expected src/auth/middleware.ts in files")
	}
	if mw.TouchCount != 2 {
		t.Errorf("middleware.ts TouchCount: got %d, want 2", mw.TouchCount)
	}

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
	_, err := parseClaudeFile("/nonexistent/path/session.jsonl", since)
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
