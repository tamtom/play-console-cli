package shared

import (
	"flag"
	"fmt"
)

// CommandUsageError carries a user-facing usage failure while preserving
// flag.ErrHelp semantics so ffcli renders the selected command's help.
type CommandUsageError struct {
	Message string
}

func (e *CommandUsageError) Error() string { return "Error: " + e.Message }
func (e *CommandUsageError) Unwrap() error { return flag.ErrHelp }

// UsageError returns a structured usage error.
// This is the standard way to report missing/invalid flags.
// It results in exit code 2 (usage error) when structured exit codes are wired.
func UsageError(msg string) error {
	return &CommandUsageError{Message: msg}
}

// UsageErrorf is like UsageError but with fmt.Sprintf formatting.
func UsageErrorf(format string, args ...any) error {
	return UsageError(fmt.Sprintf(format, args...))
}
