package cmd

import (
	"strings"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"
)

// allCommandNames returns all 49 top-level command names registered in gplay.
func allCommandNames() []string {
	return []string{
		"auth", "apps", "edits", "bundles", "apks", "tracks", "users",
		"listings", "images", "init", "status", "reviews", "details", "testers",
		"availability", "deobfuscation", "release", "promote", "rollout",
		"sync", "validate", "vitals", "iap", "subscriptions", "baseplans",
		"offers", "onetimeproducts", "purchase-options", "otp-offers",
		"pricing", "orders", "purchases", "external-transactions",
		"generated-apks", "grants", "internal-sharing", "system-apks",
		"expansion", "recovery", "data-safety", "device-tiers", "notify",
		"migrate", "release-notes", "reports", "docs", "update",
		"completion", "version",
	}
}

func buildRootCommand(names []string) *ffcli.Command {
	var subs []*ffcli.Command
	for _, name := range names {
		subs = append(subs, &ffcli.Command{
			Name:      name,
			ShortHelp: "Help for " + name,
		})
	}
	return &ffcli.Command{
		Name:        "gplay",
		ShortUsage:  "gplay <command> [flags]",
		ShortHelp:   "A CLI for Google Play Console.",
		Subcommands: subs,
	}
}

func TestRootUsageFunc_GroupsCommands(t *testing.T) {
	root := buildRootCommand(allCommandNames())
	got := RootUsageFunc(root)

	expectedGroups := []string{
		"GETTING STARTED",
		"APP MANAGEMENT",
		"RELEASES & TRACKS",
		"TESTING",
		"VITALS & REVIEWS",
		"MONETIZATION",
		"ACCOUNT & ACCESS",
		"AUTOMATION",
		"UTILITIES",
	}

	for _, group := range expectedGroups {
		if !strings.Contains(got, group) {
			t.Errorf("expected group %q in output, got:\n%s", group, got)
		}
	}
}

func TestRootUsageFunc_AllCommandsAppear(t *testing.T) {
	names := allCommandNames()
	root := buildRootCommand(names)
	got := RootUsageFunc(root)

	for _, name := range names {
		if !strings.Contains(got, name) {
			t.Errorf("command %q not found in root usage output", name)
		}
	}
}

func TestRootUsageFunc_DeprecatedCommandsHidden(t *testing.T) {
	root := &ffcli.Command{
		Name:       "gplay",
		ShortUsage: "gplay <command> [flags]",
		ShortHelp:  "A CLI for Google Play Console.",
		Subcommands: []*ffcli.Command{
			{Name: "apps", ShortHelp: "List apps"},
			{Name: "old-cmd", ShortHelp: "DEPRECATED: use apps instead"},
		},
	}

	got := RootUsageFunc(root)
	if !strings.Contains(got, "apps") {
		t.Errorf("expected 'apps' command in output, got:\n%s", got)
	}
	if strings.Contains(got, "old-cmd") {
		t.Errorf("deprecated command 'old-cmd' should be hidden, got:\n%s", got)
	}
}

func TestRootUsageFunc_UngroupedCommandsInAdditional(t *testing.T) {
	root := &ffcli.Command{
		Name:       "gplay",
		ShortUsage: "gplay <command> [flags]",
		ShortHelp:  "A CLI for Google Play Console.",
		Subcommands: []*ffcli.Command{
			{Name: "auth", ShortHelp: "Manage auth"},
			{Name: "custom-cmd", ShortHelp: "A custom command"},
		},
	}

	got := RootUsageFunc(root)
	if !strings.Contains(got, "ADDITIONAL COMMANDS") {
		t.Errorf("expected ADDITIONAL COMMANDS section for ungrouped command, got:\n%s", got)
	}
	if !strings.Contains(got, "custom-cmd") {
		t.Errorf("expected 'custom-cmd' in ADDITIONAL COMMANDS, got:\n%s", got)
	}
}

func TestRootUsageFunc_UsageAndDescription(t *testing.T) {
	root := &ffcli.Command{
		Name:       "gplay",
		ShortUsage: "gplay <command> [flags]",
		ShortHelp:  "A CLI for Google Play Console.",
		Subcommands: []*ffcli.Command{
			{Name: "apps", ShortHelp: "List apps"},
		},
	}

	got := RootUsageFunc(root)
	if !strings.Contains(got, "USAGE") {
		t.Errorf("expected USAGE section, got:\n%s", got)
	}
	if !strings.Contains(got, "gplay <command> [flags]") {
		t.Errorf("expected short usage text, got:\n%s", got)
	}
}

func TestRootUsageFunc_NoSubcommands(t *testing.T) {
	root := &ffcli.Command{
		Name:       "gplay",
		ShortUsage: "gplay <command> [flags]",
	}

	got := RootUsageFunc(root)
	// Should not panic and should still have USAGE
	if !strings.Contains(got, "USAGE") {
		t.Errorf("expected USAGE section even with no subcommands, got:\n%s", got)
	}
}
