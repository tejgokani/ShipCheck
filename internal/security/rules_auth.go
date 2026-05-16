package security

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

// WildcardCORS detects Access-Control-Allow-Origin set to *.
// AI applies this to fix CORS errors — it disables the same-origin protection entirely.
var WildcardCORS = Rule{
	ID:        "SEC-301",
	Name:      "Wildcard CORS",
	Severity:  Medium,
	Languages: []string{"go", "typescript", "javascript", "python"},
	Check:     checkWildcardCORS,
}

var corsWildcardRe = regexp.MustCompile(`(?i)Access-Control-Allow-Origin['":\s,]+[*]`)

func checkWildcardCORS(path string, content []byte) []Finding {
	var findings []Finding
	sc := bufio.NewScanner(bytes.NewReader(content))
	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := sc.Text()
		if corsWildcardRe.MatchString(line) {
			findings = append(findings, Finding{
				Rule:      "SEC-301",
				Severity:  Medium,
				File:      path,
				Line:      lineNum,
				Message:   "CORS set to wildcard (*) — allows any origin",
				Evidence:  strings.TrimSpace(line),
				FixPrompt: fmt.Sprintf("Replace the wildcard CORS at %s:%d with a specific allowed origin from an env var.", path, lineNum),
			})
		}
	}
	return findings
}

// SSLVerifyDisabled detects verify=False in Python HTTP requests.
// AI adds this to fix SSL certificate errors — disabling cert verification entirely.
var SSLVerifyDisabled = Rule{
	ID:        "SEC-302",
	Name:      "SSL Verification Disabled",
	Severity:  High,
	Languages: []string{"python"},
	Check:     checkPatternWithSeverity("SEC-302", "SSL verification disabled (verify=False)", High, regexp.MustCompile(`\bverify\s*=\s*False\b`)),
}

// DebugMode detects DEBUG=True or debug: true left from development scaffolding.
var DebugMode = Rule{
	ID:        "SEC-305",
	Name:      "Debug Mode Enabled",
	Severity:  Medium,
	Languages: []string{"python", "typescript", "javascript", "env", "yaml"},
	Check:     checkPatternWithSeverity("SEC-305", "debug mode enabled in non-test file", Medium, regexp.MustCompile(`(?i)\bDEBUG\s*[=:]\s*(True|true)\b|app\.run\s*\(.*debug\s*=\s*(True|true)`)),
}

// checkPatternWithSeverity returns a Check function for a regex at a given severity.
func checkPatternWithSeverity(id, message string, sev Severity, re *regexp.Regexp) func(string, []byte) []Finding {
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
					Severity:  sev,
					File:      path,
					Line:      lineNum,
					Message:   message,
					Evidence:  strings.TrimSpace(line),
					FixPrompt: fmt.Sprintf("Fix %s at %s:%d.", message, path, lineNum),
				})
			}
		}
		return findings
	}
}
