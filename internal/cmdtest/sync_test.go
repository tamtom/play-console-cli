package cmdtest_test

import (
	"context"
	"strings"
	"testing"
)

func TestSync_HelpExposesResumableOneShotCommands(t *testing.T) {
	root := RootCommand("test")
	stdout, stderr := captureOutput(t, func() {
		_ = root.Parse([]string{"sync"})
		_ = root.Run(context.Background())
	})
	combined := stdout + stderr
	for _, subcommand := range []string{"plan", "apply", "run"} {
		if !strings.Contains(combined, subcommand) {
			t.Fatalf("sync help should list %q, got: %q", subcommand, combined)
		}
	}
}

func TestSyncApply_RequiresPlanBeforeAuthentication(t *testing.T) {
	_, _, err := runCommand(t, "sync", "apply")
	if err == nil || !strings.Contains(err.Error(), "--plan-file is required") {
		t.Fatalf("expected local --plan-file validation, got %v", err)
	}
}
