package customapps

import (
	"context"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// --- CustomAppsCommand tests ---

func TestCustomAppsCommand_Name(t *testing.T) {
	cmd := CustomAppsCommand()
	if cmd.Name != "custom-apps" {
		t.Errorf("Name = %q, want %q", cmd.Name, "custom-apps")
	}
}

func TestCustomAppsCommand_HasCreateSubcommand(t *testing.T) {
	cmd := CustomAppsCommand()
	if len(cmd.Subcommands) != 1 {
		t.Fatalf("expected 1 subcommand, got %d", len(cmd.Subcommands))
	}
	if cmd.Subcommands[0].Name != "create" {
		t.Errorf("subcommand = %q, want %q", cmd.Subcommands[0].Name, "create")
	}
}

func TestCustomAppsCommand_UsageFunc(t *testing.T) {
	cmd := CustomAppsCommand()
	if cmd.UsageFunc == nil {
		t.Fatal("UsageFunc should be set")
	}
}

func TestCustomAppsCommand_Help(t *testing.T) {
	cmd := CustomAppsCommand()
	if cmd.ShortHelp == "" {
		t.Error("ShortHelp should not be empty")
	}
	if cmd.LongHelp == "" {
		t.Error("LongHelp should not be empty")
	}
}

// --- CreateCommand tests ---

func TestCreateCommand_UsageFunc(t *testing.T) {
	cmd := CreateCommand()
	if cmd.UsageFunc == nil {
		t.Fatal("UsageFunc should be set")
	}
	expected := shared.DefaultUsageFunc(cmd)
	if got := cmd.UsageFunc(cmd); got != expected {
		t.Error("UsageFunc should match shared.DefaultUsageFunc")
	}
}

func execCreate(t *testing.T, args []string) error {
	t.Helper()
	cmd := CreateCommand()
	if err := cmd.FlagSet.Parse(args); err != nil {
		t.Fatalf("parse args: %v", err)
	}
	return cmd.Exec(context.Background(), cmd.FlagSet.Args())
}

func TestCreateCommand_RequiresDeveloper(t *testing.T) {
	err := execCreate(t, []string{"--title", "T", "--apk", "app.apk"})
	if err == nil || !strings.Contains(err.Error(), "--developer") {
		t.Fatalf("expected --developer error, got %v", err)
	}
}

func TestCreateCommand_DeveloperMustBeNumeric(t *testing.T) {
	err := execCreate(t, []string{"--developer", "not-a-number", "--title", "T", "--apk", "app.apk"})
	if err == nil || !strings.Contains(err.Error(), "numeric") {
		t.Fatalf("expected numeric account error, got %v", err)
	}
}

func TestCreateCommand_RequiresTitle(t *testing.T) {
	err := execCreate(t, []string{"--developer", "123", "--apk", "app.apk"})
	if err == nil || !strings.Contains(err.Error(), "--title") {
		t.Fatalf("expected --title error, got %v", err)
	}
}

func TestCreateCommand_RequiresAPK(t *testing.T) {
	err := execCreate(t, []string{"--developer", "123", "--title", "T"})
	if err == nil || !strings.Contains(err.Error(), "--apk") {
		t.Fatalf("expected --apk error, got %v", err)
	}
}

func TestCreateCommand_RejectsBadOutput(t *testing.T) {
	err := execCreate(t, []string{"--developer", "123", "--title", "T", "--apk", "app.apk", "--output", "xml"})
	if err == nil {
		t.Fatal("expected error for invalid --output")
	}
}
