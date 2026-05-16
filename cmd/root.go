// Package cmd implements the shipcheck CLI commands.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgDir      string
	cfgFormat   string
	cfgFailOn   string
	cfgJSON     bool
	cfgHTML     bool
	cfgQuiet    bool
	cfgNoColor  bool
	cfgHookMode bool // set by post-session hook; suppresses "no sessions" warnings
)

// rootCmd is the base command — running shipcheck with no subcommand triggers scan.
var rootCmd = &cobra.Command{
	Use:   "shipcheck [directory]",
	Short: "Post-session audit tool for agentic coding sessions",
	Long: `shipcheck audits your project after an AI coding session.

It reports session cost, which files the agent kept struggling with (heatmap),
and security issues introduced by the agent — all offline, zero API calls.`,
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runScan,
}

// Execute runs the CLI.
func Execute(version string) {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "shipcheck:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgDir, "dir", "d", ".", "directory to scan")
	rootCmd.PersistentFlags().StringVar(&cfgFormat, "format", "tui", "output format: tui|json|html")
	rootCmd.PersistentFlags().StringVar(&cfgFailOn, "fail-on", "none", "exit 1 if findings >= severity: critical|high|medium|low|none")
	rootCmd.PersistentFlags().BoolVar(&cfgJSON, "json", false, "shorthand for --format=json")
	rootCmd.PersistentFlags().BoolVar(&cfgHTML, "html", false, "shorthand for --format=html")
	rootCmd.PersistentFlags().BoolVar(&cfgQuiet, "quiet", false, "print only the score number")
	rootCmd.PersistentFlags().BoolVar(&cfgNoColor, "no-color", false, "disable color output")
	rootCmd.PersistentFlags().BoolVar(&cfgHookMode, "hook-mode", false, "running as a post-session hook (suppresses warnings)")

	viper.SetEnvPrefix("SHIPCHECK")
	viper.AutomaticEnv()

	rootCmd.AddCommand(scanCmd, initCmd, reportCmd, versionCmd)
}
