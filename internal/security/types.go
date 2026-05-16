// Package security provides deterministic AST-based security scanning rules.
package security

// Severity represents the impact level of a security finding.
type Severity int

const (
	Info Severity = iota
	Low
	Medium
	High
	Critical
)

// String returns the human-readable severity name.
func (s Severity) String() string {
	switch s {
	case Critical:
		return "CRITICAL"
	case High:
		return "HIGH"
	case Medium:
		return "MEDIUM"
	case Low:
		return "LOW"
	default:
		return "INFO"
	}
}

// Finding is a single security issue located in source code.
type Finding struct {
	Rule      string
	Severity  Severity
	File      string
	Line      int
	Column    int
	Message   string
	Evidence  string // the actual code snippet
	FixPrompt string // ready-to-paste prompt for Claude Code
}

// Rule is a named, language-scoped check function.
type Rule struct {
	ID          string
	Name        string
	Severity    Severity
	Description string
	Languages   []string // ["go", "typescript", "python", "*"]
	Check       func(path string, content []byte) []Finding
}
