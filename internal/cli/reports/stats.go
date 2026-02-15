package reports

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/gcsclient"
)

var validStatsTypes = map[string]bool{
	"installs":          true,
	"ratings":           true,
	"crashes":           true,
	"store_performance": true,
	"subscriptions":     true,
}

// statsPrefixes maps stats types to their GCS prefix under stats/.
var statsPrefixes = map[string]string{
	"installs":          "stats/installs/",
	"ratings":           "stats/ratings/",
	"crashes":           "stats/crashes/",
	"store_performance": "stats/store_performance/",
	"subscriptions":     "stats/subscriptions/",
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

// statsPrefixesForType returns the GCS prefixes to search for a given stats type.
func statsPrefixesForType(statsType string) []string {
	if statsType == "all" {
		return []string{
			"stats/installs/",
			"stats/ratings/",
			"stats/crashes/",
			"stats/store_performance/",
			"stats/subscriptions/",
		}
	}
	if p, ok := statsPrefixes[statsType]; ok {
		return []string{p}
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
	developer := fs.String("developer", "", "Developer ID (required)")
	pkg := fs.String("package", "", "Package name (filters results by package)")
	from := fs.String("from", "", "Start month in YYYY-MM format")
	to := fs.String("to", "", "End month in YYYY-MM format")
	statsType := fs.String("type", "all", "Stats type: installs, ratings, crashes, store_performance, subscriptions, all")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay reports stats list --developer <id> [flags]",
		ShortHelp:  "List available statistics reports.",
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
			if err := validateStatsType(*statsType); err != nil {
				return err
			}

			svc, err := newGCSServiceFunc(ctx)
			if err != nil {
				return err
			}

			bucket := bucketName(*developer)
			prefixes := statsPrefixesForType(*statsType)

			var reports []gcsclient.ObjectInfo
			for _, prefix := range prefixes {
				objects, err := svc.ListObjects(ctx, bucket, prefix)
				if err != nil {
					return err
				}
				for _, obj := range objects {
					if *pkg != "" && !strings.Contains(obj.Name, *pkg) {
						continue
					}
					if !matchesDateRange(obj.Name, *from, *to) {
						continue
					}
					reports = append(reports, obj)
				}
			}

			result := map[string]interface{}{
				"developer": *developer,
				"bucket":    bucket,
				"reports":   reports,
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}

// StatsDownloadCommand returns the stats download subcommand.
func StatsDownloadCommand() *ffcli.Command {
	fs := flag.NewFlagSet("stats download", flag.ExitOnError)
	developer := fs.String("developer", "", "Developer ID (required)")
	pkg := fs.String("package", "", "Package name (required)")
	from := fs.String("from", "", "Start month in YYYY-MM format (required)")
	to := fs.String("to", "", "End month in YYYY-MM format (defaults to --from)")
	statsType := fs.String("type", "", "Stats type: installs, ratings, crashes, store_performance, subscriptions (required)")
	dir := fs.String("dir", ".", "Output directory")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "download",
		ShortUsage: "gplay reports stats download --developer <id> --package <name> --from <YYYY-MM> --type <type> [flags]",
		ShortHelp:  "Download statistics reports.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*developer) == "" {
				return fmt.Errorf("--developer is required")
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
			if *statsType == "all" {
				return fmt.Errorf("--type must be one of: installs, ratings, crashes, store_performance, subscriptions (got \"all\")")
			}

			svc, err := newGCSServiceFunc(ctx)
			if err != nil {
				return err
			}

			bucket := bucketName(*developer)
			prefix := statsPrefixes[*statsType]

			objects, err := svc.ListObjects(ctx, bucket, prefix)
			if err != nil {
				return err
			}

			var downloaded []map[string]interface{}
			for _, obj := range objects {
				if !strings.Contains(obj.Name, *pkg) {
					continue
				}
				if !matchesDateRange(obj.Name, *from, effectiveTo) {
					continue
				}
				localPath := filepath.Join(*dir, filepath.Base(obj.Name))
				if err := downloadFile(ctx, svc, bucket, obj.Name, localPath); err != nil {
					return fmt.Errorf("failed to download %s: %w", obj.Name, err)
				}
				downloaded = append(downloaded, map[string]interface{}{
					"name": obj.Name,
					"path": localPath,
					"size": obj.Size,
				})
			}

			result := map[string]interface{}{
				"developer": *developer,
				"package":   *pkg,
				"type":      *statsType,
				"from":      *from,
				"to":        effectiveTo,
				"dir":       *dir,
				"files":     downloaded,
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}
