package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/tejgokani/shipcheck/internal/heatmap"
	"github.com/tejgokani/shipcheck/internal/report"
	"github.com/tejgokani/shipcheck/internal/security"
	"github.com/tejgokani/shipcheck/internal/session"
)

var reportCmd = &cobra.Command{
	Use:   "report [directory]",
	Short: "Generate HTML report and open in browser",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runReport,
}

func runReport(cmd *cobra.Command, args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("report: resolve dir: %w", err)
	}

	result := report.AuditResult{
		Directory:   absDir,
		GeneratedAt: time.Now(),
	}

	since := parseSince("24h")
	parsers := session.AllParsers()
	for _, p := range parsers {
		if !p.Detect() {
			continue
		}
		sessions, err := p.Parse(since)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: %s session parse: %v\n", p.ToolName(), err)
			continue
		}
		result.Sessions = append(result.Sessions, sessions...)
	}

	hotFiles, err := heatmap.Build(result.Sessions, absDir, 50)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: heatmap: %v\n", err)
	}
	result.HotFiles = hotFiles

	scanner := security.New(absDir)
	findings, err := scanner.Scan()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: security scan: %v\n", err)
	}
	result.Findings = findings

	sr := report.ComputeScore(result.Findings)
	result.Score = sr.Score
	result.Label = sr.Label

	outPath := filepath.Join(absDir, "shipcheck-report.html")
	if err := report.RenderHTML(result, outPath); err != nil {
		return err
	}
	fmt.Printf("Report saved to %s\n", outPath)
	return nil
}
