package vitals

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

func TestVitalsCommandName(t *testing.T) {
	cmd := VitalsCommand()
	if cmd.Name != "vitals" {
		t.Errorf("expected command name 'vitals', got %q", cmd.Name)
	}
}

func TestVitalsCommandHasSubcommands(t *testing.T) {
	cmd := VitalsCommand()
	if len(cmd.Subcommands) == 0 {
		t.Fatal("expected vitals command to have subcommands")
	}
	found := false
	for _, sub := range cmd.Subcommands {
		if sub.Name == "crashes" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected vitals to have 'crashes' subcommand")
	}
}

func TestCrashesCommandHasSubcommands(t *testing.T) {
	cmd := CrashesCommand()
	if cmd.Name != "crashes" {
		t.Errorf("expected command name 'crashes', got %q", cmd.Name)
	}
	expectedSubs := map[string]bool{"query": false, "anomalies": false}
	for _, sub := range cmd.Subcommands {
		if _, ok := expectedSubs[sub.Name]; ok {
			expectedSubs[sub.Name] = true
		}
	}
	for name, found := range expectedSubs {
		if !found {
			t.Errorf("expected crashes to have %q subcommand", name)
		}
	}
}

func TestVitalsNoArgsReturnsHelp(t *testing.T) {
	cmd := VitalsCommand()
	err := cmd.Exec(context.Background(), nil)
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

func TestCrashesNoArgsReturnsHelp(t *testing.T) {
	cmd := CrashesCommand()
	err := cmd.Exec(context.Background(), nil)
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

func TestCrashesQueryRequiresPackage(t *testing.T) {
	cmd := CrashesQueryCommand()
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when --package is missing")
	}
	if !strings.Contains(err.Error(), "--package is required") {
		t.Errorf("expected '--package is required' error, got: %v", err)
	}
}

func TestCrashesQueryInvalidType(t *testing.T) {
	cmd := CrashesQueryCommand()
	_ = cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--type", "invalid"})
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid --type")
	}
	if !strings.Contains(err.Error(), "--type must be") {
		t.Errorf("expected '--type must be' error, got: %v", err)
	}
}

func TestCrashesQueryValidTypeCrash(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}
	installMockVitalsReportingService(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"rows":[{"metrics":[{"metric":"crashRate","decimalValue":{"value":"0.02"}}]}]}`)
	})

	cmd := CrashesQueryCommand()
	_ = cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--type", "crash", "--dimension", "versionCode"})
	stdout, err := captureVitalsStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if gotPath != "/v1beta1/apps/com.example.app/crashRateMetricSet:query" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if !requestListContains(gotBody["metrics"], "crashRate") {
		t.Fatalf("expected crashRate metric in body: %#v", gotBody["metrics"])
	}
	if !requestListContains(gotBody["dimensions"], "versionCode") {
		t.Fatalf("expected versionCode dimension in body: %#v", gotBody["dimensions"])
	}
	if !strings.Contains(stdout, "crashRate") {
		t.Fatalf("expected response in output, got %s", stdout)
	}
}

func TestCrashesQueryValidTypeANR(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}
	installMockVitalsReportingService(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"rows":[{"metrics":[{"metric":"anrRate","decimalValue":{"value":"0.01"}}]}]}`)
	})

	cmd := CrashesQueryCommand()
	_ = cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--type", "anr"})
	_, err := captureVitalsStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if gotPath != "/v1beta1/apps/com.example.app/anrRateMetricSet:query" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if !requestListContains(gotBody["metrics"], "anrRate") {
		t.Fatalf("expected anrRate metric in body: %#v", gotBody["metrics"])
	}
}

func TestCrashesQueryInvalidFromDate(t *testing.T) {
	cmd := CrashesQueryCommand()
	_ = cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--from", "not-a-date"})
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid --from date")
	}
	if !strings.Contains(err.Error(), "--from") {
		t.Errorf("expected error to mention --from, got: %v", err)
	}
}

func TestCrashesQueryInvalidToDate(t *testing.T) {
	cmd := CrashesQueryCommand()
	_ = cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--to", "2025/01/01"})
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid --to date")
	}
	if !strings.Contains(err.Error(), "--to") {
		t.Errorf("expected error to mention --to, got: %v", err)
	}
}

