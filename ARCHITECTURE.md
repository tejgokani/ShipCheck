# ARCHITECTURE.md — shipcheck System Design

All decisions here are **locked**. Agents do not debate these. If a decision needs revisiting, update this file first via the LEAD agent.

---

## Core Design Principles

### 1. Zero network calls at runtime
The scanner never makes HTTP requests. No phoning home, no telemetry, no update checks during scan. The pricing table is compiled into the binary and updated on releases.

### 2. Fail gracefully, always
Missing session logs → warn and continue with code-only scan.
Unparseable JSONL line → log the line number, skip, continue.
No git repo → skip heatmap, continue with security scan.
Binary scan target → skip the file silently.

### 3. Fast by default
The scanner must complete on a 50k LOC project in under 5 seconds. Use goroutines for file scanning (bounded by `runtime.NumCPU()` workers). Profile before optimizing — don't prematurely optimize.

### 4. Deterministic output
Same input always produces same output. No randomness, no timestamps in findings. The score for a given codebase state is always identical. This makes CI integration reliable.

### 5. Config over convention, but good defaults
Everything is configurable via flags or config file. But defaults must be sane — running `shipcheck` with no flags in any Go/TypeScript/Python project should work correctly.

---

## Data Flow

```
┌─────────────────────────────────────────────────────────┐
│                    cmd/scan.go                          │
│  (orchestrates the pipeline, no business logic here)    │
└──────────────────────┬──────────────────────────────────┘
                       │
          ┌────────────┼────────────┐
          ▼            ▼            ▼
  ┌──────────────┐ ┌──────────┐ ┌──────────────┐
  │session parser│ │heatmap   │ │security      │
  │              │ │          │ │scanner       │
  │Reads:        │ │Reads:    │ │              │
  │~/.claude/    │ │git log   │ │Reads:        │
  │~/.cursor/    │ │+ session │ │source files  │
  │~/.codex/     │ │timestamps│ │in target dir │
  └──────┬───────┘ └────┬─────┘ └──────┬───────┘
         │              │              │
         └──────────────┼──────────────┘
                        ▼
                ┌───────────────┐
                │ AuditResult   │
                │ (types.go)    │
                └───────┬───────┘
                        │
           ┌────────────┼────────────┐
           ▼            ▼            ▼
      ┌─────────┐  ┌─────────┐  ┌─────────┐
      │TUI      │  │HTML     │  │JSON     │
      │renderer │  │renderer │  │renderer │
      └─────────┘  └─────────┘  └─────────┘
```

---

## Session Log Parsing Architecture

### Strategy pattern for multiple tools

Each tool (Claude Code, Cursor, Codex) has its own parser implementing:

```go
type SessionParser interface {
    Detect() bool                    // Can this parser find logs on this machine?
    Parse(since time.Time) ([]Session, error)
    ToolName() string
}
```

`cmd/scan.go` iterates all registered parsers, calls `Detect()`, and aggregates results. This makes adding new tools (Gemini CLI, Aider, etc.) trivial.

### JSONL parsing approach

Read line by line — never load entire session files into memory. This handles multi-GB session logs from long agent runs without OOMing.

```go
scanner := bufio.NewScanner(f)
for scanner.Scan() {
    var entry SessionEntry
    if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
        log.Printf("warn: skipping malformed line %d in %s", lineNum, path)
        continue
    }
    // process entry
}
```

---

## Security Scanner Architecture

### Rule execution model

Rules are pure functions. No side effects. No shared state between rules.

```go
type Rule struct {
    ID          string
    Name        string
    Severity    Severity
    Description string
    Languages   []string  // ["go", "typescript", "python", "*"]
    Check       func(path string, content []byte) []Finding
}
```

Rules are registered in `internal/security/scanner.go`:

```go
var allRules = []Rule{
    secrets.HardcodedAPIKey,
    secrets.NextPublicLeak,
    secrets.SupabaseServiceRole,
    injection.SQLStringConcat,
    auth.WildcardCORS,
    // ...
}
```

### Concurrency model

Files are scanned concurrently using a worker pool:

```go
sem := make(chan struct{}, runtime.NumCPU())
var wg sync.WaitGroup
var mu sync.Mutex
var findings []Finding

for _, file := range files {
    wg.Add(1)
    go func(f string) {
        defer wg.Done()
        sem <- struct{}{}
        defer func() { <-sem }()

        content, _ := os.ReadFile(f)
        for _, rule := range applicableRules(f) {
            hits := rule.Check(f, content)
            if len(hits) > 0 {
                mu.Lock()
                findings = append(findings, hits...)
                mu.Unlock()
            }
        }
    }(file)
}
wg.Wait()
```

