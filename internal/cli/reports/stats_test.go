package reports

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/gcsclient"
)

// --- stats (parent) ---

func TestStatsCommand_NoArgs_ReturnsHelp(t *testing.T) {
	err := execCommand(t, []string{"stats"})
	if err == nil {
		t.Fatal("expected flag.ErrHelp, got nil")
	}
}

// --- stats list ---

func TestStatsList_MissingDeveloper(t *testing.T) {
	err := execCommand(t, []string{"stats", "list"})
	if err == nil {
		t.Fatal("expected error for missing --developer")
	}
	if !strings.Contains(err.Error(), "--developer is required") {
		t.Errorf("expected '--developer is required' error, got: %v", err)
	}
}

func TestStatsList_InvalidFromMonth(t *testing.T) {
	err := execCommand(t, []string{"stats", "list", "--developer", "12345", "--from", "2024-13"})
	if err == nil {
		t.Fatal("expected error for invalid --from month")
	}
	if !strings.Contains(err.Error(), "--from must be in YYYY-MM format") {
		t.Errorf("expected month format error, got: %v", err)
	}
}

func TestStatsList_InvalidToMonth(t *testing.T) {
	err := execCommand(t, []string{"stats", "list", "--developer", "12345", "--to", "bad"})
	if err == nil {
		t.Fatal("expected error for invalid --to month")
	}
	if !strings.Contains(err.Error(), "--to must be in YYYY-MM format") {
		t.Errorf("expected month format error, got: %v", err)
	}
}

func TestStatsList_InvalidType(t *testing.T) {
	err := execCommand(t, []string{"stats", "list", "--developer", "12345", "--type", "unknown"})
	if err == nil {
		t.Fatal("expected error for invalid --type")
	}
	if !strings.Contains(err.Error(), "--type must be one of") {
		t.Errorf("expected type validation error, got: %v", err)
	}
}

