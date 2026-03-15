package cmdtest_test

import (
	"context"
	"strings"
	"testing"
)

func TestApps_Help(t *testing.T) {
	root := RootCommand("test")
	stdout, stderr := captureOutput(t, func() {
		root.Parse([]string{"apps"})
		root.Run(context.Background())
	})
	combined := stdout + stderr
	if !strings.Contains(combined, "apps") {
		t.Errorf("help should mention apps, got: %q", combined)
	}
	if !strings.Contains(combined, "list") {
		t.Errorf("help should list 'list' subcommand, got: %q", combined)
	}
}

func TestApps_List_SubcommandExists(t *testing.T) {
	root := RootCommand("test")
	var parseErr error
	captureOutput(t, func() {
		parseErr = root.Parse([]string{"apps", "list"})
	})
	if parseErr != nil {
		t.Errorf("apps list should be a valid subcommand, parse error: %v", parseErr)
	}
}
