package vitals

import (
	"context"
	"flag"
	"strings"
	"testing"
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
	if err != flag.ErrHelp {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

func TestCrashesNoArgsReturnsHelp(t *testing.T) {
	cmd := CrashesCommand()
	err := cmd.Exec(context.Background(), nil)
	if err != flag.ErrHelp {
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
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--type", "invalid"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid --type")
	}
	if !strings.Contains(err.Error(), "--type must be") {
		t.Errorf("expected '--type must be' error, got: %v", err)
	}
}

func TestCrashesQueryValidTypeCrash(t *testing.T) {
	cmd := CrashesQueryCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--type", "crash"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error (stub implementation)")
	}
	if !strings.Contains(err.Error(), "crashRateMetricSet") {
		t.Errorf("expected error to mention crashRateMetricSet, got: %v", err)
	}
}

func TestCrashesQueryValidTypeANR(t *testing.T) {
	cmd := CrashesQueryCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--type", "anr"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error (stub implementation)")
	}
	if !strings.Contains(err.Error(), "anrRateMetricSet") {
		t.Errorf("expected error to mention anrRateMetricSet, got: %v", err)
	}
}

func TestCrashesQueryInvalidFromDate(t *testing.T) {
	cmd := CrashesQueryCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--from", "not-a-date"}); err != nil {
		t.Fatal(err)
	}
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
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--to", "2025/01/01"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid --to date")
	}
	if !strings.Contains(err.Error(), "--to") {
		t.Errorf("expected error to mention --to, got: %v", err)
	}
}

func TestCrashesQueryValidDates(t *testing.T) {
	cmd := CrashesQueryCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--from", "2025-01-01", "--to", "2025-01-31"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error (stub implementation)")
	}
	// Should reach the stub error, not a date validation error
	if strings.Contains(err.Error(), "--from") || strings.Contains(err.Error(), "--to") {
		t.Errorf("expected stub error, not date validation error, got: %v", err)
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

func TestAnomaliesStubWithPackage(t *testing.T) {
	cmd := AnomaliesCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example.app"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error (stub implementation)")
	}
	if !strings.Contains(err.Error(), "not yet connected") {
		t.Errorf("expected stub error, got: %v", err)
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

func TestVitalsUnknownSubcommand(t *testing.T) {
	cmd := VitalsCommand()
	err := cmd.Exec(context.Background(), []string{"unknown"})
	if err != flag.ErrHelp {
		t.Errorf("expected flag.ErrHelp for unknown subcommand, got %v", err)
	}
}

func TestCrashesUnknownSubcommand(t *testing.T) {
	cmd := CrashesCommand()
	err := cmd.Exec(context.Background(), []string{"unknown"})
	if err != flag.ErrHelp {
		t.Errorf("expected flag.ErrHelp for unknown subcommand, got %v", err)
	}
}
