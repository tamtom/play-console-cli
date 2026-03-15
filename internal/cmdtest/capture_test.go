package cmdtest_test

import (
	"context"
	"strings"
	"testing"
)

func TestRootCommand_VersionSubcommand(t *testing.T) {
	root := RootCommand("1.2.3-test")
	stdout, _ := captureOutput(t, func() {
		if err := root.Parse([]string{"version"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		_ = root.Run(context.Background())
	})
	if !strings.Contains(stdout, "1.2.3-test") {
		t.Errorf("expected version in stdout, got: %q", stdout)
	}
}

func TestRootCommand_NoArgs_ShowsHelp(t *testing.T) {
	root := RootCommand("test")
	_, stderr := captureOutput(t, func() {
		_ = root.Parse([]string{})
		_ = root.Run(context.Background())
	})
	if !strings.Contains(stderr, "gplay") {
		t.Errorf("expected help output mentioning gplay, got: %q", stderr)
	}
}

func TestRootCommand_UnknownCommand(t *testing.T) {
	root := RootCommand("test")
	_, stderr := captureOutput(t, func() {
		_ = root.Parse([]string{"nonexistent-command"})
		_ = root.Run(context.Background())
	})
	if !strings.Contains(stderr, "unknown command") && !strings.Contains(stderr, "Unknown") {
		t.Errorf("expected unknown command error, got: %q", stderr)
	}
}
