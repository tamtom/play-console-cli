package testers

import (
	"context"
	"flag"
	"strings"
	"testing"
)

func TestTestersCommand_Name(t *testing.T) {
	cmd := TestersCommand()
	if cmd.Name != "testers" {
		t.Errorf("expected name %q, got %q", "testers", cmd.Name)
	}
}

func TestTestersCommand_ShortHelp(t *testing.T) {
	cmd := TestersCommand()
	if cmd.ShortHelp == "" {
		t.Error("expected non-empty ShortHelp")
	}
}

func TestTestersCommand_UsageFunc(t *testing.T) {
	cmd := TestersCommand()
	if cmd.UsageFunc == nil {
		t.Error("expected UsageFunc to be set")
	}
}

func TestTestersCommand_HasSubcommands(t *testing.T) {
	cmd := TestersCommand()
	if len(cmd.Subcommands) == 0 {
		t.Error("expected subcommands")
	}
}

func TestTestersCommand_SubcommandNames(t *testing.T) {
	cmd := TestersCommand()
	expected := map[string]bool{
		"get":    false,
		"update": false,
		"patch":  false,
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

func TestTestersCommand_SubcommandsHaveUsageFunc(t *testing.T) {
	cmd := TestersCommand()
	for _, sub := range cmd.Subcommands {
		if sub.UsageFunc == nil {
			t.Errorf("subcommand %q missing UsageFunc", sub.Name)
		}
	}
}

func TestTestersCommand_SubcommandsHaveShortHelp(t *testing.T) {
	cmd := TestersCommand()
	for _, sub := range cmd.Subcommands {
		if sub.ShortHelp == "" {
			t.Errorf("subcommand %q missing ShortHelp", sub.Name)
		}
	}
}

func TestTestersCommand_NoArgs_ReturnsHelp(t *testing.T) {
	cmd := TestersCommand()
	err := cmd.Exec(context.Background(), nil)
	if err != flag.ErrHelp {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

// --- testers get ---

func TestTestersGetCommand_Name(t *testing.T) {
	cmd := GetCommand()
	if cmd.Name != "get" {
		t.Errorf("expected name %q, got %q", "get", cmd.Name)
	}
}

func TestTestersGetCommand_MissingEdit(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{"--track", "internal"}); err != nil {
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

func TestTestersGetCommand_MissingTrack(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{"--edit", "abc123"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --track")
	}
	if !strings.Contains(err.Error(), "--track") {
		t.Errorf("error should mention --track, got: %s", err.Error())
	}
}

func TestTestersGetCommand_WhitespaceEdit(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{"--edit", "   ", "--track", "internal"}); err != nil {
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

func TestTestersGetCommand_WhitespaceTrack(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{"--edit", "abc123", "--track", "   "}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --track")
	}
	if !strings.Contains(err.Error(), "--track") {
		t.Errorf("error should mention --track, got: %s", err.Error())
	}
}

func TestTestersGetCommand_InvalidOutputFormat(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "xml"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

func TestTestersGetCommand_PrettyWithTable(t *testing.T) {
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

// --- testers update ---

func TestTestersUpdateCommand_Name(t *testing.T) {
	cmd := UpdateCommand()
	if cmd.Name != "update" {
		t.Errorf("expected name %q, got %q", "update", cmd.Name)
	}
}

func TestTestersUpdateCommand_MissingEdit(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--track", "internal"}); err != nil {
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

func TestTestersUpdateCommand_MissingTrack(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--edit", "abc123"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --track")
	}
	if !strings.Contains(err.Error(), "--track") {
		t.Errorf("error should mention --track, got: %s", err.Error())
	}
}

func TestTestersUpdateCommand_PrettyWithMarkdown(t *testing.T) {
	cmd := UpdateCommand()
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

// --- testers patch ---

func TestTestersPatchCommand_Name(t *testing.T) {
	cmd := PatchCommand()
	if cmd.Name != "patch" {
		t.Errorf("expected name %q, got %q", "patch", cmd.Name)
	}
}

func TestTestersPatchCommand_MissingEdit(t *testing.T) {
	cmd := PatchCommand()
	if err := cmd.FlagSet.Parse([]string{"--track", "internal"}); err != nil {
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

func TestTestersPatchCommand_MissingTrack(t *testing.T) {
	cmd := PatchCommand()
	if err := cmd.FlagSet.Parse([]string{"--edit", "abc123"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --track")
	}
	if !strings.Contains(err.Error(), "--track") {
		t.Errorf("error should mention --track, got: %s", err.Error())
	}
}
