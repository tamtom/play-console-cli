package docs

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// ShowCommand returns the docs show subcommand.
func ShowCommand() *ffcli.Command {
	fs := flag.NewFlagSet("docs show", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "show",
		ShortUsage: "gplay docs show <topic>",
		ShortHelp:  "Print an embedded documentation guide.",
		LongHelp: `Print an embedded documentation guide.

Use 'gplay docs list' to see available topics.

Examples:
  gplay docs show auth-setup
  gplay docs show release-workflow
  gplay docs show troubleshooting`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			if len(args) > 1 {
				fmt.Fprintln(os.Stderr, "Error: docs show accepts exactly one topic name")
				return flag.ErrHelp
			}

			topicName := strings.TrimSpace(args[0])
			topic, ok := findTopic(topicName)
			if !ok {
				fmt.Fprintf(os.Stderr, "Error: unknown topic %q\n", topicName)
				fmt.Fprintf(os.Stderr, "Available topics: %s\n", strings.Join(topicSlugs(), ", "))
				return flag.ErrHelp
			}

			content := topic.Content
			if !strings.HasSuffix(content, "\n") {
				content += "\n"
			}
			fmt.Fprint(os.Stdout, content)
			return nil
		},
	}
}

// ListCommand returns the docs list subcommand.
func ListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("docs list", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay docs list",
		ShortHelp:  "List available documentation topics.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			fmt.Fprintln(os.Stdout, "Available documentation topics:")
			fmt.Fprintln(os.Stdout)

			tw := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
			for _, topic := range topicRegistry {
				fmt.Fprintf(tw, "  %s\t%s\n", topic.Slug, topic.Description)
			}
			tw.Flush()

			fmt.Fprintln(os.Stdout)
			fmt.Fprintln(os.Stdout, "Use 'gplay docs show <topic>' to view a topic.")
			return nil
		},
	}
}
