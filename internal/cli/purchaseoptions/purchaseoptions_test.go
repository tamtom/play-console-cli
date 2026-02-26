package purchaseoptions

import (
	"context"
	"errors"
	"flag"
	"strings"
	"testing"
)

func TestPurchaseOptionsCommand_Name(t *testing.T) {
	cmd := PurchaseOptionsCommand()
	if cmd.Name != "purchase-options" {
		t.Errorf("expected name %q, got %q", "purchase-options", cmd.Name)
	}
}

func TestPurchaseOptionsCommand_ShortHelp(t *testing.T) {
	cmd := PurchaseOptionsCommand()
	if cmd.ShortHelp == "" {
		t.Error("expected non-empty ShortHelp")
	}
}

func TestPurchaseOptionsCommand_LongHelp(t *testing.T) {
	cmd := PurchaseOptionsCommand()
	if cmd.LongHelp == "" {
		t.Error("expected non-empty LongHelp")
	}
}

func TestPurchaseOptionsCommand_UsageFunc(t *testing.T) {
	cmd := PurchaseOptionsCommand()
	if cmd.UsageFunc == nil {
		t.Error("expected UsageFunc to be set")
	}
}

func TestPurchaseOptionsCommand_HasSubcommands(t *testing.T) {
	cmd := PurchaseOptionsCommand()
	if len(cmd.Subcommands) == 0 {
		t.Error("expected subcommands")
	}
}

func TestPurchaseOptionsCommand_SubcommandNames(t *testing.T) {
	cmd := PurchaseOptionsCommand()
	expected := map[string]bool{
		"batch-update-states": false,
		"batch-delete":        false,
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

func TestPurchaseOptionsCommand_SubcommandsHaveUsageFunc(t *testing.T) {
	cmd := PurchaseOptionsCommand()
	for _, sub := range cmd.Subcommands {
		if sub.UsageFunc == nil {
			t.Errorf("subcommand %q missing UsageFunc", sub.Name)
		}
	}
}

func TestPurchaseOptionsCommand_NoArgs_ReturnsHelp(t *testing.T) {
	cmd := PurchaseOptionsCommand()
	err := cmd.Exec(context.Background(), nil)
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

// --- batch-update-states ---

func TestBatchUpdateStatesCommand_Name(t *testing.T) {
	cmd := BatchUpdateStatesCommand()
	if cmd.Name != "batch-update-states" {
		t.Errorf("expected name %q, got %q", "batch-update-states", cmd.Name)
	}
}

func TestBatchUpdateStatesCommand_MissingProductID(t *testing.T) {
	cmd := BatchUpdateStatesCommand()
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

func TestBatchUpdateStatesCommand_MissingJson(t *testing.T) {
	cmd := BatchUpdateStatesCommand()
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

func TestBatchUpdateStatesCommand_InvalidOutputFormat(t *testing.T) {
	cmd := BatchUpdateStatesCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "xml"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

// --- batch-delete ---

func TestBatchDeleteCommand_Name(t *testing.T) {
	cmd := BatchDeleteCommand()
	if cmd.Name != "batch-delete" {
		t.Errorf("expected name %q, got %q", "batch-delete", cmd.Name)
	}
}

func TestBatchDeleteCommand_MissingProductID(t *testing.T) {
	cmd := BatchDeleteCommand()
	if err := cmd.FlagSet.Parse([]string{"--json", `{}`, "--confirm"}); err != nil {
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

func TestBatchDeleteCommand_MissingJson(t *testing.T) {
	cmd := BatchDeleteCommand()
	if err := cmd.FlagSet.Parse([]string{"--product-id", "test", "--confirm"}); err != nil {
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

func TestBatchDeleteCommand_MissingConfirm(t *testing.T) {
	cmd := BatchDeleteCommand()
	if err := cmd.FlagSet.Parse([]string{"--product-id", "test", "--json", `{}`}); err != nil {
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
