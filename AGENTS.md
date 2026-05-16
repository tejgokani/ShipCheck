# AGENTS.md — shipcheck Agent Team

This file defines the agent team for building shipcheck in parallel using
Claude Code's Agent Teams feature. Each agent owns a specific layer of the
codebase and should not touch files outside its domain without coordination.

---

## Agent Team Overview

```
                    ┌─────────────────────┐
                    │   LEAD (Orchestrator)│
                    │   Coordinates all    │
                    │   agent work         │
                    └──────────┬──────────┘
          ┌───────────┬────────┴────────┬───────────┐
          ▼           ▼                 ▼           ▼
    ┌──────────┐ ┌──────────┐   ┌──────────┐ ┌──────────┐
    │  PARSER  │ │ SECURITY │   │  OUTPUT  │ │  INFRA   │
    │  Agent   │ │  Agent   │   │  Agent   │ │  Agent   │
    └──────────┘ └──────────┘   └──────────┘ └──────────┘
```

---

## Agent Definitions

### LEAD — Orchestrator

**Responsibility:** Owns `cmd/`, `main.go`, `go.mod`. Coordinates work across agents. Makes architectural decisions. Writes integration tests.

**Files owned:**
```
main.go
cmd/root.go
cmd/scan.go
cmd/init.go
cmd/report.go
cmd/version.go
go.mod
go.sum
```

**Instructions:**
- You are the orchestrator. Spawn the other agents to build their layers in parallel.
- After each agent reports completion, integrate their work via the `cmd/scan.go` pipeline.
- Never implement business logic in `cmd/` — only wire up calls to `internal/`.
- Run `go build ./...` after each integration step to catch import errors early.
- Run `go test ./...` before marking any milestone complete.
- Keep `cmd/scan.go` under 100 lines — it should only orchestrate calls to `internal/`.

**Coordination rules:**
- Assign PARSER agent first (other agents depend on its types)
- SECURITY and OUTPUT agents can run in parallel after PARSER defines `internal/session/types.go`
- INFRA agent runs in parallel from the start — it has no code dependencies
- Block on PARSER completing `types.go` before allowing SECURITY or OUTPUT to proceed

---

### PARSER — Session & Cost Agent

**Responsibility:** Owns all session log parsing and cost calculation logic.

**Files owned:**
```
internal/session/
internal/cost/
internal/heatmap/
```

**Instructions:**
- Start with `internal/session/types.go` — define the shared `Session`, `SessionFile`, `TokenUsage` structs. This unblocks all other agents.
- Parse JSONL session logs for Claude Code, Cursor, and Codex.
- Claude Code session logs are at `~/.claude/projects/*/sessions/*.jsonl`
- Each line is a JSON object — key fields: `type`, `cost`, `tokensIn`, `tokensOut`, `cacheReads`, `cacheWrites`, `timestamp`
- Cursor logs are at `~/.cursor/logs/` — format varies by version, handle gracefully
- Codex logs are at `~/.codex/sessions/` — JSONL per session
- Handle missing log directories with a warning, not a fatal error
- Cost calculation: multiply token counts by model pricing from `internal/cost/models.go`
- Heatmap: use `git log --follow --name-only` output to count file touches, correlated with session timestamps
- Write table-driven tests for every parser — use fixtures in `testdata/sessions/`
- Never panic on malformed JSON — log the line and continue

**Key types to define first (other agents depend on these):**

```go
// internal/session/types.go
type Session struct {
    Tool       string        // "claude-code" | "cursor" | "codex"
    StartTime  time.Time
    EndTime    time.Time
    Files      []SessionFile
    TotalCost  float64
    Usage      TokenUsage
    ModelID    string
}

type SessionFile struct {
    Path        string
    TouchCount  int
    TokensSpent int64
}

type TokenUsage struct {
    InputTokens  int64
    OutputTokens int64
    CacheReads   int64
    CacheWrites  int64
}
```

**Report to LEAD when:** `internal/session/types.go` is written and compiled successfully.

---

### SECURITY — Security Scanner Agent

**Responsibility:** Owns all security rule logic and scanning.

**Files owned:**
```
internal/security/
rules/
testdata/projects/
```

**Instructions:**
- Wait for PARSER to define `internal/session/types.go` before starting (you need the `SessionFile` type)
- Build deterministic AST-based rules — NO LLM calls, NO external API calls
- Use `go/ast` for Go files, `github.com/smacker/go-tree-sitter` for JS/TS/Python
- Each rule is a function: `func(file string, content []byte) []Finding`
- Rules live in `internal/security/rules/` — one file per category
- Every rule MUST have a comment explaining why AI generates this specific pattern
- Load rule definitions from `rules/*.yaml` for the human-readable rule list
- Respect `.gitignore` — use `github.com/sabhiram/go-gitignore` library
- Also respect `.shipcheckirgnore` if present (same format as .gitignore)
- Scan performance target: 10,000 source files in under 2 seconds
- Write tests using fixtures in `testdata/projects/` — create both vulnerable and clean versions

**Finding type:**

```go
// internal/security/types.go
type Finding struct {
    Rule        string
    Severity    Severity  // Critical | High | Medium | Low | Info
    File        string
    Line        int
    Column      int
    Message     string
    Evidence    string    // the actual code snippet
    FixPrompt   string    // ready-to-paste prompt for Claude Code
}

type Severity int
const (
    Info Severity = iota
    Low
    Medium
    High
    Critical
)
```

