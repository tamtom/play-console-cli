package docs

import (
	"bytes"
	"context"
	"flag"
	"strings"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

func TestDocsCommand_Name(t *testing.T) {
	cmd := DocsCommand()
	if cmd.Name != "docs" {
		t.Errorf("Name = %q, want %q", cmd.Name, "docs")
	}
}

func TestGenerateMarkdown(t *testing.T) {
	// Create a mock command tree
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	fs.String("output", "json", "Output format")

	root := &ffcli.Command{
		Name:       "gplay",
		ShortUsage: "gplay <command> [flags]",
		ShortHelp:  "A CLI tool.",
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			{
				Name:       "test",
				ShortUsage: "gplay test [flags]",
				ShortHelp:  "A test command.",
				FlagSet:    fs,
				UsageFunc:  shared.DefaultUsageFunc,
				Exec:       func(ctx context.Context, args []string) error { return nil },
			},
		},
	}

	var buf bytes.Buffer
	generateMarkdown(&buf, root, "")
	out := buf.String()

	if !strings.Contains(out, "# gplay CLI Reference") {
		t.Error("expected header")
	}
	if !strings.Contains(out, "## gplay test") {
		t.Error("expected test command section")
	}
	if !strings.Contains(out, "--output") {
		t.Error("expected flag listing")
	}
}
