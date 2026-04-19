package bundles

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/bundleanalysis"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// AnalyzeCommand parses an AAB/APK and reports size breakdown.
func AnalyzeCommand() *ffcli.Command {
	fs := flag.NewFlagSet("bundles analyze", flag.ExitOnError)
	file := fs.String("file", "", "Path to .aab or .apk (required)")
	top := fs.Int("top-files", 20, "Number of largest individual files to include")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "analyze",
		ShortUsage: "gplay bundles analyze --file <app.aab>",
		ShortHelp:  "Analyze a local AAB/APK: size per module, bucket, and largest files.",
		LongHelp: `Analyze an AAB/APK offline (no Play API calls).

The analyzer parses the archive's ZIP structure and groups entries into
buckets (dex/resources/native/assets/etc.) and AAB modules. Use --top-files
to surface the largest individual entries.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*file) == "" {
				return fmt.Errorf("--file is required")
			}
			if *top < 0 {
				return fmt.Errorf("--top-files must be non-negative")
			}
			result, err := bundleanalysis.Analyze(*file, bundleanalysis.Options{TopFiles: *top})
			if err != nil {
				return err
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}

// CompareCommand diffs two AAB/APK files and optionally flags regressions.
func CompareCommand() *ffcli.Command {
	fs := flag.NewFlagSet("bundles compare", flag.ExitOnError)
	base := fs.String("base", "", "Baseline AAB/APK (required)")
	candidate := fs.String("candidate", "", "Candidate AAB/APK (required)")
	threshold := fs.String("threshold", "", "Regression threshold in bytes (e.g. 500K, 2M, 1G)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "compare",
		ShortUsage: "gplay bundles compare --base <a.aab> --candidate <b.aab> [--threshold 2M]",
		ShortHelp:  "Diff two AAB/APK files and flag size regressions.",
		LongHelp: `Diff two AAB/APK files and flag size regressions for CI.

When --threshold is set, exits non-zero if the uncompressed delta exceeds it.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*base) == "" {
				return fmt.Errorf("--base is required")
			}
			if strings.TrimSpace(*candidate) == "" {
				return fmt.Errorf("--candidate is required")
			}
			t, err := bundleanalysis.ParseSizeThreshold(*threshold)
			if err != nil {
				return fmt.Errorf("--threshold: %w", err)
			}
			baseA, err := bundleanalysis.Analyze(*base, bundleanalysis.Options{})
			if err != nil {
				return fmt.Errorf("analyze base: %w", err)
			}
			candA, err := bundleanalysis.Analyze(*candidate, bundleanalysis.Options{})
			if err != nil {
				return fmt.Errorf("analyze candidate: %w", err)
			}
			diff := bundleanalysis.Compare(baseA, candA, t)
			if err := shared.PrintOutput(diff, *outputFlag, *pretty); err != nil {
				return err
			}
			if diff.Regression {
				return shared.NewReportedError(fmt.Errorf(
					"bundle regression: +%d uncompressed bytes exceeds threshold %d",
					diff.DeltaUncompr, t,
				))
			}
			return nil
		},
	}
}
