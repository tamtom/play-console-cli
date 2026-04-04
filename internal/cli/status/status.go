package status

import (
	"context"
	"flag"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/playdeveloperreporting/v1beta1"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
	"github.com/tamtom/play-console-cli/internal/reportingclient"
)

const defaultPollInterval = 30 * time.Second
const vitalsWindowDays = 7

var (
	newPlayService      = playclient.NewService
	newReportingService = reportingclient.NewService
	nowFunc             = time.Now
	afterFunc           = time.After
)

// StatusCommand returns the top-level gplay status command.
func StatusCommand() *ffcli.Command {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	watch := fs.Bool("watch", false, "Continuously refresh the status snapshot")
	pollInterval := fs.Duration("poll-interval", defaultPollInterval, "Polling interval for --watch")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "status",
		ShortUsage: "gplay status --package <name> [flags]",
		ShortHelp:  "Show a deterministic release-health snapshot.",
		LongHelp: `Show a single Play-specific status snapshot for the current app.

The snapshot combines current release-track state with a vitals summary.
Use --watch to refresh the snapshot on an interval until interrupted.

Examples:
  gplay status --package com.example.app
  gplay status --package com.example.app --watch --poll-interval 30s
  gplay status --package com.example.app --pretty`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if strings.TrimSpace(*packageName) == "" {
				return fmt.Errorf("--package is required")
			}
			if *watch && *pollInterval <= 0 {
				return fmt.Errorf("--poll-interval must be greater than 0")
			}

			return run(ctx, statusOptions{
				packageName:  *packageName,
				watch:        *watch,
				pollInterval: *pollInterval,
				pretty:       *pretty,
			})
		},
	}
}

type statusOptions struct {
	packageName  string
	watch        bool
	pollInterval time.Duration
	pretty       bool
}

type StatusReport struct {
	Package             string          `json:"package"`
	GeneratedAt         time.Time       `json:"generated_at"`
	Watch               bool            `json:"watch"`
	PollIntervalSeconds int64           `json:"poll_interval_seconds,omitempty"`
	Status              string          `json:"status"`
	Sources             []SourceStatus  `json:"sources"`
	Tracks              *TracksSnapshot `json:"tracks,omitempty"`
	Vitals              *VitalsSnapshot `json:"vitals,omitempty"`
}

