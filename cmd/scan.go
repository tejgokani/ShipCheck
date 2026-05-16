package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/tejgokani/shipcheck/internal/config"
	"github.com/tejgokani/shipcheck/internal/heatmap"
	"github.com/tejgokani/shipcheck/internal/report"
	"github.com/tejgokani/shipcheck/internal/security"
	"github.com/tejgokani/shipcheck/internal/session"
)

var (
	scanNoSession  bool
	scanNoSecurity bool
	scanNoHeatmap  bool
	scanDepth      int
	scanSince      string
	scanCostOnly   bool
	scanHeatOnly   bool
	scanSecOnly    bool
)

var scanCmd = &cobra.Command{
	Use:   "scan [directory]",
	Short: "Run full audit (cost + heatmap + security)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runScan,
}

func init() {
	// Register scan flags on both rootCmd (shipcheck ...) and scanCmd (shipcheck scan ...).
	// rootCmd.RunE = runScan so both entry points need the same flag set.
	for _, fs := range []*pflag.FlagSet{rootCmd.Flags(), scanCmd.Flags()} {
		fs.BoolVar(&scanNoSession, "no-session", false, "skip session log analysis")
		fs.BoolVar(&scanNoSecurity, "no-security", false, "skip security scan")
		fs.BoolVar(&scanNoHeatmap, "no-heatmap", false, "skip heatmap")
		fs.IntVar(&scanDepth, "depth", 50, "git history depth for heatmap")
		fs.StringVar(&scanSince, "since", "24h", "only scan sessions since (e.g. 24h, 7d)")
		fs.BoolVar(&scanCostOnly, "cost", false, "cost + burn analysis only")
		fs.BoolVar(&scanHeatOnly, "heat", false, "heatmap only")
		fs.BoolVar(&scanSecOnly, "sec", false, "security scan only")
	}
}

func runScan(cmd *cobra.Command, args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	if cfgDir != "." {
		dir = cfgDir
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("scan: resolve dir: %w", err)
	}

	cfg, err := config.Load(absDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: config load: %v\n", err)
	}
	cfg.Dir = absDir

	// Apply flag overrides — flags beat config file values.
	if cmd.Flags().Changed("since") {
		cfg.Since = scanSince
	}
	if cmd.Flags().Changed("depth") {
		cfg.Depth = scanDepth
	}
	if scanNoSession || scanSecOnly || scanHeatOnly {
		cfg.NoSession = true
	}
	if scanNoSecurity || scanCostOnly || scanHeatOnly {
		cfg.NoSecurity = true
	}
	if scanNoHeatmap || scanCostOnly || scanSecOnly {
		cfg.NoHeatmap = true
	}

	result := report.AuditResult{
		Directory:   absDir,
		GeneratedAt: time.Now(),
	}

	// Parse session logs.
	if !cfg.NoSession {
		since := parseSince(cfg.Since)
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
	}

	// Build heatmap.
	if !cfg.NoHeatmap {
		hotFiles, err := heatmap.Build(result.Sessions, absDir, cfg.Depth)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: heatmap: %v\n", err)
		}
		result.HotFiles = hotFiles
	}

	// Run security scan.
	if !cfg.NoSecurity {
		scanner := security.New(absDir)
		findings, err := scanner.Scan()
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: security scan: %v\n", err)
		}
		result.Findings = findings
	}

	// Compute score.
	sr := report.ComputeScore(result.Findings)
	result.Score = sr.Score
	result.Label = sr.Label

	// Output.
	format := cfg.Format
	if cfgJSON {
		format = "json"
	}
	if cfgHTML {
		format = "html"
	}

	switch format {
	case "json":
		return report.RenderJSON(os.Stdout, result)
	case "html":
		outPath := filepath.Join(absDir, "shipcheck-report.html")
		if err := report.RenderHTML(result, outPath); err != nil {
			return err
		}
		fmt.Printf("Report saved to %s\n", outPath)
	default:
		if cfgQuiet {
			fmt.Println(result.Score)
			return nil
		}
		report.RenderTUI(os.Stdout, result)
	}

	// Exit 1 if --fail-on threshold exceeded.
	if cfg.FailOn != "none" && cfg.FailOn != "" {
		threshold := severityFromString(cfg.FailOn)
		for _, f := range result.Findings {
			if f.Severity >= threshold {
				os.Exit(1)
			}
		}
	}

	return nil
}

func parseSince(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	// Support day shorthand (e.g. "7d") since time.ParseDuration only handles h and below.
	if len(s) > 1 && s[len(s)-1] == 'd' {
		n := 0
		for _, ch := range s[:len(s)-1] {
			if ch < '0' || ch > '9' {
				n = -1
				break
			}
			n = n*10 + int(ch-'0')
		}
		if n > 0 {
			return time.Now().Add(-time.Duration(n) * 24 * time.Hour)
		}
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return time.Now().Add(-24 * time.Hour)
	}
	return time.Now().Add(-d)
}

func severityFromString(s string) security.Severity {
	switch s {
	case "critical":
		return security.Critical
	case "high":
		return security.High
	case "medium":
		return security.Medium
	case "low":
		return security.Low
	default:
		return security.Info
	}
}
