package listings

import (
	"context"
	"flag"
	"strings"
	"testing"
)

func TestListingsCommand_Name(t *testing.T) {
	cmd := ListingsCommand()
	if cmd.Name != "listings" {
		t.Errorf("expected name %q, got %q", "listings", cmd.Name)
	}
}

func TestListingsCommand_ShortHelp(t *testing.T) {
	cmd := ListingsCommand()
	if cmd.ShortHelp == "" {
		t.Error("expected non-empty ShortHelp")
	}
}

func TestListingsCommand_UsageFunc(t *testing.T) {
	cmd := ListingsCommand()
	if cmd.UsageFunc == nil {
		t.Error("expected UsageFunc to be set")
	}
}

func TestListingsCommand_HasSubcommands(t *testing.T) {
	cmd := ListingsCommand()
	if len(cmd.Subcommands) == 0 {
		t.Error("expected subcommands")
	}
}

func TestListingsCommand_SubcommandNames(t *testing.T) {
	cmd := ListingsCommand()
	expected := map[string]bool{
		"list":       false,
		"get":        false,
		"update":     false,
		"patch":      false,
		"delete":     false,
		"delete-all": false,
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

func TestListingsCommand_SubcommandsHaveUsageFunc(t *testing.T) {
	cmd := ListingsCommand()
	for _, sub := range cmd.Subcommands {
		if sub.UsageFunc == nil {
			t.Errorf("subcommand %q missing UsageFunc", sub.Name)
		}
	}
}

func TestListingsCommand_NoArgs_ReturnsHelp(t *testing.T) {
	cmd := ListingsCommand()
	err := cmd.Exec(context.Background(), nil)
	if err != flag.ErrHelp {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

// --- listings get ---

func TestListingsGetCommand_Name(t *testing.T) {
	cmd := GetCommand()
	if cmd.Name != "get" {
		t.Errorf("expected name %q, got %q", "get", cmd.Name)
	}
}

func TestListingsGetCommand_MissingLocale(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{"--edit", "abc123"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --locale")
	}
	if !strings.Contains(err.Error(), "--locale") {
		t.Errorf("error should mention --locale, got: %s", err.Error())
	}
}

func TestListingsGetCommand_WhitespaceLocale(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{"--locale", "   ", "--edit", "abc123"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --locale")
	}
	if !strings.Contains(err.Error(), "--locale") {
		t.Errorf("error should mention --locale, got: %s", err.Error())
	}
}

func TestListingsGetCommand_InvalidOutputFormat(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "xml"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

func TestListingsGetCommand_PrettyWithTable(t *testing.T) {
	cmd := GetCommand()
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

// --- listings delete ---

func TestListingsDeleteCommand_Name(t *testing.T) {
	cmd := DeleteCommand()
	if cmd.Name != "delete" {
		t.Errorf("expected name %q, got %q", "delete", cmd.Name)
	}
}

func TestListingsDeleteCommand_MissingLocale(t *testing.T) {
	cmd := DeleteCommand()
	if err := cmd.FlagSet.Parse([]string{"--confirm"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --locale")
	}
	if !strings.Contains(err.Error(), "--locale") {
		t.Errorf("error should mention --locale, got: %s", err.Error())
	}
}

func TestListingsDeleteCommand_MissingConfirm(t *testing.T) {
	cmd := DeleteCommand()
	if err := cmd.FlagSet.Parse([]string{"--locale", "en-US"}); err != nil {
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

// --- listings delete-all ---

func TestListingsDeleteAllCommand_Name(t *testing.T) {
	cmd := DeleteAllCommand()
	if cmd.Name != "delete-all" {
		t.Errorf("expected name %q, got %q", "delete-all", cmd.Name)
	}
}

func TestListingsDeleteAllCommand_MissingConfirm(t *testing.T) {
	cmd := DeleteAllCommand()
	if err := cmd.FlagSet.Parse([]string{}); err != nil {
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

// --- listings update ---

func TestListingsUpdateCommand_Name(t *testing.T) {
	cmd := UpdateCommand()
	if cmd.Name != "update" {
		t.Errorf("expected name %q, got %q", "update", cmd.Name)
	}
}

func TestListingsUpdateCommand_MissingLocale(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--edit", "abc123"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --locale")
	}
	if !strings.Contains(err.Error(), "--locale") {
		t.Errorf("error should mention --locale, got: %s", err.Error())
	}
}

func TestListingsUpdateCommand_PrettyWithTable(t *testing.T) {
	cmd := UpdateCommand()
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

// --- listings patch ---

func TestListingsPatchCommand_Name(t *testing.T) {
	cmd := PatchCommand()
	if cmd.Name != "patch" {
		t.Errorf("expected name %q, got %q", "patch", cmd.Name)
	}
}

func TestListingsPatchCommand_MissingLocale(t *testing.T) {
	cmd := PatchCommand()
	if err := cmd.FlagSet.Parse([]string{"--edit", "abc123"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --locale")
	}
	if !strings.Contains(err.Error(), "--locale") {
		t.Errorf("error should mention --locale, got: %s", err.Error())
	}
}

// --- listings list ---

func TestListingsListCommand_Name(t *testing.T) {
	cmd := ListCommand()
	if cmd.Name != "list" {
		t.Errorf("expected name %q, got %q", "list", cmd.Name)
	}
}

func TestListingsListCommand_InvalidOutputFormat(t *testing.T) {
	cmd := ListCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "yaml"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}
