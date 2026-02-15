package reports

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

var validStatsTypes = map[string]bool{
	"installs":          true,
	"ratings":           true,
	"crashes":           true,
	"store_performance": true,
	"subscriptions":     true,
}

// validateStatsType checks that a stats type is valid.
func validateStatsType(value string) error {
	if value == "all" {
		return nil
	}
	if !validStatsTypes[value] {
		return fmt.Errorf("--type must be one of: installs, ratings, crashes, store_performance, subscriptions, all (got %q)", value)
	}
	return nil
}

// StatsCommand returns the stats subcommand group.
func StatsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("stats", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "stats",
		ShortUsage: "gplay reports stats <subcommand> [flags]",
		ShortHelp:  "Download and list aggregated statistics reports (installs, ratings, crashes).",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			StatsListCommand(),
			StatsDownloadCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// StatsListCommand returns the stats list subcommand.
func StatsListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("stats list", flag.ExitOnError)
	pkg := fs.String("package", "", "Package name (required)")
	from := fs.String("from", "", "Start month in YYYY-MM format")
	to := fs.String("to", "", "End month in YYYY-MM format")
	statsType := fs.String("type", "all", "Stats type: installs, ratings, crashes, store_performance, subscriptions, all")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay reports stats list --package <name> [flags]",
		ShortHelp:  "List available statistics reports.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*pkg) == "" {
				return fmt.Errorf("--package is required")
			}
			if *from != "" {
				if err := validateMonth(*from, "from"); err != nil {
					return err
				}
			}
			if *to != "" {
				if err := validateMonth(*to, "to"); err != nil {
					return err
				}
			}
			if err := validateStatsType(*statsType); err != nil {
				return err
			}

			// Stub: in a real implementation this would list statistics CSVs from
			// Google Cloud Storage bucket pubsite_prod_rev_<developer_id>.
			result := map[string]interface{}{
				"package": *pkg,
				"type":    *statsType,
				"from":    *from,
				"to":      *to,
				"reports": []interface{}{},
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}

// StatsDownloadCommand returns the stats download subcommand.
func StatsDownloadCommand() *ffcli.Command {
	fs := flag.NewFlagSet("stats download", flag.ExitOnError)
	pkg := fs.String("package", "", "Package name (required)")
	from := fs.String("from", "", "Start month in YYYY-MM format (required)")
	to := fs.String("to", "", "End month in YYYY-MM format (defaults to --from)")
	statsType := fs.String("type", "", "Stats type: installs, ratings, crashes, store_performance, subscriptions (required)")
	dir := fs.String("dir", ".", "Output directory")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "download",
		ShortUsage: "gplay reports stats download --package <name> --from <YYYY-MM> --type <type> [flags]",
		ShortHelp:  "Download statistics reports.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*pkg) == "" {
				return fmt.Errorf("--package is required")
			}
			if strings.TrimSpace(*from) == "" {
				return fmt.Errorf("--from is required")
			}
			if err := validateMonth(*from, "from"); err != nil {
				return err
			}
			effectiveTo := *to
			if effectiveTo == "" {
				effectiveTo = *from
			} else {
				if err := validateMonth(effectiveTo, "to"); err != nil {
					return err
				}
			}
			if strings.TrimSpace(*statsType) == "" {
				return fmt.Errorf("--type is required")
			}
			if err := validateStatsType(*statsType); err != nil {
				return err
			}
			// "all" is not valid for download — you must pick a specific type.
			if *statsType == "all" {
				return fmt.Errorf("--type must be one of: installs, ratings, crashes, store_performance, subscriptions (got \"all\")")
			}

			// Stub: in a real implementation this would download statistics CSVs from
			// Google Cloud Storage bucket pubsite_prod_rev_<developer_id>.
			result := map[string]interface{}{
				"package": *pkg,
				"type":    *statsType,
				"from":    *from,
				"to":      effectiveTo,
				"dir":     *dir,
				"files":   []interface{}{},
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}
