package cmdtest_test

import (
	"context"
	"strings"
	"testing"
)

func TestImages_Help(t *testing.T) {
	root := RootCommand("test")
	stdout, stderr := captureOutput(t, func() {
		_ = root.Parse([]string{"images"})
		_ = root.Run(context.Background())
	})
	combined := stdout + stderr
	if !strings.Contains(combined, "images") {
		t.Fatalf("help should mention images, got: %q", combined)
	}
	for _, sub := range []string{"list", "upload", "delete", "delete-all", "plan", "pull", "sync"} {
		if !strings.Contains(combined, sub) {
			t.Fatalf("help should list %q subcommand, got: %q", sub, combined)
		}
	}
}

func TestImages_Plan_SubcommandExists(t *testing.T) {
	root := RootCommand("test")
	var parseErr error
	captureOutput(t, func() {
		parseErr = root.Parse([]string{"images", "plan"})
	})
	if parseErr != nil {
		t.Fatalf("images plan should be a valid subcommand, parse error: %v", parseErr)
	}
}

func TestImages_Pull_SubcommandExists(t *testing.T) {
	root := RootCommand("test")
	var parseErr error
	captureOutput(t, func() {
		parseErr = root.Parse([]string{"images", "pull"})
	})
	if parseErr != nil {
		t.Fatalf("images pull should be a valid subcommand, parse error: %v", parseErr)
	}
}

func TestImages_Sync_SubcommandExists(t *testing.T) {
	root := RootCommand("test")
	var parseErr error
	captureOutput(t, func() {
		parseErr = root.Parse([]string{"images", "sync"})
	})
	if parseErr != nil {
		t.Fatalf("images sync should be a valid subcommand, parse error: %v", parseErr)
	}
}
