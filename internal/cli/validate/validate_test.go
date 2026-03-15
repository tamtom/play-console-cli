package validate

import (
	"context"
	"errors"
	"flag"
	"strings"
	"testing"
)

// --- validate command group ---

func TestValidateCommand_Name(t *testing.T) {
	cmd := ValidateCommand()
	if cmd.Name != "validate" {
		t.Errorf("expected name %q, got %q", "validate", cmd.Name)
	}
}

func TestValidateCommand_ShortHelp(t *testing.T) {
	cmd := ValidateCommand()
	if cmd.ShortHelp == "" {
		t.Error("expected non-empty ShortHelp")
	}
}

func TestValidateCommand_UsageFunc(t *testing.T) {
	cmd := ValidateCommand()
	if cmd.UsageFunc == nil {
		t.Error("expected UsageFunc to be set")
	}
}

func TestValidateCommand_HasSubcommands(t *testing.T) {
	cmd := ValidateCommand()
	if len(cmd.Subcommands) == 0 {
		t.Error("expected subcommands")
	}
}

func TestValidateCommand_SubcommandNames(t *testing.T) {
	cmd := ValidateCommand()
	expected := map[string]bool{
		"bundle":      false,
		"listing":     false,
		"screenshots": false,
		"submission":  false,
	}
	for _, sub := range cmd.Subcommands {
		if _, ok := expected[sub.Name]; ok {
			expected[sub.Name] = true
		}
	}
	for name, found := range expected {
		if !found {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

func TestValidateCommand_SubcommandsHaveUsageFunc(t *testing.T) {
	cmd := ValidateCommand()
	for _, sub := range cmd.Subcommands {
		if sub.UsageFunc == nil {
			t.Errorf("subcommand %q missing UsageFunc", sub.Name)
		}
	}
}

func TestValidateCommand_NoArgs_ReturnsHelp(t *testing.T) {
	cmd := ValidateCommand()
	err := cmd.Exec(context.Background(), nil)
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

// --- validate submission ---

func TestSubmissionCommand_Name(t *testing.T) {
	cmd := SubmissionCommand()
	if cmd.Name != "submission" {
		t.Errorf("expected name %q, got %q", "submission", cmd.Name)
	}
}

func TestSubmissionCommand_ShortHelp(t *testing.T) {
	cmd := SubmissionCommand()
	if cmd.ShortHelp == "" {
		t.Error("expected non-empty ShortHelp")
	}
}

func TestSubmissionCommand_UsageFunc(t *testing.T) {
	cmd := SubmissionCommand()
	if cmd.UsageFunc == nil {
		t.Error("expected UsageFunc to be set")
	}
}

func TestSubmissionCommand_MissingPackage(t *testing.T) {
	cmd := SubmissionCommand()
	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --package")
	}
	if !strings.Contains(err.Error(), "package") {
		t.Errorf("error should mention package, got: %s", err.Error())
	}
}

func TestSubmissionCommand_InvalidOutputFormat(t *testing.T) {
	cmd := SubmissionCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--output", "xml"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

func TestSubmissionCommand_PrettyWithTable(t *testing.T) {
	cmd := SubmissionCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--output", "table", "--pretty"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for --pretty with table output")
	}
	if !strings.Contains(err.Error(), "--pretty") {
		t.Errorf("error should mention --pretty, got: %s", err.Error())
	}
}

// --- existing subcommands: bundle ---

func TestBundleCommand_Name(t *testing.T) {
	cmd := BundleCommand()
	if cmd.Name != "bundle" {
		t.Errorf("expected name %q, got %q", "bundle", cmd.Name)
	}
}

func TestBundleCommand_MissingFile(t *testing.T) {
	cmd := BundleCommand()
	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --file")
	}
	if !strings.Contains(err.Error(), "--file") {
		t.Errorf("error should mention --file, got: %s", err.Error())
	}
}

// --- existing subcommands: listing ---

func TestListingCommand_Name(t *testing.T) {
	cmd := ListingCommand()
	if cmd.Name != "listing" {
		t.Errorf("expected name %q, got %q", "listing", cmd.Name)
	}
}

// --- existing subcommands: screenshots ---

func TestScreenshotsCommand_Name(t *testing.T) {
	cmd := ScreenshotsCommand()
	if cmd.Name != "screenshots" {
		t.Errorf("expected name %q, got %q", "screenshots", cmd.Name)
	}
}
