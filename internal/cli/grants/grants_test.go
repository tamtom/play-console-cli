package grants

import (
	"context"
	"flag"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// --- GrantsCommand tests ---

func TestGrantsCommand_Name(t *testing.T) {
	cmd := GrantsCommand()
	if cmd.Name != "grants" {
		t.Errorf("Name = %q, want %q", cmd.Name, "grants")
	}
}

func TestGrantsCommand_HasSubcommands(t *testing.T) {
	cmd := GrantsCommand()
	if len(cmd.Subcommands) == 0 {
		t.Fatal("expected subcommands, got none")
	}

	names := make(map[string]bool)
	for _, sub := range cmd.Subcommands {
		names[sub.Name] = true
	}

	for _, want := range []string{"create", "update", "delete"} {
		if !names[want] {
			t.Errorf("missing subcommand %q", want)
		}
	}
}

func TestGrantsCommand_UsageFunc(t *testing.T) {
	cmd := GrantsCommand()
	if cmd.UsageFunc == nil {
		t.Fatal("UsageFunc is nil, want shared.DefaultUsageFunc")
	}
	// Verify it produces the same output as shared.DefaultUsageFunc
	got := cmd.UsageFunc(cmd)
	want := shared.DefaultUsageFunc(cmd)
	if got != want {
		t.Error("UsageFunc does not match shared.DefaultUsageFunc")
	}
}

func TestGrantsCommand_ExecNoArgs(t *testing.T) {
	cmd := GrantsCommand()
	err := cmd.Exec(context.Background(), []string{})
	if err != flag.ErrHelp {
		t.Errorf("Exec() = %v, want flag.ErrHelp", err)
	}
}

func TestGrantsCommand_ExecWithArgs(t *testing.T) {
	cmd := GrantsCommand()
	err := cmd.Exec(context.Background(), []string{"unknown"})
	if err != flag.ErrHelp {
		t.Errorf("Exec() = %v, want flag.ErrHelp", err)
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
		t.Fatal("UsageFunc is nil, want shared.DefaultUsageFunc")
	}
	got := cmd.UsageFunc(cmd)
	want := shared.DefaultUsageFunc(cmd)
	if got != want {
		t.Error("UsageFunc does not match shared.DefaultUsageFunc")
	}
}

func TestListCommand_ReturnsAPILimitationError(t *testing.T) {
	cmd := ListCommand()
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "not implemented") {
		t.Errorf("error should mention 'not implemented', got: %v", msg)
	}
	if !strings.Contains(msg, "List method") {
		t.Errorf("error should mention 'List method', got: %v", msg)
	}
	if !strings.Contains(msg, "API v3") {
		t.Errorf("error should mention 'API v3', got: %v", msg)
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
		t.Fatal("UsageFunc is nil, want shared.DefaultUsageFunc")
	}
	got := cmd.UsageFunc(cmd)
	want := shared.DefaultUsageFunc(cmd)
	if got != want {
		t.Error("UsageFunc does not match shared.DefaultUsageFunc")
	}
}

func TestCreateCommand_RequiresDeveloper(t *testing.T) {
	cmd := CreateCommand()
	if err := cmd.FlagSet.Parse([]string{"--email", "test@example.com", "--json", `{"appLevelPermissions":[]}`}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error without --developer")
	}
	if !strings.Contains(err.Error(), "--developer") {
		t.Errorf("error should mention --developer, got: %v", err)
	}
}

func TestCreateCommand_RequiresEmail(t *testing.T) {
	cmd := CreateCommand()
	if err := cmd.FlagSet.Parse([]string{"--developer", "123", "--json", `{"appLevelPermissions":[]}`}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error without --email")
	}
	if !strings.Contains(err.Error(), "--email") {
		t.Errorf("error should mention --email, got: %v", err)
	}
}

