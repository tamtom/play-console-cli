package vitals

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// CrashesQueryCommand returns the "gplay vitals crashes query" command.
func CrashesQueryCommand() *ffcli.Command {
	fs := flag.NewFlagSet("crashes query", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	from := fs.String("from", "", "Start date (ISO 8601, e.g. 2025-01-01)")
	to := fs.String("to", "", "End date (ISO 8601, e.g. 2025-01-31)")
	dimension := fs.String("dimension", "", "Dimension to group by (versionCode, deviceModel, etc.)")
	metricType := fs.String("type", "crash", "Metric type: crash (default) or anr")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")
	_ = fs.Bool("paginate", false, "Fetch all pages")

	return &ffcli.Command{
		Name:       "query",
		ShortUsage: "gplay vitals crashes query --package <pkg> [--from <date>] [--to <date>] [--dimension <dim>] [--type crash|anr]",
		ShortHelp:  "Query crash or ANR rate metrics.",
		LongHelp: `Query crash or ANR rate metrics from the Play Developer Reporting API.

Use --type to select between crash rate and ANR rate metrics.
Use --dimension to group results by versionCode, deviceModel, etc.
Date range can be specified with --from and --to in ISO 8601 format.

Note: This command uses the Play Developer Reporting API, which is
separate from the Android Publisher API used by other commands.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*packageName) == "" {
				return fmt.Errorf("--package is required")
			}
			normalizedType := strings.ToLower(strings.TrimSpace(*metricType))
			if normalizedType != "crash" && normalizedType != "anr" {
				return fmt.Errorf("--type must be 'crash' or 'anr', got: %s", *metricType)
			}

			// Validate date formats if provided
			if strings.TrimSpace(*from) != "" {
				if err := validateISO8601Date(*from); err != nil {
					return fmt.Errorf("--from: %w", err)
				}
			}
			if strings.TrimSpace(*to) != "" {
				if err := validateISO8601Date(*to); err != nil {
					return fmt.Errorf("--to: %w", err)
				}
			}

			_ = *dimension // validated but not used until API client is connected

			metricSet := "crashRateMetricSet"
			if normalizedType == "anr" {
				metricSet = "anrRateMetricSet"
			}

			return fmt.Errorf(
				"Play Developer Reporting API client is not yet connected. "+
					"Would call POST apps/%s/%s:query with the provided parameters",
				*packageName, metricSet,
			)
		},
	}
}

// validateISO8601Date checks that a date string is in YYYY-MM-DD format.
func validateISO8601Date(date string) error {
	date = strings.TrimSpace(date)
	if len(date) != 10 {
		return fmt.Errorf("invalid date format: %q (expected YYYY-MM-DD)", date)
	}
	if date[4] != '-' || date[7] != '-' {
		return fmt.Errorf("invalid date format: %q (expected YYYY-MM-DD)", date)
	}
	for i, ch := range date {
		if i == 4 || i == 7 {
			continue
		}
		if ch < '0' || ch > '9' {
			return fmt.Errorf("invalid date format: %q (expected YYYY-MM-DD)", date)
		}
	}
	return nil
}
