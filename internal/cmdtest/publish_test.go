package cmdtest_test

import (
	"context"
	"strings"
	"testing"
)

func TestPublish_Help(t *testing.T) {
	root := RootCommand("test")
	stdout, stderr := captureOutput(t, func() {
		_ = root.Parse([]string{"publish"})
		_ = root.Run(context.Background())
	})
	combined := stdout + stderr
	if !strings.Contains(combined, "publish") {
		t.Fatalf("help should mention publish, got %q", combined)
	}
	if !strings.Contains(combined, "track") {
		t.Fatalf("help should mention track subcommand, got %q", combined)
	}
}
