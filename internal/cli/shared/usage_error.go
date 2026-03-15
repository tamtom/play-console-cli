package shared

import (
	"flag"
	"fmt"
	"os"
)

// UsageError prints msg to stderr and returns flag.ErrHelp.
// This is the standard way to report missing/invalid flags.
// It results in exit code 2 (usage error) when structured exit codes are wired.
func UsageError(msg string) error {
	fmt.Fprintln(os.Stderr, "Error: "+msg)
	return flag.ErrHelp
}

// UsageErrorf is like UsageError but with fmt.Sprintf formatting.
func UsageErrorf(format string, args ...any) error {
	return UsageError(fmt.Sprintf(format, args...))
}
