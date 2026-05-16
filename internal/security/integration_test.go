package security

import (
	"path/filepath"
	"runtime"
	"testing"
)

// testdataDir resolves the path to testdata/projects relative to this file.
func testdataDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "projects")
}

func TestScan_VulnerableProject(t *testing.T) {
	dir := filepath.Join(testdataDir(), "vulnerable")
	s := New(dir)
	findings, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if len(findings) == 0 {
		t.Fatal("expected findings in vulnerable project, got none")
	}

	// Check that specific rules fire.
	want := []string{"SEC-301", "SEC-201", "SEC-108", "SEC-203", "SEC-302", "SEC-305", "SEC-401", "SEC-501"}
	ruleSet := make(map[string]bool, len(findings))
	for _, f := range findings {
		ruleSet[f.Rule] = true
	}
	for _, id := range want {
		if !ruleSet[id] {
			t.Errorf("expected rule %s to fire in vulnerable project, but it didn't (findings: %v)", id, ruleIDs(findings))
		}
	}
}

func TestScan_CleanProject(t *testing.T) {
	dir := filepath.Join(testdataDir(), "clean")
	s := New(dir)
	findings, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	// Clean project should have zero findings (the .env only has placeholders / env refs).
	critical := 0
	for _, f := range findings {
		if f.Severity == Critical {
			critical++
			t.Logf("unexpected critical finding: %s %s:%d %s", f.Rule, f.File, f.Line, f.Message)
		}
	}
	if critical > 0 {
		t.Errorf("expected zero critical findings in clean project, got %d", critical)
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
