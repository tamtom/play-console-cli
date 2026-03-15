package cmdtest_test

import (
	"context"
	"strings"
	"testing"
)

func TestListings_Help(t *testing.T) {
	root := RootCommand("test")
	stdout, stderr := captureOutput(t, func() {
		root.Parse([]string{"listings"})
		root.Run(context.Background())
	})
	combined := stdout + stderr
	if !strings.Contains(combined, "listings") {
		t.Errorf("help should mention listings, got: %q", combined)
	}
	for _, sub := range []string{"list", "get", "update", "patch", "delete", "delete-all", "locales"} {
		if !strings.Contains(combined, sub) {
			t.Errorf("help should list %q subcommand, got: %q", sub, combined)
		}
	}
}

func TestListings_List_SubcommandExists(t *testing.T) {
	root := RootCommand("test")
	var parseErr error
	captureOutput(t, func() {
		parseErr = root.Parse([]string{"listings", "list"})
	})
	if parseErr != nil {
		t.Errorf("listings list should be a valid subcommand, parse error: %v", parseErr)
	}
}

func TestListings_Update_SubcommandExists(t *testing.T) {
	root := RootCommand("test")
	var parseErr error
	captureOutput(t, func() {
		parseErr = root.Parse([]string{"listings", "update"})
	})
	if parseErr != nil {
		t.Errorf("listings update should be a valid subcommand, parse error: %v", parseErr)
	}
}
