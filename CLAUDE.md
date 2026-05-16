# shipcheck — Claude Code Project Brain

## What This Project Is

`shipcheck` is a **Go CLI tool** that runs after any agentic coding session (Claude Code, Cursor, Codex) and gives the developer a full post-session audit report. It does three things in one command:

1. **Cost & Burn** — parses session logs from `~/.claude/`, `~/.cursor/`, `~/.codex/` and shows token spend, cost estimate, and which files burned the most tokens
2. **Heatmap** — reads git history + session logs to show which files were touched/retried most by the agent (the "hot files" that keep breaking)
3. **Security Scan** — runs deterministic AST-based security rules against the source code, looking specifically for AI-generated code failure patterns

**One command. Zero API calls. Zero tokens spent. 100% offline. MIT licensed.**

---

## Tech Stack

| Layer | Choice | Why |
|---|---|---|
| Language | Go 1.22+ | Single static binary, cross-platform, fast startup |
| CLI framework | `cobra` | Industry standard for Go CLIs |
| TUI | `bubbletea` + `lipgloss` | Beautiful terminal output |
| AST parsing | `tree-sitter-go` bindings | Multi-language code analysis |
| Config | `viper` | Config file + env var support |
| Testing | `go test` + `testify` | Standard Go testing |
| Build | `goreleaser` | Cross-platform binary + Homebrew formula |

---

## Project Structure

```
shipcheck/
├── CLAUDE.md                  ← You are here
├── AGENTS.md                  ← Agent team definitions
├── ARCHITECTURE.md            ← System architecture decisions
├── CONTRIBUTING.md            ← How to contribute
├── README.md                  ← Public-facing docs
├── go.mod
├── go.sum
├── main.go                    ← Entry point only, delegates to cmd/
├── cmd/
│   ├── root.go                ← Root cobra command, global flags
│   ├── scan.go                ← `shipcheck scan` — main audit command
│   ├── init.go                ← `shipcheck init` — set up hooks
│   ├── report.go              ← `shipcheck report` — generate HTML report
│   └── version.go             ← `shipcheck version`
├── internal/
│   ├── session/
│   │   ├── claude.go          ← Parse ~/.claude/ session logs
│   │   ├── cursor.go          ← Parse ~/.cursor/ session logs
│   │   ├── codex.go           ← Parse ~/.codex/ session logs
│   │   └── types.go           ← Shared session data types
│   ├── cost/
│   │   ├── calculator.go      ← Token → dollar cost calculation
│   │   ├── models.go          ← Model pricing table (updated regularly)
│   │   └── burn.go            ← Per-file burn analysis
│   ├── heatmap/
│   │   ├── git.go             ← Git log analysis for retry detection
│   │   ├── heatmap.go         ← File heat scoring algorithm
│   │   └── types.go
│   ├── security/
│   │   ├── scanner.go         ← Main scan orchestrator
│   │   ├── rules/
│   │   │   ├── secrets.go     ← Hardcoded secrets, API keys, tokens
│   │   │   ├── injection.go   ← SQL injection, command injection
│   │   │   ├── auth.go        ← Missing auth, insecure shortcuts
│   │   │   ├── frontend.go    ← NEXT_PUBLIC_ leaks, client-side secrets
│   │   │   └── deps.go        ← Hallucinated/vulnerable dependencies
│   │   └── types.go           ← Finding, Severity types
│   ├── report/
│   │   ├── tui.go             ← Terminal output renderer
│   │   ├── html.go            ← HTML report generator
│   │   ├── json.go            ← JSON output for CI
│   │   └── score.go           ← Score calculation (0-100)
│   └── config/
│       ├── config.go          ← Config loading (viper)
│       └── defaults.go        ← Default config values
├── rules/                     ← YAML rule definitions (human-readable)
│   ├── secrets.yaml
│   ├── injection.yaml
│   ├── auth.yaml
│   └── frontend.yaml
├── testdata/                  ← Fixtures for testing
│   ├── sessions/              ← Sample session log files
│   └── projects/              ← Sample vulnerable projects for scan tests
├── scripts/
│   ├── install.sh             ← curl | sh installer
│   └── hooks/
│       ├── claude-code-hook   ← Post-session hook script
│       └── cursor-hook        ← Post-session hook script
└── .goreleaser.yaml           ← Release config for goreleaser
```

---

## Core Commands

```bash
shipcheck              # Run full audit on current directory (default)
shipcheck scan         # Explicit scan — same as default
shipcheck scan --cost  # Cost + burn analysis only
shipcheck scan --heat  # Heatmap only
shipcheck scan --sec   # Security scan only
shipcheck report       # Generate HTML report, open in browser
shipcheck init         # Set up post-session hooks
shipcheck version      # Print version
```

