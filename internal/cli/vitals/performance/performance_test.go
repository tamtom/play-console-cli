package performance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/tamtom/play-console-cli/internal/reportingclient"
)

func TestPerformanceCommand_Structure(t *testing.T) {
	cmd := PerformanceCommand()

	if cmd.Name != "performance" {
		t.Errorf("expected command name 'performance', got %q", cmd.Name)
	}

	expected := map[string]bool{
		"startup":   false,
		"rendering": false,
		"battery":   false,
	}

	for _, sub := range cmd.Subcommands {
		if _, ok := expected[sub.Name]; !ok {
			t.Errorf("unexpected subcommand: %q", sub.Name)
		}
		expected[sub.Name] = true
	}

	for name, found := range expected {
		if !found {
			t.Errorf("missing subcommand: %q", name)
		}
	}
}

func TestStartupCommand_Name(t *testing.T) {
	cmd := StartupCommand()
	if cmd.Name != "startup" {
		t.Errorf("expected command name 'startup', got %q", cmd.Name)
	}
}

func TestRenderingCommand_Name(t *testing.T) {
	cmd := RenderingCommand()
	if cmd.Name != "rendering" {
		t.Errorf("expected command name 'rendering', got %q", cmd.Name)
	}
}

func TestBatteryCommand_Name(t *testing.T) {
	cmd := BatteryCommand()
	if cmd.Name != "battery" {
		t.Errorf("expected command name 'battery', got %q", cmd.Name)
	}
}

func TestStartupCommand_PackageRequired(t *testing.T) {
	cmd := StartupCommand()
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when --package is missing")
	}
	if !strings.Contains(err.Error(), "--package is required") {
		t.Errorf("expected '--package is required' error, got: %v", err)
	}
}

func TestRenderingCommand_PackageRequired(t *testing.T) {
	cmd := RenderingCommand()
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when --package is missing")
	}
	if !strings.Contains(err.Error(), "--package is required") {
		t.Errorf("expected '--package is required' error, got: %v", err)
	}
}

func TestBatteryCommand_PackageRequired(t *testing.T) {
	cmd := BatteryCommand()
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when --package is missing")
	}
	if !strings.Contains(err.Error(), "--package is required") {
		t.Errorf("expected '--package is required' error, got: %v", err)
	}
}

func TestBatteryCommand_InvalidType(t *testing.T) {
	cmd := BatteryCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--type", "invalid"}); err != nil {
		t.Fatalf("flag parse error: %v", err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid --type")
	}
	if !strings.Contains(err.Error(), "--type must be 'wakeup' or 'wakelock'") {
		t.Errorf("expected type validation error, got: %v", err)
	}
}

func TestBatteryCommand_ValidType(t *testing.T) {
	for _, tc := range []struct {
		name         string
		metricType   string
		expectedPath string
	}{
		{name: "wakeup", metricType: "wakeup", expectedPath: "/v1beta1/apps/com.example.app/excessiveWakeupRateMetricSet:query"},
		{name: "wakelock", metricType: "wakelock", expectedPath: "/v1beta1/apps/com.example.app/stuckBackgroundWakelockRateMetricSet:query"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests []capturedRequest
			installMockReportingService(t, func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("read request body: %v", err)
				}
				var decoded map[string]interface{}
				if err := json.Unmarshal(body, &decoded); err != nil {
					t.Fatalf("decode request body: %v", err)
				}
				requests = append(requests, capturedRequest{
					Method: r.Method,
					Path:   r.URL.Path,
					Body:   decoded,
				})
				writeJSONResponse(t, w, map[string]interface{}{
					"rows": []map[string]interface{}{
						{
							"metrics": []map[string]interface{}{
								{
									"metric":       "distinctUsers",
									"decimalValue": map[string]string{"value": "100"},
								},
							},
						},
					},
				})
			})

			cmd := BatteryCommand()
			if err := cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--type", tc.metricType}); err != nil {
				t.Fatalf("flag parse error: %v", err)
			}
			stdout, err := captureOutput(func() error {
				return cmd.Exec(context.Background(), nil)
			})
			if err != nil {
				t.Fatalf("expected no error for valid --type %q, got: %v", tc.metricType, err)
			}
			if len(requests) != 1 {
				t.Fatalf("expected 1 request, got %d", len(requests))
			}
			if requests[0].Method != http.MethodPost {
				t.Fatalf("expected POST request, got %s", requests[0].Method)
			}
			if requests[0].Path != tc.expectedPath {
				t.Fatalf("expected request path %q, got %q", tc.expectedPath, requests[0].Path)
			}
			if !strings.Contains(stdout, "distinctUsers") {
				t.Fatalf("expected output to contain metric response, got: %s", stdout)
			}
		})
	}
}

