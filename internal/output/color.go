package output

import (
	"fmt"
	"os"
)

// ANSI escape codes.
const (
	ansiReset  = "\033[0m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiCyan   = "\033[36m"
	ansiBold   = "\033[1m"
	ansiDim    = "\033[2m"
)

// colorEnabled tracks whether ANSI color output is active.
var colorEnabled bool

func init() {
	initColors()
}

// initColors determines whether colors should be enabled based on the NO_COLOR
// environment variable (https://no-color.org) and whether stderr is a TTY.
func initColors() {
	// NO_COLOR spec: when the variable is present (regardless of value),
	// disable color output.
	if _, present := os.LookupEnv("NO_COLOR"); present {
		colorEnabled = false
		return
	}

	// Check if stderr is a terminal (character device).
	fi, err := os.Stderr.Stat()
	if err != nil {
		colorEnabled = false
		return
	}
	colorEnabled = fi.Mode()&os.ModeCharDevice != 0
}

// ColorsEnabled reports whether ANSI color output is currently active.
func ColorsEnabled() bool {
	return colorEnabled
}

// wrap returns s wrapped in the given ANSI code if colors are enabled,
// otherwise returns s unchanged.
func wrap(code, s string) string {
	if !colorEnabled {
		return s
	}
	return fmt.Sprintf("%s%s%s", code, s, ansiReset)
}

// Green wraps s in green ANSI color.
func Green(s string) string { return wrap(ansiGreen, s) }

// Yellow wraps s in yellow ANSI color.
func Yellow(s string) string { return wrap(ansiYellow, s) }

// Red wraps s in red ANSI color.
func Red(s string) string { return wrap(ansiRed, s) }

// Bold wraps s in bold ANSI style.
func Bold(s string) string { return wrap(ansiBold, s) }

// Cyan wraps s in cyan ANSI color.
func Cyan(s string) string { return wrap(ansiCyan, s) }

// Dim wraps s in dim ANSI style.
func Dim(s string) string { return wrap(ansiDim, s) }

// StatusColor returns the status string wrapped in a color appropriate to its
// meaning: green for completed/active, yellow for draft/inProgress, red for
// halted/failed. Unknown statuses are returned unmodified.
func StatusColor(status string) string {
	switch status {
	case "completed", "active":
		return Green(status)
	case "draft", "inProgress":
		return Yellow(status)
	case "halted", "failed":
		return Red(status)
	default:
		return status
	}
}
