package vitals

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestAnomaliesMetricTypeValid(t *testing.T) {
	cases := []struct {
		in        string
		wantSet   string
		wantError bool
	}{
		{"crash", "crashRateMetricSet", false},
		{"anr", "anrRateMetricSet", false},
		{"errors", "errorCountMetricSet", false},
		{"performance", "slowRenderingRateMetricSet", false},
		{"all", "*", false},
		{"", "*", false},
		{"nonsense", "", true},
	}
	for _, c := range cases {
		got, err := resolveAnomalyMetric(c.in)
		if c.wantError {
			if err == nil {
				t.Errorf("expected error for %q", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("%q unexpected error: %v", c.in, err)
		}
		if got != c.wantSet {
			t.Errorf("%q -> %q, want %q", c.in, got, c.wantSet)
		}
	}
}

func TestAnomaliesDefaultDates(t *testing.T) {
	from, to, err := resolveAnomalyDates("", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := validateISO8601Date(from); err != nil {
		t.Errorf("default from invalid: %v", err)
	}
	if err := validateISO8601Date(to); err != nil {
		t.Errorf("default to invalid: %v", err)
	}
}

func TestAnomaliesInvalidDates(t *testing.T) {
	if _, _, err := resolveAnomalyDates("garbage", ""); err == nil {
		t.Error("expected error on invalid from")
	}
	if _, _, err := resolveAnomalyDates("", "garbage"); err == nil {
		t.Error("expected error on invalid to")
	}
}

func TestAnomaliesFlagValidation(t *testing.T) {
	cmd := AnomaliesCommand()
	_ = cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--type", "bogus"})
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--type") {
		t.Fatalf("expected --type error, got %v", err)
	}
}

func TestAnomaliesLimitBounds(t *testing.T) {
	cmd := AnomaliesCommand()
	_ = cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--limit", "0"})
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--limit") {
		t.Fatalf("expected limit error, got %v", err)
	}

	cmd = AnomaliesCommand()
	_ = cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--limit", "2000"})
	err = cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--limit") {
		t.Fatalf("expected limit error, got %v", err)
	}
}

func TestAnomaliesFiltersByMetricSet(t *testing.T) {
	installMockVitalsReportingService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"anomalies":[{"metricSet":"apps/com.example.app/crashRateMetricSet"},{"metricSet":"apps/com.example.app/anrRateMetricSet"}]}`)
	})

	cmd := AnomaliesCommand()
	_ = cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--type", "crash", "--limit", "10"})
	stdout, err := captureVitalsStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(stdout, "crashRateMetricSet") {
		t.Fatalf("expected crash anomaly in output, got %s", stdout)
	}
	if strings.Contains(stdout, "anrRateMetricSet") {
		t.Fatalf("did not expect ANR anomaly after crash filter, got %s", stdout)
	}
}
