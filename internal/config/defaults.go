package config

// DefaultConfig returns a Config populated with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Dir:    ".",
		Format: "tui",
		FailOn: "none",
		Depth:  50,
		Since:  "24h",
	}
}
