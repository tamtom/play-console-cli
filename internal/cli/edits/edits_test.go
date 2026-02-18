package edits

import (
	"context"
	"errors"
	"flag"
	"strings"
	"testing"
)

func TestEditsCommand_Name(t *testing.T) {
	cmd := EditsCommand()
	if cmd.Name != "edits" {
		t.Errorf("expected name %q, got %q", "edits", cmd.Name)
	}
}

func TestEditsCommand_ShortHelp(t *testing.T) {
	cmd := EditsCommand()
	if cmd.ShortHelp == "" {
		t.Error("expected non-empty ShortHelp")
	}
}

func TestEditsCommand_UsageFunc(t *testing.T) {
	cmd := EditsCommand()
	if cmd.UsageFunc == nil {
		t.Error("expected UsageFunc to be set")
	}
}

func TestEditsCommand_HasSubcommands(t *testing.T) {
	cmd := EditsCommand()
	if len(cmd.Subcommands) == 0 {
		t.Error("expected subcommands")
	}
}

func TestEditsCommand_SubcommandNames(t *testing.T) {
	cmd := EditsCommand()
	expected := map[string]bool{
		"create":   false,
		"get":      false,
		"validate": false,
		"commit":   false,
		"delete":   false,
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

func TestEditsCommand_SubcommandsHaveUsageFunc(t *testing.T) {
	cmd := EditsCommand()
	for _, sub := range cmd.Subcommands {
		if sub.UsageFunc == nil {
			t.Errorf("subcommand %q missing UsageFunc", sub.Name)
		}
	}
}

func TestEditsCommand_SubcommandsHaveShortHelp(t *testing.T) {
	cmd := EditsCommand()
	for _, sub := range cmd.Subcommands {
		if sub.ShortHelp == "" {
			t.Errorf("subcommand %q missing ShortHelp", sub.Name)
		}
	}
}

func TestEditsCommand_NoArgs_ReturnsHelp(t *testing.T) {
	cmd := EditsCommand()
	err := cmd.Exec(context.Background(), nil)
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

// --- edits get ---

func TestEditsGetCommand_Name(t *testing.T) {
	cmd := GetCommand()
	if cmd.Name != "get" {
		t.Errorf("expected name %q, got %q", "get", cmd.Name)
	}
}

func TestEditsGetCommand_MissingEdit(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example.app"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --edit")
	}
	if !strings.Contains(err.Error(), "--edit") {
		t.Errorf("error should mention --edit, got: %s", err.Error())
	}
}

func TestEditsGetCommand_WhitespaceEdit(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{"--edit", "   "}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --edit")
	}
	if !strings.Contains(err.Error(), "--edit") {
		t.Errorf("error should mention --edit, got: %s", err.Error())
	}
}

func TestEditsGetCommand_InvalidOutputFormat(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "xml"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

func TestEditsGetCommand_PrettyWithTable(t *testing.T) {
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

// --- edits validate ---

func TestEditsValidateCommand_Name(t *testing.T) {
	cmd := ValidateCommand()
	if cmd.Name != "validate" {
		t.Errorf("expected name %q, got %q", "validate", cmd.Name)
	}
}

func TestEditsValidateCommand_MissingEdit(t *testing.T) {
	cmd := ValidateCommand()
	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --edit")
	}
	if !strings.Contains(err.Error(), "--edit") {
		t.Errorf("error should mention --edit, got: %s", err.Error())
	}
}

func TestEditsValidateCommand_InvalidOutputFormat(t *testing.T) {
	cmd := ValidateCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "yaml"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

// --- edits commit ---

func TestEditsCommitCommand_Name(t *testing.T) {
	cmd := CommitCommand()
	if cmd.Name != "commit" {
		t.Errorf("expected name %q, got %q", "commit", cmd.Name)
	}
}

func TestEditsCommitCommand_MissingEdit(t *testing.T) {
	cmd := CommitCommand()
	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --edit")
	}
	if !strings.Contains(err.Error(), "--edit") {
		t.Errorf("error should mention --edit, got: %s", err.Error())
	}
}

func TestEditsCommitCommand_WhitespaceEdit(t *testing.T) {
	cmd := CommitCommand()
	if err := cmd.FlagSet.Parse([]string{"--edit", "  "}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --edit")
	}
	if !strings.Contains(err.Error(), "--edit") {
		t.Errorf("error should mention --edit, got: %s", err.Error())
	}
}

func TestEditsCommitCommand_PrettyWithMarkdown(t *testing.T) {
	cmd := CommitCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "markdown", "--pretty"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for --pretty with markdown output")
	}
	if !strings.Contains(err.Error(), "--pretty") {
		t.Errorf("error should mention --pretty, got: %s", err.Error())
	}
}

// --- edits delete ---

func TestEditsDeleteCommand_Name(t *testing.T) {
	cmd := DeleteCommand()
	if cmd.Name != "delete" {
		t.Errorf("expected name %q, got %q", "delete", cmd.Name)
	}
}

func TestEditsDeleteCommand_MissingEdit(t *testing.T) {
	cmd := DeleteCommand()
	if err := cmd.FlagSet.Parse([]string{"--confirm"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --edit")
	}
	if !strings.Contains(err.Error(), "--edit") {
		t.Errorf("error should mention --edit, got: %s", err.Error())
	}
}

func TestEditsDeleteCommand_MissingConfirm(t *testing.T) {
	cmd := DeleteCommand()
	if err := cmd.FlagSet.Parse([]string{"--edit", "abc123"}); err != nil {
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

func TestEditsDeleteCommand_WhitespaceEdit(t *testing.T) {
	cmd := DeleteCommand()
	if err := cmd.FlagSet.Parse([]string{"--edit", "  ", "--confirm"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --edit")
	}
	if !strings.Contains(err.Error(), "--edit") {
		t.Errorf("error should mention --edit, got: %s", err.Error())
	}
}

// --- edits create ---

func TestEditsCreateCommand_Name(t *testing.T) {
	cmd := CreateCommand()
	if cmd.Name != "create" {
		t.Errorf("expected name %q, got %q", "create", cmd.Name)
	}
}

func TestEditsCreateCommand_InvalidOutputFormat(t *testing.T) {
	cmd := CreateCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "csv"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

func TestEditsCreateCommand_PrettyWithTable(t *testing.T) {
	cmd := CreateCommand()
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
