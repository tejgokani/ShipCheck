package security

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

// HardcodedAPIKey detects generic API key/secret assignments.
// AI does this when it gets an auth error and hardcodes the key to "fix" it.
var HardcodedAPIKey = Rule{
	ID:        "SEC-101",
	Name:      "Generic API Key",
	Severity:  Critical,
	Languages: []string{"*"},
	Check:     checkHardcodedAPIKey,
}

var (
	apiKeyNameRe = regexp.MustCompile(`(?i)(api[_-]?key|apikey|secret|token|password|passwd|pwd)\s*[:=]\s*["']([^"']{20,})["']`)
	// envVarRefRe matches patterns indicating the value is loaded from env, not hardcoded.
	// Covers Go, Python, JS/TS, and template languages (goreleaser, Helm, etc.).
	envVarRefRe = regexp.MustCompile(`(?i)(process\.env|os\.getenv|os\.environ|os\.Getenv|config\.|settings\.|{{\s*\.Env\.|getenv\()`)
)

func checkHardcodedAPIKey(path string, content []byte) []Finding {
	var findings []Finding
	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if envVarRefRe.MatchString(line) {
			continue
		}
		if m := apiKeyNameRe.FindStringSubmatch(line); m != nil {
			val := m[2]
			if looksLikePlaceholder(val) {
				continue
			}
			findings = append(findings, Finding{
				Rule:      "SEC-101",
				Severity:  Critical,
				File:      path,
				Line:      lineNum,
				Message:   fmt.Sprintf("possible hardcoded %s", m[1]),
				Evidence:  strings.TrimSpace(line),
				FixPrompt: fmt.Sprintf("Move the hardcoded %s at %s:%d into an environment variable.", m[1], path, lineNum),
			})
		}
	}
	return findings
}

// OpenAIKey detects hardcoded OpenAI API keys.
var OpenAIKey = Rule{
	ID:        "SEC-102",
	Name:      "OpenAI API Key",
	Severity:  Critical,
	Languages: []string{"*"},
	Check:     checkPatternCritical("SEC-102", "OpenAI API key", regexp.MustCompile(`sk-proj-[a-zA-Z0-9]{20,}|sk-[a-zA-Z0-9]{48,}`)),
}

// AnthropicKey detects hardcoded Anthropic API keys.
var AnthropicKey = Rule{
	ID:        "SEC-103",
	Name:      "Anthropic API Key",
	Severity:  Critical,
	Languages: []string{"*"},
	Check:     checkPatternCritical("SEC-103", "Anthropic API key", regexp.MustCompile(`sk-ant-api03-[a-zA-Z0-9_-]{50,}`)),
}

// StripeSecretKey detects live Stripe secret keys.
var StripeSecretKey = Rule{
	ID:        "SEC-104",
	Name:      "Stripe Secret Key",
	Severity:  Critical,
	Languages: []string{"*"},
	Check:     checkPatternCritical("SEC-104", "Stripe live secret key", regexp.MustCompile(`(sk_live_|rk_live_)[a-zA-Z0-9]{24,}`)),
}

// SupabaseServiceRole detects supabase service_role key usage (bypasses RLS).
var SupabaseServiceRole = Rule{
	ID:        "SEC-105",
	Name:      "Supabase Service Role Key",
	Severity:  Critical,
	Languages: []string{"*"},
	Check:     checkPatternCritical("SEC-105", "Supabase service_role key — bypasses all RLS policies", regexp.MustCompile(`SERVICE_ROLE[_-]?KEY\s*[:=]\s*["'][^"']{20,}["']`)),
}

// JWTSecret detects placeholder JWT secrets copied from boilerplate.
// AI copies "secret" from examples and forgets to replace it.
var JWTSecret = Rule{
	ID:        "SEC-106",
	Name:      "JWT Placeholder Secret",
	Severity:  Critical,
	Languages: []string{"*"},
	Check:     checkJWTSecret,
}

var jwtSignRe        = regexp.MustCompile(`(?i)(jwt\.sign|verify\(token)`)
var jwtPlaceholderRe = regexp.MustCompile(`["'](secret|your-secret|jwt-secret|mysecret|change-me|changeme)["']`)

func checkJWTSecret(path string, content []byte) []Finding {
	var findings []Finding
	lines := bytes.Split(content, []byte("\n"))
	for i, line := range lines {
		if jwtSignRe.Match(line) || jwtPlaceholderRe.Match(line) {
			start := maxIdx(0, i-3)
			end := minIdx(len(lines)-1, i+3)
			window := bytes.Join(lines[start:end+1], []byte("\n"))
			if jwtSignRe.Match(window) && jwtPlaceholderRe.Match(window) {
				findings = append(findings, Finding{
					Rule:      "SEC-106",
					Severity:  Critical,
					File:      path,
					Line:      i + 1,
					Message:   "JWT secret is a placeholder value",
					Evidence:  strings.TrimSpace(string(line)),
					FixPrompt: fmt.Sprintf("Replace the placeholder JWT secret at %s:%d with os.Getenv(\"JWT_SECRET\").", path, i+1),
				})
				break
			}
		}
	}
	return findings
}

// SendGridKey detects hardcoded SendGrid API keys.
var SendGridKey = Rule{
	ID:        "SEC-107",
	Name:      "SendGrid API Key",
	Severity:  Critical,
	Languages: []string{"*"},
	Check:     checkPatternCritical("SEC-107", "SendGrid API key", regexp.MustCompile(`SG\.[a-zA-Z0-9_-]{22,}\.[a-zA-Z0-9_-]{43}`)),
}

// DatabaseURL detects database connection strings with embedded credentials.
var DatabaseURL = Rule{
	ID:        "SEC-108",
	Name:      "Database URL with Credentials",
	Severity:  Critical,
	Languages: []string{"*"},
	Check:     checkPatternCritical("SEC-108", "database URL with embedded credentials", regexp.MustCompile(`(postgres|postgresql|mysql|mongodb)://[^:]+:[^@]+@[^\s"']+`)),
}

// GitHubPAT detects hardcoded GitHub personal access tokens.
var GitHubPAT = Rule{
	ID:        "SEC-110",
	Name:      "GitHub Personal Access Token",
	Severity:  Critical,
	Languages: []string{"*"},
	Check:     checkPatternCritical("SEC-110", "GitHub personal access token", regexp.MustCompile(`(ghp_[a-zA-Z0-9]{36}|github_pat_[a-zA-Z0-9_]{82})`)),
}

// checkPatternCritical returns a Check function for a critical-severity regex pattern.
func checkPatternCritical(id, message string, re *regexp.Regexp) func(string, []byte) []Finding {
	return func(path string, content []byte) []Finding {
		var findings []Finding
		sc := bufio.NewScanner(bytes.NewReader(content))
		lineNum := 0
		for sc.Scan() {
			lineNum++
			line := sc.Text()
			if re.MatchString(line) {
				findings = append(findings, Finding{
					Rule:      id,
					Severity:  Critical,
					File:      path,
					Line:      lineNum,
					Message:   message,
					Evidence:  strings.TrimSpace(line),
					FixPrompt: fmt.Sprintf("Remove the hardcoded %s at %s:%d and load it from an environment variable.", message, path, lineNum),
				})
			}
		}
		return findings
	}
}

func looksLikePlaceholder(s string) bool {
	lower := strings.ToLower(s)
	for _, p := range []string{"example", "placeholder", "your-", "change-me", "xxx", "todo", "fixme", "test", "dummy"} {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func maxIdx(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minIdx(a, b int) int {
	if a < b {
		return a
	}
	return b
}
