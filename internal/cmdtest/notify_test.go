package cmdtest_test

import (
	"context"
	"strings"
	"testing"
)

func TestNotify_Help(t *testing.T) {
	root := RootCommand("test")
	stdout, stderr := captureOutput(t, func() {
		root.Parse([]string{"notify"})
		root.Run(context.Background())
	})
	combined := stdout + stderr
	if !strings.Contains(combined, "notify") {
		t.Errorf("help should mention notify, got: %q", combined)
	}
	if !strings.Contains(combined, "send") {
		t.Errorf("help should list 'send' subcommand, got: %q", combined)
	}
}

func TestNotify_Send_RequiresWebhookURL(t *testing.T) {
	root := RootCommand("test")
	var runErr error
	_, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"notify", "send"}); err != nil {
			return
		}
		runErr = root.Run(context.Background())
	})
	combined := stderr
	if runErr != nil {
		combined += runErr.Error()
	}
	if !strings.Contains(combined, "webhook-url") {
		t.Errorf("should require --webhook-url, got stderr: %q, err: %v", stderr, runErr)
	}
}

func TestNotify_Send_SubcommandExists(t *testing.T) {
	root := RootCommand("test")
	var parseErr error
	captureOutput(t, func() {
		parseErr = root.Parse([]string{"notify", "send"})
	})
	if parseErr != nil {
		t.Errorf("notify send should be a valid subcommand, parse error: %v", parseErr)
	}
}
