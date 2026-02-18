package output

import "regexp"

// ansiPattern matches ANSI CSI and OSC escape sequences.
var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\][^\x07]*\x07|\x1b\[[0-9;]*m`)

// SanitizeTerminal strips ANSI escape sequences and control characters
// from s. Newlines (\n), carriage returns (\r), and tabs (\t) are preserved.
func SanitizeTerminal(s string) string {
	// Strip ANSI sequences first
	s = ansiPattern.ReplaceAllString(s, "")

	// Strip control chars (0x00-0x08, 0x0B-0x0C, 0x0E-0x1F, 0x7F)
	// but preserve \n (0x0A), \r (0x0D), \t (0x09)
	var result []rune
	for _, r := range s {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			result = append(result, r)
		case r < 0x20 || r == 0x7F:
			// skip control characters
		default:
			result = append(result, r)
		}
	}
	return string(result)
}
