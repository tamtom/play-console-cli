package cmdtest_test

import (
	"strings"
	"testing"
)

func TestExperimentsBoundaryIsDiscoverable(t *testing.T) {
	root := RootCommand("test")
	var usage string
	for _, command := range root.Subcommands {
		if command.Name == "experiments" {
			usage = command.UsageFunc(command)
			break
		}
	}
	for _, want := range []string{"lifecycle and results remain manual", "support", "apply-winner"} {
		if !strings.Contains(strings.ToLower(usage), want) {
			t.Fatalf("experiments help missing %q:\n%s", want, usage)
		}
	}
}
