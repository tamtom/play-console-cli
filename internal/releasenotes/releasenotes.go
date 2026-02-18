package releasenotes

import (
	"fmt"
	"strings"
)

// Format converts commits into a bullet-point release notes string.
// Truncates to maxChars (Google Play limit is 500 chars).
// Truncation removes whole lines to avoid cutting mid-sentence,
// and appends "..." if any lines were dropped.
func Format(commits []GitCommit, maxChars int) string {
	if len(commits) == 0 {
		return ""
	}

	var lines []string
	for _, c := range commits {
		lines = append(lines, fmt.Sprintf("- %s", c.Subject))
	}

	result := strings.Join(lines, "\n")
	if maxChars <= 0 || len(result) <= maxChars {
		return result
	}

	// Truncate at line boundary
	const ellipsis = "\n..."
	budget := maxChars - len(ellipsis)
	if budget <= 0 {
		return "..."
	}

	var kept []string
	total := 0
	for i, line := range lines {
		lineLen := len(line)
		if i > 0 {
			lineLen++ // account for newline separator
		}
		if total+lineLen > budget {
			break
		}
		kept = append(kept, line)
		total += lineLen
	}

	if len(kept) == 0 {
		// Even a single line is too long; truncate it
		return lines[0][:budget] + ellipsis
	}

	return strings.Join(kept, "\n") + ellipsis
}
