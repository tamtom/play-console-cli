package reports

import (
	"strings"
	"testing"
)

// --- stats (parent) ---

func TestStatsCommand_NoArgs_ReturnsHelp(t *testing.T) {
	err := execCommand(t, []string{"stats"})
	if err == nil {
		t.Fatal("expected flag.ErrHelp, got nil")
	}
}

// --- stats list ---

func TestStatsList_MissingPackage(t *testing.T) {
	err := execCommand(t, []string{"stats", "list"})
	if err == nil {
		t.Fatal("expected error for missing --package")
	}
	if !strings.Contains(err.Error(), "--package is required") {
		t.Errorf("expected '--package is required' error, got: %v", err)
	}
}

func TestStatsList_InvalidFromMonth(t *testing.T) {
	err := execCommand(t, []string{"stats", "list", "--package", "com.example.app", "--from", "2024-13"})
	if err == nil {
		t.Fatal("expected error for invalid --from month")
	}
	if !strings.Contains(err.Error(), "--from must be in YYYY-MM format") {
		t.Errorf("expected month format error, got: %v", err)
	}
}

func TestStatsList_InvalidToMonth(t *testing.T) {
	err := execCommand(t, []string{"stats", "list", "--package", "com.example.app", "--to", "bad"})
	if err == nil {
		t.Fatal("expected error for invalid --to month")
	}
	if !strings.Contains(err.Error(), "--to must be in YYYY-MM format") {
		t.Errorf("expected month format error, got: %v", err)
	}
}

func TestStatsList_InvalidType(t *testing.T) {
	err := execCommand(t, []string{"stats", "list", "--package", "com.example.app", "--type", "unknown"})
	if err == nil {
		t.Fatal("expected error for invalid --type")
	}
	if !strings.Contains(err.Error(), "--type must be one of") {
		t.Errorf("expected type validation error, got: %v", err)
	}
}

