package vitals

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
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

func installMetricReportingService(t *testing.T, server *httptest.Server) {
	t.Helper()
	original := newMetricReportingService
	newMetricReportingService = func(context.Context) (*reportingclient.Service, error) {
		return reportingclient.NewServiceWithClient(context.Background(), server.Client(), server.URL+"/")
	}
	t.Cleanup(func() { newMetricReportingService = original })
}
