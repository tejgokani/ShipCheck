// Package security provides deterministic AST-based security scanning rules.
package security

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	gitignore "github.com/sabhiram/go-gitignore"
)

// langByExt maps file extensions to language identifiers.
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

// allRules is the full registered rule set (rules live in rules_*.go in this package).
var allRules = []Rule{
	HardcodedAPIKey,
	OpenAIKey,
	AnthropicKey,
	StripeSecretKey,
	SupabaseServiceRole,
	JWTSecret,
	SendGridKey,
	DatabaseURL,
	GitHubPAT,
	SQLStringConcat,
	EvalOnInput,
	CommandInjection,
	WildcardCORS,
	SSLVerifyDisabled,
	DebugMode,
	NextPublicLeak,
	DeferredSecurityComment,
}

// Scanner runs all registered rules over a directory.
type Scanner struct {
	dir string
	ig  *gitignore.GitIgnore
	si  *gitignore.GitIgnore // .shipcheckirgnore
}

// New creates a Scanner for the given directory.
func New(dir string) *Scanner {
	ig, _ := gitignore.CompileIgnoreFile(filepath.Join(dir, ".gitignore"))
	si, _ := gitignore.CompileIgnoreFile(filepath.Join(dir, ".shipcheckirgnore"))
	return &Scanner{dir: dir, ig: ig, si: si}
}

// Scan walks dir and applies all rules, returning all findings.
func (s *Scanner) Scan() ([]Finding, error) {
	var files []string
	err := filepath.Walk(s.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(s.dir, path)
		if s.ig != nil && s.ig.MatchesPath(rel) {
			return nil
		}
		if s.si != nil && s.si.MatchesPath(rel) {
			return nil
		}
		if _, ok := langByExt[filepath.Ext(path)]; ok {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sem := make(chan struct{}, runtime.NumCPU())
	var wg sync.WaitGroup
	var mu sync.Mutex
	var findings []Finding

	for _, f := range files {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			content, err := os.ReadFile(path)
			if err != nil {
				log.Printf("warn: security: cannot read %s: %v", path, err)
				return
			}
			lang := langByExt[filepath.Ext(path)]
			for _, rule := range allRules {
				if !appliesToLang(rule.Languages, lang) {
					continue
				}
				hits := rule.Check(path, content)
				if len(hits) > 0 {
					mu.Lock()
					findings = append(findings, hits...)
					mu.Unlock()
				}
			}
		}(f)
	}
	wg.Wait()
	return findings, nil
}

func appliesToLang(languages []string, lang string) bool {
	for _, l := range languages {
		if l == "*" || l == lang {
			return true
		}
	}
	return false
}
