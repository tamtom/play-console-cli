package shared

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func TestDeprecatedAliasLeafCommand_ShortHelpPrefix(t *testing.T) {
	original := &ffcli.Command{
		Name:       "new-cmd",
		ShortUsage: "gplay new-cmd [flags]",
		ShortHelp:  "Does something useful",
		Exec: func(ctx context.Context, args []string) error {
			return nil
		},
	}

	alias := DeprecatedAliasLeafCommand(original, "old-cmd", "gplay new-cmd")

	if alias.Name != "old-cmd" {
		t.Errorf("Name = %q; want %q", alias.Name, "old-cmd")
	}
	if !strings.HasPrefix(alias.ShortHelp, "DEPRECATED:") {
		t.Errorf("ShortHelp = %q; want prefix 'DEPRECATED:'", alias.ShortHelp)
	}
	if !strings.Contains(alias.ShortHelp, "gplay new-cmd") {
		t.Errorf("ShortHelp = %q; want it to contain 'gplay new-cmd'", alias.ShortHelp)
	}
}

func TestDeprecatedAliasLeafCommand_ExecPrintsWarningAndDelegates(t *testing.T) {
	called := false
	original := &ffcli.Command{
		Name:       "new-cmd",
		ShortUsage: "gplay new-cmd [flags]",
		ShortHelp:  "Does something useful",
		Exec: func(ctx context.Context, args []string) error {
			called = true
			return nil
		},
	}

	alias := DeprecatedAliasLeafCommand(original, "old-cmd", "gplay new-cmd")

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := alias.Exec(context.Background(), nil)
	w.Close()
	os.Stderr = oldStderr

	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if !called {
		t.Error("original Exec was not called")
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	stderr := buf.String()

	if !strings.Contains(stderr, "deprecated") {
		t.Errorf("stderr = %q; want deprecation warning", stderr)
	}
	if !strings.Contains(stderr, "gplay new-cmd") {
		t.Errorf("stderr = %q; want it to mention 'gplay new-cmd'", stderr)
	}
}

func TestDeprecatedAliasLeafCommand_NilExec(t *testing.T) {
	original := &ffcli.Command{
		Name:       "new-cmd",
		ShortUsage: "gplay new-cmd [flags]",
		ShortHelp:  "Does something useful",
		Exec:       nil,
	}

	alias := DeprecatedAliasLeafCommand(original, "old-cmd", "gplay new-cmd")

	// Capture stderr to suppress warning output
	oldStderr := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w

	err := alias.Exec(context.Background(), nil)
	w.Close()
	os.Stderr = oldStderr

	if err != nil {
		t.Fatalf("Exec returned error when original.Exec is nil: %v", err)
	}
}

func TestDeprecatedAliasLeafCommand_PropagatesError(t *testing.T) {
	expectedErr := errors.New("something failed")
	original := &ffcli.Command{
		Name:       "new-cmd",
		ShortUsage: "gplay new-cmd [flags]",
		ShortHelp:  "Does something useful",
		Exec: func(ctx context.Context, args []string) error {
			return expectedErr
		},
	}

	alias := DeprecatedAliasLeafCommand(original, "old-cmd", "gplay new-cmd")

	// Capture stderr to suppress warning output
	oldStderr := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w

	err := alias.Exec(context.Background(), nil)
	w.Close()
	os.Stderr = oldStderr

	if !errors.Is(err, expectedErr) {
		t.Errorf("Exec error = %v; want %v", err, expectedErr)
	}
}

func TestVisibleUsageFunc_HidesDeprecatedCommands(t *testing.T) {
	parent := &ffcli.Command{
		Name:       "gplay",
		ShortUsage: "gplay <subcommand>",
		ShortHelp:  "Google Play CLI",
		FlagSet:    flag.NewFlagSet("gplay", flag.ContinueOnError),
		Subcommands: []*ffcli.Command{
			{
				Name:      "visible-cmd",
				ShortHelp: "A visible command",
			},
			{
				Name:      "old-cmd",
				ShortHelp: "DEPRECATED: use `gplay visible-cmd` instead.",
			},
			{
				Name:      "another-visible",
				ShortHelp: "Another visible command",
			},
		},
	}

	output := VisibleUsageFunc(parent)

	if strings.Contains(output, "old-cmd") {
		t.Errorf("output should not contain deprecated 'old-cmd'; got:\n%s", output)
	}
	if !strings.Contains(output, "visible-cmd") {
		t.Errorf("output should contain 'visible-cmd'; got:\n%s", output)
	}
	if !strings.Contains(output, "another-visible") {
		t.Errorf("output should contain 'another-visible'; got:\n%s", output)
	}
}

func TestVisibleUsageFunc_NoSubcommands(t *testing.T) {
	cmd := &ffcli.Command{
		Name:       "gplay",
		ShortUsage: "gplay [flags]",
		ShortHelp:  "Google Play CLI",
		FlagSet:    flag.NewFlagSet("gplay", flag.ContinueOnError),
	}

	// Should not panic
	output := VisibleUsageFunc(cmd)
	if output == "" {
		t.Error("output should not be empty")
	}
}
