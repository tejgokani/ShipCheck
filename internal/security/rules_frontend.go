package security

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

// NextPublicLeak detects NEXT_PUBLIC_ prefix on backend secrets.
// AI prefixes backend env vars with NEXT_PUBLIC_ to "fix" browser env var errors,
// which exposes them to every browser that loads the page.
var NextPublicLeak = Rule{
	ID:        "SEC-401",
	Name:      "NEXT_PUBLIC_ Secret Leak",
	Severity:  Critical,
	Languages: []string{"env", "typescript", "javascript"},
	Check:     checkNextPublicLeak,
}

var safePubRe       = regexp.MustCompile(`(?i)(PUBLISHABLE|PUBLIC_KEY|ANON_KEY|VAPID)`)
var sensitiveNameRe = regexp.MustCompile(`(?i)NEXT_PUBLIC_(DATABASE|SECRET|SERVICE_ROLE|STRIPE_SECRET|API_KEY|TOKEN|PASSWORD|PRIVATE)`)

func checkNextPublicLeak(path string, content []byte) []Finding {
	var findings []Finding
	sc := bufio.NewScanner(bytes.NewReader(content))
	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := sc.Text()
		if sensitiveNameRe.MatchString(line) && !safePubRe.MatchString(line) {
			findings = append(findings, Finding{
				Rule:      "SEC-401",
				Severity:  Critical,
				File:      path,
				Line:      lineNum,
				Message:   "NEXT_PUBLIC_ prefix exposes backend secret to browser",
				Evidence:  strings.TrimSpace(line),
				FixPrompt: fmt.Sprintf("Remove NEXT_PUBLIC_ prefix from the env var at %s:%d and access it server-side only.", path, lineNum),
			})
		}
	}
	return findings
}

// DeferredSecurityComment detects TODO/FIXME comments that defer security work.
// AI commonly generates these placeholders that never get addressed.
var DeferredSecurityComment = Rule{
	ID:        "SEC-501",
	Name:      "Deferred Security TODO",
	Severity:  Low,
	Languages: []string{"*"},
	Check:     checkDeferredSecurity,
}

var deferredRe = regexp.MustCompile(`(?i)(TODO|FIXME)[:\s].*(auth|authenticat|validat|sanitiz|secur|input|env\s*var|hardcod)`)

func checkDeferredSecurity(path string, content []byte) []Finding {
	var findings []Finding
	sc := bufio.NewScanner(bytes.NewReader(content))
	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := sc.Text()
		if deferredRe.MatchString(line) {
			findings = append(findings, Finding{
				Rule:      "SEC-501",
				Severity:  Low,
				File:      path,
				Line:      lineNum,
				Message:   "deferred security work — TODO comment marks a known vulnerability",
				Evidence:  strings.TrimSpace(line),
				FixPrompt: fmt.Sprintf("Address the security TODO at %s:%d before shipping.", path, lineNum),
			})
		}
	}
	return findings
}
