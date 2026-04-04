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
