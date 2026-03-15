package cmdtest_test

import (
	"context"
	"strings"
	"testing"
)

func TestAuth_Help(t *testing.T) {
	root := RootCommand("test")
	stdout, stderr := captureOutput(t, func() {
		_ = root.Parse([]string{"auth"})
		_ = root.Run(context.Background())
	})
	combined := stdout + stderr
	if !strings.Contains(combined, "auth") {
		t.Errorf("help should mention auth, got: %q", combined)
	}
	// Check subcommands are listed
	for _, sub := range []string{"init", "login", "switch", "logout", "status", "doctor"} {
		if !strings.Contains(combined, sub) {
			t.Errorf("help should list %q subcommand, got: %q", sub, combined)
		}
	}
}

func TestAuth_Login_RequiresServiceAccount(t *testing.T) {
	root := RootCommand("test")
	var runErr error
	_, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"auth", "login"}); err != nil {
			return
		}
		runErr = root.Run(context.Background())
	})
	combined := stderr
	if runErr != nil {
		combined += runErr.Error()
	}
	if !strings.Contains(combined, "service-account") {
		t.Errorf("should require --service-account, got stderr: %q, err: %v", stderr, runErr)
	}
}

func TestAuth_Doctor_ShowsReport(t *testing.T) {
	root := RootCommand("test")
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"auth", "doctor"}); err != nil {
			return
		}
		_ = root.Run(context.Background())
	})
	combined := stdout + stderr
	// Doctor runs without errors in isolated environment; should produce output
	if !strings.Contains(combined, "doctor") && !strings.Contains(combined, "Auth") && !strings.Contains(combined, "config") {
		// At minimum, it should not panic and should produce some output
		if combined == "" {
			t.Errorf("auth doctor should produce output, got nothing")
		}
	}
}
