package vitals

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/reportingclient"
)

func TestMetricSetGet_SupportsNewMemoryDescriptorViaOfficialREST(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"name":"apps/dev.example/bitmapMemoryUsageMetricSet"}`)
	}))
	defer server.Close()
	installMetricReportingService(t, server)

	cmd := MetricSetGetCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "dev.example", "--metric-set", "bitmap-memory"}); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Exec(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1beta1/apps/dev.example/bitmapMemoryUsageMetricSet" {
		t.Fatalf("path = %q", gotPath)
	}
}

func TestMetricSetQuery_RequiresRequestJSON(t *testing.T) {
	cmd := MetricSetQueryCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "dev.example", "--metric-set", "lmk"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--json") {
		t.Fatalf("expected --json error, got %v", err)
	}
}

func TestEveryMetricSetEndpoint_SuccessAndAPIError(t *testing.T) {
	names := make([]string, 0, len(metricSetResources))
	for name := range metricSetResources {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		resource := metricSetResources[name]
		for _, commandKind := range []string{"get", "query"} {
			for _, status := range []int{http.StatusOK, http.StatusForbidden} {
				caseName := name + " " + commandKind + " success"
				if status != http.StatusOK {
					caseName = name + " " + commandKind + " API error"
				}
				t.Run(caseName, func(t *testing.T) {
					var gotMethod, gotPath string
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						gotMethod, gotPath = r.Method, r.URL.Path
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(status)
						if status == http.StatusOK {
							_, _ = io.WriteString(w, `{}`)
							return
						}
						_, _ = io.WriteString(w, `{"error":{"message":"denied"}}`)
					}))
					t.Cleanup(server.Close)
					installMetricReportingService(t, server)

					cmd := MetricSetGetCommand()
					args := []string{"--package", "dev.example", "--metric-set", name}
					wantMethod := http.MethodGet
					wantPath := "/v1beta1/apps/dev.example/" + resource
					if commandKind == "query" {
						cmd = MetricSetQueryCommand()
						args = append(args, "--json", `{"metrics":["distinctUsers"]}`)
						wantMethod = http.MethodPost
						wantPath += ":query"
					}
					if err := cmd.FlagSet.Parse(args); err != nil {
						t.Fatal(err)
					}
					err := cmd.Exec(context.Background(), nil)
					if status != http.StatusOK {
						if err == nil || !strings.Contains(err.Error(), name) || !strings.Contains(err.Error(), "reporting request failed") {
							t.Fatalf("expected contextual API error, got %v", err)
						}
						return
					}
					if err != nil {
						t.Fatal(err)
					}
					if gotMethod != wantMethod || gotPath != wantPath {
						t.Fatalf("request = %s %s, want %s %s", gotMethod, gotPath, wantMethod, wantPath)
					}
				})
			}
		}
	}
}

func TestReleaseFiltersEndpoint_SuccessAndAPIError(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var gotPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				if status == http.StatusOK {
					_, _ = io.WriteString(w, `{}`)
					return
				}
				_, _ = io.WriteString(w, `{"error":{"message":"denied"}}`)
			}))
			t.Cleanup(server.Close)
			installMetricReportingService(t, server)
			cmd := ReleaseFiltersCommand()
			if err := cmd.FlagSet.Parse([]string{"--package", "dev.example"}); err != nil {
				t.Fatal(err)
			}
			err := cmd.Exec(context.Background(), nil)
			if status != http.StatusOK {
				if err == nil || !strings.Contains(err.Error(), "release-filter") {
					t.Fatalf("expected contextual API error, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if gotPath != "/v1beta1/apps/dev.example:fetchReleaseFilterOptions" {
				t.Fatalf("path = %q", gotPath)
			}
		})
	}
}

func installMetricReportingService(t *testing.T, server *httptest.Server) {
	t.Helper()
	original := newMetricReportingService
	newMetricReportingService = func(context.Context) (*reportingclient.Service, error) {
		return reportingclient.NewServiceWithClient(context.Background(), server.Client(), server.URL+"/")
	}
	t.Cleanup(func() { newMetricReportingService = original })
}