**Priority rules to implement first:**
1. Hardcoded secrets (regex — fastest to implement, highest value)
2. `NEXT_PUBLIC_` secrets leak
3. SQL string concatenation
4. `Access-Control-Allow-Origin: *`
5. `verify=False` / `ssl=False`

**Report to LEAD when:** `internal/security/scanner.go` compiles and at least 5 rules are passing tests.

---

### OUTPUT — Report & TUI Agent

**Responsibility:** Owns all output rendering — TUI, HTML, JSON.

**Files owned:**
```
internal/report/
```

**Instructions:**
- Wait for PARSER to define `internal/session/types.go` before starting
- The TUI output uses `lipgloss` for styling — keep it clean, no unnecessary decorations
- Color palette: use only these colors for consistency:
  - Critical: `#E24B4A` (red)
  - High: `#EF9F27` (amber)
  - Medium: `#378ADD` (blue)
  - Low: `#888780` (gray)
  - Good/score: `#1D9E75` (teal)
  - Score bar background: `#D3D1C7`
- The HTML report must be a single self-contained file — inline all CSS and JS
- HTML report must work offline — no CDN links
- HTML report must have a visible score (big number, top center), three cards (Cost / Heat / Security), and a copy-to-clipboard fix prompt button
- JSON output must be valid JSON that CI tools can parse with `jq`
- Score calculation lives in `internal/report/score.go` — import the formula from CLAUDE.md
- The fix prompt must be automatically generated from the findings list

**Score output format:**

```go
// internal/report/score.go
type AuditResult struct {
    Score      int
    Label      string    // CLEAN | GOOD | REVIEW | RISKY | DANGER
    Sessions   []session.Session
    Findings   []security.Finding
    HotFiles   []heatmap.HotFile
    GeneratedAt time.Time
    Directory  string
}
```

**Report to LEAD when:** `internal/report/tui.go` renders a complete mock result without panicking.

---

### INFRA — Build, CI & Distribution Agent

**Responsibility:** Owns everything related to building, testing, releasing, and distribution. Has no runtime code dependencies — can start immediately.

**Files owned:**
```
.goreleaser.yaml
.github/
scripts/
Makefile
.shipcheckirgnore (template)
```

**Instructions:**
- No dependencies on other agents — start immediately in parallel
- Set up `.goreleaser.yaml` for cross-platform builds: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`
- Create `Makefile` with these targets: `build`, `test`, `lint`, `release-dry`, `install-hooks`
- Write `scripts/install.sh` — the `curl | sh` installer that detects OS/arch and downloads the right binary from GitHub releases
- Write `scripts/hooks/claude-code-hook` — a shell script that runs `shipcheck --quiet` after a Claude Code session ends, only if the user ran `shipcheck init`
- Set up GitHub Actions in `.github/workflows/`:
  - `ci.yml` — runs `go test ./...` and `go vet ./...` on every PR
  - `release.yml` — triggers goreleaser on new tags
  - `security.yml` — runs shipcheck itself on every PR (dogfooding)
- Create `.shipcheckirgnore` template with sensible defaults (node_modules, .git, vendor, dist, build, etc.)

**Makefile targets:**

```makefile
build:
    go build -ldflags="-s -w -X main.version=$(VERSION)" -o bin/shipcheck .

test:
    go test ./... -race -coverprofile=coverage.out

lint:
    golangci-lint run ./...

release-dry:
    goreleaser release --snapshot --clean

install-hooks:
    ./scripts/install-hooks.sh
```

**Report to LEAD when:** `.goreleaser.yaml` is written and `goreleaser check` passes (dry-run validation).

---

## Shared Rules for All Agents

1. **Type first** — if your package exports types that other agents consume, define types.go first and report to LEAD
2. **No globals** — no package-level mutable state; pass dependencies explicitly
3. **Errors wrapped** — `fmt.Errorf("packagename: action: %w", err)` — always add context
4. **Tests alongside code** — `scanner_test.go` lives next to `scanner.go`
5. **No premature optimization** — write readable code first, benchmark second
6. **Import paths** — always use the full module path: `github.com/tejgokani/shipcheck/internal/session`
7. **Comments on exported symbols** — every exported function/type needs a doc comment
8. **Never break compilation** — if you need a type from another agent that isn't ready, use a stub and leave a `// TODO(agent-name): replace stub` comment

---

## Coordination Protocol

When spawning the team, use this sequence:

```
1. Spawn INFRA immediately (no dependencies)
2. Spawn PARSER immediately (defines foundational types)
3. Wait for PARSER to report "types.go done"
4. Spawn SECURITY and OUTPUT in parallel
5. Collect all agents → integrate in cmd/scan.go
6. Run go test ./... → fix any integration issues
7. Run goreleaser release --snapshot → verify binary builds
```

---

## Milestone Checkpoints

| Milestone | Criteria |
|---|---|
| M0 — Foundation | `go build ./...` succeeds, types defined, project structure in place |
| M1 — Parser works | Session logs parsed for Claude Code, cost calculated, tests passing |
| M2 — Security works | 5+ rules scanning, findings produced with line numbers |
| M3 — Output works | TUI renders cleanly, HTML report generates, JSON is valid |
| M4 — Integration | `shipcheck` runs end-to-end on a real project without panic |
| M5 — Distribution | Goreleaser builds binaries, install.sh works, hooks install |
| M6 — v0.1.0 | README done, all tests green, binary published to GitHub Releases |
