package insights

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf16"
)

func TestWeeklyAggregatesOfficialStatisticsReports(t *testing.T) {
	dir := t.TempDir()
	installs := writeFixture(t, dir, "installs.csv", `Date,Package Name,Country,Daily Device Installs,Daily Device Uninstalls,Daily User Installs,Daily User Uninstalls
2026-08-10,com.example.app,US,6,2,5,1
2026-08-16,com.example.app,GB,8,1,7,1
2026-08-17,com.example.app,US,12,3,10,2
2026-08-23,com.example.app,GB,15,2,14,1
2026-08-23,com.other.app,US,999,999,999,999
`)
	crashes := writeFixture(t, dir, "crashes.csv", `Date,Package Name,Device,Daily Crashes,Daily ANRs
2026-08-10,com.example.app,pixel,3,1
2026-08-16,com.example.app,pixel,2,0
2026-08-17,com.example.app,pixel,1,0
2026-08-23,com.example.app,pixel,2,1
`)
	store := writeFixture(t, dir, "store.csv", `Date,Package name,Country,Store listing acquisitions,Store listing visitors,Store listing conversion rate
2026-08-10,com.example.app,US,20,100,0.2
2026-08-16,com.example.app,GB,30,100,0.3
2026-08-17,com.example.app,US,40,100,0.4
2026-08-23,com.example.app,GB,50,100,0.5
`)

	report, err := Weekly(WeeklyRequest{
		Package: "com.example.app", Week: "2026-08-17",
		InstallsFile: installs, CrashesFile: crashes, StorePerformanceFile: store,
	})
	if err != nil {
		t.Fatalf("Weekly: %v", err)
	}
	if report.Week.Start != "2026-08-17" || report.Week.End != "2026-08-23" {
		t.Fatalf("week = %#v", report.Week)
	}
	assertMetric(t, report, "daily_user_installs", 24, 12, 12, 100)
	assertMetric(t, report, "daily_user_uninstalls", 3, 2, 1, 50)
	assertMetric(t, report, "daily_device_installs", 27, 14, 13, 92.85714285714286)
	assertMetric(t, report, "daily_crashes", 3, 5, -2, -40)
	assertMetric(t, report, "daily_anrs", 1, 1, 0, 0)
	assertMetric(t, report, "store_listing_acquisitions", 90, 50, 40, 80)
	assertMetric(t, report, "store_listing_conversion_rate", 0.45, 0.25, 0.2, 80)
}

func TestWeeklyMarksMissingSourcesUnavailable(t *testing.T) {
	report, err := Weekly(WeeklyRequest{Package: "com.example.app", Week: "2026-08-17"})
	if err != nil {
		t.Fatalf("Weekly: %v", err)
	}
	for _, metric := range report.Metrics {
		if metric.Status != "unavailable" || metric.Reason == "" {
			t.Fatalf("metric %#v should be explicitly unavailable", metric)
		}
	}
}

func TestWeeklyReadsOfficialUTF16CSVExports(t *testing.T) {
	text := "Date,Package Name,Country,Daily Device Installs,Daily Device Uninstalls,Daily User Installs,Daily User Uninstalls\n2026-08-10,com.example.app,US,1,0,2,0\n2026-08-17,com.example.app,US,2,0,4,0\n"
	units := utf16.Encode([]rune(text))
	data := make([]byte, 2+len(units)*2)
	data[0], data[1] = 0xff, 0xfe
	for i, unit := range units {
		binary.LittleEndian.PutUint16(data[2+i*2:], unit)
	}
	path := filepath.Join(t.TempDir(), "installs-utf16.csv")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write UTF-16 fixture: %v", err)
	}
	report, err := Weekly(WeeklyRequest{Package: "com.example.app", Week: "2026-08-17", InstallsFile: path})
	if err != nil {
		t.Fatalf("Weekly: %v", err)
	}
	assertMetric(t, report, "daily_user_installs", 4, 2, 2, 100)
}

func TestDailyComparesSelectedDateWithPreviousDate(t *testing.T) {
	path := writeFixture(t, t.TempDir(), "crashes.csv", `Date,Package Name,Device,Daily Crashes,Daily ANRs
2026-08-23,com.example.app,pixel,5,2
2026-08-24,com.example.app,pixel,3,1
`)
	report, err := Daily(DailyRequest{Package: "com.example.app", Date: "2026-08-24", CrashesFile: path})
	if err != nil {
		t.Fatalf("Daily: %v", err)
	}
	if report.Date != "2026-08-24" || report.PreviousDate != "2026-08-23" {
		t.Fatalf("dates = %s / %s", report.Date, report.PreviousDate)
	}
	assertDailyMetric(t, report, "daily_crashes", 3, 5, -2, -40)
}

func writeFixture(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func assertMetric(t *testing.T, report Report, name string, current, previous, delta, percent float64) {
	t.Helper()
	for _, metric := range report.Metrics {
		if metric.Name != name {
			continue
		}
		if metric.Status != "available" || metric.Current == nil || metric.Previous == nil || metric.Delta == nil || metric.DeltaPercent == nil {
			t.Fatalf("metric %s incomplete: %#v", name, metric)
		}
		if *metric.Current != current || *metric.Previous != previous || *metric.Delta != delta || *metric.DeltaPercent != percent {
			t.Fatalf("metric %s = current=%v previous=%v delta=%v percent=%v", name, *metric.Current, *metric.Previous, *metric.Delta, *metric.DeltaPercent)
		}
		return
	}
	t.Fatalf("metric %s not found", name)
}

func assertDailyMetric(t *testing.T, report DailyReport, name string, current, previous, delta, percent float64) {
	t.Helper()
	for _, metric := range report.Metrics {
		if metric.Name != name {
			continue
		}
		if metric.Current == nil || metric.Previous == nil || metric.Delta == nil || metric.DeltaPercent == nil {
			t.Fatalf("metric %s incomplete: %#v", name, metric)
		}
		if *metric.Current != current || *metric.Previous != previous || *metric.Delta != delta || *metric.DeltaPercent != percent {
			t.Fatalf("metric %s values = %#v", name, metric)
		}
		return
	}
	t.Fatalf("metric %s not found", name)
}
