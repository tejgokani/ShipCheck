// Package heatmap identifies which files an agent touched most during a session.
package heatmap

import "time"

// HotFile represents a file that was frequently modified during an agentic session.
type HotFile struct {
	Path        string
	TouchCount  int
	HeatScore   float64   // normalized 0–1; 1 = most-touched file
	LastTouched time.Time
}
