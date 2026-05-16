// Package session provides types and parsers for agentic tool session logs.
package session

import "time"

// Session represents a single agentic coding session from any supported tool.
type Session struct {
	Tool      string        // "claude-code" | "cursor" | "codex"
	StartTime time.Time
	EndTime   time.Time
	Files     []SessionFile
	TotalCost float64
	Usage     TokenUsage
	ModelID   string
}

// SessionFile records how much agent activity occurred on a specific file.
type SessionFile struct {
	Path        string
	TouchCount  int
	TokensSpent int64
}

// TokenUsage holds the token breakdown for a session.
type TokenUsage struct {
	InputTokens  int64
	OutputTokens int64
	CacheReads   int64
	CacheWrites  int64
}

// SessionParser is the strategy interface for tool-specific log parsers.
type SessionParser interface {
	// Detect reports whether this parser can find logs on the current machine.
	Detect() bool
	// Parse reads session logs since the given time and returns sessions found.
	Parse(since time.Time) ([]Session, error)
	// ToolName returns the human-readable tool name.
	ToolName() string
}
