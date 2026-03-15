package metadata

import (
	"context"
	"errors"
	"flag"
	"testing"
)

func TestMetadataCommand_Name(t *testing.T) {
	cmd := MetadataCommand()
	if cmd.Name != "metadata" {
		t.Errorf("expected name %q, got %q", "metadata", cmd.Name)
	}
}

func TestMetadataCommand_ShortHelp(t *testing.T) {
	cmd := MetadataCommand()
	if cmd.ShortHelp == "" {
		t.Error("expected non-empty ShortHelp")
	}
}

func TestMetadataCommand_UsageFunc(t *testing.T) {
	cmd := MetadataCommand()
	if cmd.UsageFunc == nil {
		t.Error("expected UsageFunc to be set")
	}
}

func TestMetadataCommand_HasSubcommands(t *testing.T) {
	cmd := MetadataCommand()
	if len(cmd.Subcommands) == 0 {
		t.Error("expected subcommands")
	}
}

func TestMetadataCommand_SubcommandNames(t *testing.T) {
	cmd := MetadataCommand()
	expected := map[string]bool{
		"pull":     false,
		"push":     false,
		"validate": false,
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

func TestMetadataCommand_SubcommandsHaveUsageFunc(t *testing.T) {
	cmd := MetadataCommand()
	for _, sub := range cmd.Subcommands {
		if sub.UsageFunc == nil {
			t.Errorf("subcommand %q missing UsageFunc", sub.Name)
		}
	}
}

func TestMetadataCommand_NoArgs_ReturnsHelp(t *testing.T) {
	cmd := MetadataCommand()
	err := cmd.Exec(context.Background(), nil)
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}
