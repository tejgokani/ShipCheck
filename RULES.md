# RULES.md — Security Rule Catalog

Complete reference for all security rules in shipcheck. Rules are deterministic (no LLM required). Every rule has a documented reason why AI generates this specific pattern.

---

## Rule ID Format

`SEC-<category><number>`
- `SEC-1xx` — Secrets & Tokens
- `SEC-2xx` — Injection
- `SEC-3xx` — Auth & Access Control
- `SEC-4xx` — Frontend / Client-side
- `SEC-5xx` — AI-specific patterns
- `SEC-6xx` — Dependencies

---

## Category 1: Secrets & Tokens

### SEC-101 — Generic API Key Pattern
**Severity:** Critical
**Languages:** All
**Why AI does this:** When an agent gets an API authentication error, it "fixes" it by hardcoding the key directly in the code instead of using env vars.

Pattern: Variable name contains `api_key`, `apikey`, `api-key`, `secret`, `token`, `password` and value is a string literal of 20+ characters that looks like a token.

```
// CAUGHT
const OPENAI_API_KEY = "sk-proj-abc123..."
apiKey := "sk-ant-api03-..."

// IGNORED
const apiKey = process.env.OPENAI_API_KEY
```

---

### SEC-102 — OpenAI API Key
**Severity:** Critical
**Languages:** All
**Pattern:** `sk-proj-[a-zA-Z0-9]{48}` or `sk-[a-zA-Z0-9]{48}`

---

### SEC-103 — Anthropic API Key
**Severity:** Critical
**Languages:** All
**Pattern:** `sk-ant-api03-[a-zA-Z0-9_-]{93}`

---

### SEC-104 — Stripe Secret Key
**Severity:** Critical
**Languages:** All
**Pattern:** `sk_live_[a-zA-Z0-9]{24,}` or `rk_live_[a-zA-Z0-9]{24,}`
**Note:** Test keys (`sk_test_`) are flagged as Medium — they reveal patterns.

---

### SEC-105 — Supabase Service Role Key
**Severity:** Critical
**Languages:** All
**Why AI does this:** When Row Level Security blocks a query, the agent "fixes" it by switching from the `anon` key to the `service_role` key — which bypasses ALL security policies.

Pattern: `service_role` key used in frontend code OR in a client-side bundle.

```
// CAUGHT
const supabase = createClient(url, process.env.SUPABASE_SERVICE_ROLE_KEY)
// in any frontend file (Next.js, React, etc.)

// IGNORED
// service_role key used in server-only code (API routes, server actions)
```

---

### SEC-106 — JWT Secret Placeholder
**Severity:** Critical
**Languages:** All
**Why AI does this:** Boilerplate examples always use `"secret"` as the JWT secret. AI copies this and forgets to replace it.

Pattern: `jwt.sign(`, `verify(token,` followed within 3 lines by string literal: `"secret"`, `"your-secret"`, `"jwt-secret"`, `"mysecret"`, `"change-me"`

---

### SEC-107 — SendGrid API Key
**Severity:** Critical
**Pattern:** `SG\.[a-zA-Z0-9_-]{22,}\.[a-zA-Z0-9_-]{43}`

---

### SEC-108 — Database URL with Credentials
**Severity:** Critical
**Languages:** All
**Pattern:** Connection string containing username:password@ (e.g. `postgresql://user:pass@host/db`)

---

### SEC-109 — Vercel Token
**Severity:** Critical
**Pattern:** `vercel_[a-zA-Z0-9]{24}` or `vc_[a-zA-Z0-9_-]{40,}`

---

### SEC-110 — GitHub Personal Access Token
**Severity:** Critical
**Pattern:** `ghp_[a-zA-Z0-9]{36}` or `github_pat_[a-zA-Z0-9_]{82}`

---

### SEC-111 — Railway Token
**Severity:** High
**Pattern:** `railway_[a-zA-Z0-9]{32}`

---

### SEC-112 — Cloudflare API Token
**Severity:** High
**Pattern:** 40-char hex string near `cloudflare` variable name

---

## Category 2: Injection

