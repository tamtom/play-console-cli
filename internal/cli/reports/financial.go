package reports

import (
	"context"
	"flag"
	"fmt"
	"regexp"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

var monthRegex = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

var validReportTypes = map[string]bool{
	"earnings": true,
	"sales":    true,
	"payouts":  true,
}

// validateMonth checks that a month string matches YYYY-MM format.
func validateMonth(value, flagName string) error {
	if !monthRegex.MatchString(value) {
		return fmt.Errorf("--%s must be in YYYY-MM format (got %q)", flagName, value)
	}
	return nil
}

// validateReportType checks that a report type is valid.
func validateReportType(value string) error {
	if value == "all" {
		return nil
	}
	if !validReportTypes[value] {
		return fmt.Errorf("--type must be one of: earnings, sales, payouts, all (got %q)", value)
	}
	return nil
}

// FinancialCommand returns the financial subcommand group.
func FinancialCommand() *ffcli.Command {
	fs := flag.NewFlagSet("financial", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "financial",
		ShortUsage: "gplay reports financial <subcommand> [flags]",
		ShortHelp:  "Manage financial reports (earnings, sales, payouts).",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			FinancialListCommand(),
			FinancialDownloadCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// FinancialListCommand returns the financial list subcommand.
func FinancialListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("financial list", flag.ExitOnError)
	developer := fs.String("developer", "", "Developer ID (required)")
	from := fs.String("from", "", "Start month in YYYY-MM format")
	to := fs.String("to", "", "End month in YYYY-MM format")
	reportType := fs.String("type", "all", "Report type: earnings, sales, payouts, all")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay reports financial list --developer <id> [flags]",
		ShortHelp:  "List available financial reports.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*developer) == "" {
				return fmt.Errorf("--developer is required")
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
			if err := validateReportType(*reportType); err != nil {
				return err
			}

			// Stub: in a real implementation this would list reports from
			// Google Cloud Storage bucket pubsite_prod_rev_<developer_id>.
			result := map[string]interface{}{
				"developer": *developer,
				"type":      *reportType,
				"from":      *from,
				"to":        *to,
				"reports":   []interface{}{},
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}

// FinancialDownloadCommand returns the financial download subcommand.
func FinancialDownloadCommand() *ffcli.Command {
	fs := flag.NewFlagSet("financial download", flag.ExitOnError)
	developer := fs.String("developer", "", "Developer ID (required)")
	from := fs.String("from", "", "Start month in YYYY-MM format (required)")
	to := fs.String("to", "", "End month in YYYY-MM format (defaults to --from)")
	reportType := fs.String("type", "earnings", "Report type: earnings, sales, payouts")
	dir := fs.String("dir", ".", "Output directory")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "download",
		ShortUsage: "gplay reports financial download --developer <id> --from <YYYY-MM> [flags]",
		ShortHelp:  "Download financial reports.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*developer) == "" {
				return fmt.Errorf("--developer is required")
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
			if err := validateReportType(*reportType); err != nil {
				return err
			}
			// "all" is not valid for download — you must pick a specific type.
			if *reportType == "all" {
				return fmt.Errorf("--type must be one of: earnings, sales, payouts (got \"all\")")
			}

			// Stub: in a real implementation this would download from
			// Google Cloud Storage bucket pubsite_prod_rev_<developer_id>.
			result := map[string]interface{}{
				"developer": *developer,
				"type":      *reportType,
				"from":      *from,
				"to":        effectiveTo,
				"dir":       *dir,
				"files":     []interface{}{},
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}
