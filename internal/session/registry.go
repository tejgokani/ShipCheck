package session

// AllParsers returns all registered session log parsers.
// cmd/scan.go iterates these, calls Detect(), and aggregates results.
func AllParsers() []SessionParser {
	return []SessionParser{
		NewClaudeParser(),
		NewCursorParser(),
		NewCodexParser(),
	}
}
