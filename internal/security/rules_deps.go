package security

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// ── SEC-504: Hallucinated package import ─────────────────────────────────────
// AI invents plausible-sounding package names that don't exist on npm/PyPI.
// These fail at install time but can also become supply-chain vectors if someone
// registers the name after the AI "creates" it.

var HallucinatedImport = Rule{
	ID:        "SEC-504",
	Name:      "Hallucinated Package Import",
	Severity:  Medium,
	Languages: []string{"typescript", "javascript", "python"},
	Check:     checkHallucinatedImport,
}

// knownHallucinations is a curated list of packages AI commonly invents.
// Keyed by the fake name, value is the real alternative (empty = no known alt).
var knownHallucinations = map[string]string{
	// OpenAI ecosystem — AI invents sub-packages that don't exist
	"@openai/utils":        "openai",
	"@openai/tools":        "openai",
	"@openai/client":       "openai",
	"@openai/functions":    "openai",
	"openai-node":          "openai",
	"openai-api":           "openai",
	// Anthropic ecosystem
	"@anthropic/sdk":       "@anthropic-ai/sdk",
	"@anthropic/client":    "@anthropic-ai/sdk",
	"anthropic-sdk":        "@anthropic-ai/sdk",
	"anthropic-ai":         "@anthropic-ai/sdk",
	"anthropic":            "@anthropic-ai/sdk",
	// LangChain — AI often uses wrong package names
	"langchain-tools":      "langchain",
	"langchain-agents":     "langchain",
	"langchain-openai":     "@langchain/openai",
	// React ecosystem hallucinations
	"react-auth-hooks":     "",
	"react-fetch-hooks":    "",
	"react-api-hooks":      "",
	"use-auth":             "",
	// Common Python hallucinations
	"openai-utils":         "openai",
	"anthropic-python":     "anthropic",
	"langchain-utils":      "langchain",
	"fastapi-auth":         "",
	"pydantic-settings-v2": "pydantic-settings",
}

var (
	// Matches: import X from "pkg", import { X } from 'pkg', require("pkg"), import "pkg"
	jsImportRe = regexp.MustCompile(`(?:from\s+['"]|require\s*\(\s*['"]|import\s+['"])([^'"]+)['"]`)
	pyImportRe = regexp.MustCompile(`^(?:import|from)\s+([\w\-\.]+)`)
)

func checkHallucinatedImport(path string, content []byte) []Finding {
	var findings []Finding
	ext := fileExt(path)
	sc := bufio.NewScanner(bytes.NewReader(content))
	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := sc.Text()

		var pkgName string
		switch ext {
		case ".ts", ".tsx", ".js", ".jsx":
			if m := jsImportRe.FindStringSubmatch(line); m != nil {
				pkgName = strings.TrimSpace(m[1])
			}
		case ".py":
			if m := pyImportRe.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
				pkgName = strings.TrimSpace(m[1])
			}
		}

		if pkgName == "" {
			continue
		}

		alt, known := knownHallucinations[pkgName]
		if !known {
			continue
		}

		msg := fmt.Sprintf("import of %q — known AI-hallucinated package name", pkgName)
		fix := fmt.Sprintf("Remove import of %q at %s:%d.", pkgName, path, lineNum)
		if alt != "" {
			msg = fmt.Sprintf("import of %q — AI-hallucinated; use %q instead", pkgName, alt)
			fix = fmt.Sprintf("Replace %q with %q at %s:%d.", pkgName, alt, path, lineNum)
		}

		findings = append(findings, Finding{
			Rule:      "SEC-504",
			Severity:  Medium,
			File:      path,
			Line:      lineNum,
			Message:   msg,
			Evidence:  strings.TrimSpace(line),
			FixPrompt: fix,
		})
	}
	return findings
}

// ── SEC-601: Package with known CVE ──────────────────────────────────────────
// AI pins specific package versions copied from examples or old docs.
// These may be vulnerable versions with publicly known CVEs.

var KnownVulnerablePackage = Rule{
	ID:        "SEC-601",
	Name:      "Package with Known CVE",
	Severity:  High,
	Languages: []string{"json", "yaml", "python"}, // python covers requirements.txt via specialFiles
	Check:     checkKnownVulnerablePackage,
}

// vulnPackage describes a known-vulnerable version range.
type vulnPackage struct {
	name    string
	fixedAt string // first safe version (semver)
	cve     string
	summary string
}

// knownVulnPackages is a curated list of high-impact CVEs commonly seen in
// AI-generated dependency files. Updated on each shipcheck release.
var knownVulnPackages = []vulnPackage{
	// npm
	{name: "lodash", fixedAt: "4.17.21", cve: "CVE-2021-23337", summary: "command injection via template"},
	{name: "axios", fixedAt: "0.21.1", cve: "CVE-2020-28168", summary: "SSRF via crafted URL"},
	{name: "jsonwebtoken", fixedAt: "9.0.0", cve: "CVE-2022-23529", summary: "remote code execution"},
	{name: "node-fetch", fixedAt: "2.6.7", cve: "CVE-2022-0235", summary: "exposure of sensitive info"},
	{name: "minimist", fixedAt: "1.2.6", cve: "CVE-2021-44906", summary: "prototype pollution"},
	{name: "express", fixedAt: "4.19.2", cve: "CVE-2024-29041", summary: "open redirect"},
	{name: "multer", fixedAt: "1.4.5-lts.1", cve: "CVE-2022-24434", summary: "denial of service"},
	{name: "semver", fixedAt: "7.5.2", cve: "CVE-2022-25883", summary: "ReDoS via crafted version"},
	{name: "tough-cookie", fixedAt: "4.1.3", cve: "CVE-2023-26136", summary: "prototype pollution"},
	// PyPI
	{name: "Pillow", fixedAt: "9.0.0", cve: "CVE-2022-22816", summary: "buffer overflow in path_getbbox"},
	{name: "Django", fixedAt: "3.2.19", cve: "CVE-2023-31047", summary: "file upload bypass"},
	{name: "cryptography", fixedAt: "41.0.0", cve: "CVE-2023-38325", summary: "NULL pointer dereference"},
	{name: "urllib3", fixedAt: "1.26.5", cve: "CVE-2021-33503", summary: "ReDoS via crafted URL"},
	{name: "requests", fixedAt: "2.28.0", cve: "CVE-2023-32681", summary: "credential leak on redirect"},
	{name: "paramiko", fixedAt: "3.4.0", cve: "CVE-2023-48795", summary: "prefix truncation attack"},
}

