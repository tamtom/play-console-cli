package auth

import (
	"context"
	"errors"
	"flag"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/config"
)

func TestAuthCommand_Name(t *testing.T) {
	cmd := AuthCommand()
	if cmd.Name != "auth" {
		t.Errorf("expected name %q, got %q", "auth", cmd.Name)
	}
}

func TestAuthCommand_ShortHelp(t *testing.T) {
	cmd := AuthCommand()
	if cmd.ShortHelp == "" {
		t.Error("expected non-empty ShortHelp")
	}
}

func TestAuthCommand_UsageFunc(t *testing.T) {
	cmd := AuthCommand()
	if cmd.UsageFunc == nil {
		t.Error("expected UsageFunc to be set")
	}
}

func TestAuthCommand_HasSubcommands(t *testing.T) {
	cmd := AuthCommand()
	if len(cmd.Subcommands) == 0 {
		t.Error("expected subcommands")
	}
}

func TestAuthCommand_SubcommandNames(t *testing.T) {
	cmd := AuthCommand()
	expected := map[string]bool{
		"init":   false,
		"login":  false,
		"switch": false,
		"logout": false,
		"status": false,
		"doctor": false,
	}
	for _, sub := range cmd.Subcommands {
		if _, ok := expected[sub.Name]; ok {
			expected[sub.Name] = true
		} else {
			t.Errorf("unexpected subcommand: %s", sub.Name)
		}
	}
	for name, found := range expected {
		if !found {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

func TestAuthCommand_SubcommandsHaveUsageFunc(t *testing.T) {
	cmd := AuthCommand()
	for _, sub := range cmd.Subcommands {
		if sub.UsageFunc == nil {
			t.Errorf("subcommand %q missing UsageFunc", sub.Name)
		}
	}
}

func TestAuthCommand_SubcommandsHaveShortHelp(t *testing.T) {
	cmd := AuthCommand()
	for _, sub := range cmd.Subcommands {
		if sub.ShortHelp == "" {
			t.Errorf("subcommand %q missing ShortHelp", sub.Name)
		}
	}
}

func TestAuthCommand_NoArgs_ReturnsHelp(t *testing.T) {
	cmd := AuthCommand()
	err := cmd.Exec(context.Background(), nil)
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

func TestAuthCommand_UnknownSubcommand_ReturnsHelp(t *testing.T) {
	cmd := AuthCommand()
	err := cmd.Exec(context.Background(), []string{"nonexistent"})
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

// --- auth login ---

func TestAuthLoginCommand_Name(t *testing.T) {
	cmd := AuthLoginCommand()
	if cmd.Name != "login" {
		t.Errorf("expected name %q, got %q", "login", cmd.Name)
	}
}

func TestAuthLoginCommand_MissingServiceAccount(t *testing.T) {
	cmd := AuthLoginCommand()
	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --service-account")
	}
	if !strings.Contains(err.Error(), "--service-account") {
		t.Errorf("error should mention --service-account, got: %s", err.Error())
	}
}

func TestAuthLoginCommand_WhitespaceServiceAccount(t *testing.T) {
	cmd := AuthLoginCommand()
	if err := cmd.FlagSet.Parse([]string{"--service-account", "   "}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --service-account")
	}
	if !strings.Contains(err.Error(), "--service-account") {
		t.Errorf("error should mention --service-account, got: %s", err.Error())
	}
}

func TestAuthLoginCommand_WhitespaceProfile(t *testing.T) {
	cmd := AuthLoginCommand()
	if err := cmd.FlagSet.Parse([]string{"--service-account", "/path/to/key.json", "--profile", "   "}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --profile")
	}
	if !strings.Contains(err.Error(), "--profile") {
		t.Errorf("error should mention --profile, got: %s", err.Error())
	}
}

// --- auth logout ---

func TestAuthLogoutCommand_Name(t *testing.T) {
	cmd := AuthLogoutCommand()
	if cmd.Name != "logout" {
		t.Errorf("expected name %q, got %q", "logout", cmd.Name)
	}
}

func TestAuthLogoutCommand_MissingProfile(t *testing.T) {
	cmd := AuthLogoutCommand()
	if err := cmd.FlagSet.Parse([]string{"--confirm"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --profile")
	}
	if !strings.Contains(err.Error(), "--profile") {
		t.Errorf("error should mention --profile, got: %s", err.Error())
	}
}

func TestAuthLogoutCommand_MissingConfirm(t *testing.T) {
	cmd := AuthLogoutCommand()
	if err := cmd.FlagSet.Parse([]string{"--profile", "default"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --confirm")
	}
	if !strings.Contains(err.Error(), "--confirm") {
		t.Errorf("error should mention --confirm, got: %s", err.Error())
	}
}

func TestAuthLogoutCommand_WhitespaceProfile(t *testing.T) {
	cmd := AuthLogoutCommand()
	if err := cmd.FlagSet.Parse([]string{"--profile", "  ", "--confirm"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --profile")
	}
	if !strings.Contains(err.Error(), "--profile") {
		t.Errorf("error should mention --profile, got: %s", err.Error())
	}
}

// --- auth switch ---

func TestAuthSwitchCommand_Name(t *testing.T) {
	cmd := AuthSwitchCommand()
	if cmd.Name != "switch" {
		t.Errorf("expected name %q, got %q", "switch", cmd.Name)
	}
}

func TestAuthSwitchCommand_MissingProfile(t *testing.T) {
	cmd := AuthSwitchCommand()
	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --profile")
	}
	if !strings.Contains(err.Error(), "--profile") {
		t.Errorf("error should mention --profile, got: %s", err.Error())
	}
}

func TestAuthSwitchCommand_WhitespaceProfile(t *testing.T) {
	cmd := AuthSwitchCommand()
	if err := cmd.FlagSet.Parse([]string{"--profile", "   "}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --profile")
	}
	if !strings.Contains(err.Error(), "--profile") {
		t.Errorf("error should mention --profile, got: %s", err.Error())
	}
}

// --- auth status ---

func TestAuthStatusCommand_Name(t *testing.T) {
	cmd := AuthStatusCommand()
	if cmd.Name != "status" {
		t.Errorf("expected name %q, got %q", "status", cmd.Name)
	}
}

func TestAuthStatusCommand_InvalidOutputFormat(t *testing.T) {
	cmd := AuthStatusCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "xml"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
	if !strings.Contains(err.Error(), "xml") {
		t.Errorf("error should mention the invalid format, got: %s", err.Error())
	}
}

func TestAuthStatusCommand_PrettyWithTable(t *testing.T) {
	cmd := AuthStatusCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "table", "--pretty"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for --pretty with table output")
	}
	if !strings.Contains(err.Error(), "--pretty") {
		t.Errorf("error should mention --pretty, got: %s", err.Error())
	}
}

// --- auth doctor ---

func TestAuthDoctorCommand_Name(t *testing.T) {
	cmd := AuthDoctorCommand()
	if cmd.Name != "doctor" {
		t.Errorf("expected name %q, got %q", "doctor", cmd.Name)
	}
}

func TestAuthDoctorCommand_InvalidOutputFormat(t *testing.T) {
	cmd := AuthDoctorCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "yaml"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
	if !strings.Contains(err.Error(), "unsupported format") {
		t.Errorf("error should mention unsupported format, got: %s", err.Error())
	}
}

func TestAuthDoctorCommand_PrettyWithText(t *testing.T) {
	cmd := AuthDoctorCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "text", "--pretty"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for --pretty with text output")
	}
	if !strings.Contains(err.Error(), "--pretty") {
		t.Errorf("error should mention --pretty, got: %s", err.Error())
	}
}

// --- auth init ---

func TestAuthInitCommand_Name(t *testing.T) {
	cmd := AuthInitCommand()
	if cmd.Name != "init" {
		t.Errorf("expected name %q, got %q", "init", cmd.Name)
	}
}

// --- helper functions ---

func TestUpsertProfile_AddsNew(t *testing.T) {
	existing := []config.Profile{{Name: "a"}}
	result := upsertProfile(existing, config.Profile{Name: "b"})
	if len(result) != 2 {
		t.Errorf("expected 2 profiles, got %d", len(result))
	}
}

func TestUpsertProfile_UpdatesExisting(t *testing.T) {
	existing := []config.Profile{{Name: "a", KeyPath: "old.json"}}
	result := upsertProfile(existing, config.Profile{Name: "a", KeyPath: "new.json"})
	if len(result) != 1 {
		t.Errorf("expected 1 profile, got %d", len(result))
	}
	if result[0].KeyPath != "new.json" {
		t.Errorf("expected KeyPath %q, got %q", "new.json", result[0].KeyPath)
	}
}

func TestRemoveProfile_RemovesExisting(t *testing.T) {
	existing := []config.Profile{{Name: "a"}, {Name: "b"}}
	result, removed := removeProfile(existing, "a")
	if !removed {
		t.Error("expected removed to be true")
	}
	if len(result) != 1 {
		t.Errorf("expected 1 profile, got %d", len(result))
	}
}

func TestRemoveProfile_NotFound(t *testing.T) {
	existing := []config.Profile{{Name: "a"}}
	_, removed := removeProfile(existing, "b")
	if removed {
		t.Error("expected removed to be false")
	}
}

func TestFindProfile_Found(t *testing.T) {
	existing := []config.Profile{{Name: "a"}, {Name: "b"}}
	found := findProfile(existing, "b")
	if !found {
		t.Error("expected to find profile")
	}
}

func TestFindProfile_NotFound(t *testing.T) {
	existing := []config.Profile{{Name: "a"}}
	found := findProfile(existing, "z")
	if found {
		t.Error("expected not to find profile")
	}
}
