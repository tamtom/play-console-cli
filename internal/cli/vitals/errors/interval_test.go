package errors

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tamtom/play-console-cli/internal/reportingclient"
)

func TestBuildSearchInterval_Empty(t *testing.T) {
	interval, err := buildSearchInterval("", "  ")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if interval != nil {
		t.Errorf("expected nil interval when no dates given, got %+v", interval)
	}
}

func TestBuildSearchInterval_BothDates(t *testing.T) {
	interval, err := buildSearchInterval("2025-01-01", "2025-01-31")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if interval == nil {
		t.Fatal("expected interval, got nil")
	}
	wantStart := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	if !interval.start.Equal(wantStart) {
		t.Errorf("expected start %v, got %v", wantStart, interval.start)
	}
	// Inclusive --to date becomes an exclusive next-day bound.
	wantEnd := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	if !interval.end.Equal(wantEnd) {
		t.Errorf("expected exclusive end %v, got %v", wantEnd, interval.end)
	}
}

func TestBuildSearchInterval_FromOnly(t *testing.T) {
	interval, err := buildSearchInterval("2025-01-01", "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if interval == nil {
		t.Fatal("expected interval, got nil")
	}
	if interval.start.IsZero() {
		t.Error("expected start to be set")
	}
	if !interval.end.IsZero() {
		t.Errorf("expected zero end, got %v", interval.end)
	}
}

func TestBuildSearchInterval_InvalidDate(t *testing.T) {
	if _, err := buildSearchInterval("01/02/2025", ""); err == nil || !strings.Contains(err.Error(), "YYYY-MM-DD") {
		t.Errorf("expected YYYY-MM-DD format error, got: %v", err)
	}
	if _, err := buildSearchInterval("", "not-a-date"); err == nil || !strings.Contains(err.Error(), "YYYY-MM-DD") {
		t.Errorf("expected YYYY-MM-DD format error, got: %v", err)
	}
}

func TestBuildSearchInterval_FromAfterTo(t *testing.T) {
	_, err := buildSearchInterval("2025-02-01", "2025-01-01")
	if err == nil || !strings.Contains(err.Error(), "--from must be on or before --to") {
		t.Errorf("expected ordering error, got: %v", err)
	}
}

func installMockErrorsReportingService(t *testing.T, handler http.HandlerFunc) {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	original := newReportingService
	newReportingService = func(ctx context.Context) (*reportingclient.Service, error) {
		return reportingclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
	}
	t.Cleanup(func() {
		newReportingService = original
	})
}

func captureErrorsStdout(fn func() error) (string, error) {
	origStdout := os.Stdout
	rOut, wOut, err := os.Pipe()
	if err != nil {
		return "", err
	}

	os.Stdout = wOut

	var buf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(&buf, rOut)
	}()

	execErr := fn()

	wOut.Close()
	os.Stdout = origStdout
	wg.Wait()

	return buf.String(), execErr
}

func TestIssuesCommand_IntervalQueryParams(t *testing.T) {
	var gotQuery map[string]string
	installMockErrorsReportingService(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = map[string]string{}
		for key, values := range r.URL.Query() {
			gotQuery[key] = values[0]
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"errorIssues":[]}`)
	})

	cmd := IssuesCommand()
	_ = cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--from", "2025-01-01", "--to", "2025-01-31"})
	_, err := captureErrorsStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	want := map[string]string{
		"interval.startTime.year":  "2025",
		"interval.startTime.month": "1",
		"interval.startTime.day":   "1",
		"interval.endTime.year":    "2025",
		"interval.endTime.month":   "2",
		"interval.endTime.day":     "1",
	}
	for key, value := range want {
		if gotQuery[key] != value {
			t.Errorf("expected query param %s=%s, got %q", key, value, gotQuery[key])
		}
	}
}

func TestIssuesCommand_NoIntervalParamsByDefault(t *testing.T) {
	var gotRawQuery string
	installMockErrorsReportingService(t, func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"errorIssues":[]}`)
	})

	cmd := IssuesCommand()
	_ = cmd.FlagSet.Parse([]string{"--package", "com.example.app"})
	_, err := captureErrorsStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if strings.Contains(gotRawQuery, "interval.") {
		t.Errorf("expected no interval params without --from/--to, got query: %s", gotRawQuery)
	}
}

func TestReportsCommand_IntervalQueryParams(t *testing.T) {
	var gotQuery map[string]string
	installMockErrorsReportingService(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = map[string]string{}
		for key, values := range r.URL.Query() {
			gotQuery[key] = values[0]
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"errorReports":[]}`)
	})

	cmd := ReportsCommand()
	_ = cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--from", "2025-06-15", "--to", "2025-06-15"})
	_, err := captureErrorsStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	want := map[string]string{
		"interval.startTime.year":  "2025",
		"interval.startTime.month": "6",
		"interval.startTime.day":   "15",
		"interval.endTime.year":    "2025",
		"interval.endTime.month":   "6",
		"interval.endTime.day":     "16",
	}
	for key, value := range want {
		if gotQuery[key] != value {
			t.Errorf("expected query param %s=%s, got %q", key, value, gotQuery[key])
		}
	}
}

func TestIssuesCommand_InvalidFromDate(t *testing.T) {
	cmd := IssuesCommand()
	_ = cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--from", "bad-date"})
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "YYYY-MM-DD") {
		t.Errorf("expected date format error, got: %v", err)
	}
}

func TestReportsCommand_FromAfterTo(t *testing.T) {
	cmd := ReportsCommand()
	_ = cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--from", "2025-02-01", "--to", "2025-01-01"})
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--from must be on or before --to") {
		t.Errorf("expected ordering error, got: %v", err)
	}
}
