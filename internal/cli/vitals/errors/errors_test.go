package errors

import (
	"bytes"
	"context"
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"
)

// Helper to run a command with args and capture stderr.
func runCommand(t *testing.T, cmd *ffcli.Command, args []string) error {
	t.Helper()
	// Redirect stderr to capture error messages
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() {
		os.Stderr = oldStderr
	}()

	err := cmd.ParseAndRun(context.Background(), args)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)

	return err
}

func TestErrorsCommand_Structure(t *testing.T) {
	cmd := ErrorsCommand()
	if cmd.Name != "errors" {
		t.Errorf("expected command name 'errors', got %q", cmd.Name)
	}
	if len(cmd.Subcommands) != 2 {
		t.Fatalf("expected 2 subcommands, got %d", len(cmd.Subcommands))
	}

	names := make(map[string]bool)
	for _, sub := range cmd.Subcommands {
		names[sub.Name] = true
	}
	for _, expected := range []string{"issues", "reports"} {
		if !names[expected] {
			t.Errorf("expected subcommand %q not found", expected)
		}
	}
}

func TestErrorsCommand_NoSubcommand(t *testing.T) {
	cmd := ErrorsCommand()
	err := runCommand(t, cmd, []string{})
	if err != flag.ErrHelp {
		t.Errorf("expected flag.ErrHelp with no subcommand, got %v", err)
	}
}

func TestErrorsCommand_UnknownSubcommand(t *testing.T) {
	cmd := ErrorsCommand()
	err := runCommand(t, cmd, []string{"unknown"})
	if err != flag.ErrHelp {
		t.Errorf("expected flag.ErrHelp for unknown subcommand, got %v", err)
	}
}

func TestIssuesCommand_Structure(t *testing.T) {
	cmd := IssuesCommand()
	if cmd.Name != "issues" {
		t.Errorf("expected command name 'issues', got %q", cmd.Name)
	}
	if cmd.UsageFunc == nil {
		t.Error("expected UsageFunc to be set")
	}
	if !strings.Contains(cmd.ShortUsage, "--package") {
		t.Error("expected ShortUsage to mention --package")
	}
}

func TestIssuesCommand_Flags(t *testing.T) {
	cmd := IssuesCommand()
	expectedFlags := []string{"package", "filter", "order-by", "page-size", "paginate", "output", "pretty"}
	for _, name := range expectedFlags {
		f := cmd.FlagSet.Lookup(name)
		if f == nil {
			t.Errorf("expected flag --%s to be registered", name)
		}
	}
}

func TestIssuesCommand_MissingPackage(t *testing.T) {
	// Clear env to ensure no fallback package is set.
	origPkg := os.Getenv("GPLAY_PACKAGE_NAME")
	os.Unsetenv("GPLAY_PACKAGE_NAME")
	defer func() {
		if origPkg != "" {
			os.Setenv("GPLAY_PACKAGE_NAME", origPkg)
		}
	}()

	// Also clear config path to avoid loading a config with a package name.
	origCfg := os.Getenv("GPLAY_CONFIG_PATH")
	os.Setenv("GPLAY_CONFIG_PATH", "/tmp/nonexistent-gplay-config.json")
	defer func() {
		if origCfg != "" {
			os.Setenv("GPLAY_CONFIG_PATH", origCfg)
		} else {
			os.Unsetenv("GPLAY_CONFIG_PATH")
		}
	}()

	// Also unset auth env vars to ensure auth failure path.
	origSA := os.Getenv("GPLAY_SERVICE_ACCOUNT_JSON")
	os.Unsetenv("GPLAY_SERVICE_ACCOUNT_JSON")
	origOAuth := os.Getenv("GPLAY_OAUTH_TOKEN_PATH")
	os.Unsetenv("GPLAY_OAUTH_TOKEN_PATH")
	defer func() {
		if origSA != "" {
			os.Setenv("GPLAY_SERVICE_ACCOUNT_JSON", origSA)
		}
		if origOAuth != "" {
			os.Setenv("GPLAY_OAUTH_TOKEN_PATH", origOAuth)
		}
	}()

	cmd := IssuesCommand()
	err := cmd.ParseAndRun(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error for missing --package, got nil")
	}
	// The error could be either auth failure or missing package depending
	// on which happens first. Both are valid failures.
	errMsg := err.Error()
	if !strings.Contains(errMsg, "--package") && !strings.Contains(errMsg, "auth") && !strings.Contains(errMsg, "credentials") {
		t.Errorf("expected error about --package or auth, got: %s", errMsg)
	}
}

func TestIssuesCommand_InvalidOutputFormat(t *testing.T) {
	cmd := IssuesCommand()
	err := cmd.ParseAndRun(context.Background(), []string{"--output", "xml"})
	// This may fail on output validation or auth - both are acceptable.
	// We just verify it doesn't succeed.
	if err == nil {
		t.Fatal("expected error for invalid output format, got nil")
	}
}

