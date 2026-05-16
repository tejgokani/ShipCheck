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

// codexParser implements SessionParser for OpenAI Codex session logs.
type codexParser struct{}

// NewCodexParser returns a SessionParser for Codex logs.
func NewCodexParser() SessionParser {
	return &codexParser{}
}

// ToolName returns the human-readable tool name.
func (p *codexParser) ToolName() string {
	return "codex"
}

// Detect reports whether Codex session log directories exist on this machine.
func (p *codexParser) Detect() bool {
	for _, dir := range codexLogDirs() {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return true
		}
	}
	return false
}

// Parse reads Codex session logs created since the given time.
func (p *codexParser) Parse(since time.Time) ([]Session, error) {
	var sessions []Session

	dirs := codexLogDirs()
	found := false
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		found = true

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Printf("warn: session/codex: walk error at %s: %v", path, err)
				return nil
			}
			if info.IsDir() {
				return nil
			}
			ext := filepath.Ext(path)
			if ext != ".jsonl" && ext != ".json" {
				return nil
			}
			s, err := parseCodexFile(path, since)
			if err != nil {
				log.Printf("warn: session/codex: skipping %s: %v", path, err)
				return nil
			}
			if s != nil {
				sessions = append(sessions, *s)
			}
			return nil
		})
		if err != nil {
			log.Printf("warn: session/codex: walk %s: %v", dir, err)
		}
	}

	if !found {
		log.Printf("warn: session/codex: no Codex log directories found (checked: %v)", dirs)
	}

	return sessions, nil
}

// codexEntry represents a single JSONL line in a Codex session log.
type codexEntry struct {
	Type      string      `json:"type"`
	Timestamp string      `json:"timestamp"`
	Model     string      `json:"model"`
	Cost      float64     `json:"cost"`
	Usage     codexUsage  `json:"usage"`
	File      string      `json:"file"`
}

type codexUsage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	TotalTokens      int64 `json:"total_tokens"`
	// Some versions use these field names
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
}

// parseCodexFile parses one Codex JSONL session file.
func parseCodexFile(path string, since time.Time) (*Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	sess := &Session{
		Tool: "codex",
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

		var entry codexEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			log.Printf("warn: session/codex: malformed JSON at %s line %d: %v", path, lineNum, err)
			continue
		}

		// Normalise token fields: Codex uses prompt_tokens/completion_tokens,
		// newer versions may use input_tokens/output_tokens.
		inputTokens := entry.Usage.InputTokens
		outputTokens := entry.Usage.OutputTokens
		if inputTokens == 0 {
			inputTokens = entry.Usage.PromptTokens
		}
		if outputTokens == 0 {
			outputTokens = entry.Usage.CompletionTokens
		}

		if inputTokens == 0 && outputTokens == 0 && entry.Cost == 0 {
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
		sess.Usage.InputTokens += inputTokens
		sess.Usage.OutputTokens += outputTokens

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
			sf.TokensSpent += inputTokens + outputTokens
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

// codexLogDirs returns candidate directories for Codex logs.
func codexLogDirs() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(home, ".codex", "sessions"),
		filepath.Join(home, ".openai", "codex-logs"),
	}
}
