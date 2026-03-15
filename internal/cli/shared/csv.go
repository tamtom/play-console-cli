package shared

import "strings"

// SplitCSV splits a comma-separated string, trims whitespace from each element,
// and removes empty strings.
func SplitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// SplitUniqueCSV splits a comma-separated string, trims whitespace,
// removes empties, and deduplicates (preserving first occurrence order).
func SplitUniqueCSV(s string) []string {
	parts := SplitCSV(s)
	seen := make(map[string]bool, len(parts))
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if !seen[p] {
			seen[p] = true
			result = append(result, p)
		}
	}
	return result
}
