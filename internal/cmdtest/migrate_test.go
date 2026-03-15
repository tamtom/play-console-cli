package cmdtest_test

import (
	"context"
	"strings"
	"testing"
)

func TestMigrate_Help(t *testing.T) {
	root := RootCommand("test")
	stdout, stderr := captureOutput(t, func() {
		root.Parse([]string{"migrate"})
		root.Run(context.Background())
	})
	combined := stdout + stderr
	if !strings.Contains(combined, "migrate") {
		t.Errorf("help should mention migrate, got: %q", combined)
	}
	if !strings.Contains(combined, "fastlane") {
		t.Errorf("help should list 'fastlane' subcommand, got: %q", combined)
	}
}

func TestMigrate_Fastlane_RequiresSource(t *testing.T) {
	root := RootCommand("test")
	var runErr error
	_, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"migrate", "fastlane"}); err != nil {
			return
		}
		runErr = root.Run(context.Background())
	})
	combined := stderr
	if runErr != nil {
		combined += runErr.Error()
	}
	if !strings.Contains(combined, "source") {
		t.Errorf("should require --source, got stderr: %q, err: %v", stderr, runErr)
	}
}