### Language detection

Detect language by file extension — do NOT use content sniffing (slow):

```go
var langByExt = map[string]string{
    ".go":   "go",
    ".ts":   "typescript",
    ".tsx":  "typescript",
    ".js":   "javascript",
    ".jsx":  "javascript",
    ".py":   "python",
    ".env":  "env",
    ".yaml": "yaml",
    ".yml":  "yaml",
    ".json": "json",
}
```

---

## Heatmap Architecture

The heatmap answers: "which files did the agent keep touching and retrying?"

### Algorithm

1. Get all git commits within the session timeframe (using session start/end timestamps)
2. For each commit, get the list of changed files (`git diff-tree --no-commit-id -r --name-only <hash>`)
3. Count how many times each file appears across commits in the session window
4. Optionally correlate with session log `file` fields if available
5. Sort descending by count
6. Top 10 = the heatmap

```go
type HotFile struct {
    Path        string
    TouchCount  int
    HeatScore   float64  // normalized 0-1
    LastTouched time.Time
}
```

### Heatmap scoring

```
HeatScore = TouchCount / MaxTouchCount  (normalized to 0-1 across all files)
```

Displayed as a bar: 10 blocks max, each block = 10% heat.

---

## Score Architecture

Score is computed in `internal/report/score.go` as a pure function:

```go
func ComputeScore(findings []security.Finding) ScoreResult {
    base := 100
    criticalCount := countBySeverity(findings, Critical)
    highCount := countBySeverity(findings, High)
    mediumCount := countBySeverity(findings, Medium)
    lowCount := countBySeverity(findings, Low)

    deduction := 0
    deduction += min(criticalCount*15, 60)
    deduction += min(highCount*8, 32)
    deduction += min(mediumCount*3, 15)
    deduction += min(lowCount*1, 5)

    score := max(base-deduction, 0)

    if criticalCount == 0 { score = min(score+5, 100) }
    if criticalCount == 0 && highCount == 0 { score = min(score+3, 100) }

    return ScoreResult{
        Score: score,
        Label: labelForScore(score),
    }
}
```

---

## Configuration Architecture

Config is loaded in priority order (highest wins):

1. CLI flags (`--fail-on critical`)
2. Environment variables (`SHIPCHECK_FAIL_ON=critical`)
3. Project config file (`.shipcheck.yaml` in project root)
4. User config file (`~/.config/shipcheck/config.yaml`)
5. Built-in defaults

Config struct:

```go
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
```

---

## HTML Report Architecture

Single self-contained HTML file. No external dependencies.

Structure:
```html
<!DOCTYPE html>
<html>
<head>
  <style>/* All CSS inlined — ~3KB */</style>
</head>
<body>
  <!-- Header: score, project path, timestamp -->
  <!-- Card 1: Cost & session summary -->
  <!-- Card 2: Heatmap (CSS bar charts, no JS required) -->
  <!-- Card 3: Security findings table -->
  <!-- Footer: fix prompt + copy button -->
  <script>/* Copy to clipboard only — ~200 bytes */</script>
</body>
</html>
```

Generated by `internal/report/html.go` using Go's `html/template`. Template stored as embedded bytes using `//go:embed`.

---

## Hook Architecture

`shipcheck init` installs post-session hooks. Each hook is a shell script that:
1. Checks if `shipcheck` binary is in PATH
2. Runs `shipcheck --quiet --no-session` (security scan only by default for hooks)
3. If score is below 70, prints a warning and the number of findings
4. Never blocks the user's workflow (runs in background with timeout)

### Claude Code hook location
Claude Code supports hooks via `~/.claude/settings.json`:
```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": ".*",
        "hooks": [{ "type": "command", "command": "shipcheck --hook-mode --quiet" }]
      }
    ]
  }
}
```

`shipcheck init` writes this config entry automatically.

---

## Decisions Log

| Date | Decision | Reason |
|------|----------|--------|
| 2026-05 | Go over Rust | Faster development, excellent stdlib, easier for contributors |
| 2026-05 | cobra over urfave/cli | More familiar, better documentation, active maintenance |
| 2026-05 | lipgloss over charm | Same team, more composable, better for our layout needs |
| 2026-05 | Deterministic rules over LLM-based | Zero cost to run, no false positives from hallucination, CI-safe |
| 2026-05 | JSONL streaming over full file load | Memory safety for large session logs |
| 2026-05 | MIT over AGPL | Open source, embeddable, community-friendly |
| 2026-05 | Goreleaser over manual CI | Single source of truth for releases, Homebrew integration built-in |
