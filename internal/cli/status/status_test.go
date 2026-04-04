package status

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/playdeveloperreporting/v1beta1"

	"github.com/tamtom/play-console-cli/internal/config"
	"github.com/tamtom/play-console-cli/internal/playclient"
	"github.com/tamtom/play-console-cli/internal/reportingclient"
)

func TestBuildStatusReport_Success(t *testing.T) {
	originalPlayFactory := newPlayService
	originalReportingFactory := newReportingService
	originalNow := nowFunc
	t.Cleanup(func() {
		newPlayService = originalPlayFactory
		newReportingService = originalReportingFactory
		nowFunc = originalNow
	})

	playServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/edits"):
			writeJSON(t, w, map[string]any{"id": "edit-123"})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/edits/edit-123/tracks"):
			writeJSON(t, w, map[string]any{
				"tracks": []any{
					map[string]any{
						"track": "production",
						"releases": []any{
							map[string]any{
								"status":       "inProgress",
								"versionCodes": []any{"42"},
								"userFraction": 0.2,
							},
						},
					},
				},
			})
		case r.Method == http.MethodDelete && strings.HasSuffix(r.URL.Path, "/edits/edit-123"):
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected play request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(playServer.Close)

	reportingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/slowStartRateMetricSet:query"):
			writeJSON(t, w, map[string]any{
				"rows": []any{
					map[string]any{
						"aggregationPeriod": "DAILY",
						"startTime":         map[string]any{"year": 2026, "month": 4, "day": 3},
						"metrics": []any{
							map[string]any{"metric": "slowStartRate", "decimalValue": map[string]any{"value": "0.12"}},
						},
					},
				},
			})
		case strings.HasSuffix(r.URL.Path, "/slowRenderingRateMetricSet:query"):
			writeJSON(t, w, map[string]any{
				"rows": []any{
					map[string]any{
						"aggregationPeriod": "DAILY",
						"startTime":         map[string]any{"year": 2026, "month": 4, "day": 3},
						"metrics": []any{
							map[string]any{"metric": "slowRenderingRate20Fps", "decimalValue": map[string]any{"value": "0.08"}},
						},
					},
				},
			})
		case strings.HasSuffix(r.URL.Path, "/excessiveWakeupRateMetricSet:query"):
			writeJSON(t, w, map[string]any{
				"rows": []any{
					map[string]any{
						"aggregationPeriod": "DAILY",
						"startTime":         map[string]any{"year": 2026, "month": 4, "day": 3},
						"metrics": []any{
							map[string]any{"metric": "excessiveWakeupRate", "decimalValue": map[string]any{"value": "0.02"}},
						},
					},
				},
			})
		case strings.HasSuffix(r.URL.Path, "/stuckBackgroundWakelockRateMetricSet:query"):
			writeJSON(t, w, map[string]any{
				"rows": []any{
					map[string]any{
						"aggregationPeriod": "DAILY",
						"startTime":         map[string]any{"year": 2026, "month": 4, "day": 3},
						"metrics": []any{
							map[string]any{"metric": "stuckBgWakelockRate", "decimalValue": map[string]any{"value": "0.03"}},
						},
					},
				},
			})
		default:
			t.Fatalf("unexpected reporting request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(reportingServer.Close)

	newPlayService = func(context.Context) (*playclient.Service, error) {
		api, err := androidpublisher.NewService(context.Background(), option.WithHTTPClient(playServer.Client()))
		if err != nil {
			return nil, err
		}
		api.BasePath = playServer.URL + "/"
		return &playclient.Service{API: api, Cfg: &config.Config{}}, nil
	}
	newReportingService = func(context.Context) (*reportingclient.Service, error) {
		api, err := playdeveloperreporting.NewService(context.Background(), option.WithHTTPClient(reportingServer.Client()))
		if err != nil {
			return nil, err
		}
		api.BasePath = reportingServer.URL + "/"
		return &reportingclient.Service{API: api, Cfg: &config.Config{}}, nil
	}
	nowFunc = func() time.Time { return time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC) }

	report, err := buildStatusReport(context.Background(), statusOptions{packageName: "com.example.app"})
	if err != nil {
		t.Fatalf("buildStatusReport: %v", err)
	}
	if report.Package != "com.example.app" {
		t.Fatalf("package = %q", report.Package)
	}
	if report.Status != "ok" {
		t.Fatalf("status = %q", report.Status)
	}
	if len(report.Sources) != 2 || !report.Sources[0].OK || !report.Sources[1].OK {
		t.Fatalf("unexpected sources: %#v", report.Sources)
	}
	if report.Tracks == nil || len(report.Tracks.Tracks) != 1 {
		t.Fatalf("unexpected tracks snapshot: %#v", report.Tracks)
	}
	if report.Vitals == nil || report.Vitals.Startup == nil || report.Vitals.Battery == nil {
		t.Fatalf("unexpected vitals snapshot: %#v", report.Vitals)
	}
	if got := report.Vitals.Window.Start; got != "2026-03-28" {
		t.Fatalf("window start = %q", got)
	}
}

