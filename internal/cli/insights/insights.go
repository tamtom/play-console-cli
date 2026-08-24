package insights

import (
	"context"
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	domain "github.com/tamtom/play-console-cli/internal/insights"
	"github.com/tamtom/play-console-cli/internal/output"
)

func init() {
	output.RegisterType(domain.Report{}, []string{"METRIC", "CURRENT", "PREVIOUS", "DELTA", "DELTA %", "STATUS", "REASON"}, func(data any) [][]string {
		return metricRows(data.(domain.Report).Metrics)
	})
	output.RegisterType(domain.DailyReport{}, []string{"METRIC", "CURRENT", "PREVIOUS", "DELTA", "DELTA %", "STATUS", "REASON"}, func(data any) [][]string {
		return metricRows(data.(domain.DailyReport).Metrics)
	})
}

// InsightsCommand returns the local-first insights command group.
func InsightsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("insights", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "insights",
		ShortUsage: "gplay insights <subcommand> [flags]",
		ShortHelp:  "Compare trends from official Google Play report exports.",
		LongHelp: `Compare Google Play trends from official CSV report exports.

Insights are local and deterministic: they use no credentials, make no network
requests, and never modify a Play account. Download source files with
"gplay reports stats download" or Play Console's documented report export.`,
		FlagSet: fs, UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{WeeklyCommand(), DailyCommand()},
		Exec:        func(context.Context, []string) error { return flag.ErrHelp },
	}
}

// DailyCommand compares one official report date with the preceding date.
func DailyCommand() *ffcli.Command {
	fs := flag.NewFlagSet("insights daily", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	date := fs.String("date", "", "UTC report date to compare (YYYY-MM-DD)")
	installsFile := fs.String("installs-file", "", "Official installs statistics CSV for one breakdown")
	crashesFile := fs.String("crashes-file", "", "Official crashes statistics CSV for one breakdown")
	storeFile := fs.String("store-performance-file", "", "Official store performance CSV for one breakdown")
	outputFlags := shared.BindOutputFlags(fs)
	return &ffcli.Command{
		Name:       "daily",
		ShortUsage: "gplay insights daily --package <name> --date <YYYY-MM-DD> [report files]",
		ShortHelp:  "Compare one day with the preceding day using official local CSV exports.",
		LongHelp: `Compare one UTC report date with the preceding date.

The same official, local-only input and unavailable-metric guarantees as
"insights weekly" apply. Supply at least one report file.

Example:
  gplay insights daily --package com.example.app --date 2026-08-24 --crashes-file ./crashes.csv --output table`,
		FlagSet: fs, UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			_ = ctx
			if len(args) != 0 {
				return shared.UsageErrorf("unexpected argument(s): %s", strings.Join(args, " "))
			}
			if strings.TrimSpace(*packageName) == "" {
				return shared.UsageError("--package is required")
			}
			if strings.TrimSpace(*date) == "" {
				return shared.UsageError("--date is required")
			}
			if strings.TrimSpace(*installsFile) == "" && strings.TrimSpace(*crashesFile) == "" && strings.TrimSpace(*storeFile) == "" {
				return shared.UsageError("at least one of --installs-file, --crashes-file, or --store-performance-file is required")
			}
			if err := shared.ValidateOutputFlags(outputFlags.Format(), outputFlags.IsPretty()); err != nil {
				return err
			}
			report, err := domain.Daily(domain.DailyRequest{
				Package:              *packageName,
				Date:                 *date,
				InstallsFile:         *installsFile,
				CrashesFile:          *crashesFile,
				StorePerformanceFile: *storeFile,
			})
			if err != nil {
				return fmt.Errorf("build daily insights: %w", err)
			}
			return shared.PrintOutputContext(ctx, report, outputFlags.Format(), outputFlags.IsPretty())
		},
	}
}

// WeeklyCommand compares one Monday-to-Sunday window with the prior week.
func WeeklyCommand() *ffcli.Command {
	fs := flag.NewFlagSet("insights weekly", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	week := fs.String("week", "", "Monday starting the comparison week (YYYY-MM-DD)")
	installsFile := fs.String("installs-file", "", "Official installs statistics CSV for one breakdown")
	crashesFile := fs.String("crashes-file", "", "Official crashes statistics CSV for one breakdown")
	storeFile := fs.String("store-performance-file", "", "Official store performance CSV for one breakdown")
	outputFlags := shared.BindOutputFlags(fs)
	return &ffcli.Command{
		Name:       "weekly",
		ShortUsage: "gplay insights weekly --package <name> --week <Monday> [report files]",
		ShortHelp:  "Compare weekly trends with the previous week using official local CSV exports.",
		LongHelp: `Compare a Monday-to-Sunday window with the immediately preceding week.

Inputs are the official Google Play monthly CSV exports. Supply at least one
file. Each file must represent one breakdown/dimension so totals are not double
counted. UTF-8 and Google Play's UTF-16 exports are supported.

This command is entirely local: it uses no credentials, contacts no API, and
does not change Google Play. Missing sources or columns are reported as
unavailable instead of inventing values.

Examples:
  gplay insights weekly --package com.example.app --week 2026-08-17 --installs-file ./installs_com.example.app_202608_country.csv
  gplay insights weekly --package com.example.app --week 2026-08-17 --installs-file ./installs.csv --crashes-file ./crashes.csv --store-performance-file ./store.csv --output table`,
		FlagSet: fs, UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			_ = ctx
			if len(args) != 0 {
				return shared.UsageErrorf("unexpected argument(s): %s", strings.Join(args, " "))
			}
			if strings.TrimSpace(*packageName) == "" {
				return shared.UsageError("--package is required")
			}
			if strings.TrimSpace(*week) == "" {
				return shared.UsageError("--week is required")
			}
			if strings.TrimSpace(*installsFile) == "" && strings.TrimSpace(*crashesFile) == "" && strings.TrimSpace(*storeFile) == "" {
				return shared.UsageError("at least one of --installs-file, --crashes-file, or --store-performance-file is required")
			}
			if err := shared.ValidateOutputFlags(outputFlags.Format(), outputFlags.IsPretty()); err != nil {
				return err
			}
			report, err := domain.Weekly(domain.WeeklyRequest{
				Package:              *packageName,
				Week:                 *week,
				InstallsFile:         *installsFile,
				CrashesFile:          *crashesFile,
				StorePerformanceFile: *storeFile,
			})
			if err != nil {
				return fmt.Errorf("build weekly insights: %w", err)
			}
			return shared.PrintOutputContext(ctx, report, outputFlags.Format(), outputFlags.IsPretty())
		},
	}
}

func formatNumber(value *float64) string {
	if value == nil {
		return "—"
	}
	return strconv.FormatFloat(*value, 'f', -1, 64)
}

func metricRows(metrics []domain.Metric) [][]string {
	rows := make([][]string, 0, len(metrics))
	for _, metric := range metrics {
		rows = append(rows, []string{
			metric.Name,
			formatNumber(metric.Current),
			formatNumber(metric.Previous),
			formatNumber(metric.Delta),
			formatNumber(metric.DeltaPercent),
			metric.Status,
			metric.Reason,
		})
	}
	return rows
}
