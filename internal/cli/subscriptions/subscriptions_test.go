package subscriptions

import (
	"context"
	"errors"
	"flag"
	"strings"
	"testing"
)

func TestSubscriptionsCommand_Name(t *testing.T) {
	cmd := SubscriptionsCommand()
	if cmd.Name != "subscriptions" {
		t.Errorf("expected name %q, got %q", "subscriptions", cmd.Name)
	}
}

func TestSubscriptionsCommand_ShortHelp(t *testing.T) {
	cmd := SubscriptionsCommand()
	if cmd.ShortHelp == "" {
		t.Error("expected non-empty ShortHelp")
	}
}

func TestSubscriptionsCommand_LongHelp(t *testing.T) {
	cmd := SubscriptionsCommand()
	if cmd.LongHelp == "" {
		t.Error("expected non-empty LongHelp")
	}
}

func TestSubscriptionsCommand_UsageFunc(t *testing.T) {
	cmd := SubscriptionsCommand()
	if cmd.UsageFunc == nil {
		t.Error("expected UsageFunc to be set")
	}
}

func TestSubscriptionsCommand_HasSubcommands(t *testing.T) {
	cmd := SubscriptionsCommand()
	if len(cmd.Subcommands) == 0 {
		t.Error("expected subcommands")
	}
}

func TestSubscriptionsCommand_SubcommandNames(t *testing.T) {
	cmd := SubscriptionsCommand()
	expected := map[string]bool{
		"list":         false,
		"get":          false,
		"create":       false,
		"update":       false,
		"delete":       false,
		"archive":      false,
		"batch-get":    false,
		"batch-update": false,
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

func TestSubscriptionsCommand_SubcommandsHaveUsageFunc(t *testing.T) {
	cmd := SubscriptionsCommand()
	for _, sub := range cmd.Subcommands {
		if sub.UsageFunc == nil {
			t.Errorf("subcommand %q missing UsageFunc", sub.Name)
		}
	}
}

func TestSubscriptionsCommand_NoArgs_ReturnsHelp(t *testing.T) {
	cmd := SubscriptionsCommand()
	err := cmd.Exec(context.Background(), nil)
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

// --- get ---

func TestGetCommand_MissingProductID(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --product-id")
	}
	if !strings.Contains(err.Error(), "--product-id") {
		t.Errorf("error should mention --product-id, got: %s", err.Error())
	}
}

// --- create ---

func TestCreateCommand_MissingProductID(t *testing.T) {
	cmd := CreateCommand()
	if err := cmd.FlagSet.Parse([]string{"--json", `{}`}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --product-id")
	}
	if !strings.Contains(err.Error(), "--product-id") {
		t.Errorf("error should mention --product-id, got: %s", err.Error())
	}
}

func TestCreateCommand_MissingJson(t *testing.T) {
	cmd := CreateCommand()
	if err := cmd.FlagSet.Parse([]string{"--product-id", "test"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --json")
	}
	if !strings.Contains(err.Error(), "--json") {
		t.Errorf("error should mention --json, got: %s", err.Error())
	}
}

// --- update ---

func TestUpdateCommand_EmptyJSON_NoUpdateMask_ReturnsError(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--product-id", "test", "--json", `{}`}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for empty JSON without --update-mask")
	}
	if !strings.Contains(err.Error(), "no updatable fields") {
		t.Errorf("error should mention no updatable fields, got: %s", err.Error())
	}
}

func TestUpdateCommand_OnlyImmutableFields_ReturnsError(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--product-id", "test", "--json", `{"packageName":"com.example","productId":"test"}`}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for only immutable fields")
	}
	if !strings.Contains(err.Error(), "no updatable fields") {
		t.Errorf("error should mention no updatable fields, got: %s", err.Error())
	}
}

func TestUpdateCommand_WithExplicitUpdateMask_SkipsDeriving(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--product-id", "test", "--json", `{}`, "--update-mask", "listings"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err != nil && strings.Contains(err.Error(), "no updatable fields") {
		t.Errorf("explicit --update-mask should skip derive; got: %s", err.Error())
	}
}

func TestUpdateCommand_MissingProductID(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--json", `{}`}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --product-id")
	}
	if !strings.Contains(err.Error(), "--product-id") {
		t.Errorf("error should mention --product-id, got: %s", err.Error())
	}
}

func TestUpdateCommand_MissingJson(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--product-id", "test"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --json")
	}
	if !strings.Contains(err.Error(), "--json") {
		t.Errorf("error should mention --json, got: %s", err.Error())
	}
}

// --- delete ---

func TestDeleteCommand_MissingProductID(t *testing.T) {
	cmd := DeleteCommand()
	if err := cmd.FlagSet.Parse([]string{"--confirm"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --product-id")
	}
	if !strings.Contains(err.Error(), "--product-id") {
		t.Errorf("error should mention --product-id, got: %s", err.Error())
	}
}

func TestDeleteCommand_MissingConfirm(t *testing.T) {
	cmd := DeleteCommand()
	if err := cmd.FlagSet.Parse([]string{"--product-id", "test"}); err != nil {
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

// --- batch-get ---

func TestBatchGetCommand_Name(t *testing.T) {
	cmd := BatchGetCommand()
	if cmd.Name != "batch-get" {
		t.Errorf("expected name %q, got %q", "batch-get", cmd.Name)
	}
}

func TestBatchGetCommand_MissingProductIDs(t *testing.T) {
	cmd := BatchGetCommand()
	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --product-ids")
	}
	if !strings.Contains(err.Error(), "--product-ids") {
		t.Errorf("error should mention --product-ids, got: %s", err.Error())
	}
}

func TestBatchGetCommand_WhitespaceProductIDs(t *testing.T) {
	cmd := BatchGetCommand()
	if err := cmd.FlagSet.Parse([]string{"--product-ids", "   "}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --product-ids")
	}
	if !strings.Contains(err.Error(), "--product-ids") {
		t.Errorf("error should mention --product-ids, got: %s", err.Error())
	}
}

func TestBatchGetCommand_InvalidOutputFormat(t *testing.T) {
	cmd := BatchGetCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "xml"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

// --- batch-update ---

func TestBatchUpdateCommand_Name(t *testing.T) {
	cmd := BatchUpdateCommand()
	if cmd.Name != "batch-update" {
		t.Errorf("expected name %q, got %q", "batch-update", cmd.Name)
	}
}

func TestBatchUpdateCommand_MissingJson(t *testing.T) {
	cmd := BatchUpdateCommand()
	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --json")
	}
	if !strings.Contains(err.Error(), "--json") {
		t.Errorf("error should mention --json, got: %s", err.Error())
	}
}

func TestBatchUpdateCommand_WhitespaceJson(t *testing.T) {
	cmd := BatchUpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--json", "   "}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --json")
	}
	if !strings.Contains(err.Error(), "--json") {
		t.Errorf("error should mention --json, got: %s", err.Error())
	}
}
