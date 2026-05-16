package heatmap

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

// commitInfo holds a git commit hash and its author timestamp.
type commitInfo struct {
	Hash string
	Time time.Time
}

// gitCommitsInWindow returns up to depth commits whose timestamps fall within [start, end].
// If no git repo is found, it returns nil, nil (caller should skip heatmap gracefully).
func gitCommitsInWindow(dir string, depth int, start, end time.Time) ([]commitInfo, error) {
	// git log --pretty=format:"%H %aI" -n <depth>
	args := []string{
		"-C", dir,
		"log",
		fmt.Sprintf("--pretty=format:%%H %%aI"),
		fmt.Sprintf("-n%d", depth),
	}
	out, err := runGit(args...)
	if err != nil {
		// Not a git repo or git not available — callers treat nil result as "skip".
		log.Printf("warn: heatmap/git: git log failed (not a git repo?): %v", err)
		return nil, nil
	}

	var commits []commitInfo
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		hash := parts[0]
		ts, err := time.Parse(time.RFC3339, parts[1])
		if err != nil {
			// Try ISO 8601 with offset (git aI format).
			ts, err = time.Parse("2006-01-02T15:04:05-07:00", parts[1])
			if err != nil {
				log.Printf("warn: heatmap/git: unparseable timestamp for commit %s: %q", hash, parts[1])
				continue
			}
		}

		// Filter by time window only when both bounds are non-zero.
		if !start.IsZero() && ts.Before(start) {
			continue
		}
		if !end.IsZero() && ts.After(end) {
			continue
		}

		commits = append(commits, commitInfo{Hash: hash, Time: ts})
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("heatmap/git: scan log output: %w", err)
	}
	return commits, nil
}

// filesChangedByCommit returns the list of files changed in a given commit.
func filesChangedByCommit(dir, hash string) ([]string, error) {
	args := []string{
		"-C", dir,
		"diff-tree",
		"--no-commit-id",
		"-r",
		"--name-only",
		hash,
	}
	out, err := runGit(args...)
	if err != nil {
		return nil, fmt.Errorf("heatmap/git: diff-tree %s: %w", hash, err)
	}

	var files []string
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" {
			files = append(files, line)
		}
	}
	return files, sc.Err()
}

// runGit executes a git command and returns its combined stdout output.
func runGit(args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %v: %w", args, err)
	}
	return out, nil
}
