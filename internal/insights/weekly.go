// Package insights builds deterministic comparisons from official Google Play
// report exports. It performs no network or credential access.
package insights

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	textunicode "golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// WeeklyRequest selects one Monday-to-Sunday window and optional official
// report exports. Each file should contain one breakdown/dimension.
type WeeklyRequest struct {
	Package              string
	Week                 string
	InstallsFile         string
	CrashesFile          string
	StorePerformanceFile string
}

type Range struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type Source struct {
	Type         string `json:"type"`
	Path         string `json:"path,omitempty"`
	CurrentRows  int    `json:"current_rows"`
	PreviousRows int    `json:"previous_rows"`
	Status       string `json:"status"`
	Reason       string `json:"reason,omitempty"`
}

type Metric struct {
	Name         string   `json:"name"`
	Unit         string   `json:"unit"`
	Current      *float64 `json:"current,omitempty"`
	Previous     *float64 `json:"previous,omitempty"`
	Delta        *float64 `json:"delta,omitempty"`
	DeltaPercent *float64 `json:"delta_percent,omitempty"`
	Status       string   `json:"status"`
	Reason       string   `json:"reason,omitempty"`
}

type Report struct {
	Package      string   `json:"package"`
	Source       string   `json:"source"`
	Week         Range    `json:"week"`
	PreviousWeek Range    `json:"previous_week"`
	Sources      []Source `json:"sources"`
	Metrics      []Metric `json:"metrics"`
}

type DailyRequest struct {
	Package              string
	Date                 string
	InstallsFile         string
	CrashesFile          string
	StorePerformanceFile string
}

type DailyReport struct {
	Package      string   `json:"package"`
	Source       string   `json:"source"`
	Date         string   `json:"date"`
	PreviousDate string   `json:"previous_date"`
	Sources      []Source `json:"sources"`
	Metrics      []Metric `json:"metrics"`
}

type metricSpec struct {
	name   string
	header string
	unit   string
}

var installsMetrics = []metricSpec{
	{name: "daily_user_installs", header: "Daily User Installs", unit: "count"},
	{name: "daily_user_uninstalls", header: "Daily User Uninstalls", unit: "count"},
	{name: "daily_device_installs", header: "Daily Device Installs", unit: "count"},
	{name: "daily_device_uninstalls", header: "Daily Device Uninstalls", unit: "count"},
}

var crashMetrics = []metricSpec{
	{name: "daily_crashes", header: "Daily Crashes", unit: "count"},
	{name: "daily_anrs", header: "Daily ANRs", unit: "count"},
}

var storeMetrics = []metricSpec{
	{name: "store_listing_acquisitions", header: "Store listing acquisitions", unit: "count"},
	{name: "store_listing_visitors", header: "Store listing visitors", unit: "count"},
}

// Weekly compares the selected week with the immediately preceding week.
func Weekly(request WeeklyRequest) (Report, error) {
	packageName := strings.TrimSpace(request.Package)
	if packageName == "" {
		return Report{}, fmt.Errorf("package is required")
	}
	weekStart, err := time.Parse("2006-01-02", strings.TrimSpace(request.Week))
	if err != nil {
		return Report{}, fmt.Errorf("week must use YYYY-MM-DD: %w", err)
	}
	if weekStart.Weekday() != time.Monday {
		return Report{}, fmt.Errorf("week must be a Monday")
	}
	weekEnd := weekStart.AddDate(0, 0, 6)
	previousStart := weekStart.AddDate(0, 0, -7)
	report := Report{
		Package: packageName,
		Source:  "official-google-play-csv",
		Week: Range{
			Start: weekStart.Format("2006-01-02"),
			End:   weekEnd.Format("2006-01-02"),
		},
		PreviousWeek: Range{
			Start: previousStart.Format("2006-01-02"),
			End:   weekStart.AddDate(0, 0, -1).Format("2006-01-02"),
		},
	}

	installs, source, err := aggregateSource("installs", request.InstallsFile, packageName, previousStart, weekStart, 7, installsMetrics)
	if err != nil {
		return Report{}, err
	}
	report.Sources = append(report.Sources, source)
	report.Metrics = append(report.Metrics, metricsFromAggregate(installs, installsMetrics)...)

	crashes, source, err := aggregateSource("crashes", request.CrashesFile, packageName, previousStart, weekStart, 7, crashMetrics)
	if err != nil {
		return Report{}, err
	}
	report.Sources = append(report.Sources, source)
	report.Metrics = append(report.Metrics, metricsFromAggregate(crashes, crashMetrics)...)

	store, source, err := aggregateSource("store_performance", request.StorePerformanceFile, packageName, previousStart, weekStart, 7, storeMetrics)
	if err != nil {
		return Report{}, err
	}
	report.Sources = append(report.Sources, source)
	report.Metrics = append(report.Metrics, metricsFromAggregate(store, storeMetrics)...)
	report.Metrics = append(report.Metrics, conversionMetric(store))
	return report, nil
}

