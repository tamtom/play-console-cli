package performance

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// BatteryCommand returns the battery performance metrics subcommand.
func BatteryCommand() *ffcli.Command {
	fs := flag.NewFlagSet("performance battery", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	from := fs.String("from", "", "Start date (YYYY-MM-DD)")
	to := fs.String("to", "", "End date (YYYY-MM-DD)")
	dimension := fs.String("dimension", "", "Breakdown dimension (e.g. apiLevel, deviceModel, country)")
	metricType := fs.String("type", "", "Battery metric type: wakeup or wakelock")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")
	_ = fs.Bool("paginate", false, "Fetch all pages")

	return &ffcli.Command{
		Name:       "battery",
		ShortUsage: "gplay vitals performance battery --package <name> [flags]",
		ShortHelp:  "Get battery usage metrics.",
		LongHelp: `Get battery usage metrics.

Returns excessive wakeup or partial wake lock metrics. Use --type to select
between "wakeup" (excessive wake-ups) and "wakelock" (stuck partial wake locks).
Use --dimension to break down results by API level, device model, or country.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*packageName) == "" {
				return fmt.Errorf("--package is required")
			}
			if strings.TrimSpace(*metricType) != "" {
				t := strings.ToLower(strings.TrimSpace(*metricType))
				if t != "wakeup" && t != "wakelock" {
					return fmt.Errorf("--type must be 'wakeup' or 'wakelock'")
				}
			}

			// Stub: API client not connected yet.
			_ = from
			_ = to
			_ = dimension

			result := map[string]string{
				"status":  "stub",
				"message": "vitals performance battery: API client not connected yet",
				"package": *packageName,
				"type":    *metricType,
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}
