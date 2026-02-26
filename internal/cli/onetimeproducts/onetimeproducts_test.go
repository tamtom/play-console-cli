package onetimeproducts

import (
	"context"
	"errors"
	"flag"
	"strings"
	"testing"
)

func TestOneTimeProductsCommand_Name(t *testing.T) {
	cmd := OneTimeProductsCommand()
	if cmd.Name != "onetimeproducts" {
		t.Errorf("expected name %q, got %q", "onetimeproducts", cmd.Name)
	}
}

func TestOneTimeProductsCommand_ShortHelp(t *testing.T) {
	cmd := OneTimeProductsCommand()
	if cmd.ShortHelp == "" {
		t.Error("expected non-empty ShortHelp")
	}
}

func TestOneTimeProductsCommand_LongHelp(t *testing.T) {
	cmd := OneTimeProductsCommand()
	if cmd.LongHelp == "" {
		t.Error("expected non-empty LongHelp")
	}
}

func TestOneTimeProductsCommand_UsageFunc(t *testing.T) {
	cmd := OneTimeProductsCommand()
	if cmd.UsageFunc == nil {
		t.Error("expected UsageFunc to be set")
	}
}

func TestOneTimeProductsCommand_HasSubcommands(t *testing.T) {
	cmd := OneTimeProductsCommand()
	if len(cmd.Subcommands) == 0 {
		t.Error("expected subcommands")
	}
}

func TestOneTimeProductsCommand_SubcommandNames(t *testing.T) {
	cmd := OneTimeProductsCommand()
	expected := map[string]bool{
		"list":         false,
		"get":          false,
		"create":       false,
		"patch":        false,
		"delete":       false,
		"batch-get":    false,
		"batch-update": false,
		"batch-delete": false,
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

func TestOneTimeProductsCommand_SubcommandsHaveUsageFunc(t *testing.T) {
	cmd := OneTimeProductsCommand()
	for _, sub := range cmd.Subcommands {
		if sub.UsageFunc == nil {
			t.Errorf("subcommand %q missing UsageFunc", sub.Name)
		}
	}
}

func TestOneTimeProductsCommand_NoArgs_ReturnsHelp(t *testing.T) {
	cmd := OneTimeProductsCommand()
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

func TestGetCommand_InvalidOutputFormat(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "xml"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

// --- create ---

func TestCreateCommand_Name(t *testing.T) {
	cmd := CreateCommand()
	if cmd.Name != "create" {
		t.Errorf("expected name %q, got %q", "create", cmd.Name)
	}
}

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

func TestCreateCommand_HasRegionsVersionFlag(t *testing.T) {
	cmd := CreateCommand()
	if err := cmd.FlagSet.Parse([]string{"--regions-version", "2024001", "--product-id", "test", "--json", `{}`}); err != nil {
		t.Fatal(err)
	}
	f := cmd.FlagSet.Lookup("regions-version")
	if f == nil {
		t.Fatal("expected --regions-version flag")
	}
	if f.Value.String() != "2024001" {
		t.Errorf("expected regions-version %q, got %q", "2024001", f.Value.String())
	}
}

// --- patch ---

func TestPatchCommand_Name(t *testing.T) {
	cmd := PatchCommand()
	if cmd.Name != "patch" {
		t.Errorf("expected name %q, got %q", "patch", cmd.Name)
	}
}

func TestPatchCommand_MissingProductID(t *testing.T) {
	cmd := PatchCommand()
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

func TestPatchCommand_MissingJson(t *testing.T) {
	cmd := PatchCommand()
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

func TestPatchCommand_HasRegionsVersionFlag(t *testing.T) {
	cmd := PatchCommand()
	if err := cmd.FlagSet.Parse([]string{"--regions-version", "2024001", "--product-id", "test", "--json", `{}`}); err != nil {
		t.Fatal(err)
	}
	f := cmd.FlagSet.Lookup("regions-version")
	if f == nil {
		t.Fatal("expected --regions-version flag")
	}
}

func TestPatchCommand_HasAllowMissingFlag(t *testing.T) {
	cmd := PatchCommand()
	if err := cmd.FlagSet.Parse([]string{"--allow-missing", "--product-id", "test", "--json", `{}`}); err != nil {
		t.Fatal(err)
	}
	f := cmd.FlagSet.Lookup("allow-missing")
	if f == nil {
		t.Fatal("expected --allow-missing flag")
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

// --- batch-delete ---

func TestBatchDeleteCommand_Name(t *testing.T) {
	cmd := BatchDeleteCommand()
	if cmd.Name != "batch-delete" {
		t.Errorf("expected name %q, got %q", "batch-delete", cmd.Name)
	}
}

func TestBatchDeleteCommand_MissingJson(t *testing.T) {
	cmd := BatchDeleteCommand()
	if err := cmd.FlagSet.Parse([]string{"--confirm"}); err != nil {
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
	if err := cmd.FlagSet.Parse([]string{"--json", `{}`}); err != nil {
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