func TestStartupCommand_HelpOutput(t *testing.T) {
	cmd := StartupCommand()
	if cmd.ShortHelp == "" {
		t.Error("expected non-empty ShortHelp")
	}
	if cmd.ShortUsage == "" {
		t.Error("expected non-empty ShortUsage")
	}
	if !strings.Contains(cmd.ShortUsage, "--package") {
		t.Errorf("expected ShortUsage to mention --package, got: %s", cmd.ShortUsage)
	}
}

func TestPerformanceCommand_HelpOutput(t *testing.T) {
	cmd := PerformanceCommand()
	if cmd.ShortHelp == "" {
		t.Error("expected non-empty ShortHelp")
	}
	if !strings.Contains(cmd.ShortUsage, "gplay vitals performance") {
		t.Errorf("expected ShortUsage to contain 'gplay vitals performance', got: %s", cmd.ShortUsage)
	}
}

func TestPerformanceCommand_NoArgsReturnsHelp(t *testing.T) {
	cmd := PerformanceCommand()
	err := cmd.Exec(context.Background(), nil)
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp with no args, got: %v", err)
	}
}

func TestPerformanceCommand_UnknownSubcommand(t *testing.T) {
	cmd := PerformanceCommand()
	var stderr bytes.Buffer
	// Redirect stderr to capture output - can't easily do this without os.Pipe,
	// but we can at least verify the error type.
	_ = stderr
	err := cmd.Exec(context.Background(), []string{"unknown"})
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp for unknown subcommand, got: %v", err)
	}
}

func TestStartupCommand_Flags(t *testing.T) {
	cmd := StartupCommand()
	expectedFlags := []string{"package", "from", "to", "dimension", "output", "pretty", "paginate"}
	for _, name := range expectedFlags {
		if cmd.FlagSet.Lookup(name) == nil {
			t.Errorf("expected flag --%s to be defined", name)
		}
	}
}

func TestRenderingCommand_Flags(t *testing.T) {
	cmd := RenderingCommand()
	expectedFlags := []string{"package", "from", "to", "dimension", "output", "pretty", "paginate"}
	for _, name := range expectedFlags {
		if cmd.FlagSet.Lookup(name) == nil {
			t.Errorf("expected flag --%s to be defined", name)
		}
	}
}

func TestBatteryCommand_Flags(t *testing.T) {
	cmd := BatteryCommand()
	expectedFlags := []string{"package", "from", "to", "dimension", "type", "output", "pretty", "paginate"}
	for _, name := range expectedFlags {
		if cmd.FlagSet.Lookup(name) == nil {
			t.Errorf("expected flag --%s to be defined", name)
		}
	}
}

func TestStartupCommand_PrettyWithTableOutputInvalid(t *testing.T) {
	cmd := StartupCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--output", "table", "--pretty"}); err != nil {
		t.Fatalf("flag parse error: %v", err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for --pretty with table output")
	}
	if !strings.Contains(err.Error(), "--pretty is only valid with JSON output") {
		t.Errorf("expected pretty validation error, got: %v", err)
	}
}

func TestStartupCommand_Success(t *testing.T) {
	var requests []capturedRequest
	installMockReportingService(t, func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		var decoded map[string]interface{}
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		requests = append(requests, capturedRequest{
			Method: r.Method,
			Path:   r.URL.Path,
			Body:   decoded,
		})
		writeJSONResponse(t, w, map[string]interface{}{
			"rows": []map[string]interface{}{
				{
					"metrics": []map[string]interface{}{
						{
							"metric":       "slowStartRate",
							"decimalValue": map[string]string{"value": "0.12"},
						},
					},
				},
			},
		})
	})

	cmd := StartupCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--package", "com.example.app",
		"--from", "2026-03-01",
		"--to", "2026-03-03",
		"--dimension", "apiLevel",
	}); err != nil {
		t.Fatalf("flag parse error: %v", err)
	}
	stdout, err := captureOutput(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}
	if requests[0].Path != "/v1beta1/apps/com.example.app/slowStartRateMetricSet:query" {
		t.Fatalf("unexpected request path: %s", requests[0].Path)
	}
	assertRequestFields(t, requests[0].Body, []string{"slowStartRate", "slowStartRate7dUserWeighted", "slowStartRate28dUserWeighted", "distinctUsers"}, "apiLevel", "2026-03-01", "2026-03-04")
	if !strings.Contains(stdout, "slowStartRate") {
		t.Fatalf("expected output to include slowStartRate response, got: %s", stdout)
	}
}

