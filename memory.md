# shipcheck — Claude Code Session Memory

This file is auto-read by Claude Code at the start of every session in this project.
It provides persistent context so the agent doesn't re-read the entire codebase each time.

---

## Current State

**Phase:** Pre-development — documentation phase complete, implementation starting
**Next milestone:** M0 — Foundation (go.mod, main.go, cmd/root.go, types defined)

## What's Done
- All documentation files written (CLAUDE.md, AGENTS.md, ARCHITECTURE.md, CONTRIBUTING.md, RULES.md, README.md, CHANGELOG.md)
- Project structure decided
- Tech stack decided: Go 1.22, cobra, bubbletea, lipgloss, tree-sitter
- All architecture decisions locked in ARCHITECTURE.md

## What's Next (in order)
1. `go mod init github.com/tejgokani/shipcheck`
2. Add dependencies to go.mod (cobra, bubbletea, lipgloss, viper, testify)
3. Write `internal/session/types.go` (foundational — other packages depend on it)
4. Write `cmd/root.go` and `cmd/scan.go` (thin wrappers)
5. Implement Claude Code parser (`internal/session/claude.go`)
6. Implement cost calculator (`internal/cost/calculator.go`)
7. Implement first 5 security rules (secrets category)
8. Wire up TUI output (`internal/report/tui.go`)
9. End-to-end test on a real project

## Key Decisions (don't re-debate these)
- Language: Go (not Rust, not Node.js)
- No LLM calls at runtime — deterministic rules only
- MIT license
- Session log paths documented in CLAUDE.md
- Score formula documented in CLAUDE.md
- TUI output format documented in CLAUDE.md

## Module Name
`github.com/tejgokani/shipcheck`

## Import Paths Pattern
Always use full paths: `github.com/tejgokani/shipcheck/internal/session`

## Test Command
`go test ./... -race`

## Build Command
`go build -o bin/shipcheck .`
