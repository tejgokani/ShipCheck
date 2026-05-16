package security

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

// SQLStringConcat detects SQL queries built by string concatenation or f-strings.
// AI generates the simplest working SQL first — parameterization is an afterthought.
var SQLStringConcat = Rule{
	ID:        "SEC-201",
	Name:      "SQL String Concatenation",
	Severity:  Critical,
	Languages: []string{"go", "typescript", "javascript", "python"},
	Check:     checkSQLConcat,
}

var (
	sqlConcatPy = regexp.MustCompile(`(?i)(execute|query)\s*\(\s*f["'].*\{|["']\s*\+\s*\w+\s*\+\s*["'].*(?:SELECT|INSERT|UPDATE|DELETE|WHERE)`)
	sqlConcatGo = regexp.MustCompile(`(?i)(fmt\.Sprintf|"\s*\+\s*).*(?:SELECT|INSERT|UPDATE|DELETE|WHERE)`)
	sqlConcatJS = regexp.MustCompile("(?i)(`|['\"])\\s*(?:SELECT|INSERT|UPDATE|DELETE|WHERE).*\\$\\{")
)

func checkSQLConcat(path string, content []byte) []Finding {
	var findings []Finding
	ext := fileExt(path)
	sc := bufio.NewScanner(bytes.NewReader(content))
	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := sc.Text()
		var matched bool
		switch ext {
		case ".py":
			matched = sqlConcatPy.MatchString(line)
		case ".go":
			matched = sqlConcatGo.MatchString(line)
		case ".ts", ".tsx", ".js", ".jsx":
			matched = sqlConcatJS.MatchString(line)
		}
		if matched {
			findings = append(findings, Finding{
				Rule:      "SEC-201",
				Severity:  Critical,
				File:      path,
				Line:      lineNum,
				Message:   "SQL query built with string concatenation — SQL injection risk",
				Evidence:  strings.TrimSpace(line),
				FixPrompt: fmt.Sprintf("Replace the string-concatenated SQL at %s:%d with parameterized queries.", path, lineNum),
			})
		}
	}
	return findings
}

// EvalOnInput detects eval() or exec() called with potentially user-controlled input.
var EvalOnInput = Rule{
	ID:        "SEC-202",
	Name:      "eval() on User Input",
	Severity:  Critical,
	Languages: []string{"typescript", "javascript", "python"},
	Check:     checkPatternWithSeverity("SEC-202", "eval() or exec() with user-controlled input", Critical, regexp.MustCompile(`\beval\s*\(|(?:^|\s)exec\s*\(`)),
}

// CommandInjection detects shell=True or string interpolation into shell commands.
// AI uses shell=True or os.system() with interpolation when building CLI wrappers.
var CommandInjection = Rule{
	ID:        "SEC-203",
	Name:      "Command Injection via Shell",
	Severity:  Critical,
	Languages: []string{"go", "typescript", "javascript", "python"},
	Check:     checkCommandInjection,
}

var (
	shellTrueRe = regexp.MustCompile(`shell\s*=\s*True`)
	osSystemRe  = regexp.MustCompile(`os\.system\s*\(`)
)

func checkCommandInjection(path string, content []byte) []Finding {
	var findings []Finding
	sc := bufio.NewScanner(bytes.NewReader(content))
	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := sc.Text()
		if shellTrueRe.MatchString(line) || osSystemRe.MatchString(line) {
			findings = append(findings, Finding{
				Rule:      "SEC-203",
				Severity:  Critical,
				File:      path,
				Line:      lineNum,
				Message:   "shell=True or os.system() enables command injection",
				Evidence:  strings.TrimSpace(line),
				FixPrompt: fmt.Sprintf("Replace the shell command at %s:%d with a list-form subprocess call (no shell=True).", path, lineNum),
			})
		}
	}
	return findings
}

func fileExt(path string) string {
	dot := strings.LastIndex(path, ".")
	if dot == -1 {
		return ""
	}
	return path[dot:]
}
