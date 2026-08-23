package capabilities

import (
	"context"
	"flag"
	"fmt"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/capability"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/output"
)

func init() {
	output.RegisterType([]capability.Capability{}, []string{"ID", "STATUS", "PROVIDER", "COMMAND", "INTENT"}, func(data any) [][]string {
		items := data.([]capability.Capability)
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			rows = append(rows, []string{item.ID, string(item.Status), item.Provider, item.Command, item.Intent})
		}
		return rows
	})
}

// Command returns the policy-aware capability inventory command.
func Command() *ffcli.Command {
	fs := flag.NewFlagSet("capabilities", flag.ExitOnError)
	status := fs.String("status", "", "Filter by status: official, manual, unsupported")
	provider := fs.String("provider", "", "Filter by provider")
	query := fs.String("query", "", "Filter by ID, intent, command, provider, or notes")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "capabilities",
		ShortUsage: "gplay capabilities [flags]",
		ShortHelp:  "Show policy-aware Google Play workflow capabilities.",
		LongHelp: `Show which workflows use documented Google APIs and which require a
manual Play Console handoff. This command is a static, offline inventory: it
does not authenticate, contact Google, or change any account.

Examples:
  gplay capabilities
  gplay capabilities --status manual
  gplay capabilities --provider android-publisher-api --output table`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("unexpected arguments: %v", args)
			}
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			items, err := capability.List(capability.Filter{
				Status:   capability.Status(*status),
				Provider: *provider,
				Query:    *query,
			})
			if err != nil {
				return err
			}
			return shared.PrintOutput(items, *outputFlag, *pretty)
		},
	}
}
