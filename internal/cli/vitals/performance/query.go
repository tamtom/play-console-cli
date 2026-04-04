package performance

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/api/playdeveloperreporting/v1beta1"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/reportingclient"
)

const defaultMetricQueryPageSize int64 = 1000

var newReportingService = reportingclient.NewService

var (
	startupMetrics = []string{
		"slowStartRate",
		"slowStartRate7dUserWeighted",
		"slowStartRate28dUserWeighted",
		"distinctUsers",
	}
	renderingMetrics = []string{
		"slowRenderingRate20Fps",
		"slowRenderingRate20Fps7dUserWeighted",
		"slowRenderingRate20Fps28dUserWeighted",
		"slowRenderingRate30Fps",
		"slowRenderingRate30Fps7dUserWeighted",
		"slowRenderingRate30Fps28dUserWeighted",
		"distinctUsers",
	}
	excessiveWakeupMetrics = []string{
		"excessiveWakeupRate",
		"excessiveWakeupRate7dUserWeighted",
		"excessiveWakeupRate28dUserWeighted",
		"distinctUsers",
	}
	stuckBackgroundWakelockMetrics = []string{
		"stuckBgWakelockRate",
		"stuckBgWakelockRate7dUserWeighted",
		"stuckBgWakelockRate28dUserWeighted",
		"distinctUsers",
	}
)

type queryOptions struct {
	from      string
	to        string
	dimension string
	paginate  bool
}

func executeStartupQuery(ctx context.Context, packageName string, opts queryOptions, outputFlag string, pretty bool) error {
	service, err := newReportingService(ctx)
	if err != nil {
		return err
	}
	pkg := shared.ResolvePackageName(packageName, service.Cfg)
	if strings.TrimSpace(pkg) == "" {
		return fmt.Errorf("--package is required")
	}

	ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
	defer cancel()

	result, err := querySlowStartRate(ctx, service, pkg, opts)
	if err != nil {
		return err
	}
	return shared.PrintOutput(result, outputFlag, pretty)
}

func executeRenderingQuery(ctx context.Context, packageName string, opts queryOptions, outputFlag string, pretty bool) error {
	service, err := newReportingService(ctx)
	if err != nil {
		return err
	}
	pkg := shared.ResolvePackageName(packageName, service.Cfg)
	if strings.TrimSpace(pkg) == "" {
		return fmt.Errorf("--package is required")
	}

	ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
	defer cancel()

	result, err := querySlowRenderingRate(ctx, service, pkg, opts)
	if err != nil {
		return err
	}
	return shared.PrintOutput(result, outputFlag, pretty)
}

func executeBatteryQuery(ctx context.Context, packageName, metricType string, opts queryOptions, outputFlag string, pretty bool) error {
	service, err := newReportingService(ctx)
	if err != nil {
		return err
	}
	pkg := shared.ResolvePackageName(packageName, service.Cfg)
	if strings.TrimSpace(pkg) == "" {
		return fmt.Errorf("--package is required")
	}

	ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
	defer cancel()

	normalizedType := strings.ToLower(strings.TrimSpace(metricType))
	var result interface{}
	switch normalizedType {
	case "":
		wakeup, err := queryExcessiveWakeupRate(ctx, service, pkg, opts)
		if err != nil {
			return err
		}
		wakelock, err := queryStuckBackgroundWakelockRate(ctx, service, pkg, opts)
		if err != nil {
			return err
		}
		result = map[string]interface{}{
			"wakeup":   wakeup,
			"wakelock": wakelock,
		}
	case "wakeup":
		result, err = queryExcessiveWakeupRate(ctx, service, pkg, opts)
	case "wakelock":
		result, err = queryStuckBackgroundWakelockRate(ctx, service, pkg, opts)
	}
	if err != nil {
		return err
	}
	return shared.PrintOutput(result, outputFlag, pretty)
}

