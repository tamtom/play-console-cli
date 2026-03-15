package cmdtest_test

import (
	"context"
	"strings"
	"testing"
)

func TestGrants_Help(t *testing.T) {
	root := RootCommand("test")
	stdout, stderr := captureOutput(t, func() {
		root.Parse([]string{"grants"})
		root.Run(context.Background())
	})
	combined := stdout + stderr
	if !strings.Contains(combined, "grants") {
		t.Errorf("help should mention grants, got: %q", combined)
	}
	for _, sub := range []string{"create", "update", "delete"} {
		if !strings.Contains(combined, sub) {
			t.Errorf("help should list %q subcommand, got: %q", sub, combined)
		}
	}
}

func TestGrants_Create_SubcommandExists(t *testing.T) {
	root := RootCommand("test")
	var parseErr error
	captureOutput(t, func() {
		parseErr = root.Parse([]string{"grants", "create"})
	})
	if parseErr != nil {
		t.Errorf("grants create should be a valid subcommand, parse error: %v", parseErr)
	}
}

func TestGrants_Delete_SubcommandExists(t *testing.T) {
	root := RootCommand("test")
	var parseErr error
	captureOutput(t, func() {
		parseErr = root.Parse([]string{"grants", "delete"})
	})
	if parseErr != nil {
		t.Errorf("grants delete should be a valid subcommand, parse error: %v", parseErr)
	}
}
