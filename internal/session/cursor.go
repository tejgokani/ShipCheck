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

// cursorParser implements SessionParser for Cursor session logs.
type cursorParser struct{}

// NewCursorParser returns a SessionParser for Cursor logs.
func NewCursorParser() SessionParser {
	return &cursorParser{}
}

// ToolName returns the human-readable tool name.
func (p *cursorParser) ToolName() string {
	return "cursor"
}

// Detect reports whether Cursor log directories exist on this machine.
func (p *cursorParser) Detect() bool {
	for _, dir := range cursorLogDirs() {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return true
		}
	}
	return false
}

// Parse reads Cursor session logs created since the given time.
func (p *cursorParser) Parse(since time.Time) ([]Session, error) {
	var sessions []Session

	dirs := cursorLogDirs()
	found := false
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		found = true

		// Walk directory for JSONL files.
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Printf("warn: session/cursor: walk error at %s: %v", path, err)
				return nil
			}
			if info.IsDir() {
				return nil
			}
			if filepath.Ext(path) != ".jsonl" && filepath.Ext(path) != ".json" {
				return nil
			}
			s, err := parseCursorFile(path, since)
			if err != nil {
				log.Printf("warn: session/cursor: skipping %s: %v", path, err)
				return nil
			}
			if s != nil {
				sessions = append(sessions, *s)
			}
			return nil
		})
		if err != nil {
			log.Printf("warn: session/cursor: walk %s: %v", dir, err)
		}
	}

	if !found {
		log.Printf("warn: session/cursor: no Cursor log directories found (checked: %v)", dirs)
	}

	return sessions, nil
}

// cursorEntry is a single line from a Cursor JSONL log file.
type cursorEntry struct {
	Model     string         `json:"model"`
	Timestamp string         `json:"timestamp"`
	Cost      float64        `json:"cost"`
	Usage     cursorUsage    `json:"usage"`
	File      string         `json:"file"`
}

type cursorUsage struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	CacheReads   int64 `json:"cache_read_input_tokens"`
	CacheWrites  int64 `json:"cache_creation_input_tokens"`
}

// parseCursorFile parses one Cursor JSONL file, streaming line by line.
func parseCursorFile(path string, since time.Time) (*Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	sess := &Session{
		Tool: "cursor",
	}

	fileTokens := map[string]*SessionFile{}
	lineNum := 0
	hasEntries := false

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry cursorEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			log.Printf("warn: session/cursor: malformed JSON at %s line %d: %v", path, lineNum, err)
			continue
		}

		// Skip entries with no usage data (may be other event types).
		if entry.Usage.InputTokens == 0 && entry.Usage.OutputTokens == 0 && entry.Cost == 0 {
			continue
		}

		var ts time.Time
		if entry.Timestamp != "" {
			ts, _ = time.Parse(time.RFC3339, entry.Timestamp)
			if ts.IsZero() {
				ts, _ = time.Parse("2006-01-02T15:04:05.999Z", entry.Timestamp)
			}
		}

		if !ts.IsZero() && ts.Before(since) {
			continue
		}

		hasEntries = true

		if sess.StartTime.IsZero() || (!ts.IsZero() && ts.Before(sess.StartTime)) {
			sess.StartTime = ts
		}
		if sess.EndTime.IsZero() || (!ts.IsZero() && ts.After(sess.EndTime)) {
			sess.EndTime = ts
		}

		sess.TotalCost += entry.Cost
		sess.Usage.InputTokens += entry.Usage.InputTokens
		sess.Usage.OutputTokens += entry.Usage.OutputTokens
		sess.Usage.CacheReads += entry.Usage.CacheReads
		sess.Usage.CacheWrites += entry.Usage.CacheWrites

		if entry.Model != "" {
			sess.ModelID = entry.Model
		}

		if entry.File != "" {
			sf, ok := fileTokens[entry.File]
			if !ok {
				sf = &SessionFile{Path: entry.File}
				fileTokens[entry.File] = sf
			}
			sf.TouchCount++
			sf.TokensSpent += entry.Usage.InputTokens + entry.Usage.OutputTokens
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

// cursorLogDirs returns candidate directories for Cursor logs, ordered by preference.
func cursorLogDirs() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(home, ".cursor", "logs"),
		filepath.Join(home, ".config", "Code", "User", "globalStorage", "cursor.cursor-retrieval"),
		// macOS variant
		filepath.Join(home, "Library", "Application Support", "Cursor", "logs"),
	}
}
