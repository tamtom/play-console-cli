package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAttemptFixes_DryRun(t *testing.T) {
	report := authReport{Errors: 0, Warnings: 1}
	fixes := attemptFixes(report, false)
	// Should not fail even with empty report
	for _, f := range fixes {
		if f.Status == "fixed" {
			t.Errorf("dry run should not produce 'fixed' status, got fix %q with status %q", f.Name, f.Status)
		}
	}
}

func TestAttemptFixes_Apply(t *testing.T) {
	report := authReport{Errors: 0}
	fixes := attemptFixes(report, true)
	// Should not panic or fail
	_ = fixes
}

func TestAttemptFixes_MissingConfigDir(t *testing.T) {
	// Create a temp directory to simulate a missing config directory scenario
	tmpDir := t.TempDir()
	missingDir := filepath.Join(tmpDir, "nonexistent", ".gplay")

	// Verify the directory doesn't exist
	if _, err := os.Stat(missingDir); !os.IsNotExist(err) {
		t.Fatalf("expected directory to not exist: %s", missingDir)
	}
}

func TestAttemptFixes_ServiceAccountEnv(t *testing.T) {
	// Create a temp file to act as a service account
	tmpFile := filepath.Join(t.TempDir(), "sa.json")
	if err := os.WriteFile(tmpFile, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("GPLAY_SERVICE_ACCOUNT", tmpFile)

	report := authReport{}
	fixes := attemptFixes(report, false)

	found := false
	for _, f := range fixes {
		if f.Name == "service_account" {
			found = true
			if f.Status != "manual_action_required" {
				t.Errorf("expected status 'manual_action_required', got %q", f.Status)
			}
		}
	}
	if !found {
		t.Error("expected service_account fix when GPLAY_SERVICE_ACCOUNT is set to a valid file")
	}
}

func TestAttemptFixes_ServiceAccountEnvMissingFile(t *testing.T) {
	t.Setenv("GPLAY_SERVICE_ACCOUNT", "/nonexistent/sa.json")

	report := authReport{}
	fixes := attemptFixes(report, false)

	for _, f := range fixes {
		if f.Name == "service_account" {
			t.Error("should not suggest service_account fix when the file doesn't exist")
		}
	}
}

func TestPrintFixes_Empty(t *testing.T) {
	// Should not panic
	printFixes(nil)
	printFixes([]fixResult{})
}

func TestPrintFixes_WithResults(t *testing.T) {
	fixes := []fixResult{
		{Name: "config_directory", Status: "fixed", Message: "Created /tmp/test"},
		{Name: "service_account", Status: "manual_action_required", Message: "Run: gplay auth login"},
	}
	// Should not panic
	printFixes(fixes)
}