### Output modes

```bash
shipcheck              # TUI (default) — beautiful terminal table
shipcheck --json       # JSON output — for CI pipelines
shipcheck --html       # HTML report — saves to ./shipcheck-report.html
shipcheck --quiet      # Just the score — for scripts
```

### Flags

```bash
--dir, -d       # Directory to scan (default: current)
--depth         # Git history depth for heatmap (default: 50 commits)
--since         # Only scan sessions since date (e.g. --since 24h)
--fail-on       # Exit code 1 if findings >= severity (critical|high|medium)
--no-session    # Skip session log analysis, code scan only
--no-security   # Skip security scan
--no-heatmap    # Skip heatmap
--format        # Output format: tui|json|html (default: tui)
```

---

## Session Log Paths

These are the exact file paths to read from for each tool:

### Claude Code
```
~/.claude/projects/<project-hash>/sessions/<session-id>.jsonl
~/.claude/statsig/
```
Each line is a JSON object. Key fields: `type`, `cost`, `tokensIn`, `tokensOut`, `cacheReads`, `cacheWrites`, `timestamp`, `file`

### Cursor  
```
~/.cursor/logs/
~/.config/Code/User/globalStorage/cursor.cursor-retrieval/
```
Format: JSONL with `model`, `usage.input_tokens`, `usage.output_tokens`, `cost`

### Codex
```
~/.codex/sessions/
~/.openai/codex-logs/
```
Format: JSONL per session

**Always check for platform differences:**
- macOS: `~/Library/Application Support/` for some tools
- Linux: `~/.config/` or `~/.local/share/`
- Windows: `%APPDATA%\`

---

## Security Rules — What We Scan For

These are AI-specific failure patterns. NOT generic SAST rules. Every rule should have a comment explaining WHY AI generates this pattern.

### Category 1: Secrets & Tokens
- Hardcoded API keys (regex patterns for OpenAI, Anthropic, Stripe, Supabase, etc.)
- `NEXT_PUBLIC_` prefix on backend secrets (AI does this to "fix" env var issues)
- `service_role` key exposed to frontend (Supabase god-mode key)
- JWT secrets set to `"secret"` or `"your-secret-here"` (AI defaults)
- Database URLs with credentials in connection string

### Category 2: Auth Shortcuts
- `Access-Control-Allow-Origin: *` (AI applies this to fix CORS errors)
- `verify=False` in Python requests (AI applies to fix SSL errors)
- Missing auth middleware on routes that clearly need it
- Client-side auth checks that should be server-side

### Category 3: Injection
- SQL queries built with string concatenation or f-strings
- `eval()` or `exec()` on user input
- `subprocess` or `child_process` with unescaped user input
- Path traversal via `../` in file operations

### Category 4: AI-Specific Patterns
- Packages that don't exist (hallucinated imports) — check against known package registries
- `// TODO: add auth later` comments (AI defers security)
- Retry logic that resets rate limit counters in-memory (resets on restart)
- `DEBUG=True` or `ENV=development` committed to code

### Severity levels
- `critical` — can be exploited immediately (exposed key, SQL injection)
- `high` — serious vulnerability, needs fix before prod
- `medium` — bad practice, should fix
- `low` — code smell, consider fixing
- `info` — informational, no action required

---

## Score Calculation

The score (0–100) is computed as:

```
base = 100
deduct per critical finding:  -15 (cap at -60)
deduct per high finding:       -8  (cap at -32)
deduct per medium finding:     -3  (cap at -15)
deduct per low finding:        -1  (cap at -5)
bonus for zero criticals:      +5
bonus for zero highs:          +3
floor = 0
```

Score labels:
- 90–100: `CLEAN`
- 70–89:  `GOOD`
- 50–69:  `REVIEW`
- 30–49:  `RISKY`
- 0–29:   `DANGER`

---

## Output Format — TUI

The default terminal output should look like this:

