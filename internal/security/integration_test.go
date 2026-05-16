package security

import (
	"os"
	"path/filepath"
	"testing"
)

// buildVulnerableProject writes a set of intentionally vulnerable files into dir.
func buildVulnerableProject(t *testing.T, dir string) {
	t.Helper()
	files := map[string]string{
		"server.py": `import os, requests, subprocess
from flask import Flask, request
app = Flask(__name__)
app.run(debug=True)
api_key = "sk_real_hardcoded_key_1234567890abcdef"
response = requests.get("https://api.example.com", verify=False)
user_input = request.args.get("cmd")
os.system(user_input)
subprocess.run(user_input, shell=True)
user_id = request.args.get("id")
query = f"SELECT * FROM users WHERE id = {user_id}"
# TODO: add authentication before shipping
`,
		"api.go": `package main
import (
	"fmt"
	"net/http"
)
func handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	id := r.URL.Query().Get("id")
	query := fmt.Sprintf("SELECT * FROM users WHERE id = %s", id)
	_ = query
	dsn := "postgres://admin:s3cr3tpassword@prod-db.example.com/appdb"
	_ = dsn
}
func main() { http.HandleFunc("/", handler); http.ListenAndServe(":8080", nil) }
`,
		".env": `NEXT_PUBLIC_DATABASE_URL=postgres://user:pass@localhost/db
NEXT_PUBLIC_SECRET_KEY=super-secret-backend-key-1234567890
SERVICE_ROLE_KEY="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.realservicerolekey"
`,
	}
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("buildVulnerableProject: write %s: %v", name, err)
		}
	}
}

// buildCleanProject writes files with no critical security issues into dir.
func buildCleanProject(t *testing.T, dir string) {
	t.Helper()
	files := map[string]string{
		"server.go": `package main
import (
	"net/http"
	"os"
)
func handler(w http.ResponseWriter, r *http.Request) {
	origin := os.Getenv("ALLOWED_ORIGIN")
	w.Header().Set("Access-Control-Allow-Origin", origin)
}
func main() { http.HandleFunc("/", handler); http.ListenAndServe(":8080", nil) }
`,
		".env": `DATABASE_URL=${DATABASE_URL}
STRIPE_SECRET_KEY=${STRIPE_SECRET_KEY}
NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=pk_test_placeholder
`,
	}
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("buildCleanProject: write %s: %v", name, err)
		}
	}
}

func TestScan_VulnerableProject(t *testing.T) {
	dir := t.TempDir()
	buildVulnerableProject(t, dir)

	s := New(dir)
	findings, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if len(findings) == 0 {
		t.Fatal("expected findings in vulnerable project, got none")
	}

	want := []string{"SEC-301", "SEC-201", "SEC-108", "SEC-203", "SEC-302", "SEC-305", "SEC-401", "SEC-501"}
	ruleSet := make(map[string]bool, len(findings))
	for _, f := range findings {
		ruleSet[f.Rule] = true
	}
	for _, id := range want {
		if !ruleSet[id] {
			t.Errorf("expected rule %s to fire in vulnerable project (findings: %v)", id, ruleIDs(findings))
		}
	}
}

func TestScan_CleanProject(t *testing.T) {
	dir := t.TempDir()
	buildCleanProject(t, dir)

	s := New(dir)
	findings, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	for _, f := range findings {
		if f.Severity == Critical {
			t.Errorf("unexpected critical finding: %s %s:%d %s", f.Rule, f.File, f.Line, f.Message)
		}
	}
}

func ruleIDs(findings []Finding) []string {
	seen := map[string]bool{}
	var ids []string
	for _, f := range findings {
		if !seen[f.Rule] {
			seen[f.Rule] = true
			ids = append(ids, f.Rule)
		}
	}
	return ids
}
