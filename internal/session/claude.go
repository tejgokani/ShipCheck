// Package session provides types and parsers for agentic tool session logs.
package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// claudeParser implements SessionParser for Claude Code session logs.
type claudeParser struct{}

// NewClaudeParser returns a SessionParser for Claude Code logs.
func NewClaudeParser() SessionParser {
	return &claudeParser{}
}

// ToolName returns the human-readable tool name.
func (p *claudeParser) ToolName() string {
	return "claude-code"
}

// Detect reports whether Claude Code session logs exist on this machine.
func (p *claudeParser) Detect() bool {
	base, err := claudeBaseDir()
	if err != nil {
		return false
	}
	info, err := os.Stat(base)
	return err == nil && info.IsDir()
}

// Parse reads all Claude Code session logs created since the given time.
func (p *claudeParser) Parse(since time.Time) ([]Session, error) {
	base, err := claudeBaseDir()
	if err != nil {
		return nil, fmt.Errorf("session/claude: resolve base dir: %w", err)
	}

	// Glob pattern: ~/.claude/projects/*/sessions/*.jsonl
	pattern := filepath.Join(base, "projects", "*", "sessions", "*.jsonl")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("session/claude: glob sessions: %w", err)
	}

	if len(files) == 0 {
		log.Printf("warn: no Claude Code session files found under %s", base)
		return nil, nil
	}

	var sessions []Session
	for _, path := range files {
		s, err := parseClaudeFile(path, since)
		if err != nil {
			log.Printf("warn: session/claude: skipping %s: %v", path, err)
			continue
		}
		if s != nil {
			sessions = append(sessions, *s)
		}
	}
	return sessions, nil
}

// claudeEntry is a single line parsed from a Claude Code JSONL session file.
type claudeEntry struct {
	Type        string  `json:"type"`
	Cost        float64 `json:"cost"`
	TokensIn    int64   `json:"tokensIn"`
	TokensOut   int64   `json:"tokensOut"`
	CacheReads  int64   `json:"cacheReads"`
	CacheWrites int64   `json:"cacheWrites"`
	Timestamp   string  `json:"timestamp"`
	File        string  `json:"file"`
	Model       string  `json:"model"`
}

// parseClaudeFile parses one JSONL session file, streaming line by line.
// Returns nil session if all entries are older than since.
func parseClaudeFile(path string, since time.Time) (*Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	sess := &Session{
		Tool: "claude-code",
	}

	fileTokens := map[string]*SessionFile{}
	lineNum := 0
	hasEntries := false

	scanner := bufio.NewScanner(f)
	// Increase buffer size for long lines in session logs.
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry claudeEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			log.Printf("warn: session/claude: malformed JSON at %s line %d: %v", path, lineNum, err)
			continue
		}

		// Parse timestamp — skip entries older than since.
		var ts time.Time
		if entry.Timestamp != "" {
			ts, err = time.Parse(time.RFC3339, entry.Timestamp)
			if err != nil {
				// Try other common formats.
				ts, err = time.Parse("2006-01-02T15:04:05.999999999Z07:00", entry.Timestamp)
				if err != nil {
					log.Printf("warn: session/claude: unparseable timestamp at %s line %d: %q", path, lineNum, entry.Timestamp)
				}
			}
		}

		if !ts.IsZero() && ts.Before(since) {
			continue
		}

		hasEntries = true

		// Track session start/end.
		if sess.StartTime.IsZero() || (!ts.IsZero() && ts.Before(sess.StartTime)) {
			sess.StartTime = ts
		}
		if sess.EndTime.IsZero() || (!ts.IsZero() && ts.After(sess.EndTime)) {
			sess.EndTime = ts
		}

		// Accumulate cost and usage.
		sess.TotalCost += entry.Cost
		sess.Usage.InputTokens += entry.TokensIn
		sess.Usage.OutputTokens += entry.TokensOut
		sess.Usage.CacheReads += entry.CacheReads
		sess.Usage.CacheWrites += entry.CacheWrites

		if entry.Model != "" {
			sess.ModelID = entry.Model
		}

		// Track per-file token spend.
		if entry.File != "" {
			sf, ok := fileTokens[entry.File]
			if !ok {
				sf = &SessionFile{Path: entry.File}
				fileTokens[entry.File] = sf
			}
			sf.TouchCount++
			sf.TokensSpent += entry.TokensIn + entry.TokensOut
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}

	if !hasEntries {
		return nil, nil
	}

	for _, sf := range fileTokens {
		sess.Files = append(sess.Files, *sf)
	}

	return sess, nil
}

// claudeBaseDir returns the base directory for Claude Code logs.
func claudeBaseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".claude"), nil
}
