package cmdtest_test

import (
	"context"
	"strings"
	"testing"
)

func TestVitals_Help(t *testing.T) {
	root := RootCommand("test")
	stdout, stderr := captureOutput(t, func() {
		_ = root.Parse([]string{"vitals"})
		_ = root.Run(context.Background())
	})
	combined := stdout + stderr
	if !strings.Contains(combined, "vitals") {
		t.Errorf("help should mention vitals, got: %q", combined)
	}
	// Check subcommands are listed
	for _, sub := range []string{"crashes", "performance", "errors"} {
		if !strings.Contains(combined, sub) {
			t.Errorf("help should list %q subcommand, got: %q", sub, combined)
		}
	}
}

func TestVitals_Crashes_Help(t *testing.T) {
	root := RootCommand("test")
	stdout, stderr := captureOutput(t, func() {
		_ = root.Parse([]string{"vitals", "crashes"})
		_ = root.Run(context.Background())
	})
	combined := stdout + stderr
	if !strings.Contains(combined, "crashes") {
		t.Errorf("help should mention crashes, got: %q", combined)
	}
}

func TestVitals_Performance_SubcommandExists(t *testing.T) {
	root := RootCommand("test")
	var parseErr error
	captureOutput(t, func() {
		parseErr = root.Parse([]string{"vitals", "performance"})
	})
	if parseErr != nil {
		t.Errorf("vitals performance should be a valid subcommand, parse error: %v", parseErr)
	}
}

func TestVitals_Performance_HelpListsLeafCommands(t *testing.T) {
	root := RootCommand("test")
	stdout, stderr := captureOutput(t, func() {
		_ = root.Parse([]string{"vitals", "performance"})
		_ = root.Run(context.Background())
	})
	combined := stdout + stderr
	for _, token := range []string{"startup", "rendering", "battery"} {
		if !strings.Contains(combined, token) {
			t.Fatalf("performance help should mention %q, got %q", token, combined)
		}
	}
}
