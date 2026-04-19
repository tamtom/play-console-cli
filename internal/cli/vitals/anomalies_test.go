package vitals

import (
	"context"
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
		{"performance", "slowRenderingRate20FpsMetricSet", false},
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

func TestAnomaliesHappyPathStillStub(t *testing.T) {
	cmd := AnomaliesCommand()
	_ = cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--type", "crash", "--limit", "10"})
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "not yet connected") {
		t.Fatalf("expected stub error, got %v", err)
	}
	if !strings.Contains(err.Error(), "crashRateMetricSet") {
		t.Errorf("expected metric set in error, got %v", err)
	}
}
