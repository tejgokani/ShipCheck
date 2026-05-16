package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up post-session hooks for Claude Code and Cursor",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("init: get home dir: %w", err)
	}

	// Enable the hooks opt-in flag.
	configDir := filepath.Join(home, ".config", "shipcheck")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("init: create config dir: %w", err)
	}
	flagFile := filepath.Join(configDir, ".hooks-enabled")
	if err := os.WriteFile(flagFile, []byte("1"), 0644); err != nil {
		return fmt.Errorf("init: write hooks-enabled flag: %w", err)
	}

	// Install Claude Code post-session hook via settings.json.
	if err := installClaudeHook(home); err != nil {
		fmt.Fprintf(os.Stderr, "warn: Claude Code hook install: %v\n", err)
	} else {
		fmt.Println("✓ Claude Code post-session hook installed")
	}

	fmt.Println("\nshipcheck init complete. Run `shipcheck --help` for usage.")
	return nil
}

func installClaudeHook(home string) error {
	settingsPath := filepath.Join(home, ".claude", "settings.json")

	var settings map[string]interface{}
	data, err := os.ReadFile(settingsPath)
	if err == nil {
		if jsonErr := json.Unmarshal(data, &settings); jsonErr != nil {
			return fmt.Errorf("init: parse claude settings.json: %w", jsonErr)
		}
	} else {
		settings = make(map[string]interface{})
	}

	// "Stop" fires once when the Claude Code session ends — not after every tool call.
	hooks := map[string]interface{}{
		"Stop": []map[string]interface{}{
			{
				"hooks": []map[string]interface{}{
					{
						"type":    "command",
						"command": "shipcheck --hook-mode --quiet",
					},
				},
			},
		},
	}
	settings["hooks"] = hooks

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("init: marshal claude settings: %w", err)
	}

	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0755); err != nil {
		return fmt.Errorf("init: create .claude dir: %w", err)
	}
	return os.WriteFile(settingsPath, out, 0644)
}
