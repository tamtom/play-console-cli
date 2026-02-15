package vitals

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// AnomaliesCommand returns the "gplay vitals crashes anomalies" command.
func AnomaliesCommand() *ffcli.Command {
	fs := flag.NewFlagSet("crashes anomalies", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	_ = fs.String("output", "json", "Output format: json (default), table, markdown")
	_ = fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "anomalies",
		ShortUsage: "gplay vitals crashes anomalies --package <pkg>",
		ShortHelp:  "List detected anomalies for crash and ANR metrics.",
		LongHelp: `List detected anomalies for crash and ANR metrics.

Anomalies are automatically detected deviations in crash or ANR rates
that may indicate a regression introduced by a new release.

Note: This command uses the Play Developer Reporting API, which is
separate from the Android Publisher API used by other commands.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if strings.TrimSpace(*packageName) == "" {
				return fmt.Errorf("--package is required")
			}

			return fmt.Errorf(
				"Play Developer Reporting API client is not yet connected. "+
					"Would list anomalies for app %s",
				*packageName,
			)
		},
	}
}
