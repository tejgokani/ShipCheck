// Package report handles output rendering and score computation.
package report

import (
	"time"

	"github.com/tejgokani/shipcheck/internal/heatmap"
	"github.com/tejgokani/shipcheck/internal/security"
	"github.com/tejgokani/shipcheck/internal/session"
)

// AuditResult holds the complete output of a shipcheck scan.
type AuditResult struct {
	Score       int
	Label       string // CLEAN | GOOD | REVIEW | RISKY | DANGER
	Sessions    []session.Session
	Findings    []security.Finding
	HotFiles    []heatmap.HotFile
	GeneratedAt time.Time
	Directory   string
}

// ScoreResult holds the computed score and its label.
type ScoreResult struct {
	Score int
	Label string
}

// ComputeScore calculates a 0–100 security score from a list of findings.
func ComputeScore(findings []security.Finding) ScoreResult {
	base := 100
	criticalCount := countBySeverity(findings, security.Critical)
	highCount := countBySeverity(findings, security.High)
	mediumCount := countBySeverity(findings, security.Medium)
	lowCount := countBySeverity(findings, security.Low)

	deduction := 0
	deduction += minInt(criticalCount*15, 60)
	deduction += minInt(highCount*8, 32)
	deduction += minInt(mediumCount*3, 15)
	deduction += minInt(lowCount*1, 5)

	score := maxInt(base-deduction, 0)

	if criticalCount == 0 {
		score = minInt(score+5, 100)
	}
	if criticalCount == 0 && highCount == 0 {
		score = minInt(score+3, 100)
	}

	return ScoreResult{Score: score, Label: labelForScore(score)}
}

func labelForScore(score int) string {
	switch {
	case score >= 90:
		return "CLEAN"
	case score >= 70:
		return "GOOD"
	case score >= 50:
		return "REVIEW"
	case score >= 30:
		return "RISKY"
	default:
		return "DANGER"
	}
}

func countBySeverity(findings []security.Finding, sev security.Severity) int {
	n := 0
	for _, f := range findings {
		if f.Severity == sev {
			n++
		}
	}
	return n
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
