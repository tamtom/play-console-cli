package cmdtest_test

import (
	"context"
	"strings"
	"testing"
)

func TestReleaseNotes_Help(t *testing.T) {
	root := RootCommand("test")
	stdout, stderr := captureOutput(t, func() {
		_ = root.Parse([]string{"release-notes"})
		_ = root.Run(context.Background())
	})
	combined := stdout + stderr
	if !strings.Contains(combined, "release-notes") {
		t.Errorf("help should mention release-notes, got: %q", combined)
	}
	if !strings.Contains(combined, "generate") {
		t.Errorf("help should list 'generate' subcommand, got: %q", combined)
	}
}

func TestReleaseNotes_Generate_RequiresSinceRef(t *testing.T) {
	root := RootCommand("test")
	var runErr error
	_, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"release-notes", "generate"}); err != nil {
			return
		}
		runErr = root.Run(context.Background())
	})
	combined := stderr
	if runErr != nil {
		combined += runErr.Error()
	}
	// Should require either --since-tag or --since-ref
	if !strings.Contains(combined, "since-tag") && !strings.Contains(combined, "since-ref") {
		t.Errorf("should require --since-tag or --since-ref, got stderr: %q, err: %v", stderr, runErr)
	}
}
