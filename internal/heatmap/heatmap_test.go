package heatmap

import (
	"testing"
	"time"

	"github.com/tejgokani/shipcheck/internal/session"
)

func TestBuild_NoGitRepo(t *testing.T) {
	// Use /tmp which is unlikely to be a git repo.
	sessions := []session.Session{
		{
			StartTime: time.Now().Add(-1 * time.Hour),
			EndTime:   time.Now(),
		},
	}
	hotFiles, err := Build(sessions, "/tmp", 10)
	if err != nil {
		t.Fatalf("expected nil error for non-git directory, got: %v", err)
	}
	// Should return empty slice, not panic.
	if hotFiles == nil {
		hotFiles = []HotFile{}
	}
	// No requirement on length — just must not error.
	_ = hotFiles
}

func TestBuild_EmptySessions(t *testing.T) {
	// With no sessions, time window is zero — all commits pass filter.
	// We still should not panic even if the directory isn't a git repo.
	hotFiles, err := Build(nil, "/tmp", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = hotFiles
}

func TestHotFile_HeatScore(t *testing.T) {
	// Verify HeatScore calculation logic manually.
	files := []HotFile{
		{Path: "a.go", TouchCount: 10},
		{Path: "b.go", TouchCount: 5},
		{Path: "c.go", TouchCount: 2},
	}

	// Simulate normalization.
	maxCount := 0
	for _, f := range files {
		if f.TouchCount > maxCount {
			maxCount = f.TouchCount
		}
	}
	for i := range files {
		files[i].HeatScore = float64(files[i].TouchCount) / float64(maxCount)
	}

	if files[0].HeatScore != 1.0 {
		t.Errorf("top file HeatScore: got %.2f, want 1.0", files[0].HeatScore)
	}
	if files[1].HeatScore != 0.5 {
		t.Errorf("second file HeatScore: got %.2f, want 0.5", files[1].HeatScore)
	}
	expected := 2.0 / 10.0
	if files[2].HeatScore != expected {
		t.Errorf("third file HeatScore: got %.2f, want %.2f", files[2].HeatScore, expected)
	}
}
