package cmd

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/peterbourgon/ff/v3/ffcli"
)

type commandGroup struct {
	title    string
	commands []string
}

var rootCommandGroups = []commandGroup{
	{title: "GETTING STARTED", commands: []string{"auth", "init", "docs"}},
	{title: "APP MANAGEMENT", commands: []string{"apps", "listings", "images", "details", "data-safety", "availability", "device-tiers"}},
	{title: "RELEASES & TRACKS", commands: []string{"edits", "bundles", "apks", "tracks", "release", "promote", "rollout", "sync", "validate", "deobfuscation", "expansion", "generated-apks", "system-apks"}},
	{title: "TESTING", commands: []string{"testers", "internal-sharing"}},
	{title: "VITALS & REVIEWS", commands: []string{"status", "vitals", "reviews"}},
	{title: "MONETIZATION", commands: []string{"iap", "subscriptions", "base-plans", "offers", "one-time-products", "purchase-options", "otp-offers", "pricing", "orders", "purchases", "external-transactions"}},
	{title: "ACCOUNT & ACCESS", commands: []string{"users", "grants"}},
	{title: "AUTOMATION", commands: []string{"notify", "migrate", "release-notes", "reports", "recovery"}},
	{title: "UTILITIES", commands: []string{"version", "update", "completion", "docs"}},
}

// RootUsageFunc renders grouped help for the root gplay command.
func RootUsageFunc(c *ffcli.Command) string {
	var b strings.Builder

	fmt.Fprintf(&b, "USAGE\n  %s\n\n", c.ShortUsage)
	if c.ShortHelp != "" {
		fmt.Fprintf(&b, "%s\n\n", c.ShortHelp)
	}

	// Build lookup of subcommands by name
	subByName := make(map[string]*ffcli.Command)
	for _, sub := range c.Subcommands {
		subByName[sub.Name] = sub
	}

	// Render grouped commands
	grouped := make(map[string]bool)
	for _, group := range rootCommandGroups {
		var entries []*ffcli.Command
		for _, name := range group.commands {
			if sub, ok := subByName[name]; ok {
				if !strings.HasPrefix(sub.ShortHelp, "DEPRECATED:") {
					entries = append(entries, sub)
					grouped[name] = true
				}
			}
		}
		if len(entries) == 0 {
			continue
		}
		fmt.Fprintf(&b, "%s\n", group.title)
		tw := tabwriter.NewWriter(&b, 2, 4, 2, ' ', 0)
		for _, sub := range entries {
			fmt.Fprintf(tw, "  %s\t%s\n", sub.Name, sub.ShortHelp)
		}
		tw.Flush()
		fmt.Fprintln(&b)
	}

	// Render ungrouped commands
	var ungrouped []*ffcli.Command
	for _, sub := range c.Subcommands {
		if !grouped[sub.Name] && !strings.HasPrefix(sub.ShortHelp, "DEPRECATED:") {
			ungrouped = append(ungrouped, sub)
		}
	}
	if len(ungrouped) > 0 {
		fmt.Fprintf(&b, "ADDITIONAL COMMANDS\n")
		tw := tabwriter.NewWriter(&b, 2, 4, 2, ' ', 0)
		for _, sub := range ungrouped {
			fmt.Fprintf(tw, "  %s\t%s\n", sub.Name, sub.ShortHelp)
		}
		tw.Flush()
		fmt.Fprintln(&b)
	}

	return b.String()
}
