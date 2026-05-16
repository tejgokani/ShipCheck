# shipcheck

**Post-session audit for AI-generated code. Cost. Heat. Security. One command.**

```bash
brew install tejgokani/tap/shipcheck
# or
curl -fsSL https://get.shipcheck.dev | sh
```

---

```
$ shipcheck

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
  > Fix the 3 security issues found by shipcheck: SQL injection at
    routes.go:47, exposed service_role key, and wildcard CORS in
    auth/handler.ts. Apply server-side fixes only.

  Run `shipcheck report` for full HTML report.
```

---

## Why

Every existing tool does one thing:

- Token monitors show cost but know nothing about your code
- Security scanners scan code but know nothing about your session
- **None** of them combine both. **None** run automatically.

shipcheck does all three — cost, heatmap, security — in a single offline pass that runs after every session.

---

## What It Does

### Cost & burn
Reads session logs from Claude Code, Cursor, and Codex. Shows what you spent, where you spent it, and what cache savings you got.

### Heatmap
Correlates git history with session timestamps to show which files your agent kept touching and retrying — the files that are costing you the most time and tokens.

### Security scan
Deterministic AST-based rules built specifically for AI-generated code failure patterns. Not generic SAST rules. Rules for things AI actually does wrong:

- Hardcoded API keys (OpenAI, Anthropic, Stripe, Supabase, SendGrid, and 40+ more)
- `NEXT_PUBLIC_` exposing backend secrets to the browser
- SQL queries built with string concatenation
- `Access-Control-Allow-Origin: *` applied to "fix" CORS
- `verify=False` applied to "fix" SSL errors
- Supabase `service_role` key in frontend code
- JWT secrets set to `"secret"` or placeholder values

---

## Install

### Homebrew (macOS / Linux)
```bash
brew install tejgokani/tap/shipcheck
```

### Binary (macOS / Linux — one-liner)
```bash
curl -fsSL https://github.com/tejgokani/ShipCheck/releases/latest/download/shipcheck_$(uname -s)_$(uname -m).tar.gz \
  | tar -xz && sudo mv shipcheck /usr/local/bin/
```

> **Windows:** download `shipcheck_0.1.0_Windows_x86_64.zip` from [GitHub Releases](https://github.com/tejgokani/ShipCheck/releases/latest), extract, and add to your PATH.

### Manual download
| Platform | File |
|---|---|
| macOS (Apple Silicon) | `shipcheck_0.1.0_Darwin_arm64.tar.gz` |
| macOS (Intel) | `shipcheck_0.1.0_Darwin_x86_64.tar.gz` |
| Linux arm64 | `shipcheck_0.1.0_Linux_arm64.tar.gz` |
| Linux x86_64 | `shipcheck_0.1.0_Linux_x86_64.tar.gz` |
| Windows x86_64 | `shipcheck_0.1.0_Windows_x86_64.zip` |

All assets: [github.com/tejgokani/ShipCheck/releases/latest](https://github.com/tejgokani/ShipCheck/releases/latest)

### From source
```bash
git clone https://github.com/tejgokani/shipcheck
cd shipcheck
go build -o shipcheck .
```

---

## Usage

```bash
shipcheck                  # full audit — cost + heatmap + security
shipcheck --html           # generate HTML report and open in browser
shipcheck --json           # JSON output for CI pipelines
shipcheck scan --sec       # security scan only
shipcheck scan --cost      # cost + burn only
shipcheck scan --heat      # heatmap only
shipcheck report           # open the last HTML report
shipcheck init             # install post-session hooks
```

### Flags

```
--dir, -d       directory to scan (default: current)
--since         only include sessions since (e.g. 24h, 7d)
--fail-on       exit 1 if findings >= severity: critical|high|medium
--depth         git history depth for heatmap (default: 50)
--no-session    skip session log analysis
--no-security   skip security scan
--no-heatmap    skip heatmap
--quiet         print score only (0-100)
--format        tui|json|html (default: tui)
```

### Auto-run after every session

```bash
shipcheck init
```

This installs a post-session hook so shipcheck runs automatically after every Claude Code or Cursor session. Prints a warning if your score drops below 70.

---

## CI Integration

```yaml
# .github/workflows/shipcheck.yml
name: shipcheck
on: [pull_request]
jobs:
  audit:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install shipcheck
        run: curl -fsSL https://get.shipcheck.dev | sh
      - name: Run audit
        run: shipcheck --json --fail-on high
```

---

## Score

```
CLEAN   90–100   No critical or high findings
GOOD    70–89    Minor issues only
REVIEW  50–69    Some concerning patterns
RISKY   30–49    Multiple high severity findings
DANGER  0–29     Critical issues found
```

---

## Supported Tools

| Tool | Session logs | Cost tracking |
|------|-------------|---------------|
| Claude Code | ✓ | ✓ |
| Cursor | ✓ | ✓ |
| Codex (OpenAI) | ✓ | ✓ |
| Gemini CLI | coming soon | coming soon |
| Aider | coming soon | coming soon |

---

## Supported Languages (Security Scan)

Go, TypeScript, JavaScript, Python, Rust, YAML, JSON, `.env` files

---

## Zero network calls

shipcheck runs 100% offline. Your code never leaves your machine. No API keys required. No telemetry. No update checks at runtime.

---

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md). The most impactful contribution is adding new security rules — each one helps every vibe coder who installs shipcheck.

---

## License

MIT — free to use, modify, and distribute.

---

Built because every AI coding session deserves a receipt.