type SourceStatus struct {
	Name  string `json:"name"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type TracksSnapshot struct {
	EditID string          `json:"edit_id"`
	Tracks []TrackSnapshot `json:"tracks"`
}

type TrackSnapshot struct {
	Track    string            `json:"track"`
	Releases []ReleaseSnapshot `json:"releases"`
}

type ReleaseSnapshot struct {
	Name         string   `json:"name,omitempty"`
	Status       string   `json:"status"`
	VersionCodes []int64  `json:"version_codes"`
	UserFraction *float64 `json:"user_fraction,omitempty"`
}

type VitalsSnapshot struct {
	Window    TimeWindow         `json:"window"`
	Startup   *MetricSetSnapshot `json:"startup,omitempty"`
	Rendering *MetricSetSnapshot `json:"rendering,omitempty"`
	Battery   *BatterySnapshot   `json:"battery,omitempty"`
}

type BatterySnapshot struct {
	Wakeup   *MetricSetSnapshot `json:"wakeup,omitempty"`
	Wakelock *MetricSetSnapshot `json:"wakelock,omitempty"`
}

type MetricSetSnapshot struct {
	MetricSet string            `json:"metric_set"`
	Rows      int               `json:"rows"`
	Latest    *MetricRowSummary `json:"latest,omitempty"`
}

type MetricRowSummary struct {
	StartTime         string            `json:"start_time,omitempty"`
	AggregationPeriod string            `json:"aggregation_period,omitempty"`
	Dimensions        map[string]string `json:"dimensions,omitempty"`
	Metrics           map[string]string `json:"metrics,omitempty"`
}

type TimeWindow struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

func run(ctx context.Context, opts statusOptions) error {
	for {
		report, err := buildStatusReport(ctx, opts)
		if err != nil {
			return err
		}
		if err := shared.PrintOutput(report, "json", opts.pretty); err != nil {
			return err
		}
		if !opts.watch {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-afterFunc(opts.pollInterval):
		}
	}
}

func buildStatusReport(ctx context.Context, opts statusOptions) (*StatusReport, error) {
	pkg, err := shared.RequirePackageName(opts.packageName, nil)
	if err != nil {
		return nil, err
	}

	report := &StatusReport{
		Package:     pkg,
		GeneratedAt: nowFunc().UTC(),
		Watch:       opts.watch,
		Sources: []SourceStatus{
			{Name: "tracks"},
			{Name: "vitals"},
		},
	}
	if opts.watch {
		report.PollIntervalSeconds = int64(opts.pollInterval / time.Second)
	}

	playSvc, playErr := newPlayService(ctx)
	if playErr != nil {
		report.Sources[0].Error = playErr.Error()
	} else {
		tracks, err := fetchTracksSnapshot(ctx, playSvc, pkg)
		if err != nil {
			report.Sources[0].Error = err.Error()
		} else {
			report.Sources[0].OK = true
			report.Tracks = tracks
		}
	}

	reportingSvc, reportingErr := newReportingService(ctx)
	if reportingErr != nil {
		report.Sources[1].Error = reportingErr.Error()
	} else {
		vitals, err := fetchVitalsSnapshot(ctx, reportingSvc, pkg, report.GeneratedAt)
		if err != nil {
			report.Sources[1].Error = err.Error()
		} else {
			report.Sources[1].OK = true
			report.Vitals = vitals
		}
	}

	report.Status = deriveOverallStatus(report.Sources)
	return report, nil
}

func deriveOverallStatus(sources []SourceStatus) string {
	okCount := 0
	for _, source := range sources {
		if source.OK {
			okCount++
		}
	}
	switch {
	case okCount == len(sources):
		return "ok"
	case okCount == 0:
		return "error"
	default:
		return "degraded"
	}
}

func fetchTracksSnapshot(ctx context.Context, svc *playclient.Service, pkg string) (*TracksSnapshot, error) {
	reqCtx, cancel := shared.ContextWithTimeout(ctx, svc.Cfg)
	defer cancel()

	edit, err := svc.API.Edits.Insert(pkg, &androidpublisher.AppEdit{}).Context(reqCtx).Do()
	if err != nil {
		return nil, fmt.Errorf("tracks: create edit: %w", err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := shared.ContextWithTimeout(context.Background(), svc.Cfg)
		defer cleanupCancel()
		_ = svc.API.Edits.Delete(pkg, edit.Id).Context(cleanupCtx).Do()
	}()

	resp, err := svc.API.Edits.Tracks.List(pkg, edit.Id).Context(reqCtx).Do()
	if err != nil {
		return nil, fmt.Errorf("tracks: list: %w", err)
	}

	tracks := make([]TrackSnapshot, 0, len(resp.Tracks))
	for _, track := range resp.Tracks {
		tracks = append(tracks, summarizeTrack(track))
	}
	sort.SliceStable(tracks, func(i, j int) bool {
		return tracks[i].Track < tracks[j].Track
	})

	return &TracksSnapshot{
		EditID: edit.Id,
		Tracks: tracks,
	}, nil
}

func summarizeTrack(track *androidpublisher.Track) TrackSnapshot {
	if track == nil {
		return TrackSnapshot{}
	}
	summary := TrackSnapshot{Track: track.Track}
	for _, release := range track.Releases {
		summary.Releases = append(summary.Releases, summarizeRelease(release))
	}
	sort.SliceStable(summary.Releases, func(i, j int) bool {
		if summary.Releases[i].Status != summary.Releases[j].Status {
			return summary.Releases[i].Status < summary.Releases[j].Status
		}
		if summary.Releases[i].Name != summary.Releases[j].Name {
			return summary.Releases[i].Name < summary.Releases[j].Name
		}
		return joinVersionCodes(summary.Releases[i].VersionCodes) < joinVersionCodes(summary.Releases[j].VersionCodes)
	})
	return summary
}

func summarizeRelease(release *androidpublisher.TrackRelease) ReleaseSnapshot {
	if release == nil {
		return ReleaseSnapshot{}
	}
	summary := ReleaseSnapshot{
		Name:         release.Name,
		Status:       release.Status,
		VersionCodes: append([]int64(nil), release.VersionCodes...),
	}
	if release.UserFraction > 0 {
		value := release.UserFraction
		summary.UserFraction = &value
	}
	return summary
}

func fetchVitalsSnapshot(ctx context.Context, svc *reportingclient.Service, pkg string, now time.Time) (*VitalsSnapshot, error) {
	reqCtx, cancel := shared.ContextWithTimeout(ctx, svc.Cfg)
	defer cancel()

	window := currentWindow(now)
	snapshot := &VitalsSnapshot{Window: window}

	startup, err := queryMetricSet(reqCtx, svc.API.Vitals.Slowstartrate, metricQuerySpec{
		name:     metricSetName(pkg, "slowStartRateMetricSet"),
		metrics:  startupMetrics,
		timeline: window.timelineSpec(),
		request:  "startup",
	})
	if err != nil {
		return nil, err
	}
	snapshot.Startup = startup

	rendering, err := queryMetricSet(reqCtx, svc.API.Vitals.Slowrenderingrate, metricQuerySpec{
		name:     metricSetName(pkg, "slowRenderingRateMetricSet"),
		metrics:  renderingMetrics,
		timeline: window.timelineSpec(),
		request:  "rendering",
	})
	if err != nil {
		return nil, err
	}
	snapshot.Rendering = rendering

	wakeup, err := queryMetricSet(reqCtx, svc.API.Vitals.Excessivewakeuprate, metricQuerySpec{
		name:     metricSetName(pkg, "excessiveWakeupRateMetricSet"),
		metrics:  excessiveWakeupMetrics,
		timeline: window.timelineSpec(),
		request:  "battery wakeup",
	})
	if err != nil {
		return nil, err
	}
	wakelock, err := queryMetricSet(reqCtx, svc.API.Vitals.Stuckbackgroundwakelockrate, metricQuerySpec{
		name:     metricSetName(pkg, "stuckBackgroundWakelockRateMetricSet"),
		metrics:  stuckBackgroundWakelockMetrics,
		timeline: window.timelineSpec(),
		request:  "battery wakelock",
	})
	if err != nil {
		return nil, err
	}
	snapshot.Battery = &BatterySnapshot{
		Wakeup:   wakeup,
		Wakelock: wakelock,
	}

	return snapshot, nil
}

type metricQuerySpec struct {
	name     string
	metrics  []string
	timeline *playdeveloperreporting.GooglePlayDeveloperReportingV1beta1TimelineSpec
	request  string
}

var startupMetrics = []string{
	"slowStartRate",
	"slowStartRate7dUserWeighted",
	"slowStartRate28dUserWeighted",
	"distinctUsers",
}

var renderingMetrics = []string{
	"slowRenderingRate20Fps",
	"slowRenderingRate20Fps7dUserWeighted",
	"slowRenderingRate20Fps28dUserWeighted",
	"slowRenderingRate30Fps",
	"slowRenderingRate30Fps7dUserWeighted",
	"slowRenderingRate30Fps28dUserWeighted",
	"distinctUsers",
}

var excessiveWakeupMetrics = []string{
	"excessiveWakeupRate",
	"excessiveWakeupRate7dUserWeighted",
	"excessiveWakeupRate28dUserWeighted",
	"distinctUsers",
}

var stuckBackgroundWakelockMetrics = []string{
	"stuckBgWakelockRate",
	"stuckBgWakelockRate7dUserWeighted",
	"stuckBgWakelockRate28dUserWeighted",
	"distinctUsers",
}

func queryMetricSet(ctx context.Context, service interface{}, spec metricQuerySpec) (*MetricSetSnapshot, error) {
	switch svc := service.(type) {
	case *playdeveloperreporting.VitalsSlowstartrateService:
		resp, err := svc.Query(spec.name, &playdeveloperreporting.GooglePlayDeveloperReportingV1beta1QuerySlowStartRateMetricSetRequest{
			Metrics:      append([]string(nil), spec.metrics...),
			TimelineSpec: spec.timeline,
			UserCohort:   "OS_PUBLIC",
		}).Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("status %s query failed: %w", spec.request, err)
		}
		return summarizeMetricSet(spec.name, resp.Rows), nil
	case *playdeveloperreporting.VitalsSlowrenderingrateService:
		resp, err := svc.Query(spec.name, &playdeveloperreporting.GooglePlayDeveloperReportingV1beta1QuerySlowRenderingRateMetricSetRequest{
			Metrics:      append([]string(nil), spec.metrics...),
			TimelineSpec: spec.timeline,
			UserCohort:   "OS_PUBLIC",
		}).Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("status %s query failed: %w", spec.request, err)
		}
		return summarizeMetricSet(spec.name, resp.Rows), nil
	case *playdeveloperreporting.VitalsExcessivewakeuprateService:
		resp, err := svc.Query(spec.name, &playdeveloperreporting.GooglePlayDeveloperReportingV1beta1QueryExcessiveWakeupRateMetricSetRequest{
			Metrics:      append([]string(nil), spec.metrics...),
			TimelineSpec: spec.timeline,
			UserCohort:   "OS_PUBLIC",
		}).Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("status %s query failed: %w", spec.request, err)
		}
		return summarizeMetricSet(spec.name, resp.Rows), nil
	case *playdeveloperreporting.VitalsStuckbackgroundwakelockrateService:
		resp, err := svc.Query(spec.name, &playdeveloperreporting.GooglePlayDeveloperReportingV1beta1QueryStuckBackgroundWakelockRateMetricSetRequest{
			Metrics:      append([]string(nil), spec.metrics...),
			TimelineSpec: spec.timeline,
			UserCohort:   "OS_PUBLIC",
		}).Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("status %s query failed: %w", spec.request, err)
		}
		return summarizeMetricSet(spec.name, resp.Rows), nil
	default:
		return nil, fmt.Errorf("unsupported service type %T", service)
	}
}

func summarizeMetricSet(metricSet string, rows []*playdeveloperreporting.GooglePlayDeveloperReportingV1beta1MetricsRow) *MetricSetSnapshot {
	snapshot := &MetricSetSnapshot{
		MetricSet: metricSet,
		Rows:      len(rows),
	}
	if len(rows) == 0 {
		return snapshot
	}
	snapshot.Latest = summarizeRow(rows[len(rows)-1])
	return snapshot
}

func summarizeRow(row *playdeveloperreporting.GooglePlayDeveloperReportingV1beta1MetricsRow) *MetricRowSummary {
	if row == nil {
		return nil
	}
	summary := &MetricRowSummary{
		AggregationPeriod: row.AggregationPeriod,
		Dimensions:        map[string]string{},
		Metrics:           map[string]string{},
	}
	if row.StartTime != nil {
		summary.StartTime = timeDateString(row.StartTime)
	}
	for _, dim := range row.Dimensions {
		if dim == nil || strings.TrimSpace(dim.Dimension) == "" {
			continue
		}
		summary.Dimensions[dim.Dimension] = dimensionValueString(dim)
	}
	for _, metric := range row.Metrics {
		if metric == nil || strings.TrimSpace(metric.Metric) == "" {
			continue
		}
		summary.Metrics[metric.Metric] = metricValueString(metric)
	}
	if len(summary.Dimensions) == 0 {
		summary.Dimensions = nil
	}
	if len(summary.Metrics) == 0 {
		summary.Metrics = nil
	}
	return summary
}

func metricValueString(metric *playdeveloperreporting.GooglePlayDeveloperReportingV1beta1MetricValue) string {
	if metric == nil || metric.DecimalValue == nil {
		return ""
	}
	return metric.DecimalValue.Value
}

func dimensionValueString(dim *playdeveloperreporting.GooglePlayDeveloperReportingV1beta1DimensionValue) string {
	if dim == nil {
		return ""
	}
	if dim.StringValue != "" {
		return dim.StringValue
	}
	if dim.ValueLabel != "" {
		return dim.ValueLabel
	}
	return fmt.Sprintf("%d", dim.Int64Value)
}

func joinVersionCodes(versionCodes []int64) string {
	if len(versionCodes) == 0 {
		return ""
	}
	sorted := append([]int64(nil), versionCodes...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	parts := make([]string, 0, len(sorted))
	for _, versionCode := range sorted {
		parts = append(parts, fmt.Sprintf("%d", versionCode))
	}
	return strings.Join(parts, ",")
}

func currentWindow(now time.Time) TimeWindow {
	end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	start := end.AddDate(0, 0, -vitalsWindowDays)
	return TimeWindow{
		Start: start.Format("2006-01-02"),
		End:   end.Format("2006-01-02"),
	}
}

func (w TimeWindow) timelineSpec() *playdeveloperreporting.GooglePlayDeveloperReportingV1beta1TimelineSpec {
	start := parseDateToGoogleDateTime(w.Start)
	end := parseDateToGoogleDateTime(w.End)
	return &playdeveloperreporting.GooglePlayDeveloperReportingV1beta1TimelineSpec{
		AggregationPeriod: "DAILY",
		StartTime:         start,
		EndTime:           end,
	}
}

func parseDateToGoogleDateTime(value string) *playdeveloperreporting.GoogleTypeDateTime {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil
	}
	return &playdeveloperreporting.GoogleTypeDateTime{
		Year:  int64(parsed.Year()),
		Month: int64(parsed.Month()),
		Day:   int64(parsed.Day()),
	}
}

func metricSetName(pkg, metricSet string) string {
	return fmt.Sprintf("apps/%s/%s", pkg, metricSet)
}

func timeDateString(value *playdeveloperreporting.GoogleTypeDateTime) string {
	if value == nil {
		return ""
	}
	year := int(value.Year)
	month := int(value.Month)
	day := int(value.Day)
	if year == 0 || month == 0 || day == 0 {
		return ""
	}
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
}
