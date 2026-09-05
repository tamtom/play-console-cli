package errors

import (
	"fmt"
	"strings"
	"time"

	"google.golang.org/api/playdeveloperreporting/v1beta1"

	"github.com/tamtom/play-console-cli/internal/reportingclient"
)

var newReportingService = reportingclient.NewService

// searchInterval holds UTC bounds for an error search. The end bound is
// already converted to the exclusive midnight after the inclusive --to date.
type searchInterval struct {
	start time.Time
	end   time.Time
}

// buildSearchInterval parses --from/--to (YYYY-MM-DD, interpreted as UTC)
// into a searchInterval. Returns nil when neither flag is set. The API
// requires hour-aligned UTC bounds; midnight date bounds satisfy that.
func buildSearchInterval(from, to string) (*searchInterval, error) {
	start, err := parseSearchDateFlag("--from", from)
	if err != nil {
		return nil, err
	}
	end, err := parseSearchDateFlag("--to", to)
	if err != nil {
		return nil, err
	}
	if !start.IsZero() && !end.IsZero() && start.After(end) {
		return nil, fmt.Errorf("--from must be on or before --to")
	}
	if start.IsZero() && end.IsZero() {
		return nil, nil
	}
	interval := &searchInterval{start: start}
	if !end.IsZero() {
		interval.end = end.AddDate(0, 0, 1)
	}
	return interval, nil
}

func parseSearchDateFlag(flagName, value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid %s date %q: expected YYYY-MM-DD", flagName, value)
	}
	return parsed, nil
}

func applyIssuesInterval(call *playdeveloperreporting.VitalsErrorsIssuesSearchCall, interval *searchInterval) *playdeveloperreporting.VitalsErrorsIssuesSearchCall {
	if interval == nil {
		return call
	}
	if !interval.start.IsZero() {
		call = call.
			IntervalStartTimeYear(int64(interval.start.Year())).
			IntervalStartTimeMonth(int64(interval.start.Month())).
			IntervalStartTimeDay(int64(interval.start.Day()))
	}
	if !interval.end.IsZero() {
		call = call.
			IntervalEndTimeYear(int64(interval.end.Year())).
			IntervalEndTimeMonth(int64(interval.end.Month())).
			IntervalEndTimeDay(int64(interval.end.Day()))
	}
	return call
}

func applyReportsInterval(call *playdeveloperreporting.VitalsErrorsReportsSearchCall, interval *searchInterval) *playdeveloperreporting.VitalsErrorsReportsSearchCall {
	if interval == nil {
		return call
	}
	if !interval.start.IsZero() {
		call = call.
			IntervalStartTimeYear(int64(interval.start.Year())).
			IntervalStartTimeMonth(int64(interval.start.Month())).
			IntervalStartTimeDay(int64(interval.start.Day()))
	}
	if !interval.end.IsZero() {
		call = call.
			IntervalEndTimeYear(int64(interval.end.Year())).
			IntervalEndTimeMonth(int64(interval.end.Month())).
			IntervalEndTimeDay(int64(interval.end.Day()))
	}
	return call
}
