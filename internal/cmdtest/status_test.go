package cmdtest_test

import (
	"context"
	"strings"
	"testing"
)

func TestStatus_CommandExists(t *testing.T) {
	root := RootCommand("test")
	if err := root.Parse([]string{"status"}); err != nil {
		t.Fatalf("status should be a valid top-level command: %v", err)
	}
}

func TestStatus_HelpIncludesStatus(t *testing.T) {
	root := RootCommand("test")
	stdout, stderr := captureOutput(t, func() {
		_ = root.Parse([]string{})
		_ = root.Run(context.Background())
	})
	combined := stdout + stderr
	if !strings.Contains(combined, "status") {
		t.Errorf("help should mention status, got: %q", combined)
	}
}

func TestStatus_RequiresPackage(t *testing.T) {
	_, _, err := runCommand(t, "status")
	if err == nil {
		t.Fatal("expected --package error")
	}
	if !strings.Contains(err.Error(), "--package is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStatus_WatchRejectsNonPositivePollInterval(t *testing.T) {
	_, _, err := runCommand(t,
		"status",
		"--package", "com.example.app",
		"--watch",
		"--poll-interval", "0s",
	)
	if err == nil {
		t.Fatal("expected poll interval error")
	}
	if !strings.Contains(err.Error(), "--poll-interval must be greater than 0") {
		t.Fatalf("unexpected error: %v", err)
	}
}
