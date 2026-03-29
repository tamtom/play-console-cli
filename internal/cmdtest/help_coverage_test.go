package cmdtest_test

import (
	"flag"
	"strings"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"
)

// walkCommands recursively visits every command in the tree, calling fn with
// the dotted path (e.g. "onetimeproducts.create") and the command.
func walkCommands(cmd *ffcli.Command, path string, fn func(path string, cmd *ffcli.Command)) {
	fn(path, cmd)
	for _, sub := range cmd.Subcommands {
		p := sub.Name
		if path != "" {
			p = path + "." + sub.Name
		}
		walkCommands(sub, p, fn)
	}
}

// hasFlag returns true if the command's FlagSet contains a flag with the given name.
func hasFlag(cmd *ffcli.Command, name string) bool {
	if cmd.FlagSet == nil {
		return false
	}
	found := false
	cmd.FlagSet.VisitAll(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// TestAllJSONCommandsHaveLongHelp ensures every command that accepts a --json
// flag has LongHelp with a JSON example. This prevents future commands from
// shipping without guidance on the expected JSON format.
func TestAllJSONCommandsHaveLongHelp(t *testing.T) {
	root := RootCommand("test")
	walkCommands(root, "", func(path string, cmd *ffcli.Command) {
		if !hasFlag(cmd, "json") {
			return
		}
		if cmd.LongHelp == "" {
			t.Errorf("command %q has --json flag but no LongHelp", path)
		}
		if cmd.LongHelp != "" && !strings.Contains(cmd.LongHelp, "{") {
			t.Errorf("command %q has --json flag but LongHelp has no JSON example (missing '{')", path)
		}
	})
}

// TestAllCommandsHaveUsageFunc ensures every command in the tree has
// UsageFunc set to produce consistent help output.
func TestAllCommandsHaveUsageFunc(t *testing.T) {
	root := RootCommand("test")
	walkCommands(root, "", func(path string, cmd *ffcli.Command) {
		if path == "" {
			return // root command uses its own usage func
		}
		if cmd.UsageFunc == nil {
			t.Errorf("command %q missing UsageFunc", path)
		}
	})
}
