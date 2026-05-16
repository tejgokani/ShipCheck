package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/tejgokani/shipcheck/internal/security"
)

const (
	colorCritical  = "#E24B4A"
	colorHigh      = "#EF9F27"
	colorMedium    = "#378ADD"
	colorLow       = "#888780"
	colorGood      = "#1D9E75"
	colorBarBG     = "#D3D1C7"
	colorSubtle    = "#6C6A62"
	colorBorder    = "#3D3D3A"
)

// RenderTUI writes the full TUI audit report to w.
func RenderTUI(w io.Writer, result AuditResult) {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorBorder)).
		Padding(0, 1).
		Width(65)

	header := boxStyle.Render(fmt.Sprintf(
		" shipcheck v%s\n  %s   last session: %s",
		"0.1.0",
		result.Directory,
		formatAge(result),
	))
	fmt.Fprintln(w, header)
	fmt.Fprintln(w)

	// Score — use pre-computed value if available.
	score, label := result.Score, result.Label
	if score == 0 && label == "" {
		sr := ComputeScore(result.Findings)
		score, label = sr.Score, sr.Label
	}
	scoreColor := colorGood
	if score < 50 {
		scoreColor = colorCritical
	} else if score < 70 {
		scoreColor = colorHigh
	}
	scoreStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(scoreColor))
	fmt.Fprintf(w, "  SCORE   %s   %s\n\n",
		scoreStyle.Render(fmt.Sprintf("%d / 100", score)),
		label)

	// Cost section
	fmt.Fprintln(w, "  ── COST ────────────────────────────────────────────────────")
	if len(result.Sessions) == 0 {
		fmt.Fprintln(w, "  No session logs found")
	} else {
		totalCost := 0.0
		var totalUsage session_usage
		for _, s := range result.Sessions {
			totalCost += s.TotalCost
			totalUsage.in += s.Usage.InputTokens
			totalUsage.out += s.Usage.OutputTokens
			totalUsage.cache += s.Usage.CacheReads
		}
		fmt.Fprintf(w, "  Session cost        $%.2f\n", totalCost)
		fmt.Fprintf(w, "  Tokens in           %s\n", formatTokens(totalUsage.in))
		fmt.Fprintf(w, "  Tokens out          %s\n", formatTokens(totalUsage.out))
		if totalUsage.cache > 0 {
			fmt.Fprintf(w, "  Cache hits          %s\n", formatTokens(totalUsage.cache))
		}
	}
	fmt.Fprintln(w)

	// Heatmap section
	fmt.Fprintln(w, "  ── HEATMAP (top files) ─────────────────────────────────────")
	if len(result.HotFiles) == 0 {
		fmt.Fprintln(w, "  No git history or session data found")
	} else {
		for i, hf := range result.HotFiles {
			if i >= 5 {
				break
			}
			bars := int(hf.HeatScore * 12)
			if bars < 1 {
				bars = 1
			}
			bar := strings.Repeat("█", bars)
			fmt.Fprintf(w, "  %-12s  %-40s  %d touches\n", bar, truncate(hf.Path, 40), hf.TouchCount)
		}
	}
	fmt.Fprintln(w)

	// Security section
	fmt.Fprintf(w, "  ── SECURITY (%d findings) ──────────────────────────────────\n", len(result.Findings))
	if len(result.Findings) == 0 {
		fmt.Fprintln(w, "  No findings")
	} else {
		for _, f := range result.Findings {
			sevStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(severityColor(f.Severity)))
			fmt.Fprintf(w, "  %s  %s:%d    %s\n",
				sevStyle.Render(fmt.Sprintf("%-8s", f.Severity.String())),
				truncate(f.File, 35), f.Line, f.Message)
		}
	}
	fmt.Fprintln(w)

	// Fix prompts
	if len(result.Findings) > 0 {
		fmt.Fprintln(w, "  ── FIX PROMPTS ────────────────────────────────────────────")
		fmt.Fprintln(w, "  Paste into Claude Code to fix all findings:")
		fmt.Fprintf(w, "  > %s\n", buildFixPrompt(result.Findings))
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, "  Run `shipcheck report` for full HTML report.")
}

type session_usage struct {
	in, out, cache int64
}

func formatAge(result AuditResult) string {
	if len(result.Sessions) == 0 {
		return "unknown"
	}
	// Use most recent session end time
	var latest = result.Sessions[0].EndTime
	for _, s := range result.Sessions[1:] {
		if s.EndTime.After(latest) {
			latest = s.EndTime
		}
	}
	if latest.IsZero() {
		return "unknown"
	}
	diff := result.GeneratedAt.Sub(latest)
	if diff.Minutes() < 60 {
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	}
	return fmt.Sprintf("%dh ago", int(diff.Hours()))
}

func formatTokens(n int64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%d,%03d", n/1000, n%1000)
	}
	return fmt.Sprintf("%d", n)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return "..." + s[len(s)-max+3:]
}

func severityColor(sev security.Severity) string {
	switch sev {
	case security.Critical:
		return colorCritical
	case security.High:
		return colorHigh
	case security.Medium:
		return colorMedium
	case security.Low:
		return colorLow
	default:
		return colorSubtle
	}
}

func buildFixPrompt(findings []security.Finding) string {
	if len(findings) == 0 {
		return ""
	}
	msgs := make([]string, 0, len(findings))
	for _, f := range findings {
		msgs = append(msgs, fmt.Sprintf("%s at %s:%d", f.Message, f.File, f.Line))
	}
	return fmt.Sprintf("Fix the %d security issues found by shipcheck: %s.",
		len(findings), strings.Join(msgs, "; "))
}