func TestCreateCommand_RequiresJSON(t *testing.T) {
	cmd := CreateCommand()
	if err := cmd.FlagSet.Parse([]string{"--developer", "123", "--email", "test@example.com"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error without --json")
	}
	if !strings.Contains(err.Error(), "--json") {
		t.Errorf("error should mention --json, got: %v", err)
	}
}

func TestCreateCommand_HasExpectedFlags(t *testing.T) {
	cmd := CreateCommand()
	flags := []string{"developer", "email", "package", "json", "output", "pretty"}
	for _, name := range flags {
		f := cmd.FlagSet.Lookup(name)
		if f == nil {
			t.Errorf("missing flag --%s", name)
		}
	}
}

func TestCreateCommand_DeveloperEmptyString(t *testing.T) {
	cmd := CreateCommand()
	if err := cmd.FlagSet.Parse([]string{"--developer", "  ", "--email", "test@example.com", "--json", `{}`}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error with whitespace-only --developer")
	}
	if !strings.Contains(err.Error(), "--developer") {
		t.Errorf("error should mention --developer, got: %v", err)
	}
}

func TestCreateCommand_EmailEmptyString(t *testing.T) {
	cmd := CreateCommand()
	if err := cmd.FlagSet.Parse([]string{"--developer", "123", "--email", "  ", "--json", `{}`}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error with whitespace-only --email")
	}
	if !strings.Contains(err.Error(), "--email") {
		t.Errorf("error should mention --email, got: %v", err)
	}
}

func TestCreateCommand_JSONEmptyString(t *testing.T) {
	cmd := CreateCommand()
	if err := cmd.FlagSet.Parse([]string{"--developer", "123", "--email", "test@example.com", "--json", "  "}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error with whitespace-only --json")
	}
	if !strings.Contains(err.Error(), "--json") {
		t.Errorf("error should mention --json, got: %v", err)
	}
}

func TestCreateCommand_InvalidOutputFormat(t *testing.T) {
	cmd := CreateCommand()
	if err := cmd.FlagSet.Parse([]string{"--developer", "123", "--email", "test@example.com", "--json", `{}`, "--output", "table", "--pretty"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error with --pretty and --output table")
	}
	if !strings.Contains(err.Error(), "--pretty") {
		t.Errorf("error should mention --pretty, got: %v", err)
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
		t.Fatal("UsageFunc is nil, want shared.DefaultUsageFunc")
	}
	got := cmd.UsageFunc(cmd)
	want := shared.DefaultUsageFunc(cmd)
	if got != want {
		t.Error("UsageFunc does not match shared.DefaultUsageFunc")
	}
}

func TestUpdateCommand_RequiresDeveloper(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--email", "test@example.com", "--json", `{"appLevelPermissions":[]}`}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error without --developer")
	}
	if !strings.Contains(err.Error(), "--developer") {
		t.Errorf("error should mention --developer, got: %v", err)
	}
}

func TestUpdateCommand_RequiresEmail(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--developer", "123", "--json", `{"appLevelPermissions":[]}`}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error without --email")
	}
	if !strings.Contains(err.Error(), "--email") {
		t.Errorf("error should mention --email, got: %v", err)
	}
}

func TestUpdateCommand_RequiresJSON(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--developer", "123", "--email", "test@example.com"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error without --json")
	}
	if !strings.Contains(err.Error(), "--json") {
		t.Errorf("error should mention --json, got: %v", err)
	}
}

func TestUpdateCommand_HasExpectedFlags(t *testing.T) {
	cmd := UpdateCommand()
	flags := []string{"developer", "email", "package", "json", "update-mask", "output", "pretty"}
	for _, name := range flags {
		f := cmd.FlagSet.Lookup(name)
		if f == nil {
			t.Errorf("missing flag --%s", name)
		}
	}
}

func TestUpdateCommand_DeveloperEmptyString(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--developer", "  ", "--email", "test@example.com", "--json", `{}`}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error with whitespace-only --developer")
	}
	if !strings.Contains(err.Error(), "--developer") {
		t.Errorf("error should mention --developer, got: %v", err)
	}
}

func TestUpdateCommand_EmailEmptyString(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--developer", "123", "--email", "  ", "--json", `{}`}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error with whitespace-only --email")
	}
	if !strings.Contains(err.Error(), "--email") {
		t.Errorf("error should mention --email, got: %v", err)
	}
}

func TestUpdateCommand_JSONEmptyString(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--developer", "123", "--email", "test@example.com", "--json", "  "}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error with whitespace-only --json")
	}
	if !strings.Contains(err.Error(), "--json") {
		t.Errorf("error should mention --json, got: %v", err)
	}
}

func TestUpdateCommand_InvalidOutputFormat(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--developer", "123", "--email", "test@example.com", "--json", `{}`, "--output", "markdown", "--pretty"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error with --pretty and --output markdown")
	}
	if !strings.Contains(err.Error(), "--pretty") {
		t.Errorf("error should mention --pretty, got: %v", err)
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
		t.Fatal("UsageFunc is nil, want shared.DefaultUsageFunc")
	}
	got := cmd.UsageFunc(cmd)
	want := shared.DefaultUsageFunc(cmd)
	if got != want {
		t.Error("UsageFunc does not match shared.DefaultUsageFunc")
	}
}

