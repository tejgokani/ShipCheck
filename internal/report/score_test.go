package report

import (
	"testing"

	"github.com/tejgokani/shipcheck/internal/security"
)

func makeFindings(counts map[security.Severity]int) []security.Finding {
	var out []security.Finding
	for sev, n := range counts {
		for i := 0; i < n; i++ {
			out = append(out, security.Finding{Severity: sev, Rule: "TEST"})
		}
	}
	return out
}

func TestComputeScore_NoFindings(t *testing.T) {
	r := ComputeScore(nil)
	// base 100 + 5 (no critical) + 3 (no high) = 108, capped at 100
	if r.Score != 100 {
		t.Errorf("expected 100, got %d", r.Score)
	}
	if r.Label != "CLEAN" {
		t.Errorf("expected CLEAN, got %s", r.Label)
	}
}

func TestComputeScore_OneCritical(t *testing.T) {
	findings := makeFindings(map[security.Severity]int{security.Critical: 1})
	r := ComputeScore(findings)
	// 100 - 15 = 85; no zero-critical bonus; no zero-high bonus
	if r.Score != 85 {
		t.Errorf("expected 85, got %d", r.Score)
	}
	if r.Label != "GOOD" {
		t.Errorf("expected GOOD, got %s", r.Label)
	}
}

func TestComputeScore_CriticalDeductionCapped(t *testing.T) {
	// 5 criticals = 75 deducted but capped at 60
	findings := makeFindings(map[security.Severity]int{security.Critical: 5})
	r := ComputeScore(findings)
	if r.Score != 40 {
		t.Errorf("expected 40, got %d", r.Score)
	}
	if r.Label != "RISKY" {
		t.Errorf("expected RISKY, got %s", r.Label)
	}
}

func TestComputeScore_FloorAtZero(t *testing.T) {
	findings := makeFindings(map[security.Severity]int{
		security.Critical: 10,
		security.High:     10,
		security.Medium:   10,
		security.Low:      10,
	})
	r := ComputeScore(findings)
	if r.Score != 0 {
		t.Errorf("expected floor 0, got %d", r.Score)
	}
	if r.Label != "DANGER" {
		t.Errorf("expected DANGER, got %s", r.Label)
	}
}

func TestComputeScore_ZeroCriticalBonus(t *testing.T) {
	// 1 high = -8; no criticals = +5; has high so no +3
	// 100 - 8 + 5 = 97
	findings := makeFindings(map[security.Severity]int{security.High: 1})
	r := ComputeScore(findings)
	if r.Score != 97 {
		t.Errorf("expected 97, got %d", r.Score)
	}
}

func TestComputeScore_BothBonuses(t *testing.T) {
	// 1 medium = -3; no criticals +5; no highs +3 → 100 - 3 + 5 + 3 = 105 → 100
	findings := makeFindings(map[security.Severity]int{security.Medium: 1})
	r := ComputeScore(findings)
	if r.Score != 100 {
		t.Errorf("expected 100 (capped), got %d", r.Score)
	}
}

func TestLabelBoundaries(t *testing.T) {
	cases := []struct {
		score int
		want  string
	}{
		{100, "CLEAN"},
		{90, "CLEAN"},
		{89, "GOOD"},
		{70, "GOOD"},
		{69, "REVIEW"},
		{50, "REVIEW"},
		{49, "RISKY"},
		{30, "RISKY"},
		{29, "DANGER"},
		{0, "DANGER"},
	}
	for _, tc := range cases {
		got := labelForScore(tc.score)
		if got != tc.want {
			t.Errorf("labelForScore(%d) = %q, want %q", tc.score, got, tc.want)
		}
	}
}
