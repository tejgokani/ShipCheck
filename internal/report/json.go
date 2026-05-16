package report

import (
	"encoding/json"
	"fmt"
	"io"
)

// RenderJSON writes the audit result as indented JSON to w.
func RenderJSON(w io.Writer, result AuditResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		return fmt.Errorf("report: json encode: %w", err)
	}
	return nil
}