func TestCrashesQueryValidDates(t *testing.T) {
	var gotBody map[string]interface{}
	installMockVitalsReportingService(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"rows":[]}`)
	})

	cmd := CrashesQueryCommand()
	_ = cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--from", "2025-01-01", "--to", "2025-01-31"})
	_, err := captureVitalsStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	timeline, ok := gotBody["timelineSpec"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected timelineSpec in request, got %#v", gotBody)
	}
	if got := timelineDate(timeline["startTime"]); got != "2025-1-1" {
		t.Fatalf("startTime = %s, want 2025-1-1", got)
	}
	if got := timelineDate(timeline["endTime"]); got != "2025-2-1" {
		t.Fatalf("endTime = %s, want exclusive 2025-2-1", got)
	}
}

func TestAnomaliesRequiresPackage(t *testing.T) {
	cmd := AnomaliesCommand()
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when --package is missing")
	}
	if !strings.Contains(err.Error(), "--package is required") {
		t.Errorf("expected '--package is required' error, got: %v", err)
	}
}

func TestAnomaliesCallsReportingAPI(t *testing.T) {
	var gotPath, gotFilter, gotPageSize string
	installMockVitalsReportingService(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotFilter = r.URL.Query().Get("filter")
		gotPageSize = r.URL.Query().Get("pageSize")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"anomalies":[{"name":"apps/com.example.app/anomalies/1","metricSet":"apps/com.example.app/crashRateMetricSet"}]}`)
	})

	cmd := AnomaliesCommand()
	_ = cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--type", "crash", "--from", "2026-03-01", "--to", "2026-03-31", "--limit", "10"})
	stdout, err := captureVitalsStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotPath != "/v1beta1/apps/com.example.app/anomalies" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if gotPageSize != "10" {
		t.Fatalf("pageSize = %q, want 10", gotPageSize)
	}
	if !strings.Contains(gotFilter, `activeBetween("2026-03-01T00:00:00Z", "2026-04-01T00:00:00Z")`) {
		t.Fatalf("unexpected filter: %s", gotFilter)
	}
	if !strings.Contains(stdout, "crashRateMetricSet") {
		t.Fatalf("expected anomaly in output, got %s", stdout)
	}
}

func TestValidateISO8601Date(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"2025-01-01", false},
		{"2025-12-31", false},
		{"not-a-date", true},
		{"2025/01/01", true},
		{"25-01-01", true},
		{"2025-1-1", true},
		{"", true},
		{"2025-01-0a", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := validateISO8601Date(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateISO8601Date(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestCrashesQueryHelpOutput(t *testing.T) {
	cmd := CrashesQueryCommand()
	if cmd.ShortHelp == "" {
		t.Error("expected CrashesQueryCommand to have ShortHelp")
	}
	if cmd.LongHelp == "" {
		t.Error("expected CrashesQueryCommand to have LongHelp")
	}
	if !strings.Contains(cmd.LongHelp, "Play Developer Reporting API") {
		t.Error("expected LongHelp to mention Play Developer Reporting API")
	}
}

func TestAnomaliesHelpOutput(t *testing.T) {
	cmd := AnomaliesCommand()
	if cmd.ShortHelp == "" {
		t.Error("expected AnomaliesCommand to have ShortHelp")
	}
	if cmd.LongHelp == "" {
		t.Error("expected AnomaliesCommand to have LongHelp")
	}
}

func installMockVitalsReportingService(t *testing.T, handler http.HandlerFunc) {
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

func captureVitalsStdout(fn func() error) (string, error) {
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

func requestListContains(raw interface{}, want string) bool {
	items, ok := raw.([]interface{})
	if !ok {
		return false
	}
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func timelineDate(raw interface{}) string {
	date, ok := raw.(map[string]interface{})
	if !ok {
		return ""
	}
	return strings.TrimSuffix(strings.TrimSuffix(strings.Join([]string{
		strings.TrimSuffix(strings.TrimSuffix(jsonNumber(date["year"]), ".0"), "."),
		strings.TrimSuffix(strings.TrimSuffix(jsonNumber(date["month"]), ".0"), "."),
		strings.TrimSuffix(strings.TrimSuffix(jsonNumber(date["day"]), ".0"), "."),
	}, "-"), ".0"), ".")
}

func jsonNumber(raw interface{}) string {
	switch v := raw.(type) {
	case float64:
		return fmt.Sprintf("%.0f", v)
	case json.Number:
		return v.String()
	default:
		return ""
	}
}

func TestVitalsUnknownSubcommand(t *testing.T) {
	cmd := VitalsCommand()
	err := cmd.Exec(context.Background(), []string{"unknown"})
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp for unknown subcommand, got %v", err)
	}
}

func TestCrashesUnknownSubcommand(t *testing.T) {
	cmd := CrashesCommand()
	err := cmd.Exec(context.Background(), []string{"unknown"})
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp for unknown subcommand, got %v", err)
	}
}
