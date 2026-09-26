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

			if isFormatFlag(outputFlag) {
				explicit := map[string]bool{}
				cmd.FlagSet.Visit(func(f *flag.Flag) { explicit[f.Name] = true })
				// Commands with a "text" default support their own format list,
				// so the environment default applies only to JSON-default flags.
				if value := os.Getenv("GPLAY_DEFAULT_OUTPUT"); !explicit["output"] && value != "" && outputFlag.DefValue == "json" {
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
				// A --pretty default of true applies only to JSON output.
				if pretty && !explicit["pretty"] && format != "json" {
					if err := prettyFlag.Value.Set("false"); err != nil {
						return err
					}
					pretty = false
				}
				if err := ValidateOutputFlags(format, pretty, additionalFormats...); err != nil {
					return err
				}
			}
		}

		return originalExec(ctx, args)
	}
}

// isFormatFlag reports whether an --output flag selects an output format.
// Some commands, such as generated-apks download, use --output for a
// directory; their usage text does not start with "Output format".
func isFormatFlag(f *flag.Flag) bool {
	return f != nil && strings.HasPrefix(f.Usage, "Output format")
}
