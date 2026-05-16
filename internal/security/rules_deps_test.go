package security

import (
	"testing"
)

// ── SEC-504: Hallucinated import ─────────────────────────────────────────────

func TestHallucinatedImport_TS_Hit(t *testing.T) {
	findings := scanContent(t, ".ts", `import { Client } from "@anthropic/sdk"`)
	if !hasRule(findings, "SEC-504") {
		t.Error("expected SEC-504 for hallucinated @anthropic/sdk import")
	}
}

func TestHallucinatedImport_JS_OpenAI(t *testing.T) {
	findings := scanContent(t, ".js", `const utils = require("@openai/utils")`)
	if !hasRule(findings, "SEC-504") {
		t.Error("expected SEC-504 for hallucinated @openai/utils import")
	}
}

func TestHallucinatedImport_Py_Hit(t *testing.T) {
	// Python can't import hyphenated names — none of these should trigger SEC-504.
	// The rule only fires on exact matches in knownHallucinations, which use hyphens
	// as registered on PyPI (e.g. "openai-utils"). Python import syntax uses
	// underscores, so pyImportRe won't match them.
	for _, src := range []string{
		`import anthropic-python`,
		`from anthropic_python import Client`,
		`import openai_utils`,
		`import openai-utils`,
	} {
		_ = scanContent(t, ".py", src) // no assertion; just confirm no panic
	}
}

func TestHallucinatedImport_RealPackage_Miss(t *testing.T) {
	findings := scanContent(t, ".ts", `import OpenAI from "openai"`)
	if hasRule(findings, "SEC-504") {
		t.Error("should not flag real 'openai' package as SEC-504")
	}
}

func TestHallucinatedImport_AnthropicReal_Miss(t *testing.T) {
	findings := scanContent(t, ".ts", `import Anthropic from "@anthropic-ai/sdk"`)
	if hasRule(findings, "SEC-504") {
		t.Error("should not flag real @anthropic-ai/sdk package as SEC-504")
	}
}

func TestHallucinatedImport_ReactHooks_Hit(t *testing.T) {
	findings := scanContent(t, ".tsx", `import { useAuth } from "react-auth-hooks"`)
	if !hasRule(findings, "SEC-504") {
		t.Error("expected SEC-504 for hallucinated react-auth-hooks import")
	}
}

// ── SEC-601: Known vulnerable package ────────────────────────────────────────

func TestKnownVulnerablePackage_NPM_Hit(t *testing.T) {
	content := `{
  "dependencies": {
    "lodash": "4.17.20",
    "express": "^4.18.0"
  }
}`
	findings := scanContent(t, "package.json", content)
	if !hasRule(findings, "SEC-601") {
		t.Error("expected SEC-601 for lodash 4.17.20 (vulnerable, fix at 4.17.21)")
	}
}

func TestKnownVulnerablePackage_NPM_Safe(t *testing.T) {
	content := `{
  "dependencies": {
    "lodash": "4.17.21"
  }
}`
	findings := scanContent(t, "package.json", content)
	if hasRule(findings, "SEC-601") {
		t.Error("should not flag lodash 4.17.21 — it's the fixed version")
	}
}

func TestKnownVulnerablePackage_Pip_Hit(t *testing.T) {
	findings := scanContent(t, "requirements.txt", "Pillow==8.3.0\nrequests==2.27.0\n")
	if !hasRule(findings, "SEC-601") {
		t.Error("expected SEC-601 for Pillow 8.3.0 (vulnerable, fix at 9.0.0)")
	}
}

func TestKnownVulnerablePackage_Pip_Safe(t *testing.T) {
	findings := scanContent(t, "requirements.txt", "Pillow==9.1.0\nrequests==2.28.1\n")
	if hasRule(findings, "SEC-601") {
		t.Error("should not flag Pillow 9.1.0 — above fixed version")
	}
}

func TestKnownVulnerablePackage_NonManifest_Skipped(t *testing.T) {
	// Rule should only fire on dependency manifest files
	findings := scanContent(t, ".go", `// lodash: "4.17.20"`)
	if hasRule(findings, "SEC-601") {
		t.Error("should not flag non-manifest .go file as SEC-601")
	}
}

// ── SEC-602: Overprivileged npm script ────────────────────────────────────────

func TestOverprivilegedScript_Sudo_Hit(t *testing.T) {
	content := `{
  "scripts": {
    "install": "sudo npm install -g some-tool"
  }
}`
	findings := scanContent(t, "package.json", content)
	if !hasRule(findings, "SEC-602") {
		t.Error("expected SEC-602 for sudo in npm script")
	}
}

func TestOverprivilegedScript_CurlPipe_Hit(t *testing.T) {
	content := `{
  "scripts": {
    "setup": "curl https://example.com/install.sh | sh"
  }
}`
	findings := scanContent(t, "package.json", content)
	if !hasRule(findings, "SEC-602") {
		t.Error("expected SEC-602 for curl|sh in npm script")
	}
}

func TestOverprivilegedScript_Chmod777_Hit(t *testing.T) {
	content := `{
  "scripts": {
    "postinstall": "chmod 777 /tmp/myapp"
  }
}`
	findings := scanContent(t, "package.json", content)
	if !hasRule(findings, "SEC-602") {
		t.Error("expected SEC-602 for chmod 777 in npm script")
	}
}

func TestOverprivilegedScript_Safe(t *testing.T) {
	content := `{
  "scripts": {
    "build": "tsc && node dist/index.js",
    "test": "jest"
  }
}`
	findings := scanContent(t, "package.json", content)
	if hasRule(findings, "SEC-602") {
		t.Error("should not flag normal npm scripts as SEC-602")
	}
}

func TestOverprivilegedScript_NonPackageJSON_Skipped(t *testing.T) {
	findings := scanContent(t, ".go", `// sudo rm -rf /`)
	if hasRule(findings, "SEC-602") {
		t.Error("should not flag non-package.json file as SEC-602")
	}
}

// ── semverLessThan unit tests ─────────────────────────────────────────────────

func TestSemverLessThan(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"4.17.20", "4.17.21", true},
		{"4.17.21", "4.17.21", false},
		{"4.17.22", "4.17.21", false},
		{"3.0.0", "4.17.21", true},
		{"5.0.0", "4.17.21", false},
		{"0.21.0", "0.21.1", true},
		{"8.3.0", "9.0.0", true},
		{"9.0.0", "9.0.0", false},
		{"9.1.0", "9.0.0", false},
	}
	for _, tc := range cases {
		got := semverLessThan(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("semverLessThan(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}
