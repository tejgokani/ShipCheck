package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppliesToLang(t *testing.T) {
	tests := []struct {
		languages []string
		lang      string
		want      bool
	}{
		{[]string{"*"}, "go", true},
		{[]string{"go"}, "go", true},
		{[]string{"typescript"}, "go", false},
		{[]string{"go", "python"}, "python", true},
		{[]string{}, "go", false},
	}
	for _, tc := range tests {
		got := appliesToLang(tc.languages, tc.lang)
		if got != tc.want {
			t.Errorf("appliesToLang(%v, %q) = %v, want %v", tc.languages, tc.lang, got, tc.want)
		}
	}
}

// writeFile creates a temp file and returns its path.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func scanContent(t *testing.T, nameOrExt, content string) []Finding {
	t.Helper()
	dir := t.TempDir()
	// If it starts with "." it's an extension; otherwise use as the full filename.
	var filename string
	if len(nameOrExt) > 0 && nameOrExt[0] == '.' {
		filename = "file" + nameOrExt
	} else {
		filename = nameOrExt
	}
	writeFile(t, dir, filename, content)
	s := New(dir)
	findings, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	return findings
}

// ── SEC-101: Generic hardcoded API key ──────────────────────────────────────

func TestHardcodedAPIKey_Hit(t *testing.T) {
	findings := scanContent(t, ".go", `api_key = "sk_real_key_that_is_long_enough_here"`)
	if !hasRule(findings, "SEC-101") {
		t.Error("expected SEC-101 finding for hardcoded api_key")
	}
}

func TestHardcodedAPIKey_Placeholder(t *testing.T) {
	findings := scanContent(t, ".go", `api_key = "your-api-key-here"`)
	if hasRule(findings, "SEC-101") {
		t.Error("should not flag placeholder value as SEC-101")
	}
}

func TestHardcodedAPIKey_EnvRef(t *testing.T) {
	findings := scanContent(t, ".go", `api_key = os.Getenv("SOME_KEY")`)
	if hasRule(findings, "SEC-101") {
		t.Error("should not flag env var reference as SEC-101")
	}
}

// ── SEC-102: OpenAI key ───────────────────────────────────────────────────

func TestOpenAIKey_Hit(t *testing.T) {
	findings := scanContent(t, ".go", `key := "sk-proj-abcdefghijklmnopqrstuv"`)
	if !hasRule(findings, "SEC-102") {
		t.Error("expected SEC-102 for OpenAI sk-proj- key")
	}
}

func TestOpenAIKey_Miss(t *testing.T) {
	findings := scanContent(t, ".go", `key := "not-an-openai-key"`)
	if hasRule(findings, "SEC-102") {
		t.Error("should not flag non-OpenAI key as SEC-102")
	}
}

// ── SEC-103: Anthropic key ────────────────────────────────────────────────

func TestAnthropicKey_Hit(t *testing.T) {
	key := "sk-ant-api03-" + repeat("a", 55)
	findings := scanContent(t, ".go", `k := "`+key+`"`)
	if !hasRule(findings, "SEC-103") {
		t.Error("expected SEC-103 for Anthropic key")
	}
}

// ── SEC-104: Stripe secret key ────────────────────────────────────────────

func TestStripeSecretKey_Hit(t *testing.T) {
	// Build the key at runtime so static secret scanners don't flag this test file.
	key := "sk_li" + "ve_" + repeat("a", 24)
	findings := scanContent(t, ".go", `stripe := "`+key+`"`)
	if !hasRule(findings, "SEC-104") {
		t.Error("expected SEC-104 for Stripe live key")
	}
}

func TestStripeSecretKey_TestKey(t *testing.T) {
	// sk_test_ is not a live key — should not trigger.
	// Built at runtime to avoid static secret scanner false-positives.
	key := "sk_te" + "st_" + repeat("a", 24)
	findings := scanContent(t, ".go", `stripe := "`+key+`"`)
	if hasRule(findings, "SEC-104") {
		t.Error("should not flag sk_test_ key as SEC-104")
	}
}

