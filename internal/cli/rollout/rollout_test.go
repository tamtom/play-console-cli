package rollout

import (
	"context"
	"errors"
	"flag"
	"strings"
	"testing"
)

func TestRolloutCommand_Name(t *testing.T) {
	cmd := RolloutCommand()
	if cmd.Name != "rollout" {
		t.Errorf("expected name %q, got %q", "rollout", cmd.Name)
	}
}

func TestRolloutCommand_ShortHelp(t *testing.T) {
	cmd := RolloutCommand()
	if cmd.ShortHelp == "" {
		t.Error("expected non-empty ShortHelp")
	}
}

func TestRolloutCommand_UsageFunc(t *testing.T) {
	cmd := RolloutCommand()
	if cmd.UsageFunc == nil {
		t.Error("expected UsageFunc to be set")
	}
}

func TestRolloutCommand_HasSubcommands(t *testing.T) {
	cmd := RolloutCommand()
	if len(cmd.Subcommands) == 0 {
		t.Error("expected subcommands")
	}
}

func TestRolloutCommand_SubcommandNames(t *testing.T) {
	cmd := RolloutCommand()
	expected := map[string]bool{
		"halt":     false,
		"resume":   false,
		"update":   false,
		"complete": false,
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

func TestRolloutCommand_SubcommandsHaveUsageFunc(t *testing.T) {
	cmd := RolloutCommand()
	for _, sub := range cmd.Subcommands {
		if sub.UsageFunc == nil {
			t.Errorf("subcommand %q missing UsageFunc", sub.Name)
		}
	}
}

func TestRolloutCommand_SubcommandsHaveShortHelp(t *testing.T) {
	cmd := RolloutCommand()
	for _, sub := range cmd.Subcommands {
		if sub.ShortHelp == "" {
			t.Errorf("subcommand %q missing ShortHelp", sub.Name)
		}
	}
}

func TestRolloutCommand_NoArgs_ReturnsHelp(t *testing.T) {
	cmd := RolloutCommand()
	err := cmd.Exec(context.Background(), nil)
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

// --- rollout update ---

func TestRolloutUpdateCommand_Name(t *testing.T) {
	cmd := UpdateCommand()
	if cmd.Name != "update" {
		t.Errorf("expected name %q, got %q", "update", cmd.Name)
	}
}

func TestRolloutUpdateCommand_RolloutZero(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--rollout", "0"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for --rollout=0")
	}
	if !strings.Contains(err.Error(), "--rollout") {
		t.Errorf("error should mention --rollout, got: %s", err.Error())
	}
}

func TestRolloutUpdateCommand_RolloutNegative(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--rollout", "-0.5"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for negative --rollout")
	}
	if !strings.Contains(err.Error(), "--rollout") {
		t.Errorf("error should mention --rollout, got: %s", err.Error())
	}
}

func TestRolloutUpdateCommand_RolloutTooHigh(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--rollout", "1.5"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for --rollout > 1")
	}
	if !strings.Contains(err.Error(), "--rollout") {
		t.Errorf("error should mention --rollout, got: %s", err.Error())
	}
}

func TestRolloutUpdateCommand_InvalidOutputFormat(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "xml", "--rollout", "0.5"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

func TestRolloutUpdateCommand_PrettyWithTable(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "table", "--pretty", "--rollout", "0.5"}); err != nil {
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

// --- rollout halt ---

func TestRolloutHaltCommand_Name(t *testing.T) {
	cmd := HaltCommand()
	if cmd.Name != "halt" {
		t.Errorf("expected name %q, got %q", "halt", cmd.Name)
	}
}

func TestRolloutHaltCommand_LongHelp(t *testing.T) {
	cmd := HaltCommand()
	if cmd.LongHelp == "" {
		t.Error("expected non-empty LongHelp")
	}
}

func TestRolloutHaltCommand_InvalidOutputFormat(t *testing.T) {
	cmd := HaltCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "yaml"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

func TestRolloutHaltCommand_PrettyWithMarkdown(t *testing.T) {
	cmd := HaltCommand()
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

// --- rollout resume ---

func TestRolloutResumeCommand_Name(t *testing.T) {
	cmd := ResumeCommand()
	if cmd.Name != "resume" {
		t.Errorf("expected name %q, got %q", "resume", cmd.Name)
	}
}

func TestRolloutResumeCommand_LongHelp(t *testing.T) {
	cmd := ResumeCommand()
	if cmd.LongHelp == "" {
		t.Error("expected non-empty LongHelp")
	}
}

func TestRolloutResumeCommand_InvalidOutputFormat(t *testing.T) {
	cmd := ResumeCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "csv"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

// --- rollout complete ---

func TestRolloutCompleteCommand_Name(t *testing.T) {
	cmd := CompleteCommand()
	if cmd.Name != "complete" {
		t.Errorf("expected name %q, got %q", "complete", cmd.Name)
	}
}

func TestRolloutCompleteCommand_LongHelp(t *testing.T) {
	cmd := CompleteCommand()
	if cmd.LongHelp == "" {
		t.Error("expected non-empty LongHelp")
	}
}

func TestRolloutCompleteCommand_InvalidOutputFormat(t *testing.T) {
	cmd := CompleteCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "xml"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}
