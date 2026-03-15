package cmdtest_test

import (
	"context"
	"strings"
	"testing"
)

func TestUsers_Help(t *testing.T) {
	root := RootCommand("test")
	stdout, stderr := captureOutput(t, func() {
		_ = root.Parse([]string{"users"})
		_ = root.Run(context.Background())
	})
	combined := stdout + stderr
	if !strings.Contains(combined, "users") {
		t.Errorf("help should mention users, got: %q", combined)
	}
	// Check subcommands are listed
	for _, sub := range []string{"list", "create", "delete"} {
		if !strings.Contains(combined, sub) {
			t.Errorf("help should list %q subcommand, got: %q", sub, combined)
		}
	}
}

func TestUsers_List_RequiresDeveloper(t *testing.T) {
	root := RootCommand("test")
	var runErr error
	_, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"users", "list"}); err != nil {
			return
		}
		runErr = root.Run(context.Background())
	})
	combined := stderr
	if runErr != nil {
		combined += runErr.Error()
	}
	if !strings.Contains(combined, "developer") {
		t.Errorf("should require --developer, got stderr: %q, err: %v", stderr, runErr)
	}
}

func TestUsers_Create_SubcommandExists(t *testing.T) {
	root := RootCommand("test")
	var parseErr error
	captureOutput(t, func() {
		parseErr = root.Parse([]string{"users", "create"})
	})
	if parseErr != nil {
		t.Errorf("users create should be a valid subcommand, parse error: %v", parseErr)
	}
}