// pinRe matches pinned version strings like `"lodash": "4.17.20"` or `lodash==4.17.20`.
var pinRe = regexp.MustCompile(`["']?([\w\-\.]+)["']?\s*(?::|==|>=?|~=|~>)\s*["']?\^?~?(\d+\.\d+[\.\d]*)["']?`)

func checkKnownVulnerablePackage(path string, content []byte) []Finding {
	// Only check dependency manifests.
	base := fileBase(path)
	if base != "package.json" && base != "requirements.txt" &&
		base != "Pipfile" && base != "pyproject.toml" {
		return nil
	}

	var findings []Finding
	sc := bufio.NewScanner(bytes.NewReader(content))
	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := sc.Text()
		m := pinRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		pkgName := m[1]
		pinnedVer := m[2]

		for _, vp := range knownVulnPackages {
			if !strings.EqualFold(pkgName, vp.name) {
				continue
			}
			if semverLessThan(pinnedVer, vp.fixedAt) {
				findings = append(findings, Finding{
					Rule:     "SEC-601",
					Severity: High,
					File:     path,
					Line:     lineNum,
					Message: fmt.Sprintf(
						"%s@%s has %s (%s) — upgrade to >=%s",
						pkgName, pinnedVer, vp.cve, vp.summary, vp.fixedAt,
					),
					Evidence:  strings.TrimSpace(line),
					FixPrompt: fmt.Sprintf("Upgrade %s to >=%s to fix %s at %s:%d.", pkgName, vp.fixedAt, vp.cve, path, lineNum),
				})
			}
		}
	}
	return findings
}

// ── SEC-602: Overprivileged npm script ────────────────────────────────────────
// AI adds scripts that require elevated privileges or write to system paths.
// Common in Dockerfiles and CI scaffolding generated by AI assistants.

var OverprivilegedScript = Rule{
	ID:        "SEC-602",
	Name:      "Overprivileged npm Script",
	Severity:  Medium,
	Languages: []string{"json"},
	Check:     checkOverprivilegedScript,
}

var dangerousScriptRe = regexp.MustCompile(
	`(?i)(sudo\s|chmod\s+[0-7]*7[0-7][0-7]|curl.+\|\s*(?:ba)?sh|wget.+\|\s*(?:ba)?sh|rm\s+-rf\s+/|chown\s+root)`,
)

func checkOverprivilegedScript(path string, content []byte) []Finding {
	if fileBase(path) != "package.json" {
		return nil
	}

	// Parse as JSON to locate the scripts section.
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(content, &pkg); err != nil {
		// Fall back to line-by-line scan if JSON is malformed.
		return checkOverprivilegedScriptLines(path, content)
	}
	if len(pkg.Scripts) == 0 {
		return nil
	}

	// Re-scan line by line so we can report accurate line numbers.
	return checkOverprivilegedScriptLines(path, content)
}

func checkOverprivilegedScriptLines(path string, content []byte) []Finding {
	var findings []Finding
	sc := bufio.NewScanner(bytes.NewReader(content))
	lineNum := 0
	inScripts := false
	for sc.Scan() {
		lineNum++
		line := sc.Text()
		if strings.Contains(line, `"scripts"`) {
			inScripts = true
		}
		if inScripts && dangerousScriptRe.MatchString(line) {
			findings = append(findings, Finding{
				Rule:      "SEC-602",
				Severity:  Medium,
				File:      path,
				Line:      lineNum,
				Message:   "npm script uses elevated privileges or dangerous shell pattern",
				Evidence:  strings.TrimSpace(line),
				FixPrompt: fmt.Sprintf("Review the npm script at %s:%d — avoid sudo, chmod 777, and curl|sh patterns.", path, lineNum),
			})
		}
	}
	return findings
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// semverLessThan returns true if a < b using simple major.minor.patch comparison.
// Handles the most common version formats; not a full semver parser.
func semverLessThan(a, b string) bool {
	ap := parseSemver(a)
	bp := parseSemver(b)
	for i := 0; i < 3; i++ {
		if ap[i] < bp[i] {
			return true
		}
		if ap[i] > bp[i] {
			return false
		}
	}
	return false
}

func parseSemver(v string) [3]int {
	// Strip leading v or = signs.
	v = strings.TrimLeft(v, "v=^~")
	parts := strings.SplitN(v, ".", 3)
	var out [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		// Stop at first non-numeric character (e.g. "1.2.3-beta").
		n := 0
		for _, c := range p {
			if c < '0' || c > '9' {
				break
			}
			n = n*10 + int(c-'0')
		}
		out[i] = n
	}
	return out
}

// fileBase returns just the filename portion of a path.
func fileBase(path string) string {
	idx := strings.LastIndexAny(path, "/\\")
	if idx == -1 {
		return path
	}
	return path[idx+1:]
}