func TestBuildStatusReport_PartialFailure(t *testing.T) {
	originalPlayFactory := newPlayService
	originalReportingFactory := newReportingService
	originalNow := nowFunc
	t.Cleanup(func() {
		newPlayService = originalPlayFactory
		newReportingService = originalReportingFactory
		nowFunc = originalNow
	})

	newPlayService = func(context.Context) (*playclient.Service, error) {
		return nil, fmt.Errorf("play unavailable")
	}
	reportingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/slowStartRateMetricSet:query"),
			strings.HasSuffix(r.URL.Path, "/slowRenderingRateMetricSet:query"),
			strings.HasSuffix(r.URL.Path, "/excessiveWakeupRateMetricSet:query"),
			strings.HasSuffix(r.URL.Path, "/stuckBackgroundWakelockRateMetricSet:query"):
			writeJSON(t, w, map[string]any{
				"rows": []any{
					map[string]any{
						"aggregationPeriod": "DAILY",
						"startTime":         map[string]any{"year": 2026, "month": 4, "day": 3},
						"metrics": []any{
							map[string]any{"metric": "distinctUsers", "decimalValue": map[string]any{"value": "100"}},
						},
					},
				},
			})
		default:
			t.Fatalf("unexpected reporting request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(reportingServer.Close)
	newReportingService = func(context.Context) (*reportingclient.Service, error) {
		api, err := playdeveloperreporting.NewService(context.Background(), option.WithHTTPClient(reportingServer.Client()))
		if err != nil {
			return nil, err
		}
		api.BasePath = reportingServer.URL + "/"
		return &reportingclient.Service{API: api, Cfg: &config.Config{}}, nil
	}
	nowFunc = func() time.Time { return time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC) }

	report, err := buildStatusReport(context.Background(), statusOptions{packageName: "com.example.app"})
	if err != nil {
		t.Fatalf("buildStatusReport: %v", err)
	}
	if report.Status != "degraded" {
		t.Fatalf("status = %q", report.Status)
	}
	if report.Sources[0].OK || report.Sources[0].Error == "" {
		t.Fatalf("expected play source error, got %#v", report.Sources[0])
	}
	if !report.Sources[1].OK || report.Vitals == nil {
		t.Fatalf("expected vitals source to succeed, got %#v", report.Sources[1])
	}
}

func TestRun_WatchPollsUntilCancelled(t *testing.T) {
	originalPlayFactory := newPlayService
	originalReportingFactory := newReportingService
	originalNow := nowFunc
	originalAfter := afterFunc
	t.Cleanup(func() {
		newPlayService = originalPlayFactory
		newReportingService = originalReportingFactory
		nowFunc = originalNow
		afterFunc = originalAfter
	})

	var mu sync.Mutex
	calls := 0
	ctx, cancel := context.WithCancel(context.Background())

	playServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/edits"):
			writeJSON(t, w, map[string]any{"id": "edit-123"})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/edits/edit-123/tracks"):
			writeJSON(t, w, map[string]any{
				"tracks": []any{
					map[string]any{
						"track": "production",
						"releases": []any{
							map[string]any{
								"status":       "completed",
								"versionCodes": []any{"42"},
							},
						},
					},
				},
			})
		case r.Method == http.MethodDelete && strings.HasSuffix(r.URL.Path, "/edits/edit-123"):
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected play request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(playServer.Close)
	reportingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{
			"rows": []any{
				map[string]any{
					"aggregationPeriod": "DAILY",
					"startTime":         map[string]any{"year": 2026, "month": 4, "day": 3},
					"metrics": []any{
						map[string]any{"metric": "distinctUsers", "decimalValue": map[string]any{"value": "100"}},
					},
				},
			},
		})
	}))
	t.Cleanup(reportingServer.Close)

	newPlayService = func(context.Context) (*playclient.Service, error) {
		mu.Lock()
		defer mu.Unlock()
		calls++
		if calls == 2 {
			cancel()
		}
		api, err := androidpublisher.NewService(context.Background(), option.WithHTTPClient(playServer.Client()))
		if err != nil {
			return nil, err
		}
		api.BasePath = playServer.URL + "/"
		return &playclient.Service{API: api, Cfg: &config.Config{}}, nil
	}
	newReportingService = func(context.Context) (*reportingclient.Service, error) {
		api, err := playdeveloperreporting.NewService(context.Background(), option.WithHTTPClient(reportingServer.Client()))
		if err != nil {
			return nil, err
		}
		api.BasePath = reportingServer.URL + "/"
		return &reportingclient.Service{API: api, Cfg: &config.Config{}}, nil
	}
	nowFunc = func() time.Time { return time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC) }
	afterFunc = func(time.Duration) <-chan time.Time {
		ch := make(chan time.Time, 1)
		ch <- time.Now()
		return ch
	}

	output := captureStatusOutput(t, func() error {
		return run(ctx, statusOptions{
			packageName:  "com.example.app",
			watch:        true,
			pollInterval: time.Nanosecond,
		})
	})
	if got := strings.Count(output, "\"generated_at\""); got < 2 {
		t.Fatalf("expected at least 2 status snapshots, got %d\noutput: %s", got, output)
	}
}

func captureStatusOutput(t *testing.T, fn func() error) string {
	t.Helper()

	origStdout := os.Stdout
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = wOut

	var buf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(&buf, rOut)
	}()

	runErr := fn()

	_ = wOut.Close()
	os.Stdout = origStdout
	wg.Wait()
	_ = rOut.Close()

	if runErr != nil && !errors.Is(runErr, context.Canceled) && !errors.Is(runErr, context.DeadlineExceeded) {
		t.Fatalf("run returned error: %v", runErr)
	}
	return buf.String()
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload map[string]any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("write json: %v", err)
	}
}
