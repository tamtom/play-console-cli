package reports

import (
	"context"
	"flag"
	"strings"
	"testing"
)

// execCommand is a helper that builds and executes a command with the given args.
func execCommand(t *testing.T, args []string) error {
	t.Helper()
	cmd := ReportsCommand()
	// Parse and run the command tree
	return cmd.ParseAndRun(context.Background(), args)
}

// --- reports (parent) ---

func TestReportsCommand_NoArgs_ReturnsHelp(t *testing.T) {
	err := execCommand(t, []string{})
	if err != flag.ErrHelp {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

func TestReportsCommand_Financial_NoArgs_ReturnsHelp(t *testing.T) {
	err := execCommand(t, []string{"financial"})
	if err != flag.ErrHelp {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

// --- financial list ---

func TestFinancialList_MissingDeveloper(t *testing.T) {
	err := execCommand(t, []string{"financial", "list"})
	if err == nil {
		t.Fatal("expected error for missing --developer")
	}
	if !strings.Contains(err.Error(), "--developer is required") {
		t.Errorf("expected '--developer is required' error, got: %v", err)
	}
}

func TestFinancialList_InvalidFromMonth(t *testing.T) {
	err := execCommand(t, []string{"financial", "list", "--developer", "12345", "--from", "2024-13"})
	if err == nil {
		t.Fatal("expected error for invalid --from month")
	}
	if !strings.Contains(err.Error(), "--from must be in YYYY-MM format") {
		t.Errorf("expected month format error, got: %v", err)
	}
}

func TestFinancialList_InvalidToMonth(t *testing.T) {
	err := execCommand(t, []string{"financial", "list", "--developer", "12345", "--to", "bad"})
	if err == nil {
		t.Fatal("expected error for invalid --to month")
	}
	if !strings.Contains(err.Error(), "--to must be in YYYY-MM format") {
		t.Errorf("expected month format error, got: %v", err)
	}
}

func TestFinancialList_InvalidType(t *testing.T) {
	err := execCommand(t, []string{"financial", "list", "--developer", "12345", "--type", "unknown"})
	if err == nil {
		t.Fatal("expected error for invalid --type")
	}
	if !strings.Contains(err.Error(), "--type must be one of") {
		t.Errorf("expected type validation error, got: %v", err)
	}
}

func TestFinancialList_ValidMinimalFlags(t *testing.T) {
	err := execCommand(t, []string{"financial", "list", "--developer", "12345"})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestFinancialList_ValidAllFlags(t *testing.T) {
	err := execCommand(t, []string{
		"financial", "list",
		"--developer", "12345",
		"--from", "2024-01",
		"--to", "2024-06",
		"--type", "earnings",
	})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestFinancialList_TypeAll(t *testing.T) {
	err := execCommand(t, []string{
		"financial", "list",
		"--developer", "12345",
		"--type", "all",
	})
	if err != nil {
		t.Errorf("expected no error with --type all, got: %v", err)
	}
}

func TestFinancialList_PrettyWithTable(t *testing.T) {
	err := execCommand(t, []string{
		"financial", "list",
		"--developer", "12345",
		"--output", "table",
		"--pretty",
	})
	if err == nil {
		t.Fatal("expected error for --pretty with table output")
	}
	if !strings.Contains(err.Error(), "--pretty is only valid with JSON output") {
		t.Errorf("expected output flag validation error, got: %v", err)
	}
}

// --- financial download ---

func TestFinancialDownload_MissingDeveloper(t *testing.T) {
	err := execCommand(t, []string{"financial", "download", "--from", "2024-01"})
	if err == nil {
		t.Fatal("expected error for missing --developer")
	}
	if !strings.Contains(err.Error(), "--developer is required") {
		t.Errorf("expected '--developer is required' error, got: %v", err)
	}
}

func TestFinancialDownload_MissingFrom(t *testing.T) {
	err := execCommand(t, []string{"financial", "download", "--developer", "12345"})
	if err == nil {
		t.Fatal("expected error for missing --from")
	}
	if !strings.Contains(err.Error(), "--from is required") {
		t.Errorf("expected '--from is required' error, got: %v", err)
	}
}

func TestFinancialDownload_InvalidFromMonth(t *testing.T) {
	err := execCommand(t, []string{"financial", "download", "--developer", "12345", "--from", "2024-00"})
	if err == nil {
		t.Fatal("expected error for invalid --from month")
	}
	if !strings.Contains(err.Error(), "--from must be in YYYY-MM format") {
		t.Errorf("expected month format error, got: %v", err)
	}
}

func TestFinancialDownload_InvalidToMonth(t *testing.T) {
	err := execCommand(t, []string{"financial", "download", "--developer", "12345", "--from", "2024-01", "--to", "2024-13"})
	if err == nil {
		t.Fatal("expected error for invalid --to month")
	}
	if !strings.Contains(err.Error(), "--to must be in YYYY-MM format") {
		t.Errorf("expected month format error, got: %v", err)
	}
}

func TestFinancialDownload_InvalidType(t *testing.T) {
	err := execCommand(t, []string{"financial", "download", "--developer", "12345", "--from", "2024-01", "--type", "bogus"})
	if err == nil {
		t.Fatal("expected error for invalid --type")
	}
	if !strings.Contains(err.Error(), "--type must be one of") {
		t.Errorf("expected type validation error, got: %v", err)
	}
}

func TestFinancialDownload_TypeAllNotAllowed(t *testing.T) {
	err := execCommand(t, []string{"financial", "download", "--developer", "12345", "--from", "2024-01", "--type", "all"})
	if err == nil {
		t.Fatal("expected error for --type all on download")
	}
	if !strings.Contains(err.Error(), "--type must be one of: earnings, sales, payouts") {
		t.Errorf("expected type error for 'all', got: %v", err)
	}
}

func TestFinancialDownload_ValidMinimalFlags(t *testing.T) {
	err := execCommand(t, []string{"financial", "download", "--developer", "12345", "--from", "2024-01"})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestFinancialDownload_ValidAllFlags(t *testing.T) {
	err := execCommand(t, []string{
		"financial", "download",
		"--developer", "12345",
		"--from", "2024-01",
		"--to", "2024-06",
		"--type", "sales",
		"--dir", "/tmp",
	})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestFinancialDownload_ToDefaultsToFrom(t *testing.T) {
	// When --to is omitted, it defaults to --from. This should succeed.
	err := execCommand(t, []string{
		"financial", "download",
		"--developer", "12345",
		"--from", "2024-03",
		"--type", "payouts",
	})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestFinancialDownload_PrettyWithMarkdown(t *testing.T) {
	err := execCommand(t, []string{
		"financial", "download",
		"--developer", "12345",
		"--from", "2024-01",
		"--output", "markdown",
		"--pretty",
	})
	if err == nil {
		t.Fatal("expected error for --pretty with markdown output")
	}
	if !strings.Contains(err.Error(), "--pretty is only valid with JSON output") {
		t.Errorf("expected output flag validation error, got: %v", err)
	}
}

// --- month validation unit tests ---

func TestValidateMonth_Valid(t *testing.T) {
	cases := []string{"2024-01", "2024-12", "2000-06", "1999-09"}
	for _, c := range cases {
		if err := validateMonth(c, "test"); err != nil {
			t.Errorf("expected %q to be valid, got: %v", c, err)
		}
	}
}

func TestValidateMonth_Invalid(t *testing.T) {
	cases := []string{"2024-00", "2024-13", "24-01", "2024", "2024-1", "bad", ""}
	for _, c := range cases {
		if err := validateMonth(c, "test"); err == nil {
			t.Errorf("expected %q to be invalid", c)
		}
	}
}

// --- report type validation unit tests ---

func TestValidateReportType_Valid(t *testing.T) {
	for _, rt := range []string{"earnings", "sales", "payouts", "all"} {
		if err := validateReportType(rt); err != nil {
			t.Errorf("expected %q to be valid, got: %v", rt, err)
		}
	}
}

func TestValidateReportType_Invalid(t *testing.T) {
	for _, rt := range []string{"unknown", "earning", "", "EARNINGS"} {
		if err := validateReportType(rt); err == nil {
			t.Errorf("expected %q to be invalid", rt)
		}
	}
}
