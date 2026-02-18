package reports

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/gcsclient"
)

// mockGCSServer creates an httptest server that simulates GCS list and download APIs.
// objects is a map of "bucket/prefix" → list of objects to return.
func mockGCSServer(t *testing.T, objects map[string][]gcsclient.ObjectInfo, fileContents map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// GCS list: GET /storage/v1/b/{bucket}/o?prefix=...
		if strings.Contains(r.URL.Path, "/storage/v1/b/") && strings.HasSuffix(r.URL.Path, "/o") {
			parts := strings.Split(r.URL.Path, "/")
			// Find bucket name: path is /storage/v1/b/{bucket}/o
			var bucket string
			for i, p := range parts {
				if p == "b" && i+1 < len(parts) {
					bucket = parts[i+1]
					break
				}
			}
			prefix := r.URL.Query().Get("prefix")
			key := bucket + "/" + prefix

			items, ok := objects[key]
			if !ok {
				items = nil
			}

			type gcsObject struct {
				Name    string `json:"name"`
				Size    uint64 `json:"size,string"`
				Updated string `json:"updated"`
			}
			type gcsResponse struct {
				Kind  string      `json:"kind"`
				Items []gcsObject `json:"items"`
			}
			var gcsItems []gcsObject
			for _, obj := range items {
				gcsItems = append(gcsItems, gcsObject{
					Name:    obj.Name,
					Size:    obj.Size,
					Updated: obj.Updated,
				})
			}
			resp := gcsResponse{Kind: "storage#objects", Items: gcsItems}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// GCS download: GET /storage/v1/b/{bucket}/o/{object}?alt=media
		if strings.Contains(r.URL.Path, "/storage/v1/b/") && r.URL.Query().Get("alt") == "media" {
			// Extract object name from path
			// Path format: /storage/v1/b/{bucket}/o/{object}
			idx := strings.Index(r.URL.Path, "/o/")
			if idx >= 0 {
				objectName := r.URL.Path[idx+3:]
				if content, ok := fileContents[objectName]; ok {
					w.Header().Set("Content-Type", "application/octet-stream")
					_, _ = w.Write([]byte(content))
					return
				}
			}
			http.NotFound(w, r)
			return
		}

		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// setupMockGCS configures newGCSServiceFunc to use a mock GCS server.
// Returns a cleanup function.
func setupMockGCS(t *testing.T, objects map[string][]gcsclient.ObjectInfo, fileContents map[string]string) {
	t.Helper()
	srv := mockGCSServer(t, objects, fileContents)
	original := newGCSServiceFunc
	newGCSServiceFunc = func(ctx context.Context) (*gcsclient.Service, error) {
		svc, err := gcsclient.NewServiceWithClient(ctx, srv.Client(), srv.URL+"/storage/v1/")
		if err != nil {
			return nil, fmt.Errorf("mock GCS service: %w", err)
		}
		return svc, nil
	}
	t.Cleanup(func() { newGCSServiceFunc = original })
}

// setupMockGCSEmpty configures newGCSServiceFunc with no objects (for validation tests).
func setupMockGCSEmpty(t *testing.T) {
	t.Helper()
	setupMockGCS(t, nil, nil)
}

// execCommand is a helper that builds and executes a command with the given args.
func execCommand(t *testing.T, args []string) error {
	t.Helper()
	cmd := ReportsCommand()
	return cmd.ParseAndRun(context.Background(), args)
}

// --- reports (parent) ---

func TestReportsCommand_NoArgs_ReturnsHelp(t *testing.T) {
	err := execCommand(t, []string{})
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

func TestReportsCommand_Financial_NoArgs_ReturnsHelp(t *testing.T) {
	err := execCommand(t, []string{"financial"})
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

// --- financial list ---

func TestFinancialList_MissingDeveloper(t *testing.T) {
	err := execCommand(t, []string{"financial", "list"})
	if err == nil {
		t.Fatal("expected error for missing --developer")
	}
	if !strings.Contains(err.Error(), "--developer is required") {
		t.Errorf("expected '--developer is required' error, got: %v", err)
	}
}

func TestFinancialList_InvalidFromMonth(t *testing.T) {
	err := execCommand(t, []string{"financial", "list", "--developer", "12345", "--from", "2024-13"})
	if err == nil {
		t.Fatal("expected error for invalid --from month")
	}
	if !strings.Contains(err.Error(), "--from must be in YYYY-MM format") {
		t.Errorf("expected month format error, got: %v", err)
	}
}

func TestFinancialList_InvalidToMonth(t *testing.T) {
	err := execCommand(t, []string{"financial", "list", "--developer", "12345", "--to", "bad"})
	if err == nil {
		t.Fatal("expected error for invalid --to month")
	}
	if !strings.Contains(err.Error(), "--to must be in YYYY-MM format") {
		t.Errorf("expected month format error, got: %v", err)
	}
}

func TestFinancialList_InvalidType(t *testing.T) {
	err := execCommand(t, []string{"financial", "list", "--developer", "12345", "--type", "unknown"})
	if err == nil {
		t.Fatal("expected error for invalid --type")
	}
	if !strings.Contains(err.Error(), "--type must be one of") {
		t.Errorf("expected type validation error, got: %v", err)
	}
}

func TestFinancialList_ValidMinimalFlags(t *testing.T) {
	setupMockGCSEmpty(t)
	err := execCommand(t, []string{"financial", "list", "--developer", "12345"})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestFinancialList_ValidAllFlags(t *testing.T) {
	setupMockGCSEmpty(t)
	err := execCommand(t, []string{
		"financial", "list",
		"--developer", "12345",
		"--from", "2024-01",
		"--to", "2024-06",
		"--type", "earnings",
	})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestFinancialList_TypeAll(t *testing.T) {
	setupMockGCSEmpty(t)
	err := execCommand(t, []string{
		"financial", "list",
		"--developer", "12345",
		"--type", "all",
	})
	if err != nil {
		t.Errorf("expected no error with --type all, got: %v", err)
	}
}

func TestFinancialList_PrettyWithTable(t *testing.T) {
	err := execCommand(t, []string{
		"financial", "list",
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

// --- financial download ---

func TestFinancialDownload_MissingDeveloper(t *testing.T) {
	err := execCommand(t, []string{"financial", "download", "--from", "2024-01"})
	if err == nil {
		t.Fatal("expected error for missing --developer")
	}
	if !strings.Contains(err.Error(), "--developer is required") {
		t.Errorf("expected '--developer is required' error, got: %v", err)
	}
}

func TestFinancialDownload_MissingFrom(t *testing.T) {
	err := execCommand(t, []string{"financial", "download", "--developer", "12345"})
	if err == nil {
		t.Fatal("expected error for missing --from")
	}
	if !strings.Contains(err.Error(), "--from is required") {
		t.Errorf("expected '--from is required' error, got: %v", err)
	}
}

func TestFinancialDownload_InvalidFromMonth(t *testing.T) {
	err := execCommand(t, []string{"financial", "download", "--developer", "12345", "--from", "2024-00"})
	if err == nil {
		t.Fatal("expected error for invalid --from month")
	}
	if !strings.Contains(err.Error(), "--from must be in YYYY-MM format") {
		t.Errorf("expected month format error, got: %v", err)
	}
}

func TestFinancialDownload_InvalidToMonth(t *testing.T) {
	err := execCommand(t, []string{"financial", "download", "--developer", "12345", "--from", "2024-01", "--to", "2024-13"})
	if err == nil {
		t.Fatal("expected error for invalid --to month")
	}
	if !strings.Contains(err.Error(), "--to must be in YYYY-MM format") {
		t.Errorf("expected month format error, got: %v", err)
	}
}

func TestFinancialDownload_InvalidType(t *testing.T) {
	err := execCommand(t, []string{"financial", "download", "--developer", "12345", "--from", "2024-01", "--type", "bogus"})
	if err == nil {
		t.Fatal("expected error for invalid --type")
	}
	if !strings.Contains(err.Error(), "--type must be one of") {
		t.Errorf("expected type validation error, got: %v", err)
	}
}

func TestFinancialDownload_TypeAllNotAllowed(t *testing.T) {
	err := execCommand(t, []string{"financial", "download", "--developer", "12345", "--from", "2024-01", "--type", "all"})
	if err == nil {
		t.Fatal("expected error for --type all on download")
	}
	if !strings.Contains(err.Error(), "--type must be one of: earnings, sales, payouts") {
		t.Errorf("expected type error for 'all', got: %v", err)
	}
}

func TestFinancialDownload_ValidMinimalFlags(t *testing.T) {
	setupMockGCSEmpty(t)
	err := execCommand(t, []string{"financial", "download", "--developer", "12345", "--from", "2024-01"})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestFinancialDownload_ValidAllFlags(t *testing.T) {
	setupMockGCSEmpty(t)
	err := execCommand(t, []string{
		"financial", "download",
		"--developer", "12345",
		"--from", "2024-01",
		"--to", "2024-06",
		"--type", "sales",
		"--dir", t.TempDir(),
	})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestFinancialDownload_ToDefaultsToFrom(t *testing.T) {
	setupMockGCSEmpty(t)
	err := execCommand(t, []string{
		"financial", "download",
		"--developer", "12345",
		"--from", "2024-03",
		"--type", "payouts",
	})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestFinancialDownload_PrettyWithMarkdown(t *testing.T) {
	err := execCommand(t, []string{
		"financial", "download",
		"--developer", "12345",
		"--from", "2024-01",
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

// --- month validation unit tests ---

func TestValidateMonth_Valid(t *testing.T) {
	cases := []string{"2024-01", "2024-12", "2000-06", "1999-09"}
	for _, c := range cases {
		if err := validateMonth(c, "test"); err != nil {
			t.Errorf("expected %q to be valid, got: %v", c, err)
		}
	}
}

func TestValidateMonth_Invalid(t *testing.T) {
	cases := []string{"2024-00", "2024-13", "24-01", "2024", "2024-1", "bad", ""}
	for _, c := range cases {
		if err := validateMonth(c, "test"); err == nil {
			t.Errorf("expected %q to be invalid", c)
		}
	}
}

// --- report type validation unit tests ---

func TestValidateReportType_Valid(t *testing.T) {
	for _, rt := range []string{"earnings", "sales", "payouts", "all"} {
		if err := validateReportType(rt); err != nil {
			t.Errorf("expected %q to be valid, got: %v", rt, err)
		}
	}
}

func TestValidateReportType_Invalid(t *testing.T) {
	for _, rt := range []string{"unknown", "earning", "", "EARNINGS"} {
		if err := validateReportType(rt); err == nil {
			t.Errorf("expected %q to be invalid", rt)
		}
	}
}

// --- GCS integration tests (with mock server) ---

func TestFinancialList_ReturnsObjects(t *testing.T) {
	objects := map[string][]gcsclient.ObjectInfo{
		"pubsite_prod_rev_12345/earnings/": {
			{Name: "earnings/earnings_202401_12345.zip", Size: 1024, Updated: "2024-02-01T00:00:00Z"},
			{Name: "earnings/earnings_202402_12345.zip", Size: 2048, Updated: "2024-03-01T00:00:00Z"},
			{Name: "earnings/earnings_202406_12345.zip", Size: 512, Updated: "2024-07-01T00:00:00Z"},
		},
	}
	setupMockGCS(t, objects, nil)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := execCommand(t, []string{
		"financial", "list",
		"--developer", "12345",
		"--type", "earnings",
		"--from", "2024-01",
		"--to", "2024-03",
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
	reports, ok := result["reports"].([]interface{})
	if !ok {
		t.Fatalf("expected reports array, got: %T", result["reports"])
	}
	// Should only include 202401 and 202402, not 202406
	if len(reports) != 2 {
		t.Errorf("expected 2 reports, got %d: %s", len(reports), out)
	}
}

func TestFinancialList_AllTypes(t *testing.T) {
	objects := map[string][]gcsclient.ObjectInfo{
		"pubsite_prod_rev_99/earnings/": {
			{Name: "earnings/earnings_202501_99.zip", Size: 100, Updated: "2025-02-01T00:00:00Z"},
		},
		"pubsite_prod_rev_99/sales/": {
			{Name: "sales/salesreport_202501.zip", Size: 200, Updated: "2025-02-01T00:00:00Z"},
		},
		"pubsite_prod_rev_99/payouts/": {
			{Name: "payouts/payout_202501.csv", Size: 50, Updated: "2025-02-01T00:00:00Z"},
		},
	}
	setupMockGCS(t, objects, nil)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := execCommand(t, []string{
		"financial", "list",
		"--developer", "99",
		"--type", "all",
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
	if len(reports) != 3 {
		t.Errorf("expected 3 reports (one per type), got %d: %s", len(reports), out)
	}
}

func TestFinancialDownload_WritesFiles(t *testing.T) {
	dir := t.TempDir()
	objects := map[string][]gcsclient.ObjectInfo{
		"pubsite_prod_rev_42/earnings/": {
			{Name: "earnings/earnings_202401_42.zip", Size: 11, Updated: "2024-02-01T00:00:00Z"},
		},
	}
	fileContents := map[string]string{
		"earnings/earnings_202401_42.zip": "fake-content",
	}
	setupMockGCS(t, objects, fileContents)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := execCommand(t, []string{
		"financial", "download",
		"--developer", "42",
		"--from", "2024-01",
		"--type", "earnings",
		"--dir", dir,
	})

	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Check file was written
	content, err := os.ReadFile(filepath.Join(dir, "earnings_202401_42.zip"))
	if err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}
	if string(content) != "fake-content" {
		t.Errorf("expected file content 'fake-content', got %q", content)
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

func TestFinancialDownload_DateRangeFilters(t *testing.T) {
	dir := t.TempDir()
	objects := map[string][]gcsclient.ObjectInfo{
		"pubsite_prod_rev_10/sales/": {
			{Name: "sales/salesreport_202401.zip", Size: 100, Updated: "2024-02-01T00:00:00Z"},
			{Name: "sales/salesreport_202406.zip", Size: 200, Updated: "2024-07-01T00:00:00Z"},
			{Name: "sales/salesreport_202412.zip", Size: 300, Updated: "2025-01-01T00:00:00Z"},
		},
	}
	fileContents := map[string]string{
		"sales/salesreport_202406.zip": "june-data",
	}
	setupMockGCS(t, objects, fileContents)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := execCommand(t, []string{
		"financial", "download",
		"--developer", "10",
		"--from", "2024-04",
		"--to", "2024-09",
		"--type", "sales",
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
		t.Errorf("expected 1 file (only 202406 in range), got %d: %s", len(files), out)
	}
}

// --- matchesDateRange unit tests ---

func TestMatchesDateRange(t *testing.T) {
	tests := []struct {
		name  string
		file  string
		from  string
		to    string
		match bool
	}{
		{"no range", "earnings_202401.zip", "", "", true},
		{"in range", "earnings_202403.zip", "2024-01", "2024-06", true},
		{"before range", "earnings_202312.zip", "2024-01", "2024-06", false},
		{"after range", "earnings_202407.zip", "2024-01", "2024-06", false},
		{"at start", "earnings_202401.zip", "2024-01", "2024-06", true},
		{"at end", "earnings_202406.zip", "2024-01", "2024-06", true},
		{"only from", "earnings_202406.zip", "2024-01", "", true},
		{"only from excludes", "earnings_202312.zip", "2024-01", "", false},
		{"only to", "earnings_202403.zip", "", "2024-06", true},
		{"only to excludes", "earnings_202407.zip", "", "2024-06", false},
		{"no date in filename", "README.md", "2024-01", "2024-06", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesDateRange(tt.file, tt.from, tt.to)
			if got != tt.match {
				t.Errorf("matchesDateRange(%q, %q, %q) = %v, want %v", tt.file, tt.from, tt.to, got, tt.match)
			}
		})
	}
}