func TestStatsList_ValidMinimalFlags(t *testing.T) {
	setupMockGCSEmpty(t)
	err := execCommand(t, []string{"stats", "list", "--developer", "12345"})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestStatsList_ValidAllFlags(t *testing.T) {
	setupMockGCSEmpty(t)
	err := execCommand(t, []string{
		"stats", "list",
		"--developer", "12345",
		"--package", "com.example.app",
		"--from", "2025-01",
		"--to", "2025-06",
		"--type", "installs",
	})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestStatsList_TypeAll(t *testing.T) {
	setupMockGCSEmpty(t)
	err := execCommand(t, []string{
		"stats", "list",
		"--developer", "12345",
		"--type", "all",
	})
	if err != nil {
		t.Errorf("expected no error with --type all, got: %v", err)
	}
}

func TestStatsList_AllStatsTypes(t *testing.T) {
	setupMockGCSEmpty(t)
	types := []string{"installs", "ratings", "crashes", "store_performance", "subscriptions"}
	for _, st := range types {
		err := execCommand(t, []string{
			"stats", "list",
			"--developer", "12345",
			"--type", st,
		})
		if err != nil {
			t.Errorf("expected no error for type %q, got: %v", st, err)
		}
	}
}

func TestStatsList_PrettyWithTable(t *testing.T) {
	err := execCommand(t, []string{
		"stats", "list",
		"--developer", "12345",
		"--output", "table",
		"--pretty",
	})
	if err == nil {
		t.Fatal("expected error for --pretty with table output")
	}
	if !strings.Contains(err.Error(), "--pretty is only valid with JSON output") {
		t.Errorf("expected output flag validation error, got: %v", err)
	}
}

// --- stats download ---

func TestStatsDownload_MissingDeveloper(t *testing.T) {
	err := execCommand(t, []string{"stats", "download", "--package", "com.example.app", "--from", "2025-01", "--type", "installs"})
	if err == nil {
		t.Fatal("expected error for missing --developer")
	}
	if !strings.Contains(err.Error(), "--developer is required") {
		t.Errorf("expected '--developer is required' error, got: %v", err)
	}
}

func TestStatsDownload_MissingPackage(t *testing.T) {
	err := execCommand(t, []string{"stats", "download", "--developer", "12345", "--from", "2025-01", "--type", "installs"})
	if err == nil {
		t.Fatal("expected error for missing --package")
	}
	if !strings.Contains(err.Error(), "--package is required") {
		t.Errorf("expected '--package is required' error, got: %v", err)
	}
}

func TestStatsDownload_MissingFrom(t *testing.T) {
	err := execCommand(t, []string{"stats", "download", "--developer", "12345", "--package", "com.example.app", "--type", "installs"})
	if err == nil {
		t.Fatal("expected error for missing --from")
	}
	if !strings.Contains(err.Error(), "--from is required") {
		t.Errorf("expected '--from is required' error, got: %v", err)
	}
}

func TestStatsDownload_MissingType(t *testing.T) {
	err := execCommand(t, []string{"stats", "download", "--developer", "12345", "--package", "com.example.app", "--from", "2025-01"})
	if err == nil {
		t.Fatal("expected error for missing --type")
	}
	if !strings.Contains(err.Error(), "--type is required") {
		t.Errorf("expected '--type is required' error, got: %v", err)
	}
}

func TestStatsDownload_InvalidFromMonth(t *testing.T) {
	err := execCommand(t, []string{"stats", "download", "--developer", "12345", "--package", "com.example.app", "--from", "2025-00", "--type", "installs"})
	if err == nil {
		t.Fatal("expected error for invalid --from month")
	}
	if !strings.Contains(err.Error(), "--from must be in YYYY-MM format") {
		t.Errorf("expected month format error, got: %v", err)
	}
}

func TestStatsDownload_InvalidToMonth(t *testing.T) {
	err := execCommand(t, []string{"stats", "download", "--developer", "12345", "--package", "com.example.app", "--from", "2025-01", "--to", "2025-13", "--type", "installs"})
	if err == nil {
		t.Fatal("expected error for invalid --to month")
	}
	if !strings.Contains(err.Error(), "--to must be in YYYY-MM format") {
		t.Errorf("expected month format error, got: %v", err)
	}
}

func TestStatsDownload_InvalidType(t *testing.T) {
	err := execCommand(t, []string{"stats", "download", "--developer", "12345", "--package", "com.example.app", "--from", "2025-01", "--type", "bogus"})
	if err == nil {
		t.Fatal("expected error for invalid --type")
	}
	if !strings.Contains(err.Error(), "--type must be one of") {
		t.Errorf("expected type validation error, got: %v", err)
	}
}

func TestStatsDownload_TypeAllNotAllowed(t *testing.T) {
	err := execCommand(t, []string{"stats", "download", "--developer", "12345", "--package", "com.example.app", "--from", "2025-01", "--type", "all"})
	if err == nil {
		t.Fatal("expected error for --type all on download")
	}
	if !strings.Contains(err.Error(), "--type must be one of: installs, ratings, crashes, store_performance, subscriptions") {
		t.Errorf("expected type error for 'all', got: %v", err)
	}
}

func TestStatsDownload_ValidMinimalFlags(t *testing.T) {
	setupMockGCSEmpty(t)
	err := execCommand(t, []string{"stats", "download", "--developer", "12345", "--package", "com.example.app", "--from", "2025-01", "--type", "installs"})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestStatsDownload_ValidAllFlags(t *testing.T) {
	setupMockGCSEmpty(t)
	err := execCommand(t, []string{
		"stats", "download",
		"--developer", "12345",
		"--package", "com.example.app",
		"--from", "2025-01",
		"--to", "2025-06",
		"--type", "crashes",
		"--dir", t.TempDir(),
	})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestStatsDownload_ToDefaultsToFrom(t *testing.T) {
	setupMockGCSEmpty(t)
	err := execCommand(t, []string{
		"stats", "download",
		"--developer", "12345",
		"--package", "com.example.app",
		"--from", "2025-03",
		"--type", "ratings",
	})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestStatsDownload_AllValidTypes(t *testing.T) {
	setupMockGCSEmpty(t)
	types := []string{"installs", "ratings", "crashes", "store_performance", "subscriptions"}
	for _, st := range types {
		err := execCommand(t, []string{
			"stats", "download",
			"--developer", "12345",
			"--package", "com.example.app",
			"--from", "2025-01",
			"--type", st,
		})
		if err != nil {
			t.Errorf("expected no error for type %q, got: %v", st, err)
		}
	}
}

func TestStatsDownload_PrettyWithMarkdown(t *testing.T) {
	err := execCommand(t, []string{
		"stats", "download",
		"--developer", "12345",
		"--package", "com.example.app",
		"--from", "2025-01",
		"--type", "installs",
		"--output", "markdown",
		"--pretty",
	})
	if err == nil {
		t.Fatal("expected error for --pretty with markdown output")
	}
	if !strings.Contains(err.Error(), "--pretty is only valid with JSON output") {
		t.Errorf("expected output flag validation error, got: %v", err)
	}
}

// --- stats type validation unit tests ---

func TestValidateStatsType_Valid(t *testing.T) {
	for _, st := range []string{"installs", "ratings", "crashes", "store_performance", "subscriptions", "all"} {
		if err := validateStatsType(st); err != nil {
			t.Errorf("expected %q to be valid, got: %v", st, err)
		}
	}
}

func TestValidateStatsType_Invalid(t *testing.T) {
	for _, st := range []string{"unknown", "install", "", "INSTALLS", "earning"} {
		if err := validateStatsType(st); err == nil {
			t.Errorf("expected %q to be invalid", st)
		}
	}
}

// --- GCS integration tests (with mock server) ---

func TestStatsList_ReturnsObjects(t *testing.T) {
	objects := map[string][]gcsclient.ObjectInfo{
		"pubsite_prod_rev_55/stats/installs/": {
			{Name: "stats/installs/installs_com.example.app_202501_overview.csv", Size: 512, Updated: "2025-02-01T00:00:00Z"},
			{Name: "stats/installs/installs_com.example.app_202502_overview.csv", Size: 256, Updated: "2025-03-01T00:00:00Z"},
			{Name: "stats/installs/installs_com.other.app_202501_overview.csv", Size: 128, Updated: "2025-02-01T00:00:00Z"},
		},
	}
	setupMockGCS(t, objects, nil)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := execCommand(t, []string{
		"stats", "list",
		"--developer", "55",
		"--package", "com.example.app",
		"--type", "installs",
	})

	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to parse output JSON: %v\noutput: %s", err, out)
	}
	reports := result["reports"].([]interface{})
	// Should only include com.example.app entries (2 of them), not com.other.app
	if len(reports) != 2 {
		t.Errorf("expected 2 reports for com.example.app, got %d: %s", len(reports), out)
	}
}

func TestStatsList_FiltersByDateRange(t *testing.T) {
	objects := map[string][]gcsclient.ObjectInfo{
		"pubsite_prod_rev_55/stats/ratings/": {
			{Name: "stats/ratings/ratings_com.example.app_202501_overview.csv", Size: 100, Updated: "2025-02-01T00:00:00Z"},
			{Name: "stats/ratings/ratings_com.example.app_202506_overview.csv", Size: 200, Updated: "2025-07-01T00:00:00Z"},
			{Name: "stats/ratings/ratings_com.example.app_202512_overview.csv", Size: 300, Updated: "2026-01-01T00:00:00Z"},
		},
	}
	setupMockGCS(t, objects, nil)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := execCommand(t, []string{
		"stats", "list",
		"--developer", "55",
		"--type", "ratings",
		"--from", "2025-04",
		"--to", "2025-09",
	})

	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to parse output JSON: %v", err)
	}
	reports := result["reports"].([]interface{})
	if len(reports) != 1 {
		t.Errorf("expected 1 report (only 202506 in range), got %d: %s", len(reports), out)
	}
}

func TestStatsDownload_WritesFiles(t *testing.T) {
	dir := t.TempDir()
	objects := map[string][]gcsclient.ObjectInfo{
		"pubsite_prod_rev_77/stats/crashes/": {
			{Name: "stats/crashes/crashes_com.example.app_202501_overview.csv", Size: 13, Updated: "2025-02-01T00:00:00Z"},
		},
	}
	fileContents := map[string]string{
		"stats/crashes/crashes_com.example.app_202501_overview.csv": "crash-csv-data",
	}
	setupMockGCS(t, objects, fileContents)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := execCommand(t, []string{
		"stats", "download",
		"--developer", "77",
		"--package", "com.example.app",
		"--from", "2025-01",
		"--type", "crashes",
		"--dir", dir,
	})

	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Check file was written
	content, err := os.ReadFile(filepath.Join(dir, "crashes_com.example.app_202501_overview.csv"))
	if err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}
	if string(content) != "crash-csv-data" {
		t.Errorf("expected file content 'crash-csv-data', got %q", content)
	}

	// Check output JSON
	var result map[string]interface{}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to parse output JSON: %v", err)
	}
	files := result["files"].([]interface{})
	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d", len(files))
	}
}

func TestStatsDownload_FiltersByPackage(t *testing.T) {
	dir := t.TempDir()
	objects := map[string][]gcsclient.ObjectInfo{
		"pubsite_prod_rev_88/stats/installs/": {
			{Name: "stats/installs/installs_com.example.app_202501_overview.csv", Size: 100, Updated: "2025-02-01T00:00:00Z"},
			{Name: "stats/installs/installs_com.other.app_202501_overview.csv", Size: 200, Updated: "2025-02-01T00:00:00Z"},
		},
	}
	fileContents := map[string]string{
		"stats/installs/installs_com.example.app_202501_overview.csv": "example-data",
	}
	setupMockGCS(t, objects, fileContents)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := execCommand(t, []string{
		"stats", "download",
		"--developer", "88",
		"--package", "com.example.app",
		"--from", "2025-01",
		"--type", "installs",
		"--dir", dir,
	})

	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to parse output JSON: %v", err)
	}
	files := result["files"].([]interface{})
	if len(files) != 1 {
		t.Errorf("expected 1 file (only com.example.app), got %d: %s", len(files), out)
	}
}
