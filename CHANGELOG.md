# Changelog

All notable changes to shipcheck will be documented here.

Format: [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)
Versioning: [Semantic Versioning](https://semver.org/spec/v2.0.0.html)

---

## [Unreleased]

### Added
- Initial project structure
- CLAUDE.md, AGENTS.md, ARCHITECTURE.md, CONTRIBUTING.md, RULES.md
- Claude Code session log parser (`~/.claude/projects/*/sessions/*.jsonl`)
- Token cost calculation with model pricing table
- Per-file burn analysis from session logs
- Git-based heatmap (file touch frequency during session window)
- Security scanner with 31 deterministic rules
- TUI output with score, cost, heatmap, findings
- HTML report (self-contained, shareable)
- JSON output for CI pipelines
- `shipcheck init` to install post-session hooks
- `--fail-on` flag for CI gate
- `.shipcheckirgnore` support
- Homebrew formula
- `curl | sh` install script
- GitHub Actions CI + release workflow

---

## Version Naming

`v0.x.0` — pre-1.0, breaking changes may happen between minor versions
`v1.0.0` — stable public API, JSON output format locked
