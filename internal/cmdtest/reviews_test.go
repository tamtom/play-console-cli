package cmdtest_test

import (
	"context"
	"strings"
	"testing"
)

func TestReviews_Help(t *testing.T) {
	root := RootCommand("test")
	stdout, stderr := captureOutput(t, func() {
		root.Parse([]string{"reviews"})
		root.Run(context.Background())
	})
	combined := stdout + stderr
	if !strings.Contains(combined, "reviews") {
		t.Errorf("help should mention reviews, got: %q", combined)
	}
	for _, sub := range []string{"list", "get", "reply"} {
		if !strings.Contains(combined, sub) {
			t.Errorf("help should list %q subcommand, got: %q", sub, combined)
		}
	}
}

func TestReviews_List_SubcommandExists(t *testing.T) {
	root := RootCommand("test")
	var parseErr error
	captureOutput(t, func() {
		parseErr = root.Parse([]string{"reviews", "list"})
	})
	if parseErr != nil {
		t.Errorf("reviews list should be a valid subcommand, parse error: %v", parseErr)
	}
}

func TestReviews_Reply_SubcommandExists(t *testing.T) {
	root := RootCommand("test")
	var parseErr error
	captureOutput(t, func() {
		parseErr = root.Parse([]string{"reviews", "reply"})
	})
	if parseErr != nil {
		t.Errorf("reviews reply should be a valid subcommand, parse error: %v", parseErr)
	}
}
