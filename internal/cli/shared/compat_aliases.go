package shared

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
)

// DeprecatedAliasLeafCommand creates a deprecated alias that wraps an existing command.
// When invoked, it prints a deprecation warning to stderr then runs the original command.
// The alias is hidden from parent help via the "DEPRECATED:" prefix in ShortHelp.
func DeprecatedAliasLeafCommand(original *ffcli.Command, oldName, newCommandPath string) *ffcli.Command {
	return &ffcli.Command{
		Name:        oldName,
		ShortUsage:  strings.Replace(original.ShortUsage, original.Name, oldName, 1),
		ShortHelp:   fmt.Sprintf("DEPRECATED: use `%s` instead.", newCommandPath),
		LongHelp:    fmt.Sprintf("Deprecated compatibility alias for `%s`.\n\n%s", newCommandPath, original.LongHelp),
		FlagSet:     original.FlagSet,
		Subcommands: original.Subcommands,
		UsageFunc:   original.UsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			fmt.Fprintf(os.Stderr, "Warning: %q is deprecated. Use `%s` instead.\n", oldName, newCommandPath)
			if original.Exec != nil {
				return original.Exec(ctx, args)
			}
			return nil
		},
	}
}

// VisibleUsageFunc renders help for a command, hiding deprecated subcommands.
// A subcommand is considered deprecated if its ShortHelp starts with "DEPRECATED:".
func VisibleUsageFunc(cmd *ffcli.Command) string {
	// Filter out deprecated subcommands for display
	var visible []*ffcli.Command
	for _, sub := range cmd.Subcommands {
		if !strings.HasPrefix(sub.ShortHelp, "DEPRECATED:") {
			visible = append(visible, sub)
		}
	}

	// Create a shallow copy with only visible subcommands
	clone := *cmd
	clone.Subcommands = visible
	return DefaultUsageFunc(&clone)
}
