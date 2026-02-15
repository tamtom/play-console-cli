package users

import (
	"context"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// --- UsersCommand tests ---

func TestUsersCommand_Name(t *testing.T) {
	cmd := UsersCommand()
	if cmd.Name != "users" {
		t.Errorf("Name = %q, want %q", cmd.Name, "users")
	}
}

func TestUsersCommand_HasFourSubcommands(t *testing.T) {
	cmd := UsersCommand()
	if len(cmd.Subcommands) != 4 {
		t.Fatalf("expected 4 subcommands, got %d", len(cmd.Subcommands))
	}

	want := map[string]bool{"list": true, "create": true, "update": true, "delete": true}
	for _, sub := range cmd.Subcommands {
		if !want[sub.Name] {
			t.Errorf("unexpected subcommand %q", sub.Name)
		}
		delete(want, sub.Name)
	}
	for name := range want {
		t.Errorf("missing subcommand %q", name)
	}
}

func TestUsersCommand_UsageFunc(t *testing.T) {
	cmd := UsersCommand()
	if cmd.UsageFunc == nil {
		t.Fatal("UsageFunc should be set")
	}
}

func TestUsersCommand_ShortHelp(t *testing.T) {
	cmd := UsersCommand()
	if cmd.ShortHelp == "" {
		t.Error("ShortHelp should not be empty")
	}
}

func TestUsersCommand_LongHelp(t *testing.T) {
	cmd := UsersCommand()
	if cmd.LongHelp == "" {
		t.Error("LongHelp should not be empty")
	}
}

// --- ListCommand tests ---

func TestListCommand_Name(t *testing.T) {
	cmd := ListCommand()
	if cmd.Name != "list" {
		t.Errorf("Name = %q, want %q", cmd.Name, "list")
	}
}

func TestListCommand_UsageFunc(t *testing.T) {
	cmd := ListCommand()
	if cmd.UsageFunc == nil {
		t.Fatal("UsageFunc should be set")
	}
	// Verify it produces the same output as shared.DefaultUsageFunc
	expected := shared.DefaultUsageFunc(cmd)
	got := cmd.UsageFunc(cmd)
	if got != expected {
		t.Error("UsageFunc should match shared.DefaultUsageFunc")
	}
}

func TestListCommand_RequiresDeveloper(t *testing.T) {
	cmd := ListCommand()
	cmd.FlagSet.Parse([]string{})
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error for missing --developer")
	}
	if !strings.Contains(err.Error(), "--developer") {
		t.Errorf("error should mention --developer, got: %v", err)
	}
}

func TestListCommand_RequiresDeveloper_Whitespace(t *testing.T) {
	cmd := ListCommand()
	cmd.FlagSet.Parse([]string{"--developer", "   "})
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error for whitespace-only --developer")
	}
	if !strings.Contains(err.Error(), "--developer") {
		t.Errorf("error should mention --developer, got: %v", err)
	}
}

func TestListCommand_HasExpectedFlags(t *testing.T) {
	cmd := ListCommand()
	flags := []string{"developer", "page-size", "paginate", "output", "pretty"}
	for _, name := range flags {
		if cmd.FlagSet.Lookup(name) == nil {
			t.Errorf("missing flag --%s", name)
		}
	}
}

func TestListCommand_InvalidOutput(t *testing.T) {
	cmd := ListCommand()
	cmd.FlagSet.Parse([]string{"--developer", "12345", "--output", "xml"})
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

// --- CreateCommand tests ---

func TestCreateCommand_Name(t *testing.T) {
	cmd := CreateCommand()
	if cmd.Name != "create" {
		t.Errorf("Name = %q, want %q", cmd.Name, "create")
	}
}

func TestCreateCommand_UsageFunc(t *testing.T) {
	cmd := CreateCommand()
	if cmd.UsageFunc == nil {
		t.Fatal("UsageFunc should be set")
	}
	expected := shared.DefaultUsageFunc(cmd)
	got := cmd.UsageFunc(cmd)
	if got != expected {
		t.Error("UsageFunc should match shared.DefaultUsageFunc")
	}
}

func TestCreateCommand_RequiresDeveloper(t *testing.T) {
	cmd := CreateCommand()
	cmd.FlagSet.Parse([]string{"--email", "user@example.com", "--json", `{}`})
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error for missing --developer")
	}
	if !strings.Contains(err.Error(), "--developer") {
		t.Errorf("error should mention --developer, got: %v", err)
	}
}

func TestCreateCommand_RequiresEmail(t *testing.T) {
	cmd := CreateCommand()
	cmd.FlagSet.Parse([]string{"--developer", "12345", "--json", `{}`})
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error for missing --email")
	}
	if !strings.Contains(err.Error(), "--email") {
		t.Errorf("error should mention --email, got: %v", err)
	}
}

func TestCreateCommand_RequiresJSON(t *testing.T) {
	cmd := CreateCommand()
	cmd.FlagSet.Parse([]string{"--developer", "12345", "--email", "user@example.com"})
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error for missing --json")
	}
	if !strings.Contains(err.Error(), "--json") {
		t.Errorf("error should mention --json, got: %v", err)
	}
}

