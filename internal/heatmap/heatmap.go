package heatmap

import (
	"log"
	"sort"
	"time"

	"github.com/tejgokani/shipcheck/internal/session"
)

const maxResults = 10

// Build analyses git history correlated with session timestamps to identify
// which files were touched most frequently during agentic sessions.
//
// dir is the project directory (must be a git repo).
// depth controls how many git commits to inspect.
//
// If dir is not a git repository, Build returns an empty slice and a nil error
// (the caller should continue without the heatmap).
func Build(sessions []session.Session, dir string, depth int) ([]HotFile, error) {
	// Determine the time window from all sessions.
	var start, end time.Time
	for _, s := range sessions {
		if !s.StartTime.IsZero() {
			if start.IsZero() || s.StartTime.Before(start) {
				start = s.StartTime
			}
		}
		if !s.EndTime.IsZero() {
			if end.IsZero() || s.EndTime.After(end) {
				end = s.EndTime
			}
		}
	}

	commits, err := gitCommitsInWindow(dir, depth, start, end)
	if err != nil {
		return nil, err
	}
	if len(commits) == 0 {
		log.Printf("info: heatmap: no commits found in session window (start=%v end=%v)", start, end)
		return nil, nil
	}

	// Count file appearances across commits and track last-touched time.
	type fileStats struct {
		count       int
		lastTouched time.Time
	}
	touchCounts := map[string]*fileStats{}

	for _, c := range commits {
		files, err := filesChangedByCommit(dir, c.Hash)
		if err != nil {
			log.Printf("warn: heatmap: skipping commit %s: %v", c.Hash, err)
			continue
		}
		for _, f := range files {
			fs, ok := touchCounts[f]
			if !ok {
				fs = &fileStats{}
				touchCounts[f] = fs
			}
			fs.count++
			if c.Time.After(fs.lastTouched) {
				fs.lastTouched = c.Time
			}
		}
	}

	if len(touchCounts) == 0 {
		return nil, nil
	}

	// Find max touch count for normalization.
	maxCount := 0
	for _, fs := range touchCounts {
		if fs.count > maxCount {
			maxCount = fs.count
		}
	}

	hotFiles := make([]HotFile, 0, len(touchCounts))
	for path, fs := range touchCounts {
		score := 0.0
		if maxCount > 0 {
			score = float64(fs.count) / float64(maxCount)
		}
		hotFiles = append(hotFiles, HotFile{
			Path:        path,
			TouchCount:  fs.count,
			HeatScore:   score,
			LastTouched: fs.lastTouched,
		})
	}

	// Sort descending by TouchCount, then alphabetically for determinism.
	sort.Slice(hotFiles, func(i, j int) bool {
		if hotFiles[i].TouchCount != hotFiles[j].TouchCount {
			return hotFiles[i].TouchCount > hotFiles[j].TouchCount
		}
		return hotFiles[i].Path < hotFiles[j].Path
	})

	if len(hotFiles) > maxResults {
		hotFiles = hotFiles[:maxResults]
	}

	return hotFiles, nil
}
