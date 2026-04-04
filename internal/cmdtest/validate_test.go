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
	for _, token := range []string{"bundle", "listing", "screenshots", "submission"} {
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
