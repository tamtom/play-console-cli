package performance

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// StartupCommand returns the startup performance metrics subcommand.
func StartupCommand() *ffcli.Command {
	fs := flag.NewFlagSet("performance startup", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	from := fs.String("from", "", "Start date (YYYY-MM-DD)")
	to := fs.String("to", "", "End date (YYYY-MM-DD)")
	dimension := fs.String("dimension", "", "Breakdown dimension (e.g. apiLevel, deviceModel, country)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")
	_ = fs.Bool("paginate", false, "Fetch all pages")

	return &ffcli.Command{
		Name:       "startup",
		ShortUsage: "gplay vitals performance startup --package <name> [flags]",
		ShortHelp:  "Get app startup time metrics.",
		LongHelp: `Get app startup time metrics.

Returns cold, warm, and hot start duration percentiles for the specified
package and date range. Use --dimension to break down results by API level,
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
				"message": "vitals performance startup: API client not connected yet",
				"package": *packageName,
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}