func TestCreateCommand_RequiresDeveloper_Whitespace(t *testing.T) {
	cmd := CreateCommand()
	cmd.FlagSet.Parse([]string{"--developer", "  ", "--email", "user@example.com", "--json", `{}`})
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error for whitespace-only --developer")
	}
	if !strings.Contains(err.Error(), "--developer") {
		t.Errorf("error should mention --developer, got: %v", err)
	}
}

func TestCreateCommand_RequiresEmail_Whitespace(t *testing.T) {
	cmd := CreateCommand()
	cmd.FlagSet.Parse([]string{"--developer", "12345", "--email", "  ", "--json", `{}`})
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error for whitespace-only --email")
	}
	if !strings.Contains(err.Error(), "--email") {
		t.Errorf("error should mention --email, got: %v", err)
	}
}

func TestCreateCommand_RequiresJSON_Whitespace(t *testing.T) {
	cmd := CreateCommand()
	cmd.FlagSet.Parse([]string{"--developer", "12345", "--email", "user@example.com", "--json", "  "})
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error for whitespace-only --json")
	}
	if !strings.Contains(err.Error(), "--json") {
		t.Errorf("error should mention --json, got: %v", err)
	}
}

func TestCreateCommand_HasExpectedFlags(t *testing.T) {
	cmd := CreateCommand()
	flags := []string{"developer", "email", "json", "output", "pretty"}
	for _, name := range flags {
		if cmd.FlagSet.Lookup(name) == nil {
			t.Errorf("missing flag --%s", name)
		}
	}
}

func TestCreateCommand_InvalidOutput(t *testing.T) {
	cmd := CreateCommand()
	cmd.FlagSet.Parse([]string{"--developer", "12345", "--email", "user@example.com", "--json", `{}`, "--output", "yaml"})
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

func TestCreateCommand_LongHelp(t *testing.T) {
	cmd := CreateCommand()
	if cmd.LongHelp == "" {
		t.Error("LongHelp should not be empty for create command")
	}
	if !strings.Contains(cmd.LongHelp, "developerAccountPermissions") {
		t.Error("LongHelp should document available permissions")
	}
}

// --- UpdateCommand tests ---

func TestUpdateCommand_Name(t *testing.T) {
	cmd := UpdateCommand()
	if cmd.Name != "update" {
		t.Errorf("Name = %q, want %q", cmd.Name, "update")
	}
}

func TestUpdateCommand_UsageFunc(t *testing.T) {
	cmd := UpdateCommand()
	if cmd.UsageFunc == nil {
		t.Fatal("UsageFunc should be set")
	}
	expected := shared.DefaultUsageFunc(cmd)
	got := cmd.UsageFunc(cmd)
	if got != expected {
		t.Error("UsageFunc should match shared.DefaultUsageFunc")
	}
}

func TestUpdateCommand_RequiresDeveloper(t *testing.T) {
	cmd := UpdateCommand()
	cmd.FlagSet.Parse([]string{"--email", "user@example.com", "--json", `{}`})
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error for missing --developer")
	}
	if !strings.Contains(err.Error(), "--developer") {
		t.Errorf("error should mention --developer, got: %v", err)
	}
}

func TestUpdateCommand_RequiresEmail(t *testing.T) {
	cmd := UpdateCommand()
	cmd.FlagSet.Parse([]string{"--developer", "12345", "--json", `{}`})
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error for missing --email")
	}
	if !strings.Contains(err.Error(), "--email") {
		t.Errorf("error should mention --email, got: %v", err)
	}
}

func TestUpdateCommand_RequiresJSON(t *testing.T) {
	cmd := UpdateCommand()
	cmd.FlagSet.Parse([]string{"--developer", "12345", "--email", "user@example.com"})
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error for missing --json")
	}
	if !strings.Contains(err.Error(), "--json") {
		t.Errorf("error should mention --json, got: %v", err)
	}
}

func TestUpdateCommand_RequiresDeveloper_Whitespace(t *testing.T) {
	cmd := UpdateCommand()
	cmd.FlagSet.Parse([]string{"--developer", "  ", "--email", "user@example.com", "--json", `{}`})
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error for whitespace-only --developer")
	}
	if !strings.Contains(err.Error(), "--developer") {
		t.Errorf("error should mention --developer, got: %v", err)
	}
}

func TestUpdateCommand_HasExpectedFlags(t *testing.T) {
	cmd := UpdateCommand()
	flags := []string{"developer", "email", "json", "update-mask", "output", "pretty"}
	for _, name := range flags {
		if cmd.FlagSet.Lookup(name) == nil {
			t.Errorf("missing flag --%s", name)
		}
	}
}