### SEC-201 — SQL String Concatenation
**Severity:** Critical
**Languages:** Go, TypeScript, JavaScript, Python
**Why AI does this:** AI generates the simplest working SQL first. String concatenation works — it just opens SQL injection. AI rarely adds parameterization unless explicitly asked.

```python
# CAUGHT
query = f"SELECT * FROM users WHERE id = {user_id}"
cursor.execute("SELECT * FROM users WHERE name = '" + name + "'")

# IGNORED
cursor.execute("SELECT * FROM users WHERE id = ?", (user_id,))
cursor.execute("SELECT * FROM users WHERE name = %s", (name,))
```

---

### SEC-202 — Raw `eval()` on User Input
**Severity:** Critical
**Languages:** TypeScript, JavaScript, Python
**Pattern:** `eval(` or `exec(` where the argument contains a variable that could be user-controlled (function parameter, request body, query param)

---

### SEC-203 — Command Injection via Shell
**Severity:** Critical
**Languages:** Go, TypeScript, JavaScript, Python
**Why AI does this:** When building CLI wrappers, AI often uses `shell=True` or string interpolation into shell commands.

```python
# CAUGHT
subprocess.run(f"git clone {repo_url}", shell=True)
os.system("convert " + filename + " output.png")

# IGNORED
subprocess.run(["git", "clone", repo_url])
```

---

### SEC-204 — Path Traversal
**Severity:** High
**Languages:** All
**Pattern:** User-controlled input used in file path operations without sanitization. Look for `../` concatenation or `os.ReadFile(userInput)` patterns.

---

### SEC-205 — XML/HTML Injection (XSS)
**Severity:** High
**Languages:** TypeScript, JavaScript
**Pattern:** `innerHTML =` with a variable, `dangerouslySetInnerHTML` with unsanitized content

---

## Category 3: Auth & Access Control

### SEC-301 — Wildcard CORS
**Severity:** Medium
**Languages:** Go, TypeScript, JavaScript, Python
**Why AI does this:** When CORS blocks a request during development, the agent "fixes" it with `*` instead of specifying allowed origins.

```go
// CAUGHT
w.Header().Set("Access-Control-Allow-Origin", "*")

// IGNORED
w.Header().Set("Access-Control-Allow-Origin", os.Getenv("ALLOWED_ORIGIN"))
```

---

### SEC-302 — SSL Verification Disabled
**Severity:** High
**Languages:** Python
**Why AI does this:** When an SSL certificate error occurs, the agent adds `verify=False` to "fix" it — a common AI shortcut.

```python
# CAUGHT
requests.get(url, verify=False)
httpx.get(url, verify=False)
```

---

### SEC-303 — Missing Auth on Sensitive Route
**Severity:** High
**Languages:** Go, TypeScript, JavaScript
**Pattern:** Route handlers containing keywords `delete`, `admin`, `payment`, `transfer`, `export` that don't have authentication middleware applied.

---

### SEC-304 — Hardcoded Admin Credentials
**Severity:** Critical
**Pattern:** Username/password variables set to `"admin"/"admin"`, `"root"/"root"`, `"test"/"test"`, or similar default credentials.

---

### SEC-305 — Debug Mode Enabled
**Severity:** Medium
**Languages:** Python, TypeScript, JavaScript
**Why AI does this:** AI generated code often includes `DEBUG=True` or `debug: true` left over from development scaffolding.

```python
# CAUGHT (in non-test file)
DEBUG = True
app.run(debug=True)
```

---

## Category 4: Frontend / Client-side

### SEC-401 — NEXT_PUBLIC_ Secret Leak
**Severity:** Critical
**Languages:** TypeScript, JavaScript
**Why AI does this:** When fixing "env var not found in browser" errors, AI prefixes backend env vars with `NEXT_PUBLIC_` — which exposes them to every browser that loads the page.

```
# CAUGHT (in .env or .env.local)
NEXT_PUBLIC_DATABASE_URL=postgresql://...
NEXT_PUBLIC_SUPABASE_SERVICE_ROLE_KEY=eyJ...
NEXT_PUBLIC_STRIPE_SECRET_KEY=sk_live_...

# IGNORED
NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=pk_live_...  (this one is intentionally public)
```

