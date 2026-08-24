package cmdtest_test

import (
	"strings"
	"testing"
)

func TestAndroidLocalToolsAreDiscoverable(t *testing.T) {
	root := RootCommand("test")
	var usage string
	for _, command := range root.Subcommands {
		if command.Name == "android" {
			usage = command.UsageFunc(command)
			break
		}
	}
	for _, want := range []string{"local Android", "build", "signing", "screenshots"} {
		if !strings.Contains(usage, want) {
			t.Fatalf("android help missing %q:\n%s", want, usage)
		}
	}
}
