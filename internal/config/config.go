// Package config handles loading and merging shipcheck configuration.
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all runtime configuration for a shipcheck scan.
type Config struct {
	Dir        string   `mapstructure:"dir"`
	Format     string   `mapstructure:"format"`      // tui|json|html
	FailOn     string   `mapstructure:"fail_on"`     // critical|high|medium|low|none
	Depth      int      `mapstructure:"depth"`       // git history depth
	Since      string   `mapstructure:"since"`       // e.g. "24h", "7d"
	NoSession  bool     `mapstructure:"no_session"`
	NoSecurity bool     `mapstructure:"no_security"`
	NoHeatmap  bool     `mapstructure:"no_heatmap"`
	NoColor    bool     `mapstructure:"no_color"`
	Ignore     []string `mapstructure:"ignore"`
}

// Load reads config from flags, env vars, project file, and user file (priority order).
func Load(dir string) (Config, error) {
	cfg := DefaultConfig()
	cfg.Dir = dir

	v := viper.New()
	v.SetConfigName(".shipcheck")
	v.SetConfigType("yaml")
	v.AddConfigPath(dir)

	home, _ := userHomeDir()
	if home != "" {
		v.AddConfigPath(home + "/.config/shipcheck")
	}

	v.SetEnvPrefix("SHIPCHECK")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Register defaults so Unmarshal doesn't zero-out fields absent from the config file.
	defaults := DefaultConfig()
	v.SetDefault("dir", defaults.Dir)
	v.SetDefault("format", defaults.Format)
	v.SetDefault("fail_on", defaults.FailOn)
	v.SetDefault("depth", defaults.Depth)
	v.SetDefault("since", defaults.Since)

	if err := v.ReadInConfig(); err != nil {
		// Config file missing is fine — use defaults.
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return cfg, fmt.Errorf("config: read: %w", err)
		}
	}

	if err := v.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("config: unmarshal: %w", err)
	}

	return cfg, nil
}

func userHomeDir() (string, error) {
	// Wrapper so we can test without os.UserHomeDir.
	return homeDir()
}
