package shared

import (
	"fmt"
	"strings"
)

// LevenshteinDistance computes the edit distance between two strings.
func LevenshteinDistance(a, b string) int {
	a = strings.ToLower(a)
	b = strings.ToLower(b)
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

// SuggestCommand finds the closest command name within maxDistance.
func SuggestCommand(input string, commands []string, maxDistance int) string {
	best := ""
	bestDist := maxDistance + 1
	for _, cmd := range commands {
		d := LevenshteinDistance(input, cmd)
		if d < bestDist {
			bestDist = d
			best = cmd
		}
	}
	if bestDist <= maxDistance {
		return best
	}
	return ""
}

// FormatUnknownCommand returns an error message with an optional suggestion.
func FormatUnknownCommand(input string, commands []string) string {
	suggestion := SuggestCommand(input, commands, 3)
	if suggestion != "" {
		return fmt.Sprintf("Unknown command %q. Did you mean %q?\n\nRun 'gplay --help' for a list of commands.", input, suggestion)
	}
	return fmt.Sprintf("Unknown command %q.\n\nRun 'gplay --help' for a list of commands.", input)
}