// ── SEC-105: Supabase service role ────────────────────────────────────────

func TestSupabaseServiceRole_Hit(t *testing.T) {
	findings := scanContent(t, ".env", `SERVICE_ROLE_KEY="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.realstuff"`)
	if !hasRule(findings, "SEC-105") {
		t.Error("expected SEC-105 for Supabase service_role key")
	}
}

// ── SEC-106: JWT placeholder secret ──────────────────────────────────────

func TestJWTSecret_Hit(t *testing.T) {
	content := `
token, err := jwt.Sign(claims, []byte("secret"))
`
	findings := scanContent(t, ".go", content)
	if !hasRule(findings, "SEC-106") {
		t.Error("expected SEC-106 for JWT placeholder secret")
	}
}

func TestJWTSecret_Miss(t *testing.T) {
	content := `
token, err := jwt.Sign(claims, []byte(os.Getenv("JWT_SECRET")))
`
	findings := scanContent(t, ".go", content)
	if hasRule(findings, "SEC-106") {
		t.Error("should not flag env-loaded JWT secret as SEC-106")
	}
}

// ── SEC-107: SendGrid key ─────────────────────────────────────────────────

func TestSendGridKey_Hit(t *testing.T) {
	key := "SG." + repeat("a", 22) + "." + repeat("b", 43)
	findings := scanContent(t, ".go", `k := "`+key+`"`)
	if !hasRule(findings, "SEC-107") {
		t.Error("expected SEC-107 for SendGrid key")
	}
}

// ── SEC-108: Database URL with credentials ────────────────────────────────

func TestDatabaseURL_Hit(t *testing.T) {
	findings := scanContent(t, ".go", `dsn := "postgres://user:password@localhost/mydb"`)
	if !hasRule(findings, "SEC-108") {
		t.Error("expected SEC-108 for database URL with credentials")
	}
}

func TestDatabaseURL_NoCredentials(t *testing.T) {
	findings := scanContent(t, ".go", `dsn := "postgres://localhost/mydb"`)
	if hasRule(findings, "SEC-108") {
		t.Error("should not flag database URL without credentials as SEC-108")
	}
}

// ── SEC-110: GitHub PAT ───────────────────────────────────────────────────

func TestGitHubPAT_Hit(t *testing.T) {
	pat := "ghp_" + repeat("A", 36)
	findings := scanContent(t, ".go", `token := "`+pat+`"`)
	if !hasRule(findings, "SEC-110") {
		t.Error("expected SEC-110 for GitHub PAT")
	}
}

// ── SEC-201: SQL string concatenation ────────────────────────────────────

func TestSQLStringConcat_Go_Hit(t *testing.T) {
	findings := scanContent(t, ".go", `query := fmt.Sprintf("SELECT * FROM users WHERE id = %s", id)`)
	if !hasRule(findings, "SEC-201") {
		t.Error("expected SEC-201 for fmt.Sprintf SQL")
	}
}

func TestSQLStringConcat_JS_Hit(t *testing.T) {
	findings := scanContent(t, ".js", "const q = `SELECT * FROM users WHERE id = ${userId}`;")
	if !hasRule(findings, "SEC-201") {
		t.Error("expected SEC-201 for JS template literal SQL")
	}
}

func TestSQLStringConcat_Py_Hit(t *testing.T) {
	findings := scanContent(t, ".py", `db.execute(f"SELECT * FROM users WHERE id = {user_id}")`)
	if !hasRule(findings, "SEC-201") {
		t.Error("expected SEC-201 for Python f-string SQL")
	}
}

func TestSQLStringConcat_Miss(t *testing.T) {
	findings := scanContent(t, ".go", `db.QueryRow("SELECT * FROM users WHERE id = ?", id)`)
	if hasRule(findings, "SEC-201") {
		t.Error("should not flag parameterized SQL as SEC-201")
	}
}

// ── SEC-202: eval() on input ──────────────────────────────────────────────