func TestUpdateCommand_InvalidOutput(t *testing.T) {
	cmd := UpdateCommand()
	cmd.FlagSet.Parse([]string{"--developer", "12345", "--email", "user@example.com", "--json", `{}`, "--output", "csv"})
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

// --- DeleteCommand tests ---

func TestDeleteCommand_Name(t *testing.T) {
	cmd := DeleteCommand()
	if cmd.Name != "delete" {
		t.Errorf("Name = %q, want %q", cmd.Name, "delete")
	}
}

func TestDeleteCommand_UsageFunc(t *testing.T) {
	cmd := DeleteCommand()
	if cmd.UsageFunc == nil {
		t.Fatal("UsageFunc should be set")
	}
	expected := shared.DefaultUsageFunc(cmd)
	got := cmd.UsageFunc(cmd)
	if got != expected {
		t.Error("UsageFunc should match shared.DefaultUsageFunc")
	}
}

func TestDeleteCommand_RequiresDeveloper(t *testing.T) {
	cmd := DeleteCommand()
	cmd.FlagSet.Parse([]string{"--email", "user@example.com", "--confirm"})
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error for missing --developer")
	}
	if !strings.Contains(err.Error(), "--developer") {
		t.Errorf("error should mention --developer, got: %v", err)
	}
}

func TestDeleteCommand_RequiresEmail(t *testing.T) {
	cmd := DeleteCommand()
	cmd.FlagSet.Parse([]string{"--developer", "12345", "--confirm"})
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error for missing --email")
	}
	if !strings.Contains(err.Error(), "--email") {
		t.Errorf("error should mention --email, got: %v", err)
	}
}

func TestDeleteCommand_RequiresConfirm(t *testing.T) {
	cmd := DeleteCommand()
	cmd.FlagSet.Parse([]string{"--developer", "12345", "--email", "user@example.com"})
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error for missing --confirm")
	}
	if !strings.Contains(err.Error(), "--confirm") {
		t.Errorf("error should mention --confirm, got: %v", err)
	}
}

func TestDeleteCommand_RequiresDeveloper_Whitespace(t *testing.T) {
	cmd := DeleteCommand()
	cmd.FlagSet.Parse([]string{"--developer", "  ", "--email", "user@example.com", "--confirm"})
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error for whitespace-only --developer")
	}
	if !strings.Contains(err.Error(), "--developer") {
		t.Errorf("error should mention --developer, got: %v", err)
	}
}

func TestDeleteCommand_RequiresEmail_Whitespace(t *testing.T) {
	cmd := DeleteCommand()
	cmd.FlagSet.Parse([]string{"--developer", "12345", "--email", "  ", "--confirm"})
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error for whitespace-only --email")
	}
	if !strings.Contains(err.Error(), "--email") {
		t.Errorf("error should mention --email, got: %v", err)
	}
}

func TestDeleteCommand_HasExpectedFlags(t *testing.T) {
	cmd := DeleteCommand()
	flags := []string{"developer", "email", "confirm", "output", "pretty"}
	for _, name := range flags {
		if cmd.FlagSet.Lookup(name) == nil {
			t.Errorf("missing flag --%s", name)
		}
	}
}

func TestDeleteCommand_InvalidOutput(t *testing.T) {
	cmd := DeleteCommand()
	cmd.FlagSet.Parse([]string{"--developer", "12345", "--email", "user@example.com", "--confirm", "--output", "xml"})
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

// --- Validation order tests ---

func TestCreateCommand_ValidationOrder(t *testing.T) {
	// When all flags are missing, --developer should be reported first
	cmd := CreateCommand()
	cmd.FlagSet.Parse([]string{})
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--developer") {
		t.Errorf("first validation error should be --developer, got: %v", err)
	}
}

func TestUpdateCommand_ValidationOrder(t *testing.T) {
	cmd := UpdateCommand()
	cmd.FlagSet.Parse([]string{})
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--developer") {
		t.Errorf("first validation error should be --developer, got: %v", err)
	}
}

func TestDeleteCommand_ValidationOrder(t *testing.T) {
	cmd := DeleteCommand()
	cmd.FlagSet.Parse([]string{})
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--developer") {
		t.Errorf("first validation error should be --developer, got: %v", err)
	}
}

// --- All subcommands have UsageFunc set ---

func TestAllSubcommands_HaveUsageFunc(t *testing.T) {
	cmd := UsersCommand()
	for _, sub := range cmd.Subcommands {
		if sub.UsageFunc == nil {
			t.Errorf("subcommand %q missing UsageFunc", sub.Name)
		}
	}
}

// --- All subcommands have ShortHelp ---

func TestAllSubcommands_HaveShortHelp(t *testing.T) {
	cmd := UsersCommand()
	for _, sub := range cmd.Subcommands {
		if sub.ShortHelp == "" {
			t.Errorf("subcommand %q missing ShortHelp", sub.Name)
		}
	}
}

// --- All subcommands have ShortUsage ---

func TestAllSubcommands_HaveShortUsage(t *testing.T) {
	cmd := UsersCommand()
	for _, sub := range cmd.Subcommands {
		if sub.ShortUsage == "" {
			t.Errorf("subcommand %q missing ShortUsage", sub.Name)
		}
	}
}
