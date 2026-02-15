package performance

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// RenderingCommand returns the rendering performance metrics subcommand.
func RenderingCommand() *ffcli.Command {
	fs := flag.NewFlagSet("performance rendering", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	from := fs.String("from", "", "Start date (YYYY-MM-DD)")
	to := fs.String("to", "", "End date (YYYY-MM-DD)")
	dimension := fs.String("dimension", "", "Breakdown dimension (e.g. apiLevel, deviceModel, country)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")
	_ = fs.Bool("paginate", false, "Fetch all pages")

	return &ffcli.Command{
		Name:       "rendering",
		ShortUsage: "gplay vitals performance rendering --package <name> [flags]",
		ShortHelp:  "Get slow rendering metrics.",
		LongHelp: `Get slow rendering metrics.

Returns the percentage of frames that exceeded the 16ms and 700ms render
time thresholds. Use --dimension to break down results by API level,
device model, or country.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*packageName) == "" {
				return fmt.Errorf("--package is required")
			}

			// Stub: API client not connected yet.
			_ = from
			_ = to
			_ = dimension

			result := map[string]string{
				"status":  "stub",
				"message": "vitals performance rendering: API client not connected yet",
				"package": *packageName,
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}