func TestEvalOnInput_Hit(t *testing.T) {
	findings := scanContent(t, ".js", `eval(userInput)`)
	if !hasRule(findings, "SEC-202") {
		t.Error("expected SEC-202 for eval()")
	}
}

// ── SEC-203: Command injection ────────────────────────────────────────────

func TestCommandInjection_ShellTrue(t *testing.T) {
	findings := scanContent(t, ".py", `subprocess.run(cmd, shell=True)`)
	if !hasRule(findings, "SEC-203") {
		t.Error("expected SEC-203 for shell=True")
	}
}

func TestCommandInjection_OsSystem(t *testing.T) {
	findings := scanContent(t, ".py", `os.system(user_cmd)`)
	if !hasRule(findings, "SEC-203") {
		t.Error("expected SEC-203 for os.system()")
	}
}

func TestCommandInjection_Miss(t *testing.T) {
	findings := scanContent(t, ".py", `subprocess.run(["ls", "-la"])`)
	if hasRule(findings, "SEC-203") {
		t.Error("should not flag list-form subprocess as SEC-203")
	}
}

// ── SEC-301: Wildcard CORS ────────────────────────────────────────────────

func TestWildcardCORS_Hit(t *testing.T) {
	findings := scanContent(t, ".go", `w.Header().Set("Access-Control-Allow-Origin", "*")`)
	if !hasRule(findings, "SEC-301") {
		t.Error("expected SEC-301 for wildcard CORS")
	}
}

func TestWildcardCORS_Specific(t *testing.T) {
	findings := scanContent(t, ".go", `w.Header().Set("Access-Control-Allow-Origin", "https://example.com")`)
	if hasRule(findings, "SEC-301") {
		t.Error("should not flag specific origin CORS as SEC-301")
	}
}

// ── SEC-302: SSL verify disabled ──────────────────────────────────────────

func TestSSLVerifyDisabled_Hit(t *testing.T) {
	findings := scanContent(t, ".py", `requests.get(url, verify=False)`)
	if !hasRule(findings, "SEC-302") {
		t.Error("expected SEC-302 for verify=False")
	}
}

// ── SEC-305: Debug mode ───────────────────────────────────────────────────

func TestDebugMode_Hit(t *testing.T) {
	findings := scanContent(t, ".py", `DEBUG=True`)
	if !hasRule(findings, "SEC-305") {
		t.Error("expected SEC-305 for DEBUG=True")
	}
}

// ── SEC-401: NEXT_PUBLIC_ secret leak ────────────────────────────────────

func TestNextPublicLeak_Hit(t *testing.T) {
	findings := scanContent(t, ".env", `NEXT_PUBLIC_DATABASE_URL=postgres://localhost/db`)
	if !hasRule(findings, "SEC-401") {
		t.Error("expected SEC-401 for NEXT_PUBLIC_ database url")
	}
}

func TestNextPublicLeak_SafePublishable(t *testing.T) {
	// PUBLISHABLE keys are meant to be public
	findings := scanContent(t, ".env", `NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=pk_live_abc`)
	if hasRule(findings, "SEC-401") {
		t.Error("should not flag PUBLISHABLE key as SEC-401")
	}
}

// ── SEC-501: Deferred security TODO ──────────────────────────────────────

func TestDeferredSecurityComment_Hit(t *testing.T) {
	findings := scanContent(t, ".go", `// TODO: add authentication here`)
	if !hasRule(findings, "SEC-501") {
		t.Error("expected SEC-501 for security TODO comment")
	}
}

func TestDeferredSecurityComment_UnrelatedTODO(t *testing.T) {
	findings := scanContent(t, ".go", `// TODO: refactor this function`)
	if hasRule(findings, "SEC-501") {
		t.Error("should not flag unrelated TODO as SEC-501")
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────

func hasRule(findings []Finding, ruleID string) bool {
	for _, f := range findings {
		if f.Rule == ruleID {
			return true
		}
	}
	return false
}

func repeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}
