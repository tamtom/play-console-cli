package releasenotes

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// GitCommit represents a single git commit.
type GitCommit struct {
	Hash    string `json:"hash"`
	Subject string `json:"subject"`
}

// GitLog runs `git log --no-merges --format="%h%x00%s"` between refs and returns commits.
// sinceRef is the starting ref (exclusive), untilRef is the ending ref (inclusive).
func GitLog(ctx context.Context, sinceRef, untilRef string) ([]GitCommit, error) {
	refRange := untilRef
	if sinceRef != "" {
		refRange = sinceRef + ".." + untilRef
	}

	cmd := exec.CommandContext(ctx, "git", "log", "--no-merges", "--format=%h%x00%s", refRange) // #nosec G204 -- args are ref names, not user shell input
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("git log failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	text := strings.TrimSpace(string(out))
	if text == "" {
		return nil, nil
	}

	var commits []GitCommit
	for _, line := range strings.Split(text, "\n") {
		parts := strings.SplitN(line, "\x00", 2)
		if len(parts) != 2 {
			continue
		}
		commits = append(commits, GitCommit{
			Hash:    parts[0],
			Subject: parts[1],
		})
	}
	return commits, nil
}