// Daily compares one selected UTC report date with the preceding date.
func Daily(request DailyRequest) (DailyReport, error) {
	packageName := strings.TrimSpace(request.Package)
	if packageName == "" {
		return DailyReport{}, fmt.Errorf("package is required")
	}
	date, err := time.Parse("2006-01-02", strings.TrimSpace(request.Date))
	if err != nil {
		return DailyReport{}, fmt.Errorf("date must use YYYY-MM-DD: %w", err)
	}
	previousDate := date.AddDate(0, 0, -1)
	report := DailyReport{
		Package: packageName, Source: "official-google-play-csv",
		Date: date.Format("2006-01-02"), PreviousDate: previousDate.Format("2006-01-02"),
	}
	installs, source, err := aggregateSource("installs", request.InstallsFile, packageName, previousDate, date, 1, installsMetrics)
	if err != nil {
		return DailyReport{}, err
	}
	report.Sources = append(report.Sources, source)
	report.Metrics = append(report.Metrics, metricsFromAggregate(installs, installsMetrics)...)
	crashes, source, err := aggregateSource("crashes", request.CrashesFile, packageName, previousDate, date, 1, crashMetrics)
	if err != nil {
		return DailyReport{}, err
	}
	report.Sources = append(report.Sources, source)
	report.Metrics = append(report.Metrics, metricsFromAggregate(crashes, crashMetrics)...)
	store, source, err := aggregateSource("store_performance", request.StorePerformanceFile, packageName, previousDate, date, 1, storeMetrics)
	if err != nil {
		return DailyReport{}, err
	}
	report.Sources = append(report.Sources, source)
	report.Metrics = append(report.Metrics, metricsFromAggregate(store, storeMetrics)...)
	report.Metrics = append(report.Metrics, conversionMetric(store))
	return report, nil
}

type aggregate struct {
	path         string
	missing      bool
	reason       string
	currentRows  int
	previousRows int
	present      map[string]bool
	current      map[string]float64
	previous     map[string]float64
}

func aggregateSource(sourceType, path, packageName string, previousStart, currentStart time.Time, windowDays int, specs []metricSpec) (aggregate, Source, error) {
	result := aggregate{path: path, present: map[string]bool{}, current: map[string]float64{}, previous: map[string]float64{}}
	source := Source{Type: sourceType, Path: strings.TrimSpace(path)}
	if strings.TrimSpace(path) == "" {
		result.missing = true
		result.reason = sourceType + " report not provided"
		source.Status = "unavailable"
		source.Reason = result.reason
		return result, source, nil
	}

	reader, closeFile, err := openCSV(path)
	if err != nil {
		return aggregate{}, Source{}, fmt.Errorf("open %s report: %w", sourceType, err)
	}
	defer closeFile()
	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = -1
	header, err := csvReader.Read()
	if err != nil {
		return aggregate{}, Source{}, fmt.Errorf("read %s report header: %w", sourceType, err)
	}
	indexes := make(map[string]int, len(header))
	for index, name := range header {
		indexes[normalizeHeader(name)] = index
	}
	dateIndex, ok := indexes[normalizeHeader("Date")]
	if !ok {
		return aggregate{}, Source{}, fmt.Errorf("%s report is missing Date column", sourceType)
	}
	packageIndex, ok := indexes[normalizeHeader("Package Name")]
	if !ok {
		return aggregate{}, Source{}, fmt.Errorf("%s report is missing Package Name column", sourceType)
	}
	for _, spec := range specs {
		_, result.present[spec.name] = indexes[normalizeHeader(spec.header)]
	}

	for rowNumber := 2; ; rowNumber++ {
		row, err := csvReader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return aggregate{}, Source{}, fmt.Errorf("read %s report row %d: %w", sourceType, rowNumber, err)
		}
		if field(row, packageIndex) != packageName {
			continue
		}
		date, err := time.Parse("2006-01-02", field(row, dateIndex))
		if err != nil {
			return aggregate{}, Source{}, fmt.Errorf("%s report row %d has invalid Date: %w", sourceType, rowNumber, err)
		}
		period := ""
		switch {
		case !date.Before(currentStart) && date.Before(currentStart.AddDate(0, 0, windowDays)):
			period = "current"
			result.currentRows++
		case !date.Before(previousStart) && date.Before(currentStart):
			period = "previous"
			result.previousRows++
		default:
			continue
		}
		for _, spec := range specs {
			index, present := indexes[normalizeHeader(spec.header)]
			if !present {
				continue
			}
			value, err := parseNumber(field(row, index))
			if err != nil {
				return aggregate{}, Source{}, fmt.Errorf("%s report row %d column %q: %w", sourceType, rowNumber, spec.header, err)
			}
			if period == "current" {
				result.current[spec.name] += value
			} else {
				result.previous[spec.name] += value
			}
		}
	}

	source.CurrentRows = result.currentRows
	source.PreviousRows = result.previousRows
	if result.currentRows == 0 || result.previousRows == 0 {
		result.reason = "report has no matching package rows in one or both comparison windows"
		source.Status = "unavailable"
		source.Reason = result.reason
	} else {
		source.Status = "available"
	}
	return result, source, nil
}

