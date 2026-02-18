package reports

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/gcsclient"
)

var monthRegex = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

// monthFromFilename extracts YYYYMM from a report filename.
// Looks for a 6-digit sequence that starts with 20 (e.g., 202601).
var monthFromFilenameRegex = regexp.MustCompile(`(20\d{4})`)

var validReportTypes = map[string]bool{
	"earnings": true,
	"sales":    true,
	"payouts":  true,
}

// financialPrefixes maps report types to their GCS prefix in the bucket.
var financialPrefixes = map[string]string{
	"earnings": "earnings/",
	"sales":    "sales/",
	"payouts":  "payouts/",
}

// newGCSServiceFunc is the factory for creating GCS services.
// It is overridden in tests to inject mock clients.
var newGCSServiceFunc = defaultNewGCSService

func defaultNewGCSService(ctx context.Context) (*gcsclient.Service, error) {
	return gcsclient.NewService(ctx)
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

// bucketName constructs the GCS bucket name from a developer ID.
func bucketName(developerID string) string {
	return "pubsite_prod_rev_" + developerID
}

// monthToCompact converts "2024-01" to "202401" for filename matching.
func monthToCompact(month string) string {
	return strings.ReplaceAll(month, "-", "")
}

// matchesDateRange checks if a filename's embedded YYYYMM falls within [from, to].
// If from/to are empty, no filtering is applied.
func matchesDateRange(name, from, to string) bool {
	if from == "" && to == "" {
		return true
	}
	matches := monthFromFilenameRegex.FindStringSubmatch(name)
	if len(matches) < 2 {
		return true // no date in filename — include it
	}
	fileMonth := matches[1]
	if from != "" && fileMonth < monthToCompact(from) {
		return false
	}
	if to != "" && fileMonth > monthToCompact(to) {
		return false
	}
	return true
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
	developer := fs.String("developer", "", "GCS developer ID (required; find via Play Console > Download reports > Cloud Storage URI)")
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

			svc, err := newGCSServiceFunc(ctx)
			if err != nil {
				return err
			}

			bucket := bucketName(*developer)
			prefixes := financialPrefixesForType(*reportType)

			var reports []gcsclient.ObjectInfo
			for _, prefix := range prefixes {
				objects, err := svc.ListObjects(ctx, bucket, prefix)
				if err != nil {
					return err
				}
				for _, obj := range objects {
					if matchesDateRange(obj.Name, *from, *to) {
						reports = append(reports, obj)
					}
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

// FinancialDownloadCommand returns the financial download subcommand.
func FinancialDownloadCommand() *ffcli.Command {
	fs := flag.NewFlagSet("financial download", flag.ExitOnError)
	developer := fs.String("developer", "", "GCS developer ID (required; find via Play Console > Download reports > Cloud Storage URI)")
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
			if *reportType == "all" {
				return fmt.Errorf("--type must be one of: earnings, sales, payouts (got \"all\")")
			}

			svc, err := newGCSServiceFunc(ctx)
			if err != nil {
				return err
			}

			bucket := bucketName(*developer)
			prefix := financialPrefixes[*reportType]

			objects, err := svc.ListObjects(ctx, bucket, prefix)
			if err != nil {
				return err
			}

			var downloaded []map[string]interface{}
			for _, obj := range objects {
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
				"type":      *reportType,
				"from":      *from,
				"to":        effectiveTo,
				"dir":       *dir,
				"files":     downloaded,
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}

// financialPrefixesForType returns the GCS prefixes to search for a given report type.
func financialPrefixesForType(reportType string) []string {
	if reportType == "all" {
		return []string{"earnings/", "sales/", "payouts/"}
	}
	if p, ok := financialPrefixes[reportType]; ok {
		return []string{p}
	}
	return nil
}

// downloadFile downloads a GCS object and writes it to a local file.
func downloadFile(ctx context.Context, svc *gcsclient.Service, bucket, object, localPath string) error {
	rc, err := svc.DownloadObject(ctx, bucket, object)
	if err != nil {
		return err
	}
	defer rc.Close()

	f, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, rc); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}
