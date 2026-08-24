package cmdtest_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/cmdtest"
)

func TestInsightsWeeklyIsLocalAndDeterministic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "installs.csv")
	content := "Date,Package Name,Country,Daily Device Installs,Daily Device Uninstalls,Daily User Installs,Daily User Uninstalls\n" +
		"2026-08-10,com.example.app,US,1,0,2,0\n" +
		"2026-08-17,com.example.app,US,2,0,4,0\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	result := cmdtest.Run(t, "insights", "weekly", "--package", "com.example.app", "--week", "2026-08-17", "--installs-file", path)
	if result.ExitCode != 0 {
		t.Fatalf("insights weekly failed: stderr=%s", result.Stderr)
	}
	var response struct {
		Source  string `json:"source"`
		Metrics []struct {
			Name    string   `json:"name"`
			Current *float64 `json:"current"`
			Status  string   `json:"status"`
		} `json:"metrics"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
		t.Fatalf("decode response: %v\n%s", err, result.Stdout)
	}
	if response.Source != "official-google-play-csv" {
		t.Fatalf("source = %q", response.Source)
	}
	for _, metric := range response.Metrics {
		if metric.Name == "daily_user_installs" && metric.Status == "available" && metric.Current != nil && *metric.Current == 4 {
			return
		}
	}
	t.Fatalf("daily_user_installs metric missing from %#v", response.Metrics)
}

func TestInsightsWeeklyHelpDocumentsOfficialLocalSources(t *testing.T) {
	result := cmdtest.Run(t, "insights", "weekly", "--help")
	if result.ExitCode != 0 {
		t.Fatalf("help failed: %s", result.Stderr)
	}
	help := result.Stdout + result.Stderr
	for _, want := range []string{"--installs-file", "--crashes-file", "--store-performance-file", "official", "no credentials"} {
		if !strings.Contains(strings.ToLower(help), strings.ToLower(want)) {
			t.Fatalf("help missing %q:\n%s", want, help)
		}
	}
}

func TestInsightsDailyComparesOfficialLocalReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "crashes.csv")
	content := "Date,Package Name,Device,Daily Crashes,Daily ANRs\n" +
		"2026-08-23,com.example.app,pixel,5,2\n" +
		"2026-08-24,com.example.app,pixel,3,1\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	result := cmdtest.Run(t, "insights", "daily", "--package", "com.example.app", "--date", "2026-08-24", "--crashes-file", path)
	if result.ExitCode != 0 || !strings.Contains(result.Stdout, `"previous_date":"2026-08-23"`) {
		t.Fatalf("daily insights failed: stdout=%s stderr=%s", result.Stdout, result.Stderr)
	}
}