---

### SEC-402 — VITE_ / REACT_APP_ Secret Leak
**Severity:** Critical
**Languages:** TypeScript, JavaScript
**Same pattern as SEC-401 but for Vite (`VITE_`) and CRA (`REACT_APP_`) prefixes applied to backend secrets.**

---

### SEC-403 — Client-side Auth Check
**Severity:** High
**Languages:** TypeScript, JavaScript
**Why AI does this:** AI often implements auth checks in React components/pages instead of server middleware — these can be bypassed by disabling JavaScript.

Pattern: Auth/role checks in `.tsx`/`.jsx` files that are not in a `middleware.ts` file or `getServerSideProps`.

---

## Category 5: AI-Specific Patterns

### SEC-501 — Deferred Security Comment
**Severity:** Low
**Languages:** All
**Why this exists:** AI commonly generates code with TODO comments like "add auth later" — these mark known security shortcuts that never get addressed.

```
// CAUGHT
// TODO: add authentication
// TODO: validate user input
// FIXME: this is insecure
// TODO: use env var instead of hardcoding
```

---

### SEC-502 — In-Memory Rate Limiting
**Severity:** Medium
**Languages:** Go, TypeScript, JavaScript, Python
**Why AI does this:** AI implements rate limiting with in-memory maps/counters — these reset every time the process restarts, making them useless in production.

Pattern: Rate limiting implemented with a variable like `rateLimitMap`, `requestCounts`, `ipCounts` that is a local map (not Redis/database backed).

---

### SEC-503 — Retry Loop Without Backoff
**Severity:** Low
**Languages:** All
**Pattern:** `for` loops with `retry` in variable name but no exponential backoff or sleep — AI generates retry loops that hammer APIs.

---

### SEC-504 — Hallucinated Package Import
**Severity:** Medium
**Languages:** TypeScript, JavaScript (npm), Python (pip)
**Why AI does this:** AI sometimes imports packages that don't exist or have different APIs than expected. These fail at install time but can also be supply-chain attack vectors if someone registers the package name.

Detection: Cross-reference imports against a bundled list of known-good packages. Flag anything not in the list that follows patterns of hallucinated names (e.g. `@openai/utils`, `react-auth-hooks`).

---

## Category 6: Dependencies

### SEC-601 — Package with Known CVE
**Severity:** High
**Languages:** TypeScript, JavaScript (package.json), Python (requirements.txt)
**Detection:** Check against a bundled CVE list (updated on releases). Flag packages with critical CVEs pinned to vulnerable versions.

---

### SEC-602 — Overprivileged npm Script
**Severity:** Medium
**Languages:** TypeScript, JavaScript
**Pattern:** `package.json` scripts that use `sudo`, `chmod 777`, or write to system directories.

---

## Rule Statistics

| Category | Rules | Critical | High | Medium | Low |
|----------|-------|----------|------|--------|-----|
| Secrets | 12 | 10 | 2 | 0 | 0 |
| Injection | 5 | 3 | 2 | 0 | 0 |
| Auth | 5 | 2 | 2 | 1 | 0 |
| Frontend | 3 | 2 | 1 | 0 | 0 |
| AI-Specific | 4 | 0 | 0 | 2 | 2 |
| Dependencies | 2 | 0 | 1 | 1 | 0 |
| **Total** | **31** | **17** | **8** | **4** | **2** |

---

## False Positive Suppression

Add a comment to suppress a specific finding:

```go
const apiKey = "sk-test-only-not-real"  //nolint:SEC-103
```

Or add to `.shipcheckirgnore`:
```
# Suppress SEC-301 for this specific test file
testdata/mock_server.go:SEC-301
```

---

## Suggesting New Rules

Open a GitHub issue with:
1. The pattern you want detected
2. Why AI generates this specific pattern
3. A vulnerable code example (positive case)
4. A clean code example (negative case — should NOT trigger)
5. The severity you'd assign and why