func TestDeleteCommand_RequiresConfirm(t *testing.T) {
	cmd := DeleteCommand()
	if err := cmd.FlagSet.Parse([]string{"--developer", "123", "--email", "test@example.com", "--package", "com.test"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error without --confirm")
	}
	if !strings.Contains(err.Error(), "--confirm") {
		t.Errorf("error should mention --confirm, got: %v", err)
	}
}

func TestDeleteCommand_RequiresDeveloper(t *testing.T) {
	cmd := DeleteCommand()
	if err := cmd.FlagSet.Parse([]string{"--email", "test@example.com", "--confirm"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error without --developer")
	}
	if !strings.Contains(err.Error(), "--developer") {
		t.Errorf("error should mention --developer, got: %v", err)
	}
}

func TestDeleteCommand_RequiresEmail(t *testing.T) {
	cmd := DeleteCommand()
	if err := cmd.FlagSet.Parse([]string{"--developer", "123", "--confirm"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error without --email")
	}
	if !strings.Contains(err.Error(), "--email") {
		t.Errorf("error should mention --email, got: %v", err)
	}
}

func TestDeleteCommand_HasExpectedFlags(t *testing.T) {
	cmd := DeleteCommand()
	flags := []string{"developer", "email", "package", "confirm", "output", "pretty"}
	for _, name := range flags {
		f := cmd.FlagSet.Lookup(name)
		if f == nil {
			t.Errorf("missing flag --%s", name)
		}
	}
}

func TestDeleteCommand_DeveloperEmptyString(t *testing.T) {
	cmd := DeleteCommand()
	if err := cmd.FlagSet.Parse([]string{"--developer", "  ", "--email", "test@example.com", "--confirm"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error with whitespace-only --developer")
	}
	if !strings.Contains(err.Error(), "--developer") {
		t.Errorf("error should mention --developer, got: %v", err)
	}
}

func TestDeleteCommand_EmailEmptyString(t *testing.T) {
	cmd := DeleteCommand()
	if err := cmd.FlagSet.Parse([]string{"--developer", "123", "--email", "  ", "--confirm"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error with whitespace-only --email")
	}
	if !strings.Contains(err.Error(), "--email") {
		t.Errorf("error should mention --email, got: %v", err)
	}
}

func TestDeleteCommand_InvalidOutputFormat(t *testing.T) {
	cmd := DeleteCommand()
	if err := cmd.FlagSet.Parse([]string{"--developer", "123", "--email", "test@example.com", "--confirm", "--output", "table", "--pretty"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error with --pretty and --output table")
	}
	if !strings.Contains(err.Error(), "--pretty") {
		t.Errorf("error should mention --pretty, got: %v", err)
	}
}

func TestDeleteCommand_DeveloperValidatedBeforeConfirm(t *testing.T) {
	// Without --developer and without --confirm, --developer should be validated first
	cmd := DeleteCommand()
	if err := cmd.FlagSet.Parse([]string{"--email", "test@example.com"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--developer") {
		t.Errorf("expected --developer error first, got: %v", err)
	}
}

// --- Validation order tests ---

func TestCreateCommand_ValidationOrder(t *testing.T) {
	// Validation order: output flags -> developer -> email -> json
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "invalid output first",
			args:    []string{"--output", "table", "--pretty"},
			wantErr: "--pretty",
		},
		{
			name:    "developer before email",
			args:    []string{},
			wantErr: "--developer",
		},
		{
			name:    "email before json",
			args:    []string{"--developer", "123"},
			wantErr: "--email",
		},
		{
			name:    "json required",
			args:    []string{"--developer", "123", "--email", "test@example.com"},
			wantErr: "--json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := CreateCommand()
			if err := cmd.FlagSet.Parse(tt.args); err != nil {
				t.Fatal(err)
			}
			err := cmd.Exec(context.Background(), []string{})
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestUpdateCommand_ValidationOrder(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "invalid output first",
			args:    []string{"--output", "markdown", "--pretty"},
			wantErr: "--pretty",
		},
		{
			name:    "developer before email",
			args:    []string{},
			wantErr: "--developer",
		},
		{
			name:    "email before json",
			args:    []string{"--developer", "123"},
			wantErr: "--email",
		},
		{
			name:    "json required",
			args:    []string{"--developer", "123", "--email", "test@example.com"},
			wantErr: "--json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := UpdateCommand()
			if err := cmd.FlagSet.Parse(tt.args); err != nil {
				t.Fatal(err)
			}
			err := cmd.Exec(context.Background(), []string{})
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestDeleteCommand_ValidationOrder(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "invalid output first",
			args:    []string{"--output", "table", "--pretty", "--confirm"},
			wantErr: "--pretty",
		},
		{
			name:    "developer before email",
			args:    []string{"--confirm"},
			wantErr: "--developer",
		},
		{
			name:    "email before confirm",
			args:    []string{"--developer", "123"},
			wantErr: "--email",
		},
		{
			name:    "confirm required",
			args:    []string{"--developer", "123", "--email", "test@example.com"},
			wantErr: "--confirm",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := DeleteCommand()
			if err := cmd.FlagSet.Parse(tt.args); err != nil {
				t.Fatal(err)
			}
			err := cmd.Exec(context.Background(), []string{})
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}