func TestIssuesCommand_PrettyWithTable(t *testing.T) {
	cmd := IssuesCommand()
	err := cmd.ParseAndRun(context.Background(), []string{"--output", "table", "--pretty"})
	if err == nil {
		t.Fatal("expected error for --pretty with table output, got nil")
	}
	if !strings.Contains(err.Error(), "--pretty is only valid with JSON output") {
		t.Errorf("expected --pretty validation error, got: %s", err.Error())
	}
}

func TestReportsCommand_Structure(t *testing.T) {
	cmd := ReportsCommand()
	if cmd.Name != "reports" {
		t.Errorf("expected command name 'reports', got %q", cmd.Name)
	}
	if cmd.UsageFunc == nil {
		t.Error("expected UsageFunc to be set")
	}
	if !strings.Contains(cmd.ShortUsage, "--package") {
		t.Error("expected ShortUsage to mention --package")
	}
}

func TestReportsCommand_Flags(t *testing.T) {
	cmd := ReportsCommand()
	expectedFlags := []string{"package", "filter", "page-size", "paginate", "output", "pretty"}
	for _, name := range expectedFlags {
		f := cmd.FlagSet.Lookup(name)
		if f == nil {
			t.Errorf("expected flag --%s to be registered", name)
		}
	}
}

func TestReportsCommand_MissingPackage(t *testing.T) {
	// Clear env to ensure no fallback package is set.
	origPkg := os.Getenv("GPLAY_PACKAGE_NAME")
	os.Unsetenv("GPLAY_PACKAGE_NAME")
	defer func() {
		if origPkg != "" {
			os.Setenv("GPLAY_PACKAGE_NAME", origPkg)
		}
	}()

	origCfg := os.Getenv("GPLAY_CONFIG_PATH")
	os.Setenv("GPLAY_CONFIG_PATH", "/tmp/nonexistent-gplay-config.json")
	defer func() {
		if origCfg != "" {
			os.Setenv("GPLAY_CONFIG_PATH", origCfg)
		} else {
			os.Unsetenv("GPLAY_CONFIG_PATH")
		}
	}()

	origSA := os.Getenv("GPLAY_SERVICE_ACCOUNT_JSON")
	os.Unsetenv("GPLAY_SERVICE_ACCOUNT_JSON")
	origOAuth := os.Getenv("GPLAY_OAUTH_TOKEN_PATH")
	os.Unsetenv("GPLAY_OAUTH_TOKEN_PATH")
	defer func() {
		if origSA != "" {
			os.Setenv("GPLAY_SERVICE_ACCOUNT_JSON", origSA)
		}
		if origOAuth != "" {
			os.Setenv("GPLAY_OAUTH_TOKEN_PATH", origOAuth)
		}
	}()

	cmd := ReportsCommand()
	err := cmd.ParseAndRun(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error for missing --package, got nil")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "--package") && !strings.Contains(errMsg, "auth") && !strings.Contains(errMsg, "credentials") {
		t.Errorf("expected error about --package or auth, got: %s", errMsg)
	}
}

func TestReportsCommand_PrettyWithMarkdown(t *testing.T) {
	cmd := ReportsCommand()
	err := cmd.ParseAndRun(context.Background(), []string{"--output", "markdown", "--pretty"})
	if err == nil {
		t.Fatal("expected error for --pretty with markdown output, got nil")
	}
	if !strings.Contains(err.Error(), "--pretty is only valid with JSON output") {
		t.Errorf("expected --pretty validation error, got: %s", err.Error())
	}
}

func TestReportsCommand_NoOrderByFlag(t *testing.T) {
	// Reports command should NOT have an --order-by flag (unlike issues).
	cmd := ReportsCommand()
	f := cmd.FlagSet.Lookup("order-by")
	if f != nil {
		t.Error("reports command should not have --order-by flag")
	}
}

func TestIssuesCommand_HasOrderByFlag(t *testing.T) {
	// Issues command SHOULD have an --order-by flag.
	cmd := IssuesCommand()
	f := cmd.FlagSet.Lookup("order-by")
	if f == nil {
		t.Error("issues command should have --order-by flag")
	}
}

func TestIssuesCommand_DefaultPageSize(t *testing.T) {
	cmd := IssuesCommand()
	f := cmd.FlagSet.Lookup("page-size")
	if f == nil {
		t.Fatal("expected page-size flag")
	}
	if f.DefValue != "50" {
		t.Errorf("expected default page-size 50, got %s", f.DefValue)
	}
}

func TestReportsCommand_DefaultPageSize(t *testing.T) {
	cmd := ReportsCommand()
	f := cmd.FlagSet.Lookup("page-size")
	if f == nil {
		t.Fatal("expected page-size flag")
	}
	if f.DefValue != "50" {
		t.Errorf("expected default page-size 50, got %s", f.DefValue)
	}
}
