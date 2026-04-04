package cmdtest_test

import (
	"context"
	"strings"
	"testing"
)

func TestValidate_Help(t *testing.T) {
	root := RootCommand("test")
	stdout, stderr := captureOutput(t, func() {
		_ = root.Parse([]string{"validate"})
		_ = root.Run(context.Background())
	})
	combined := stdout + stderr
	if !strings.Contains(combined, "validate") {
		t.Fatalf("help should mention validate, got %q", combined)
	}
	for _, token := range []string{"bundle", "listing", "screenshots", "submission", "--track", "--release-notes"} {
		if !strings.Contains(combined, token) {
			t.Fatalf("help should mention %q, got %q", token, combined)
		}
	}
}

func TestValidate_RootFlagsExist(t *testing.T) {
	root := RootCommand("test")
	var parseErr error
	captureOutput(t, func() {
		parseErr = root.Parse([]string{"validate", "--package", "com.example.app"})
	})
	if parseErr != nil {
		t.Fatalf("validate root flags should parse, got %v", parseErr)
	}
}

func TestValidate_RootRejectsConflictingArtifactFlags(t *testing.T) {
	_, _, err := runCommand(t,
		"validate",
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
