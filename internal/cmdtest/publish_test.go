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

func TestPublishTrack_RequiresArtifact(t *testing.T) {
	_, _, err := runCommand(t,
		"publish", "track",
		"--package", "com.example.app",
		"--track", "internal",
	)
	if err == nil {
		t.Fatal("expected missing artifact error")
	}
	if !strings.Contains(err.Error(), "either --bundle or --apk is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPublishTrack_RejectsBothArtifactFlags(t *testing.T) {
	_, _, err := runCommand(t,
		"publish", "track",
		"--package", "com.example.app",
		"--bundle", "app.aab",
		"--apk", "app.apk",
	)
	if err == nil {
		t.Fatal("expected conflicting artifact error")
	}
	if !strings.Contains(err.Error(), "use either --bundle or --apk") {
		t.Fatalf("unexpected error: %v", err)
	}
}
