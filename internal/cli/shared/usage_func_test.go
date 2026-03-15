package shared

import (
	"flag"
	"strings"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func TestDefaultUsageFunc_UsageSection(t *testing.T) {
	cmd := &ffcli.Command{
		Name:       "test",
		ShortUsage: "gplay test [flags]",
		ShortHelp:  "A test command",
	}

	got := DefaultUsageFunc(cmd)
	if !strings.Contains(got, "USAGE") {
		t.Errorf("expected USAGE section, got:\n%s", got)
	}
	if !strings.Contains(got, "gplay test [flags]") {
		t.Errorf("expected short usage in output, got:\n%s", got)
	}
}

func TestDefaultUsageFunc_SubcommandsSection(t *testing.T) {
	cmd := &ffcli.Command{
		Name:       "test",
		ShortUsage: "gplay test <subcommand>",
		Subcommands: []*ffcli.Command{
			{Name: "list", ShortHelp: "List items"},
			{Name: "get", ShortHelp: "Get an item"},
		},
	}

	got := DefaultUsageFunc(cmd)
	if !strings.Contains(got, "SUBCOMMANDS") {
		t.Errorf("expected SUBCOMMANDS section, got:\n%s", got)
	}
	if !strings.Contains(got, "list") {
		t.Errorf("expected 'list' subcommand in output, got:\n%s", got)
	}
	if !strings.Contains(got, "get") {
		t.Errorf("expected 'get' subcommand in output, got:\n%s", got)
	}
}

func TestDefaultUsageFunc_FlagsSection(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.String("package", "", "Package name")
	fs.String("output", "json", "Output format")

	cmd := &ffcli.Command{
		Name:       "test",
		ShortUsage: "gplay test [flags]",
		FlagSet:    fs,
	}

	got := DefaultUsageFunc(cmd)
	if !strings.Contains(got, "FLAGS") {
		t.Errorf("expected FLAGS section, got:\n%s", got)
	}
	if !strings.Contains(got, "--package") {
		t.Errorf("expected --package flag in output, got:\n%s", got)
	}
	if !strings.Contains(got, "--output") {
		t.Errorf("expected --output flag in output, got:\n%s", got)
	}
}

func TestDefaultUsageFunc_DeprecatedCommandsHidden(t *testing.T) {
	cmd := &ffcli.Command{
		Name:       "test",
		ShortUsage: "gplay test <subcommand>",
		Subcommands: []*ffcli.Command{
			{Name: "active", ShortHelp: "An active command"},
			{Name: "old", ShortHelp: "DEPRECATED: use 'active' instead"},
		},
	}

	got := DefaultUsageFunc(cmd)
	if !strings.Contains(got, "active") {
		t.Errorf("expected 'active' subcommand in output, got:\n%s", got)
	}
	if strings.Contains(got, "old") {
		t.Errorf("deprecated command 'old' should be hidden, got:\n%s", got)
	}
}

func TestDefaultUsageFunc_FlagDefaults(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.String("output", "json", "Output format")
	fs.Bool("verbose", false, "Enable verbose output")
	fs.Int("limit", 0, "Max items to return")

	cmd := &ffcli.Command{
		Name:       "test",
		ShortUsage: "gplay test [flags]",
		FlagSet:    fs,
	}

	got := DefaultUsageFunc(cmd)

	// Non-empty, non-false, non-zero defaults should show "(default: ...)"
	if !strings.Contains(got, "(default: json)") {
		t.Errorf("expected default value for --output, got:\n%s", got)
	}

	// false and 0 defaults should NOT show "(default: ...)" for their lines
	lines := strings.Split(got, "\n")
	for _, line := range lines {
		if strings.Contains(line, "--verbose") && strings.Contains(line, "(default:") {
			t.Errorf("--verbose (false default) should not show default, got line: %s", line)
		}
		if strings.Contains(line, "--limit") && strings.Contains(line, "(default:") {
			t.Errorf("--limit (0 default) should not show default, got line: %s", line)
		}
	}
}

func TestDefaultUsageFunc_DescriptionFromLongHelp(t *testing.T) {
	cmd := &ffcli.Command{
		Name:       "test",
		ShortUsage: "gplay test [flags]",
		LongHelp:   "This is a detailed description of the test command.",
	}

	got := DefaultUsageFunc(cmd)
	if !strings.Contains(got, "This is a detailed description") {
		t.Errorf("expected LongHelp in output, got:\n%s", got)
	}
}

func TestDefaultUsageFunc_DescriptionFromShortHelp(t *testing.T) {
	cmd := &ffcli.Command{
		Name:       "test",
		ShortUsage: "gplay test [flags]",
		ShortHelp:  "A short description",
	}

	got := DefaultUsageFunc(cmd)
	if !strings.Contains(got, "A short description") {
		t.Errorf("expected ShortHelp in output, got:\n%s", got)
	}
}

func TestDefaultUsageFunc_NoFlags(t *testing.T) {
	cmd := &ffcli.Command{
		Name:       "test",
		ShortUsage: "gplay test",
	}

	got := DefaultUsageFunc(cmd)
	if strings.Contains(got, "FLAGS") {
		t.Errorf("expected no FLAGS section when no flags defined, got:\n%s", got)
	}
}

func TestDefaultUsageFunc_NoSubcommands(t *testing.T) {
	cmd := &ffcli.Command{
		Name:       "test",
		ShortUsage: "gplay test",
	}

	got := DefaultUsageFunc(cmd)
	if strings.Contains(got, "SUBCOMMANDS") {
		t.Errorf("expected no SUBCOMMANDS section when no subcommands, got:\n%s", got)
	}
}
