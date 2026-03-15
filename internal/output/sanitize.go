package output

import (
	"regexp"
	"strings"
)

// ansiPattern matches ANSI CSI and OSC escape sequences.
var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\][^\x07]*\x07|\x1b\[[0-9;]*m`)

// SanitizeTerminal strips ANSI escape sequences and control characters
// from s. Newlines (\n), carriage returns (\r), and tabs (\t) are preserved.
func SanitizeTerminal(s string) string {
	s = ansiPattern.ReplaceAllString(s, "")
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' || (r >= 32 && r != 0x7F) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
