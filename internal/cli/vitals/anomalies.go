package vitals

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/playdeveloperreporting/v1beta1"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/reportingclient"
)

// anomalyMetricSets maps the --type flag to the Play Developer Reporting API
// metric set resource name.
var anomalyMetricSets = map[string]string{
	"crash":       "crashRateMetricSet",
	"anr":         "anrRateMetricSet",
	"errors":      "errorCountMetricSet",
	"performance": "slowRenderingRateMetricSet",
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

			service, err := newReportingService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := listAnomalies(ctx, service, pkg, metricSet, fromDate, toDate, *limit)
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func listAnomalies(ctx context.Context, service *reportingclient.Service, pkg, metricSet, fromDate, toDate string, limit int) (*playdeveloperreporting.GooglePlayDeveloperReportingV1beta1ListAnomaliesResponse, error) {
	filter, err := anomalyActiveBetweenFilter(fromDate, toDate)
	if err != nil {
		return nil, err
	}

	var anomalies []*playdeveloperreporting.GooglePlayDeveloperReportingV1beta1Anomaly
	parent := fmt.Sprintf("apps/%s", pkg)
	pageToken := ""
	nextPageToken := ""
	for len(anomalies) < limit {
		pageSize := limit - len(anomalies)
		if pageSize > 100 {
			pageSize = 100
		}
		call := service.API.Anomalies.List(parent).Context(ctx).Filter(filter).PageSize(int64(pageSize))
		if pageToken != "" {
			call.PageToken(pageToken)
		}
		resp, err := call.Do()
		if err != nil {
			return nil, shared.WrapGoogleAPIError("list anomalies", err)
		}
		nextPageToken = resp.NextPageToken
		for _, anomaly := range resp.Anomalies {
			if anomalyMetricMatches(anomaly, metricSet) {
				anomalies = append(anomalies, anomaly)
				if len(anomalies) == limit {
					break
				}
			}
		}
		if resp.NextPageToken == "" {
			nextPageToken = ""
			break
		}
		pageToken = resp.NextPageToken
	}

	return &playdeveloperreporting.GooglePlayDeveloperReportingV1beta1ListAnomaliesResponse{
		Anomalies:     anomalies,
		NextPageToken: nextPageToken,
	}, nil
}

func anomalyActiveBetweenFilter(fromDate, toDate string) (string, error) {
	from, err := time.Parse("2006-01-02", fromDate)
	if err != nil {
		return "", fmt.Errorf("--from: invalid date format: %q (expected YYYY-MM-DD)", fromDate)
	}
	to, err := time.Parse("2006-01-02", toDate)
	if err != nil {
		return "", fmt.Errorf("--to: invalid date format: %q (expected YYYY-MM-DD)", toDate)
	}
	if from.After(to) {
		return "", fmt.Errorf("--from must be on or before --to")
	}
	return fmt.Sprintf(
		`activeBetween("%sT00:00:00Z", "%sT00:00:00Z")`,
		from.Format("2006-01-02"),
		to.AddDate(0, 0, 1).Format("2006-01-02"),
	), nil
}

func anomalyMetricMatches(anomaly *playdeveloperreporting.GooglePlayDeveloperReportingV1beta1Anomaly, metricSet string) bool {
	if metricSet == "*" {
		return true
	}
	if anomaly == nil {
		return false
	}
	return anomaly.MetricSet == metricSet || strings.HasSuffix(anomaly.MetricSet, "/"+metricSet)
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