func metricsFromAggregate(values aggregate, specs []metricSpec) []Metric {
	metrics := make([]Metric, 0, len(specs))
	for _, spec := range specs {
		metric := Metric{Name: spec.name, Unit: spec.unit}
		switch {
		case values.missing || values.reason != "":
			metric.Status = "unavailable"
			metric.Reason = values.reason
		case !values.present[spec.name]:
			metric.Status = "unavailable"
			metric.Reason = fmt.Sprintf("report is missing %q column", spec.header)
		default:
			metric = availableMetric(spec.name, spec.unit, values.current[spec.name], values.previous[spec.name])
		}
		metrics = append(metrics, metric)
	}
	return metrics
}

func conversionMetric(values aggregate) Metric {
	const name = "store_listing_conversion_rate"
	metric := Metric{Name: name, Unit: "ratio"}
	if values.missing || values.reason != "" {
		metric.Status = "unavailable"
		metric.Reason = values.reason
		return metric
	}
	if !values.present["store_listing_acquisitions"] || !values.present["store_listing_visitors"] {
		metric.Status = "unavailable"
		metric.Reason = "conversion requires Store listing acquisitions and Store listing visitors columns"
		return metric
	}
	currentVisitors := values.current["store_listing_visitors"]
	previousVisitors := values.previous["store_listing_visitors"]
	if currentVisitors == 0 || previousVisitors == 0 {
		metric.Status = "unavailable"
		metric.Reason = "conversion rate is unavailable when either visitor total is zero"
		return metric
	}
	return availableMetric(name, "ratio", values.current["store_listing_acquisitions"]/currentVisitors, values.previous["store_listing_acquisitions"]/previousVisitors)
}

func availableMetric(name, unit string, current, previous float64) Metric {
	delta := current - previous
	metric := Metric{Name: name, Unit: unit, Current: pointer(current), Previous: pointer(previous), Delta: pointer(delta), Status: "available"}
	if previous != 0 {
		metric.DeltaPercent = pointer(delta / previous * 100)
	}
	return metric
}

func pointer(value float64) *float64 { return &value }

func openCSV(path string) (io.Reader, func(), error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, func() {}, err
	}
	buffered := bufio.NewReader(file)
	prefix, _ := buffered.Peek(3)
	var reader io.Reader = buffered
	switch {
	case len(prefix) >= 2 && prefix[0] == 0xff && prefix[1] == 0xfe:
		_, _ = buffered.Discard(2)
		reader = transform.NewReader(buffered, textunicode.UTF16(textunicode.LittleEndian, textunicode.IgnoreBOM).NewDecoder())
	case len(prefix) >= 2 && prefix[0] == 0xfe && prefix[1] == 0xff:
		_, _ = buffered.Discard(2)
		reader = transform.NewReader(buffered, textunicode.UTF16(textunicode.BigEndian, textunicode.IgnoreBOM).NewDecoder())
	case len(prefix) >= 3 && prefix[0] == 0xef && prefix[1] == 0xbb && prefix[2] == 0xbf:
		_, _ = buffered.Discard(3)
	}
	return reader, func() { _ = file.Close() }, nil
}

func normalizeHeader(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, strings.TrimSpace(value))
}

func field(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[index])
}

func parseNumber(value string) (float64, error) {
	value = strings.ReplaceAll(strings.TrimSpace(value), ",", "")
	if value == "" {
		return 0, fmt.Errorf("numeric value is empty")
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric value %q", value)
	}
	return number, nil
}
