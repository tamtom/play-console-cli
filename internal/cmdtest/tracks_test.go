package cmdtest_test

import (
	"context"
	"strings"
	"testing"
)

func TestTracks_Help(t *testing.T) {
	root := RootCommand("test")
	stdout, stderr := captureOutput(t, func() {
		_ = root.Parse([]string{"tracks"})
		_ = root.Run(context.Background())
	})
	combined := stdout + stderr
	if !strings.Contains(combined, "tracks") {
		t.Errorf("help should mention tracks, got: %q", combined)
	}
	for _, sub := range []string{"list", "get", "create", "update", "patch"} {
		if !strings.Contains(combined, sub) {
			t.Errorf("help should list %q subcommand, got: %q", sub, combined)
		}
	}
}

func TestTracks_List_SubcommandExists(t *testing.T) {
	root := RootCommand("test")
	var parseErr error
	captureOutput(t, func() {
		parseErr = root.Parse([]string{"tracks", "list"})
	})
	if parseErr != nil {
		t.Errorf("tracks list should be a valid subcommand, parse error: %v", parseErr)
	}
}

func TestTracks_Get_SubcommandExists(t *testing.T) {
	root := RootCommand("test")
	var parseErr error
	captureOutput(t, func() {
		parseErr = root.Parse([]string{"tracks", "get"})
	})
	if parseErr != nil {
		t.Errorf("tracks get should be a valid subcommand, parse error: %v", parseErr)
	}
}