```
┌─ shipcheck v0.1.0 ─────────────────────────────────────────────┐
│  /Users/tej/projects/nexerp              last session: 14m ago  │
└─────────────────────────────────────────────────────────────────┘

  SCORE   74 / 100   GOOD

  ── COST ────────────────────────────────────────────────────────
  Session cost        $0.84    (claude-opus-4-6)
  Tokens in           42,310
  Tokens out           8,204
  Cache hits          18,200   saved ~$0.31

  ── HEATMAP (top 5 hot files) ───────────────────────────────────
  ████████████  src/auth/middleware.ts       12 agent touches
  ██████████    src/api/routes.go             9 agent touches
  ████████      prisma/schema.prisma          7 agent touches
  ██████        src/components/Dashboard.tsx  5 agent touches
  ████          src/utils/db.ts               4 agent touches

  ── SECURITY (3 findings) ───────────────────────────────────────
  CRITICAL  src/api/routes.go:47    SQL query uses string concat
  HIGH      .env.local:3            Supabase service_role in frontend env
  MEDIUM    src/auth/handler.ts:91  CORS set to wildcard (*)

  ── FIX PROMPTS ─────────────────────────────────────────────────
  Paste into Claude Code to fix all findings:
  > Fix the 3 security issues found by shipcheck: SQL injection at
    routes.go:47, exposed service_role key, and wildcard CORS in
    auth/handler.ts. Apply server-side fixes only.

  Run `shipcheck report` for full HTML report.
```

---

## HTML Report

The HTML report (`shipcheck report`) must:
- Be a single self-contained `.html` file (all CSS/JS inlined)
- Work offline — no CDN dependencies
- Have three sections: Cost, Heatmap, Security
- Show the score prominently at the top
- Include a one-click copy button for the fix prompt
- Be shareable — designed to be posted on Twitter/Reddit as a screenshot

---

## Critical Implementation Rules

### Never do these
- Never call any external API during a scan (no LLM calls, no network requests)
- Never read files outside the target directory + session log paths
- Never write anything to the user's project directory without `--fix` flag
- Never fail silently — if a session log can't be parsed, log a warning and continue
- Never hardcode model pricing — read from `internal/cost/models.go` which is versioned

### Always do these
- Respect `.gitignore` and `.shipcheckirgnore` when scanning source files
- Handle missing session logs gracefully (user may not have run an agent recently)
- Show progress for large repos (`scanning 1,247 files...`)
- Exit 0 unless `--fail-on` threshold is breached
- Support both `~` and full paths in all path resolution
- Test on macOS + Linux — Windows is nice-to-have

### Go conventions
- All packages in `internal/` — nothing exported unless it's a public API
- Error handling: wrap with context using `fmt.Errorf("context: %w", err)`
- No global state — pass config through context or explicit parameters
- Table-driven tests for all rule matching logic
- Benchmarks for the scanner — it should scan 10k files in under 2 seconds

---

## Model Pricing Table

Keep this updated in `internal/cost/models.go`. Per million tokens:

```go
var ModelPricing = map[string]ModelPrice{
    "claude-opus-4-6":    {Input: 15.00, Output: 75.00, CacheRead: 1.50},
    "claude-sonnet-4-6":  {Input: 3.00,  Output: 15.00, CacheRead: 0.30},
    "claude-haiku-4-5":   {Input: 0.80,  Output: 4.00,  CacheRead: 0.08},
    "gpt-4o":             {Input: 2.50,  Output: 10.00, CacheRead: 1.25},
    "gpt-4o-mini":        {Input: 0.15,  Output: 0.60,  CacheRead: 0.075},
    "o3":                 {Input: 10.00, Output: 40.00, CacheRead: 2.50},
    "gemini-2.5-pro":     {Input: 1.25,  Output: 10.00, CacheRead: 0.31},
}
```

---

## Definition of Done (per feature)

A feature is done when:
- [ ] Core functionality works on macOS + Linux
- [ ] Unit tests pass (`go test ./...`)
- [ ] No `go vet` warnings
- [ ] Help text is clear (`shipcheck --help`)
- [ ] Edge cases handled (missing logs, empty repo, no git history)
- [ ] Added to CHANGELOG.md

---

## Environment Variables

```bash
SHIPCHECK_DIR         # Override scan directory
SHIPCHECK_FORMAT      # Default output format (tui|json|html)
SHIPCHECK_FAIL_ON     # Default fail threshold
SHIPCHECK_NO_COLOR    # Disable color output
NO_COLOR              # Standard no-color env var (also respected)
```

---

## Release Process

```bash
git tag v0.1.0
git push origin v0.1.0
# goreleaser handles: binary builds, GitHub release, Homebrew formula update
```

Install methods shipped at v0.1.0:
1. `brew install tejgokani/tap/shipcheck`
2. `curl -fsSL https://get.shipcheck.dev | sh`
3. Download binary from GitHub releases

---

## What Success Looks Like

After running `shipcheck` in a project for the first time, the developer should:
1. Immediately understand what they spent on their last session
2. Know which files their agent kept struggling with
3. See concrete security issues with line numbers
4. Have a ready-to-paste fix prompt for Claude Code
5. Have a shareable score they want to post

The tool should feel like a **receipt + X-ray + report card** for every agentic session.
