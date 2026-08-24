package cmd

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/registry"
	cliruntime "github.com/tamtom/play-console-cli/internal/cli/runtime"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// RootCommand constructs the root CLI command with all subcommands.
func RootCommand(version string) *ffcli.Command {
	root, _ := constructRootCommand(version)
	return root
}

func constructRootCommand(version string) (*ffcli.Command, *cliruntime.Runtime) {
	rootFS := flag.NewFlagSet("gplay", flag.ExitOnError)
	rt := cliruntime.NewRoot(rootFS)
	catalog := registry.NewCatalog(version, rt)
	return newRootCommand(rootFS, rt, catalog.All())
}

// constructRootCommandForArgs builds complete metadata for root help while
// materializing only the selected root family for normal execution.
func constructRootCommandForArgs(version string, args []string) (*ffcli.Command, *cliruntime.Runtime) {
	rootFS := flag.NewFlagSet("gplay", flag.ExitOnError)
	rt := cliruntime.NewRoot(rootFS)
	catalog := registry.NewCatalog(version, rt)
	commands := catalog.MetadataCommands()
	if selected := selectedRootCommand(args); selected != "" {
		commands = catalog.CommandsFor(selected)
	}
	return newRootCommand(rootFS, rt, commands)
}

func newRootCommand(rootFS *flag.FlagSet, rt *cliruntime.Runtime, subcommands []*ffcli.Command) (*ffcli.Command, *cliruntime.Runtime) {
	var root *ffcli.Command
	root = &ffcli.Command{
		Name:        "gplay",
		ShortUsage:  "gplay <command> [flags]",
		ShortHelp:   "A CLI for Google Play Console.",
		FlagSet:     rootFS,
		UsageFunc:   RootUsageFunc,
		Subcommands: subcommands,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			var names []string
			for _, sub := range root.Subcommands {
				names = append(names, sub.Name)
			}
			fmt.Fprintln(shared.Stderr(ctx), shared.FormatUnknownCommand(args[0], names))
			return flag.ErrHelp
		},
	}

	return root, rt
}

func selectedRootCommand(args []string) string {
	valueFlags := map[string]bool{
		"profile":     true,
		"report":      true,
		"report-file": true,
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			if i+1 < len(args) {
				return strings.TrimSpace(args[i+1])
			}
			return ""
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			return strings.TrimSpace(arg)
		}

		name := strings.TrimLeft(arg, "-")
		if _, _, found := strings.Cut(name, "="); found {
			continue
		}
		if valueFlags[name] && i+1 < len(args) {
			i++
		}
	}
	return ""
}
