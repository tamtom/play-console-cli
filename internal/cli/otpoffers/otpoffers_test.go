package otpoffers

import (
	"context"
	"errors"
	"flag"
	"strings"
	"testing"
)

func TestOTPOffersCommand_Name(t *testing.T) {
	cmd := OTPOffersCommand()
	if cmd.Name != "otp-offers" {
		t.Errorf("expected name %q, got %q", "otp-offers", cmd.Name)
	}
}

func TestOTPOffersCommand_ShortHelp(t *testing.T) {
	cmd := OTPOffersCommand()
	if cmd.ShortHelp == "" {
		t.Error("expected non-empty ShortHelp")
	}
}

func TestOTPOffersCommand_LongHelp(t *testing.T) {
	cmd := OTPOffersCommand()
	if cmd.LongHelp == "" {
		t.Error("expected non-empty LongHelp")
	}
}

func TestOTPOffersCommand_UsageFunc(t *testing.T) {
	cmd := OTPOffersCommand()
	if cmd.UsageFunc == nil {
		t.Error("expected UsageFunc to be set")
	}
}

func TestOTPOffersCommand_HasSubcommands(t *testing.T) {
	cmd := OTPOffersCommand()
	if len(cmd.Subcommands) == 0 {
		t.Error("expected subcommands")
	}
}

func TestOTPOffersCommand_SubcommandNames(t *testing.T) {
	cmd := OTPOffersCommand()
	expected := map[string]bool{
		"list":                false,
		"activate":            false,
		"deactivate":          false,
		"cancel":              false,
		"batch-get":           false,
		"batch-update":        false,
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

func TestOTPOffersCommand_SubcommandsHaveUsageFunc(t *testing.T) {
	cmd := OTPOffersCommand()
	for _, sub := range cmd.Subcommands {
		if sub.UsageFunc == nil {
			t.Errorf("subcommand %q missing UsageFunc", sub.Name)
		}
	}
}

func TestOTPOffersCommand_NoArgs_ReturnsHelp(t *testing.T) {
	cmd := OTPOffersCommand()
	err := cmd.Exec(context.Background(), nil)
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

// --- list ---

func TestListCommand_MissingProductID(t *testing.T) {
	cmd := ListCommand()
	if err := cmd.FlagSet.Parse([]string{"--purchase-option-id", "opt1"}); err != nil {
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

func TestListCommand_MissingPurchaseOptionID(t *testing.T) {
	cmd := ListCommand()
	if err := cmd.FlagSet.Parse([]string{"--product-id", "test"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --purchase-option-id")
	}
	if !strings.Contains(err.Error(), "--purchase-option-id") {
		t.Errorf("error should mention --purchase-option-id, got: %s", err.Error())
	}
}

func TestListCommand_InvalidOutputFormat(t *testing.T) {
	cmd := ListCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "xml"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

// --- activate ---

func TestActivateCommand_MissingProductID(t *testing.T) {
	cmd := ActivateCommand()
	if err := cmd.FlagSet.Parse([]string{"--purchase-option-id", "opt1", "--offer-id", "offer1"}); err != nil {
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

func TestActivateCommand_MissingPurchaseOptionID(t *testing.T) {
	cmd := ActivateCommand()
	if err := cmd.FlagSet.Parse([]string{"--product-id", "test", "--offer-id", "offer1"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --purchase-option-id")
	}
	if !strings.Contains(err.Error(), "--purchase-option-id") {
		t.Errorf("error should mention --purchase-option-id, got: %s", err.Error())
	}
}

func TestActivateCommand_MissingOfferID(t *testing.T) {
	cmd := ActivateCommand()
	if err := cmd.FlagSet.Parse([]string{"--product-id", "test", "--purchase-option-id", "opt1"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --offer-id")
	}
	if !strings.Contains(err.Error(), "--offer-id") {
		t.Errorf("error should mention --offer-id, got: %s", err.Error())
	}
}

// --- deactivate ---

func TestDeactivateCommand_MissingProductID(t *testing.T) {
	cmd := DeactivateCommand()
	if err := cmd.FlagSet.Parse([]string{"--purchase-option-id", "opt1", "--offer-id", "offer1"}); err != nil {
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

func TestDeactivateCommand_MissingOfferID(t *testing.T) {
	cmd := DeactivateCommand()
	if err := cmd.FlagSet.Parse([]string{"--product-id", "test", "--purchase-option-id", "opt1"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --offer-id")
	}
	if !strings.Contains(err.Error(), "--offer-id") {
		t.Errorf("error should mention --offer-id, got: %s", err.Error())
	}
}

// --- cancel ---

func TestCancelCommand_MissingProductID(t *testing.T) {
	cmd := CancelCommand()
	if err := cmd.FlagSet.Parse([]string{"--purchase-option-id", "opt1", "--offer-id", "offer1"}); err != nil {
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

func TestCancelCommand_MissingOfferID(t *testing.T) {
	cmd := CancelCommand()
	if err := cmd.FlagSet.Parse([]string{"--product-id", "test", "--purchase-option-id", "opt1"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --offer-id")
	}
	if !strings.Contains(err.Error(), "--offer-id") {
		t.Errorf("error should mention --offer-id, got: %s", err.Error())
	}
}

// --- batch-get ---

func TestBatchGetCommand_MissingProductID(t *testing.T) {
	cmd := BatchGetCommand()
	if err := cmd.FlagSet.Parse([]string{"--purchase-option-id", "opt1", "--json", `{}`}); err != nil {
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

func TestBatchGetCommand_MissingPurchaseOptionID(t *testing.T) {
	cmd := BatchGetCommand()
	if err := cmd.FlagSet.Parse([]string{"--product-id", "test", "--json", `{}`}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --purchase-option-id")
	}
	if !strings.Contains(err.Error(), "--purchase-option-id") {
		t.Errorf("error should mention --purchase-option-id, got: %s", err.Error())
	}
}

func TestBatchGetCommand_MissingJson(t *testing.T) {
	cmd := BatchGetCommand()
	if err := cmd.FlagSet.Parse([]string{"--product-id", "test", "--purchase-option-id", "opt1"}); err != nil {
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

// --- batch-update ---

func TestBatchUpdateCommand_MissingJson(t *testing.T) {
	cmd := BatchUpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--product-id", "test", "--purchase-option-id", "opt1"}); err != nil {
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

// --- batch-update-states ---

func TestBatchUpdateStatesCommand_MissingJson(t *testing.T) {
	cmd := BatchUpdateStatesCommand()
	if err := cmd.FlagSet.Parse([]string{"--product-id", "test", "--purchase-option-id", "opt1"}); err != nil {
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

// --- batch-delete ---

func TestBatchDeleteCommand_MissingJson(t *testing.T) {
	cmd := BatchDeleteCommand()
	if err := cmd.FlagSet.Parse([]string{"--product-id", "test", "--purchase-option-id", "opt1", "--confirm"}); err != nil {
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
	if err := cmd.FlagSet.Parse([]string{"--product-id", "test", "--purchase-option-id", "opt1", "--json", `{}`}); err != nil {
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
