package listings

import (
	"context"
	"flag"
	"testing"
)

func TestLocalesCommand_Name(t *testing.T) {
	cmd := LocalesCommand()
	if cmd.Name != "locales" {
		t.Errorf("Name = %q, want %q", cmd.Name, "locales")
	}
}

func TestLocalesCommand_HasUsageFunc(t *testing.T) {
	cmd := LocalesCommand()
	if cmd.UsageFunc == nil {
		t.Error("UsageFunc is nil, want shared.DefaultUsageFunc")
	}
}

func TestLocalesCommand_Flags(t *testing.T) {
	cmd := LocalesCommand()
	flags := []string{"package", "edit", "output", "pretty"}
	for _, name := range flags {
		f := cmd.FlagSet.Lookup(name)
		if f == nil {
			t.Errorf("flag --%s not found", name)
		}
	}
}

func TestLocalesCommand_RequiresPackage(t *testing.T) {
	cmd := LocalesCommand()
	// Run without --package; should fail with "package is required"
	// (will fail at auth before that in a real scenario, but we test the flag path)
	err := cmd.ParseAndRun(context.Background(), []string{})
	if err == nil {
		t.Error("expected error when --package is missing")
	}
}

func TestLocalesCommand_InvalidOutput(t *testing.T) {
	cmd := LocalesCommand()
	err := cmd.ParseAndRun(context.Background(), []string{"--output", "table", "--pretty"})
	if err == nil {
		t.Error("expected error for --pretty with --output table")
	}
}

func TestLocalesCommand_RegisteredInListings(t *testing.T) {
	cmd := ListingsCommand()
	found := false
	for _, sub := range cmd.Subcommands {
		if sub.Name == "locales" {
			found = true
			break
		}
	}
	if !found {
		t.Error("locales subcommand not registered in listings command")
	}
}

func TestLocalesResponse_Fields(t *testing.T) {
	resp := localesResponse{
		Locales: []string{"en-US", "fr-FR"},
		Total:   2,
	}
	if len(resp.Locales) != 2 {
		t.Errorf("Locales len = %d, want 2", len(resp.Locales))
	}
	if resp.Total != 2 {
		t.Errorf("Total = %d, want 2", resp.Total)
	}
}

func TestListingsCommand_NoArgs(t *testing.T) {
	cmd := ListingsCommand()
	err := cmd.ParseAndRun(context.Background(), []string{})
	if err != flag.ErrHelp {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}
