package vitals

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// anomalyMetricSets maps the --type flag to the Play Developer Reporting API
// metric set resource name.
var anomalyMetricSets = map[string]string{
	"crash":       "crashRateMetricSet",
	"anr":         "anrRateMetricSet",
	"errors":      "errorCountMetricSet",
	"performance": "slowRenderingRate20FpsMetricSet",
	"all":         "*",
}

// AnomaliesCommand returns the "gplay vitals crashes anomalies" command.
func AnomaliesCommand() *ffcli.Command {
	fs := flag.NewFlagSet("crashes anomalies", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	metricType := fs.String("type", "all", "Metric set: crash, anr, errors, performance, all")
	from := fs.String("from", "", "Start date (YYYY-MM-DD); defaults to 7d ago")
	to := fs.String("to", "", "End date (YYYY-MM-DD); defaults to today")
	limit := fs.Int("limit", 50, "Maximum anomalies to return (1-1000)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "anomalies",
		ShortUsage: "gplay vitals crashes anomalies --package <pkg> [flags]",
		ShortHelp:  "List detected anomalies for crash, ANR, errors, and performance metrics.",
		LongHelp: `List detected anomalies for vitals metrics.

Anomalies are automatically detected deviations in vitals metrics that may
indicate a regression introduced by a new release.

Examples:
  gplay vitals crashes anomalies --package com.example.app
  gplay vitals crashes anomalies --package com.example.app --type anr
  gplay vitals crashes anomalies --package com.example.app --from 2026-03-01 --to 2026-03-31

Note: This command uses the Play Developer Reporting API, which is separate
from the Android Publisher API used by other commands.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*packageName) == "" {
				return fmt.Errorf("--package is required")
			}
			metricSet, err := resolveAnomalyMetric(*metricType)
			if err != nil {
				return err
			}
			fromDate, toDate, err := resolveAnomalyDates(*from, *to)
			if err != nil {
				return err
			}
			if *limit < 1 || *limit > 1000 {
				return fmt.Errorf("--limit must be between 1 and 1000")
			}

			return fmt.Errorf(
				"play Developer Reporting API client is not yet connected, "+
					"would list anomalies for app %s (metricSet=%s, from=%s, to=%s, limit=%d)",
				*packageName, metricSet, fromDate, toDate, *limit,
			)
		},
	}
}

func resolveAnomalyMetric(input string) (string, error) {
	key := strings.ToLower(strings.TrimSpace(input))
	if key == "" {
		key = "all"
	}
	metric, ok := anomalyMetricSets[key]
	if !ok {
		valid := make([]string, 0, len(anomalyMetricSets))
		for k := range anomalyMetricSets {
			valid = append(valid, k)
		}
		return "", fmt.Errorf("--type must be one of: %s", strings.Join(valid, ", "))
	}
	return metric, nil
}

func resolveAnomalyDates(from, to string) (string, string, error) {
	now := time.Now().UTC()
	toDate := strings.TrimSpace(to)
	if toDate == "" {
		toDate = now.Format("2006-01-02")
	} else if err := validateISO8601Date(toDate); err != nil {
		return "", "", fmt.Errorf("--to: %w", err)
	}
	fromDate := strings.TrimSpace(from)
	if fromDate == "" {
		fromDate = now.Add(-7 * 24 * time.Hour).Format("2006-01-02")
	} else if err := validateISO8601Date(fromDate); err != nil {
		return "", "", fmt.Errorf("--from: %w", err)
	}
	return fromDate, toDate, nil
}
