package cmdtest_test

import "strings"

// stripDeprecatedCommandWarnings removes deprecation warning lines from stderr
// so test assertions can focus on actual errors.
func stripDeprecatedCommandWarnings(stderr string) string {
	lines := strings.Split(stderr, "\n")
	var kept []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Warning: ") && strings.Contains(trimmed, "is deprecated") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}
