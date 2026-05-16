package cost

import (
	"sort"

	"github.com/tejgokani/shipcheck/internal/session"
)

// FileBurn aggregates token spend and touch counts for a single file across all sessions.
type FileBurn struct {
	Path        string
	TokensSpent int64
	TouchCount  int
}

// BuildBurnReport aggregates per-file token spend across all sessions and returns
// the results sorted by TokensSpent descending (hottest files first).
func BuildBurnReport(sessions []session.Session) []FileBurn {
	totals := map[string]*FileBurn{}

	for _, sess := range sessions {
		for _, sf := range sess.Files {
			if sf.Path == "" {
				continue
			}
			b, ok := totals[sf.Path]
			if !ok {
				b = &FileBurn{Path: sf.Path}
				totals[sf.Path] = b
			}
			b.TokensSpent += sf.TokensSpent
			b.TouchCount += sf.TouchCount
		}
	}

	result := make([]FileBurn, 0, len(totals))
	for _, b := range totals {
		result = append(result, *b)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].TokensSpent != result[j].TokensSpent {
			return result[i].TokensSpent > result[j].TokensSpent
		}
		return result[i].Path < result[j].Path
	})

	return result
}
