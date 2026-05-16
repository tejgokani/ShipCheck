package cost

import (
	"testing"

	"github.com/tejgokani/shipcheck/internal/session"
)

func TestBuildBurnReport(t *testing.T) {
	sessions := []session.Session{
		{
			Tool: "claude-code",
			Files: []session.SessionFile{
				{Path: "src/auth.go", TokensSpent: 5000, TouchCount: 3},
				{Path: "src/main.go", TokensSpent: 2000, TouchCount: 1},
			},
		},
		{
			Tool: "claude-code",
			Files: []session.SessionFile{
				{Path: "src/auth.go", TokensSpent: 3000, TouchCount: 2},
				{Path: "src/db.go", TokensSpent: 8000, TouchCount: 4},
			},
		},
	}

	got := BuildBurnReport(sessions)

	if len(got) != 3 {
		t.Fatalf("expected 3 files, got %d", len(got))
	}

	// Should be sorted descending by TokensSpent.
	// src/db.go: 8000, src/auth.go: 8000 (5000+3000), src/main.go: 2000
	// db.go and auth.go are tied at 8000; they sort alphabetically.

	// Check totals.
	totals := map[string]FileBurn{}
	for _, b := range got {
		totals[b.Path] = b
	}

	authBurn, ok := totals["src/auth.go"]
	if !ok {
		t.Fatal("expected src/auth.go in burn report")
	}
	if authBurn.TokensSpent != 8000 {
		t.Errorf("auth.go TokensSpent: got %d, want 8000", authBurn.TokensSpent)
	}
	if authBurn.TouchCount != 5 {
		t.Errorf("auth.go TouchCount: got %d, want 5", authBurn.TouchCount)
	}

	dbBurn, ok := totals["src/db.go"]
	if !ok {
		t.Fatal("expected src/db.go in burn report")
	}
	if dbBurn.TokensSpent != 8000 {
		t.Errorf("db.go TokensSpent: got %d, want 8000", dbBurn.TokensSpent)
	}

	mainBurn, ok := totals["src/main.go"]
	if !ok {
		t.Fatal("expected src/main.go in burn report")
	}
	if mainBurn.TokensSpent != 2000 {
		t.Errorf("main.go TokensSpent: got %d, want 2000", mainBurn.TokensSpent)
	}

	// Verify sort order: last entry should be main.go (least tokens).
	if got[len(got)-1].Path != "src/main.go" {
		t.Errorf("last entry should be src/main.go (lowest burn), got %q", got[len(got)-1].Path)
	}
}

func TestBuildBurnReport_EmptySessions(t *testing.T) {
	got := BuildBurnReport(nil)
	if len(got) != 0 {
		t.Errorf("expected empty result for nil sessions, got %d entries", len(got))
	}
}

func TestBuildBurnReport_SkipsEmptyPaths(t *testing.T) {
	sessions := []session.Session{
		{
			Files: []session.SessionFile{
				{Path: "", TokensSpent: 100, TouchCount: 1},
				{Path: "real.go", TokensSpent: 200, TouchCount: 2},
			},
		},
	}
	got := BuildBurnReport(sessions)
	if len(got) != 1 {
		t.Errorf("expected 1 file (empty path skipped), got %d", len(got))
	}
	if got[0].Path != "real.go" {
		t.Errorf("expected real.go, got %q", got[0].Path)
	}
}
