package shared

import (
	"context"
	"flag"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
)

// WrapCommandOutputValidation recursively wraps all commands' Exec functions
// to validate output format flags before execution. This prevents API calls
// when invalid output flags are passed.
func WrapCommandOutputValidation(cmd *ffcli.Command) {
	if cmd == nil {
		return
	}

	// Wrap subcommands recursively
	for _, sub := range cmd.Subcommands {
		WrapCommandOutputValidation(sub)
	}

	// Wrap this command's Exec
	originalExec := cmd.Exec
	if originalExec == nil {
		return
	}

	cmd.Exec = func(ctx context.Context, args []string) error {
		// Check if --output flag exists and validate its value
		if cmd.FlagSet != nil {
			outputFlag := cmd.FlagSet.Lookup("output")
			prettyFlag := cmd.FlagSet.Lookup("pretty")

			if outputFlag != nil {
				explicit := false
				cmd.FlagSet.Visit(func(f *flag.Flag) {
					if f.Name == "output" {
						explicit = true
					}
				})
				if value := os.Getenv("GPLAY_DEFAULT_OUTPUT"); !explicit && value != "" {
					if err := outputFlag.Value.Set(value); err != nil {
						return err
					}
				}
				format := strings.ToLower(strings.TrimSpace(outputFlag.Value.String()))
				var additionalFormats []string
				if outputFlag.DefValue == "text" {
					additionalFormats = append(additionalFormats, "text")
				}
				if err := outputFlag.Value.Set(format); err != nil {
					return err
				}
				pretty := prettyFlag != nil && prettyFlag.Value.String() == "true"
				if err := ValidateOutputFlags(format, pretty, additionalFormats...); err != nil {
					return err
				}
			}
		}

		return originalExec(ctx, args)
	}
}
