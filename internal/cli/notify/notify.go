package notify

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// NotifyCommand returns the parent "notify" command group.
func NotifyCommand() *ffcli.Command {
	fs := flag.NewFlagSet("notify", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "notify",
		ShortUsage: "gplay notify <subcommand> [flags]",
		ShortHelp:  "Send notifications to Slack, Discord, or HTTP webhooks.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			SendCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}
