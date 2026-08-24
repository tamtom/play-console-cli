package cmdtest_test

import (
	"strings"
	"testing"
)

func TestInstallSkillsHelpAdvertisesVerificationAndSafeModes(t *testing.T) {
	root := RootCommand("test")
	var commandUsage string
	for _, command := range root.Subcommands {
		if command.Name == "install-skills" {
			commandUsage = command.UsageFunc(command)
			break
		}
	}
	if commandUsage == "" {
		t.Fatal("install-skills command is not registered")
	}
	for _, want := range []string{"immutable reviewed commit", "tree hash", "--preview", "--force", "--dest"} {
		if !strings.Contains(commandUsage, want) {
			t.Fatalf("help missing %q:\n%s", want, commandUsage)
		}
	}
}