func TestRenderingCommand_Success(t *testing.T) {
	var requests []capturedRequest
	installMockReportingService(t, func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		var decoded map[string]interface{}
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		requests = append(requests, capturedRequest{
			Method: r.Method,
			Path:   r.URL.Path,
			Body:   decoded,
		})
		writeJSONResponse(t, w, map[string]interface{}{
			"rows": []map[string]interface{}{
				{
					"metrics": []map[string]interface{}{
						{
							"metric":       "slowRenderingRate20Fps",
							"decimalValue": map[string]string{"value": "0.08"},
						},
					},
				},
			},
		})
	})

	cmd := RenderingCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--package", "com.example.app",
		"--dimension", "countryCode",
	}); err != nil {
		t.Fatalf("flag parse error: %v", err)
	}
	stdout, err := captureOutput(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}
	if requests[0].Path != "/v1beta1/apps/com.example.app/slowRenderingRateMetricSet:query" {
		t.Fatalf("unexpected request path: %s", requests[0].Path)
	}
	assertRequestFields(
		t,
		requests[0].Body,
		[]string{
			"slowRenderingRate20Fps",
			"slowRenderingRate20Fps7dUserWeighted",
			"slowRenderingRate20Fps28dUserWeighted",
			"slowRenderingRate30Fps",
			"slowRenderingRate30Fps7dUserWeighted",
			"slowRenderingRate30Fps28dUserWeighted",
			"distinctUsers",
		},
		"countryCode",
		"",
		"",
	)
	if !strings.Contains(stdout, "slowRenderingRate20Fps") {
		t.Fatalf("expected output to include rendering response, got: %s", stdout)
	}
}

type capturedRequest struct {
	Method string
	Path   string
	Body   map[string]interface{}
}

func installMockReportingService(t *testing.T, handler http.HandlerFunc) {
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

func captureOutput(fn func() error) (string, error) {
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

	runErr := fn()

	_ = wOut.Close()
	os.Stdout = origStdout
	wg.Wait()
	_ = rOut.Close()

	return buf.String(), runErr
}

func writeJSONResponse(t *testing.T, w http.ResponseWriter, payload map[string]interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func assertRequestFields(t *testing.T, body map[string]interface{}, expectedMetrics []string, expectedDimension, expectedFrom, expectedTo string) {
	t.Helper()

	rawMetrics, ok := body["metrics"].([]interface{})
	if !ok {
		t.Fatalf("expected metrics array in request body, got: %#v", body["metrics"])
	}
	var metrics []string
	for _, item := range rawMetrics {
		metrics = append(metrics, item.(string))
	}
	for _, metric := range expectedMetrics {
		if !contains(metrics, metric) {
			t.Fatalf("expected metric %q in request body, got %#v", metric, metrics)
		}
	}

	if expectedDimension != "" {
		rawDimensions, ok := body["dimensions"].([]interface{})
		if !ok || len(rawDimensions) != 1 {
			t.Fatalf("expected one dimension in request body, got: %#v", body["dimensions"])
		}
		if got := rawDimensions[0].(string); got != expectedDimension {
			t.Fatalf("expected dimension %q, got %q", expectedDimension, got)
		}
	}

	rawTimeline, ok := body["timelineSpec"].(map[string]interface{})
	if expectedFrom == "" && expectedTo == "" {
		if ok {
			t.Fatalf("expected no timeline spec, got: %#v", rawTimeline)
		}
		return
	}
	if !ok {
		t.Fatalf("expected timelineSpec in request body, got: %#v", body["timelineSpec"])
	}
	if got := rawTimeline["aggregationPeriod"]; got != "DAILY" {
		t.Fatalf("expected DAILY aggregation period, got %#v", got)
	}
	if expectedFrom != "" {
		assertDateTime(t, rawTimeline["startTime"], expectedFrom)
	}
	if expectedTo != "" {
		assertDateTime(t, rawTimeline["endTime"], expectedTo)
	}
}

func assertDateTime(t *testing.T, value interface{}, expected string) {
	t.Helper()

	parts := strings.Split(expected, "-")
	raw, ok := value.(map[string]interface{})
	if !ok {
		t.Fatalf("expected dateTime map, got %#v", value)
	}
	if got := int(raw["year"].(float64)); got != mustAtoi(t, parts[0]) {
		t.Fatalf("expected year %s, got %d", parts[0], got)
	}
	if got := int(raw["month"].(float64)); got != mustAtoi(t, parts[1]) {
		t.Fatalf("expected month %s, got %d", parts[1], got)
	}
	if got := int(raw["day"].(float64)); got != mustAtoi(t, parts[2]) {
		t.Fatalf("expected day %s, got %d", parts[2], got)
	}
}

func mustAtoi(t *testing.T, value string) int {
	t.Helper()
	var parsed int
	if _, err := fmt.Sscanf(value, "%d", &parsed); err != nil {
		t.Fatalf("parse int %q: %v", value, err)
	}
	return parsed
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
