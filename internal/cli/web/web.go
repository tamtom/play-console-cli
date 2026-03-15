package web

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

const baseURL = "https://play.google.com/console/developers/app"

// sectionPaths maps human-readable section names to Play Console URL paths.
var sectionPaths = map[string]string{
	"dashboard":     "app-dashboard",
	"store-listing": "store-listing",
	"releases":      "tracks",
	"pricing":       "pricing",
	"testers":       "closed-testing",
	"reviews":       "user-feedback/reviews",
	"vitals":        "vitals/overview",
	"statistics":    "statistics",
}

// browserOpener is a package-level function for opening URLs.
// It is overridden in tests.
var browserOpener = openBrowser

// WebCommand returns the web command group.
func WebCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "web",
		ShortUsage: "gplay web <subcommand> [flags]",
		ShortHelp:  "Open Google Play Console pages in the browser.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			OpenCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", args[0])
			return flag.ErrHelp
		},
	}
}

// OpenCommand returns the web open subcommand.
func OpenCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web open", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	section := fs.String("section", "", "Console section: "+strings.Join(validSections(), ", "))

	return &ffcli.Command{
		Name:       "open",
		ShortUsage: "gplay web open [--package <name>] [--section <section>]",
		ShortHelp:  "Open a Google Play Console page in the browser.",
		LongHelp: `Open a Google Play Console page in the default browser.

Sections:
  dashboard      App dashboard (default)
  store-listing  Store listing editor
  releases       Release tracks overview
  pricing        Pricing and distribution
  testers        Closed testing management
  reviews        User reviews and feedback
  vitals         Android vitals overview
  statistics     Statistics and metrics

Examples:
  gplay web open --package com.example.app
  gplay web open --package com.example.app --section vitals
  gplay web open --package com.example.app --section reviews`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			pkg := strings.TrimSpace(*packageName)
			if pkg == "" {
				return fmt.Errorf("--package is required")
			}

			sec := strings.TrimSpace(*section)
			if sec != "" {
				if _, ok := sectionPaths[sec]; !ok {
					return fmt.Errorf("invalid section %q; valid sections: %s", sec, strings.Join(validSections(), ", "))
				}
			}

			url := buildPlayConsoleURL(pkg, sec)
			fmt.Fprintf(os.Stderr, "Opening %s\n", url)
			return browserOpener(url)
		},
	}
}

// buildPlayConsoleURL constructs a Play Console URL for the given package and section.
func buildPlayConsoleURL(packageName, section string) string {
	section = strings.TrimSpace(section)
	path, ok := sectionPaths[section]
	if !ok {
		path = sectionPaths["dashboard"]
	}
	return fmt.Sprintf("%s/%s/%s", baseURL, packageName, path)
}

// validSections returns sorted valid section names.
func validSections() []string {
	sections := make([]string, 0, len(sectionPaths))
	for s := range sectionPaths {
		sections = append(sections, s)
	}
	sort.Strings(sections)
	return sections
}

// openBrowser opens a URL in the default browser.
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return exec.Command(cmd, args...).Start()
}
