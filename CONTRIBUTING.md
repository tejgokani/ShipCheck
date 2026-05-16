# Contributing to shipcheck

shipcheck is MIT licensed and welcomes contributions. This guide covers everything from setting up your dev environment to submitting a pull request.

---

## Development Setup

### Prerequisites

```bash
go 1.22+          # https://go.dev/dl/
git               # any recent version
golangci-lint     # https://golangci-lint.run/usage/install/
goreleaser        # https://goreleaser.com/install/ (for release testing only)
```

### Clone and build

```bash
git clone https://github.com/tejgokani/shipcheck
cd shipcheck
go mod download
go build -o bin/shipcheck .
./bin/shipcheck --help
```

### Run tests

```bash
go test ./...                    # all tests
go test ./internal/security/...  # just security rules
go test -run TestSQLInjection    # specific test
go test -race ./...              # race condition detection
```

### Run linter

```bash
golangci-lint run ./...
```

---

## Project Structure

See `CLAUDE.md` for the full directory structure and package responsibilities.

Quick summary:
- `cmd/` — CLI commands (thin wrappers, no logic)
- `internal/session/` — session log parsing
- `internal/cost/` — token cost calculation
- `internal/heatmap/` — git-based file heat analysis
- `internal/security/` — AST-based security scanner
- `internal/report/` — output renderers (TUI, HTML, JSON)
- `rules/` — human-readable YAML rule definitions
- `testdata/` — fixtures for tests

---

## Adding a New Security Rule

This is the most common contribution. Here's the exact process:

### 1. Add the rule definition to YAML

Create or edit a file in `rules/` (e.g. `rules/secrets.yaml`):

```yaml
- id: SEC-042
  name: Hardcoded SendGrid API Key
  severity: critical
  description: |
    SendGrid API keys hardcoded in source code.
    AI coding agents frequently "fix" email configuration errors by
    hardcoding API keys directly — this is a critical security issue.
  languages: [go, typescript, javascript, python]
  evidence: "SG."  # The prefix for all SendGrid API keys
```

### 2. Implement the rule function

Add to the appropriate file in `internal/security/rules/`. Use the existing rules as templates:

```go
// secrets.go

// SendGridKey detects hardcoded SendGrid API keys.
// AI agents commonly hardcode these when trying to "fix" email sending errors
// rather than using environment variables properly.
var SendGridKey = Rule{
    ID:       "SEC-042",
    Name:     "Hardcoded SendGrid API Key",
    Severity: Critical,
    Languages: []string{"go", "typescript", "javascript", "python"},
    Check: func(path string, content []byte) []Finding {
        return findPattern(content, path,
            `SG\.[a-zA-Z0-9_-]{22,}\.[a-zA-Z0-9_-]{43}`,
            "SEC-042",
            "SendGrid API key hardcoded in source",
            Critical,
            "Move this key to an environment variable. Ask your AI coding tool: 'Replace the hardcoded SendGrid API key with process.env.SENDGRID_API_KEY and add it to .env.example'",
        )
    },
}
```

### 3. Register the rule

In `internal/security/scanner.go`, add your rule to `allRules`:

```go
var allRules = []Rule{
    // ... existing rules
    secrets.SendGridKey,  // add here
}
```

### 4. Write tests

Add a test in `internal/security/rules/secrets_test.go`:

```go
func TestSendGridKey(t *testing.T) {
    tests := []struct {
        name     string
        code     string
        wantHits int
    }{
        {
            name:     "catches hardcoded key",
            code:     `const apiKey = "SG.abc123def456ghi789jkl012.mno345pqr678stu901vwx234yz5"`,
            wantHits: 1,
        },
        {
            name:     "ignores env var reference",
            code:     `const apiKey = process.env.SENDGRID_API_KEY`,
            wantHits: 0,
        },
        {
            name:     "ignores test fixtures",
            code:     `const testKey = "SG.test_key_do_not_use"`, // too short
            wantHits: 0,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            findings := secrets.SendGridKey.Check("test.ts", []byte(tt.code))
            assert.Len(t, findings, tt.wantHits)
        })
    }
}
```

### 5. Add a vulnerable fixture

Add a test file to `testdata/projects/vulnerable/` that contains the pattern, and a clean version to `testdata/projects/clean/`.

---

## Adding a New Session Log Parser

To support a new AI coding tool:

### 1. Create the parser file

```go
// internal/session/gemini.go
package session

// GeminiParser parses Gemini CLI session logs.
// Logs are located at ~/.gemini/sessions/*.jsonl
type GeminiParser struct{}

func (p *GeminiParser) Detect() bool {
    home, _ := os.UserHomeDir()
    _, err := os.Stat(filepath.Join(home, ".gemini", "sessions"))
    return err == nil
}

func (p *GeminiParser) Parse(since time.Time) ([]Session, error) {
    // implementation
}

func (p *GeminiParser) ToolName() string {
    return "gemini-cli"
}
```

### 2. Register the parser

In `internal/session/registry.go`:

```go
var allParsers = []SessionParser{
    &ClaudeCodeParser{},
    &CursorParser{},
    &CodexParser{},
    &GeminiParser{},  // add here
}
```

### 3. Add fixtures

Add sample session log files to `testdata/sessions/gemini/` for testing.

---

## Pull Request Guidelines

### What we accept
- New security rules (see Adding a New Security Rule above)
- New session log parsers (Claude Code, Cursor, Codex already done)
- Bug fixes with failing test that now passes
- Performance improvements with benchmarks showing the improvement
- Documentation improvements

### What we don't accept
- Features that require network calls at scan time
- Rules that require LLM inference
- Changes to the scoring algorithm without discussion first
- Breaking changes to the JSON output format (it's a public API)

### PR checklist

```
[ ] Tests pass: go test ./...
[ ] No lint errors: golangci-lint run ./...
[ ] New rule: YAML definition + Go implementation + tests + fixtures
[ ] New parser: Detect() + Parse() + tests + fixtures
[ ] Bug fix: failing test added that now passes
[ ] CHANGELOG.md updated under [Unreleased]
[ ] No new external dependencies without discussion
```

### Commit message format

```
feat(security): add SendGrid API key detection rule
fix(parser): handle malformed Claude Code session log lines
docs: update contributing guide for parser additions
test: add fixtures for Cursor session log format v2
```

---

## Testing Philosophy

- **Table-driven tests** for rule matching — test both positive (catches the bug) and negative (doesn't false positive on clean code)
- **Fixture files** over inline strings for complex test cases
- **Benchmark** any code that runs on every file — use `go test -bench=.`
- **Race detector** on CI — `go test -race ./...`
- Test the **edge cases**: empty files, binary files, files with only comments, 0-byte session logs

---

## Reporting Bugs

Open an issue with:
1. `shipcheck version` output
2. OS and Go version
3. What you ran
4. What you expected
5. What actually happened
6. If it's a false positive/negative rule: the code snippet that triggered it (or didn't)

---

## Roadmap

See GitHub Issues with the `roadmap` label. High-priority items are tagged `good-first-issue` for new contributors.

Current priorities:
1. Cursor session log parser (different format per version — needs more test fixtures)
2. Python AST support for injection rules (currently regex-only)
3. `--fix` flag that auto-applies fixes via git patch
4. VS Code extension that shows findings inline
