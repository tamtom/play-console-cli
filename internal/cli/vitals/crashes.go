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

const defaultCrashQueryPageSize int64 = 1000

var newReportingService = reportingclient.NewService

var (
	crashRateMetrics = []string{
		"crashRate",
		"crashRate7dUserWeighted",
		"crashRate28dUserWeighted",
		"userPerceivedCrashRate",
		"userPerceivedCrashRate7dUserWeighted",
		"userPerceivedCrashRate28dUserWeighted",
		"distinctUsers",
	}
	anrRateMetrics = []string{
		"anrRate",
		"anrRate7dUserWeighted",
		"anrRate28dUserWeighted",
		"userPerceivedAnrRate",
		"userPerceivedAnrRate7dUserWeighted",
		"userPerceivedAnrRate28dUserWeighted",
		"distinctUsers",
	}
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
	paginate := fs.Bool("paginate", false, "Fetch all pages")

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

			opts := crashQueryOptions{
				from:      *from,
				to:        *to,
				dimension: *dimension,
				paginate:  *paginate,
			}
			var result interface{}
			if normalizedType == "anr" {
				result, err = queryANRRate(ctx, service, pkg, opts)
			} else {
				result, err = queryCrashRate(ctx, service, pkg, opts)
			}
			if err != nil {
				return err
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}

type crashQueryOptions struct {
	from      string
	to        string
	dimension string
	paginate  bool
}

func queryCrashRate(ctx context.Context, service *reportingclient.Service, pkg string, opts crashQueryOptions) (interface{}, error) {
	timelineSpec, err := buildCrashTimelineSpec(opts.from, opts.to)
	if err != nil {
		return nil, err
	}

	name := fmt.Sprintf("apps/%s/crashRateMetricSet", pkg)
	if !opts.paginate {
		resp, err := service.API.Vitals.Crashrate.Query(name, &playdeveloperreporting.GooglePlayDeveloperReportingV1beta1QueryCrashRateMetricSetRequest{
			Dimensions:   buildCrashDimensions(opts.dimension),
			Metrics:      append([]string(nil), crashRateMetrics...),
			PageSize:     defaultCrashQueryPageSize,
			TimelineSpec: timelineSpec,
		}).Context(ctx).Do()
		if err != nil {
			return nil, shared.WrapGoogleAPIError("query crash rate metrics", err)
		}
		return resp, nil
	}

	var rows []*playdeveloperreporting.GooglePlayDeveloperReportingV1beta1MetricsRow
	pageToken := ""
	for {
		resp, err := service.API.Vitals.Crashrate.Query(name, &playdeveloperreporting.GooglePlayDeveloperReportingV1beta1QueryCrashRateMetricSetRequest{
			Dimensions:   buildCrashDimensions(opts.dimension),
			Metrics:      append([]string(nil), crashRateMetrics...),
			PageSize:     defaultCrashQueryPageSize,
			PageToken:    pageToken,
			TimelineSpec: timelineSpec,
		}).Context(ctx).Do()
		if err != nil {
			return nil, shared.WrapGoogleAPIError("query crash rate metrics", err)
		}
		rows = append(rows, resp.Rows...)
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return rows, nil
}

func queryANRRate(ctx context.Context, service *reportingclient.Service, pkg string, opts crashQueryOptions) (interface{}, error) {
	timelineSpec, err := buildCrashTimelineSpec(opts.from, opts.to)
	if err != nil {
		return nil, err
	}

	name := fmt.Sprintf("apps/%s/anrRateMetricSet", pkg)
	if !opts.paginate {
		resp, err := service.API.Vitals.Anrrate.Query(name, &playdeveloperreporting.GooglePlayDeveloperReportingV1beta1QueryAnrRateMetricSetRequest{
			Dimensions:   buildCrashDimensions(opts.dimension),
			Metrics:      append([]string(nil), anrRateMetrics...),
			PageSize:     defaultCrashQueryPageSize,
			TimelineSpec: timelineSpec,
		}).Context(ctx).Do()
		if err != nil {
			return nil, shared.WrapGoogleAPIError("query ANR rate metrics", err)
		}
		return resp, nil
	}

	var rows []*playdeveloperreporting.GooglePlayDeveloperReportingV1beta1MetricsRow
	pageToken := ""
	for {
		resp, err := service.API.Vitals.Anrrate.Query(name, &playdeveloperreporting.GooglePlayDeveloperReportingV1beta1QueryAnrRateMetricSetRequest{
			Dimensions:   buildCrashDimensions(opts.dimension),
			Metrics:      append([]string(nil), anrRateMetrics...),
			PageSize:     defaultCrashQueryPageSize,
			PageToken:    pageToken,
			TimelineSpec: timelineSpec,
		}).Context(ctx).Do()
		if err != nil {
			return nil, shared.WrapGoogleAPIError("query ANR rate metrics", err)
		}
		rows = append(rows, resp.Rows...)
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return rows, nil
}

func buildCrashTimelineSpec(from, to string) (*playdeveloperreporting.GooglePlayDeveloperReportingV1beta1TimelineSpec, error) {
	startTime, startDate, err := parseCrashDateFlag("--from", from)
	if err != nil {
		return nil, err
	}
	endTime, endDate, err := parseCrashDateFlag("--to", to)
	if err != nil {
		return nil, err
	}
	if !startDate.IsZero() && !endDate.IsZero() && startDate.After(endDate) {
		return nil, fmt.Errorf("--from must be on or before --to")
	}
	if startTime == nil && endTime == nil {
		return nil, nil
	}
	if endTime != nil {
		endExclusive := endDate.AddDate(0, 0, 1)
		endTime = &playdeveloperreporting.GoogleTypeDateTime{
			Year:  int64(endExclusive.Year()),
			Month: int64(endExclusive.Month()),
			Day:   int64(endExclusive.Day()),
		}
	}
	return &playdeveloperreporting.GooglePlayDeveloperReportingV1beta1TimelineSpec{
		AggregationPeriod: "DAILY",
		StartTime:         startTime,
		EndTime:           endTime,
	}, nil
}

func parseCrashDateFlag(flagName, value string) (*playdeveloperreporting.GoogleTypeDateTime, time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, time.Time{}, nil
	}
	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("invalid %s date %q: expected YYYY-MM-DD", flagName, value)
	}
	return &playdeveloperreporting.GoogleTypeDateTime{
		Year:  int64(parsed.Year()),
		Month: int64(parsed.Month()),
		Day:   int64(parsed.Day()),
	}, parsed, nil
}

func buildCrashDimensions(dimension string) []string {
	trimmed := strings.TrimSpace(dimension)
	if trimmed == "" {
		return nil
	}
	return []string{trimmed}
}

// validateISO8601Date checks that a date string is in YYYY-MM-DD format.
func validateISO8601Date(date string) error {
	date = strings.TrimSpace(date)
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return fmt.Errorf("invalid date format: %q (expected YYYY-MM-DD)", date)
	}
	return nil
}
