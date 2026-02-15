package apps

import (
	"bytes"
	"context"
	"testing"
)

func TestAppsCommand_Help(t *testing.T) {
	cmd := AppsCommand()
	if cmd.Name != "apps" {
		t.Errorf("Name = %q, want %q", cmd.Name, "apps")
	}
	if len(cmd.Subcommands) == 0 {
		t.Error("expected at least one subcommand")
	}
}

func TestListCommand_Flags(t *testing.T) {
	cmd := ListCommand()
	if cmd.Name != "list" {
		t.Errorf("Name = %q, want %q", cmd.Name, "list")
	}
	var buf bytes.Buffer
	_ = buf
}

func TestAppsCommand_UnknownSubcommand(t *testing.T) {
	cmd := AppsCommand()
	err := cmd.ParseAndRun(context.Background(), []string{"nonexistent"})
	if err == nil {
		t.Error("expected error for unknown subcommand")
	}
}
