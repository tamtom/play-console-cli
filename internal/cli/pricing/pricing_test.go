package pricing

import (
	"context"
	"errors"
	"flag"
	"strings"
	"testing"
)

func TestPricingCommand_Subcommands(t *testing.T) {
	cmd := PricingCommand()
	expected := map[string]bool{
		"convert":         false,
		"regions-version": false,
	}
	for _, sub := range cmd.Subcommands {
		if _, ok := expected[sub.Name]; ok {
			expected[sub.Name] = true
		} else {
			t.Fatalf("unexpected subcommand %q", sub.Name)
		}
	}
	for name, found := range expected {
		if !found {
			t.Fatalf("missing subcommand %q", name)
		}
	}
}

func TestPricingCommand_ShortHelpMentionsRegionsVersion(t *testing.T) {
	cmd := PricingCommand()
	if !strings.Contains(cmd.ShortHelp, "regions-version") {
		t.Fatalf("ShortHelp should make regions-version discoverable, got %q", cmd.ShortHelp)
	}
}

func TestRegionsVersionCommand_LongHelpMentionsAutoConvertCreate(t *testing.T) {
	cmd := RegionsVersionCommand()
	for _, want := range []string{
		"subscriptions create --auto-convert-regional-prices",
		"onetimeproducts create --auto-convert-regional-prices",
	} {
		if !strings.Contains(cmd.LongHelp, want) {
			t.Fatalf("LongHelp should mention %q, got:\n%s", want, cmd.LongHelp)
		}
	}
}

func TestRegionsVersionCommand_MissingPriceJSON(t *testing.T) {
	cmd := RegionsVersionCommand()
	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "--price-json is required" {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestPricingCommand_NoArgsReturnsHelp(t *testing.T) {
	cmd := PricingCommand()
	err := cmd.Exec(context.Background(), nil)
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp, got %v", err)
	}
}