func TestStatsList_ValidMinimalFlags(t *testing.T) {
	err := execCommand(t, []string{"stats", "list", "--package", "com.example.app"})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestStatsList_ValidAllFlags(t *testing.T) {
	err := execCommand(t, []string{
		"stats", "list",
		"--package", "com.example.app",
		"--from", "2025-01",
		"--to", "2025-06",
		"--type", "installs",
	})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestStatsList_TypeAll(t *testing.T) {
	err := execCommand(t, []string{
		"stats", "list",
		"--package", "com.example.app",
		"--type", "all",
	})
	if err != nil {
		t.Errorf("expected no error with --type all, got: %v", err)
	}
}

func TestStatsList_AllStatsTypes(t *testing.T) {
	types := []string{"installs", "ratings", "crashes", "store_performance", "subscriptions"}
	for _, st := range types {
		err := execCommand(t, []string{
			"stats", "list",
			"--package", "com.example.app",
			"--type", st,
		})
		if err != nil {
			t.Errorf("expected no error for type %q, got: %v", st, err)
		}
	}
}

func TestStatsList_PrettyWithTable(t *testing.T) {
	err := execCommand(t, []string{
		"stats", "list",
		"--package", "com.example.app",
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

// --- stats download ---

func TestStatsDownload_MissingPackage(t *testing.T) {
	err := execCommand(t, []string{"stats", "download", "--from", "2025-01", "--type", "installs"})
	if err == nil {
		t.Fatal("expected error for missing --package")
	}
	if !strings.Contains(err.Error(), "--package is required") {
		t.Errorf("expected '--package is required' error, got: %v", err)
	}
}

func TestStatsDownload_MissingFrom(t *testing.T) {
	err := execCommand(t, []string{"stats", "download", "--package", "com.example.app", "--type", "installs"})
	if err == nil {
		t.Fatal("expected error for missing --from")
	}
	if !strings.Contains(err.Error(), "--from is required") {
		t.Errorf("expected '--from is required' error, got: %v", err)
	}
}

func TestStatsDownload_MissingType(t *testing.T) {
	err := execCommand(t, []string{"stats", "download", "--package", "com.example.app", "--from", "2025-01"})
	if err == nil {
		t.Fatal("expected error for missing --type")
	}
	if !strings.Contains(err.Error(), "--type is required") {
		t.Errorf("expected '--type is required' error, got: %v", err)
	}
}

func TestStatsDownload_InvalidFromMonth(t *testing.T) {
	err := execCommand(t, []string{"stats", "download", "--package", "com.example.app", "--from", "2025-00", "--type", "installs"})
	if err == nil {
		t.Fatal("expected error for invalid --from month")
	}
	if !strings.Contains(err.Error(), "--from must be in YYYY-MM format") {
		t.Errorf("expected month format error, got: %v", err)
	}
}

func TestStatsDownload_InvalidToMonth(t *testing.T) {
	err := execCommand(t, []string{"stats", "download", "--package", "com.example.app", "--from", "2025-01", "--to", "2025-13", "--type", "installs"})
	if err == nil {
		t.Fatal("expected error for invalid --to month")
	}
	if !strings.Contains(err.Error(), "--to must be in YYYY-MM format") {
		t.Errorf("expected month format error, got: %v", err)
	}
}

func TestStatsDownload_InvalidType(t *testing.T) {
	err := execCommand(t, []string{"stats", "download", "--package", "com.example.app", "--from", "2025-01", "--type", "bogus"})
	if err == nil {
		t.Fatal("expected error for invalid --type")
	}
	if !strings.Contains(err.Error(), "--type must be one of") {
		t.Errorf("expected type validation error, got: %v", err)
	}
}

func TestStatsDownload_TypeAllNotAllowed(t *testing.T) {
	err := execCommand(t, []string{"stats", "download", "--package", "com.example.app", "--from", "2025-01", "--type", "all"})
	if err == nil {
		t.Fatal("expected error for --type all on download")
	}
	if !strings.Contains(err.Error(), "--type must be one of: installs, ratings, crashes, store_performance, subscriptions") {
		t.Errorf("expected type error for 'all', got: %v", err)
	}
}

func TestStatsDownload_ValidMinimalFlags(t *testing.T) {
	err := execCommand(t, []string{"stats", "download", "--package", "com.example.app", "--from", "2025-01", "--type", "installs"})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestStatsDownload_ValidAllFlags(t *testing.T) {
	err := execCommand(t, []string{
		"stats", "download",
		"--package", "com.example.app",
		"--from", "2025-01",
		"--to", "2025-06",
		"--type", "crashes",
		"--dir", "/tmp",
	})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestStatsDownload_ToDefaultsToFrom(t *testing.T) {
	err := execCommand(t, []string{
		"stats", "download",
		"--package", "com.example.app",
		"--from", "2025-03",
		"--type", "ratings",
	})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestStatsDownload_AllValidTypes(t *testing.T) {
	types := []string{"installs", "ratings", "crashes", "store_performance", "subscriptions"}
	for _, st := range types {
		err := execCommand(t, []string{
			"stats", "download",
			"--package", "com.example.app",
			"--from", "2025-01",
			"--type", st,
		})
		if err != nil {
			t.Errorf("expected no error for type %q, got: %v", st, err)
		}
	}
}

func TestStatsDownload_PrettyWithMarkdown(t *testing.T) {
	err := execCommand(t, []string{
		"stats", "download",
		"--package", "com.example.app",
		"--from", "2025-01",
		"--type", "installs",
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

// --- stats type validation unit tests ---

func TestValidateStatsType_Valid(t *testing.T) {
	for _, st := range []string{"installs", "ratings", "crashes", "store_performance", "subscriptions", "all"} {
		if err := validateStatsType(st); err != nil {
			t.Errorf("expected %q to be valid, got: %v", st, err)
		}
	}
}

func TestValidateStatsType_Invalid(t *testing.T) {
	for _, st := range []string{"unknown", "install", "", "INSTALLS", "earning"} {
		if err := validateStatsType(st); err == nil {
			t.Errorf("expected %q to be invalid", st)
		}
	}
}