func querySlowStartRate(ctx context.Context, service *reportingclient.Service, pkg string, opts queryOptions) (interface{}, error) {
	timelineSpec, err := buildDailyTimelineSpec(opts.from, opts.to)
	if err != nil {
		return nil, err
	}

	name := metricSetName(pkg, "slowStartRateMetricSet")
	if !opts.paginate {
		resp, err := service.API.Vitals.Slowstartrate.Query(name, &playdeveloperreporting.GooglePlayDeveloperReportingV1beta1QuerySlowStartRateMetricSetRequest{
			Dimensions:   buildDimensions(opts.dimension),
			Metrics:      append([]string(nil), startupMetrics...),
			PageSize:     defaultMetricQueryPageSize,
			TimelineSpec: timelineSpec,
		}).Context(ctx).Do()
		if err != nil {
			return nil, shared.WrapGoogleAPIError("query slow start rate metrics", err)
		}
		return resp, nil
	}

	var rows []*playdeveloperreporting.GooglePlayDeveloperReportingV1beta1MetricsRow
	pageToken := ""
	for {
		resp, err := service.API.Vitals.Slowstartrate.Query(name, &playdeveloperreporting.GooglePlayDeveloperReportingV1beta1QuerySlowStartRateMetricSetRequest{
			Dimensions:   buildDimensions(opts.dimension),
			Metrics:      append([]string(nil), startupMetrics...),
			PageSize:     defaultMetricQueryPageSize,
			PageToken:    pageToken,
			TimelineSpec: timelineSpec,
		}).Context(ctx).Do()
		if err != nil {
			return nil, shared.WrapGoogleAPIError("query slow start rate metrics", err)
		}
		rows = append(rows, resp.Rows...)
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return rows, nil
}

func querySlowRenderingRate(ctx context.Context, service *reportingclient.Service, pkg string, opts queryOptions) (interface{}, error) {
	timelineSpec, err := buildDailyTimelineSpec(opts.from, opts.to)
	if err != nil {
		return nil, err
	}

	name := metricSetName(pkg, "slowRenderingRateMetricSet")
	if !opts.paginate {
		resp, err := service.API.Vitals.Slowrenderingrate.Query(name, &playdeveloperreporting.GooglePlayDeveloperReportingV1beta1QuerySlowRenderingRateMetricSetRequest{
			Dimensions:   buildDimensions(opts.dimension),
			Metrics:      append([]string(nil), renderingMetrics...),
			PageSize:     defaultMetricQueryPageSize,
			TimelineSpec: timelineSpec,
		}).Context(ctx).Do()
		if err != nil {
			return nil, shared.WrapGoogleAPIError("query slow rendering rate metrics", err)
		}
		return resp, nil
	}

	var rows []*playdeveloperreporting.GooglePlayDeveloperReportingV1beta1MetricsRow
	pageToken := ""
	for {
		resp, err := service.API.Vitals.Slowrenderingrate.Query(name, &playdeveloperreporting.GooglePlayDeveloperReportingV1beta1QuerySlowRenderingRateMetricSetRequest{
			Dimensions:   buildDimensions(opts.dimension),
			Metrics:      append([]string(nil), renderingMetrics...),
			PageSize:     defaultMetricQueryPageSize,
			PageToken:    pageToken,
			TimelineSpec: timelineSpec,
		}).Context(ctx).Do()
		if err != nil {
			return nil, shared.WrapGoogleAPIError("query slow rendering rate metrics", err)
		}
		rows = append(rows, resp.Rows...)
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return rows, nil
}

func queryExcessiveWakeupRate(ctx context.Context, service *reportingclient.Service, pkg string, opts queryOptions) (interface{}, error) {
	timelineSpec, err := buildDailyTimelineSpec(opts.from, opts.to)
	if err != nil {
		return nil, err
	}

	name := metricSetName(pkg, "excessiveWakeupRateMetricSet")
	if !opts.paginate {
		resp, err := service.API.Vitals.Excessivewakeuprate.Query(name, &playdeveloperreporting.GooglePlayDeveloperReportingV1beta1QueryExcessiveWakeupRateMetricSetRequest{
			Dimensions:   buildDimensions(opts.dimension),
			Metrics:      append([]string(nil), excessiveWakeupMetrics...),
			PageSize:     defaultMetricQueryPageSize,
			TimelineSpec: timelineSpec,
		}).Context(ctx).Do()
		if err != nil {
			return nil, shared.WrapGoogleAPIError("query excessive wakeup rate metrics", err)
		}
		return resp, nil
	}

	var rows []*playdeveloperreporting.GooglePlayDeveloperReportingV1beta1MetricsRow
	pageToken := ""
	for {
		resp, err := service.API.Vitals.Excessivewakeuprate.Query(name, &playdeveloperreporting.GooglePlayDeveloperReportingV1beta1QueryExcessiveWakeupRateMetricSetRequest{
			Dimensions:   buildDimensions(opts.dimension),
			Metrics:      append([]string(nil), excessiveWakeupMetrics...),
			PageSize:     defaultMetricQueryPageSize,
			PageToken:    pageToken,
			TimelineSpec: timelineSpec,
		}).Context(ctx).Do()
		if err != nil {
			return nil, shared.WrapGoogleAPIError("query excessive wakeup rate metrics", err)
		}
		rows = append(rows, resp.Rows...)
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return rows, nil
}

func queryStuckBackgroundWakelockRate(ctx context.Context, service *reportingclient.Service, pkg string, opts queryOptions) (interface{}, error) {
	timelineSpec, err := buildDailyTimelineSpec(opts.from, opts.to)
	if err != nil {
		return nil, err
	}

	name := metricSetName(pkg, "stuckBackgroundWakelockRateMetricSet")
	if !opts.paginate {
		resp, err := service.API.Vitals.Stuckbackgroundwakelockrate.Query(name, &playdeveloperreporting.GooglePlayDeveloperReportingV1beta1QueryStuckBackgroundWakelockRateMetricSetRequest{
			Dimensions:   buildDimensions(opts.dimension),
			Metrics:      append([]string(nil), stuckBackgroundWakelockMetrics...),
			PageSize:     defaultMetricQueryPageSize,
			TimelineSpec: timelineSpec,
		}).Context(ctx).Do()
		if err != nil {
			return nil, shared.WrapGoogleAPIError("query stuck background wakelock rate metrics", err)
		}
		return resp, nil
	}

	var rows []*playdeveloperreporting.GooglePlayDeveloperReportingV1beta1MetricsRow
	pageToken := ""
	for {
		resp, err := service.API.Vitals.Stuckbackgroundwakelockrate.Query(name, &playdeveloperreporting.GooglePlayDeveloperReportingV1beta1QueryStuckBackgroundWakelockRateMetricSetRequest{
			Dimensions:   buildDimensions(opts.dimension),
			Metrics:      append([]string(nil), stuckBackgroundWakelockMetrics...),
			PageSize:     defaultMetricQueryPageSize,
			PageToken:    pageToken,
			TimelineSpec: timelineSpec,
		}).Context(ctx).Do()
		if err != nil {
			return nil, shared.WrapGoogleAPIError("query stuck background wakelock rate metrics", err)
		}
		rows = append(rows, resp.Rows...)
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return rows, nil
}

func buildDailyTimelineSpec(from, to string) (*playdeveloperreporting.GooglePlayDeveloperReportingV1beta1TimelineSpec, error) {
	startTime, startDate, err := parseDateFlag("--from", from)
	if err != nil {
		return nil, err
	}
	endTime, endDate, err := parseDateFlag("--to", to)
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
		// Play Developer Reporting treats endTime as exclusive; advance by one day
		// so a user-supplied --to date behaves as an inclusive upper bound.
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

func parseDateFlag(flagName, value string) (*playdeveloperreporting.GoogleTypeDateTime, time.Time, error) {
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

func buildDimensions(dimension string) []string {
	trimmed := strings.TrimSpace(dimension)
	if trimmed == "" {
		return nil
	}
	return []string{trimmed}
}

func metricSetName(pkg, metricSet string) string {
	return fmt.Sprintf("apps/%s/%s", pkg, metricSet)
}
