package report

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"strings"
)

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>shipcheck report — {{.Directory}}</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;background:#0f0f0e;color:#d3d1c7;min-height:100vh;padding:2rem}
.container{max-width:900px;margin:0 auto}
h1{font-size:1rem;color:#6c6a62;margin-bottom:2rem}
.score-wrap{text-align:center;margin:2rem 0 3rem}
.score{font-size:5rem;font-weight:900;line-height:1}
.label{font-size:1.2rem;margin-top:.5rem;letter-spacing:.2em}
.card{background:#1a1a18;border-radius:8px;padding:1.5rem;margin-bottom:1.5rem}
.card h2{font-size:.75rem;letter-spacing:.15em;color:#6c6a62;margin-bottom:1rem;text-transform:uppercase}
.stat{display:flex;justify-content:space-between;padding:.4rem 0;border-bottom:1px solid #2a2a28}
.stat:last-child{border:none}
.bar-row{display:flex;align-items:center;gap:1rem;padding:.4rem 0}
.bar-wrap{flex:1;background:#2a2a28;border-radius:3px;height:8px;overflow:hidden}
.bar-fill{height:100%;background:#ef9f27;border-radius:3px;transition:width .3s}
.finding{display:grid;grid-template-columns:80px 1fr;gap:.5rem;padding:.6rem 0;border-bottom:1px solid #2a2a28;font-size:.875rem}
.finding:last-child{border:none}
.sev{font-weight:700;font-size:.75rem}
.sev.critical{color:#e24b4a}.sev.high{color:#ef9f27}.sev.medium{color:#378add}.sev.low{color:#888780}
.loc{color:#6c6a62;font-size:.75rem}
.fix-box{background:#0f0f0e;border:1px solid #2a2a28;border-radius:6px;padding:1rem;position:relative;margin-top:1rem}
.fix-box pre{font-size:.8rem;white-space:pre-wrap;word-break:break-word;color:#d3d1c7}
.copy-btn{position:absolute;top:.5rem;right:.5rem;background:#1d9e75;color:#fff;border:none;border-radius:4px;padding:.3rem .8rem;cursor:pointer;font-size:.75rem}
.copy-btn:hover{background:#17876a}
.clean{color:#1d9e75}.good{color:#1d9e75}.review{color:#ef9f27}.risky{color:#e24b4a}.danger{color:#e24b4a}
</style>
</head>
<body>
<div class="container">
<h1>shipcheck report &mdash; {{.Directory}} &mdash; {{.GeneratedAt.Format "2006-01-02 15:04"}}</h1>
<div class="score-wrap">
  <div class="score {{.Label | lower}}">{{.Score}}<span style="font-size:2rem;color:#6c6a62">/100</span></div>
  <div class="label">{{.Label}}</div>
</div>

<div class="card">
  <h2>Cost &amp; Session</h2>
  {{range .Sessions}}
  <div class="stat"><span>Session cost</span><span>${{printf "%.4f" .TotalCost}}</span></div>
  <div class="stat"><span>Model</span><span>{{.ModelID}}</span></div>
  <div class="stat"><span>Tokens in</span><span>{{.Usage.InputTokens}}</span></div>
  <div class="stat"><span>Tokens out</span><span>{{.Usage.OutputTokens}}</span></div>
  {{end}}
  {{if not .Sessions}}<p style="color:#6c6a62">No session logs found.</p>{{end}}
</div>

<div class="card">
  <h2>Heatmap</h2>
  {{range .HotFiles}}
  <div class="bar-row">
    <span style="width:200px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-size:.8rem">{{.Path}}</span>
    <div class="bar-wrap"><div class="bar-fill" style="width:{{printf "%.0f" (mul .HeatScore 100)}}%"></div></div>
    <span style="width:60px;text-align:right;font-size:.8rem">{{.TouchCount}} touches</span>
  </div>
  {{end}}
  {{if not .HotFiles}}<p style="color:#6c6a62">No git history found.</p>{{end}}
</div>

<div class="card">
  <h2>Security ({{len .Findings}} findings)</h2>
  {{range .Findings}}
  <div class="finding">
    <span class="sev {{.Severity.String | lower}}">{{.Severity}}</span>
    <div>
      <div>{{.Message}}</div>
      <div class="loc">{{.File}}:{{.Line}}</div>
    </div>
  </div>
  {{end}}
  {{if not .Findings}}<p style="color:#1d9e75">No findings — clean!</p>{{end}}
</div>

{{if .Findings}}
<div class="card">
  <h2>Fix Prompt</h2>
  <p style="font-size:.8rem;color:#6c6a62;margin-bottom:.5rem">Paste into Claude Code to fix all findings:</p>
  <div class="fix-box">
    <pre id="fix-prompt">{{.FixPrompt}}</pre>
    <button class="copy-btn" onclick="copyFix()">Copy</button>
  </div>
</div>
{{end}}

</div>
<script>
function copyFix(){
  var t=document.getElementById('fix-prompt').textContent;
  navigator.clipboard.writeText(t).then(function(){
    var b=document.querySelector('.copy-btn');
    b.textContent='Copied!';
    setTimeout(function(){b.textContent='Copy'},2000);
  });
}
</script>
</body>
</html>`

// RenderHTML generates a self-contained HTML report and writes it to path.
func RenderHTML(result AuditResult, path string) error {
	// Re-compute only if caller didn't pre-populate score.
	if result.Score == 0 && result.Label == "" {
		sr := ComputeScore(result.Findings)
		result.Score = sr.Score
		result.Label = sr.Label
	}

	type templateData struct {
		AuditResult
		FixPrompt string
	}

	data := templateData{
		AuditResult: result,
		FixPrompt:   buildFixPrompt(result.Findings),
	}

	funcMap := template.FuncMap{
		"lower": strings.ToLower,
		"mul":   func(a, b float64) float64 { return a * b },
	}

	tmpl, err := template.New("report").Funcs(funcMap).Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("report: html template parse: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("report: html template execute: %w", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("report: html write: %w", err)
	}
	return nil
}
