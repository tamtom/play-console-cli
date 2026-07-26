// Package preflight wires the `gplay preflight` command.
package preflight

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/bundleanalysis"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	preflightpkg "github.com/tamtom/play-console-cli/internal/preflight"
)

// PreflightCommand is the root `gplay preflight`.
func PreflightCommand() *ffcli.Command {
	fs := flag.NewFlagSet("preflight", flag.ExitOnError)
	file := fs.String("file", "", "Path to .aab or .apk to scan (required)")
	listingsDir := fs.String("listings-dir", "", "Listings directory to validate (enables the metadata scanner)")
	only := fs.String("only", "", "Comma-separated scanners to run (default: all)")
	skip := fs.String("skip", "", "Comma-separated scanners to exclude")
	minTargetSDK := fs.Int("min-target-sdk", 0, "Minimum accepted targetSdkVersion (default: the current Play requirement)")
	maxSize := fs.String("max-size", "", "Max allowed bundle size (e.g. 150M)")
	maxDex := fs.String("max-dex", "", "Max allowed size per dex file (e.g. 64M)")
	skipSecrets := fs.Bool("skip-secrets", false, "Skip the secrets scanner (faster)")
	listScanners := fs.Bool("list-scanners", false, "Print the available scanner IDs and exit")
	severity := fs.String("fail-on", "error", "Exit non-zero when findings reach this severity: info, warning, error")
	outputFlag := fs.String("output", "text", "Output format: text (default), json, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "preflight",
		ShortUsage: "gplay preflight --file <app.aab> [flags]",
		ShortHelp:  "Run offline compliance and hygiene checks against an AAB/APK.",
		LongHelp: `Run offline checks against an AAB or APK without any API calls.

AndroidManifest.xml is fully decoded — binary AXML for APKs, aapt2 protobuf
for App Bundles — so checks read real, typed attribute values rather than
guessing from substrings.

Scanners:
  manifest     debuggable/testOnly flags, exported components, foreground
               service types, package and version sanity
  permissions  restricted permissions needing a Play declaration, sensitive
               permissions needing a Data safety disclosure, legacy storage
  native_libs  64-bit coverage, 16 KB page alignment, debug symbols
  metadata     listing text limits and real screenshot dimensions
               (requires --listings-dir)
  secrets      API keys, private keys, keystores, developer artifacts
  billing      Play Billing vs third-party payment processors
  privacy      analytics/ads SDKs and advertising ID consistency
  policy       target API level floor, restricted services, upload format
  size         download size budget, dex count, payload breakdown

Exit codes:
  0   no findings at or above --fail-on
  1   findings at or above --fail-on severity

Examples:
  gplay preflight --file app.aab
  gplay preflight --file app.aab --fail-on warning
  gplay preflight --file app.aab --listings-dir ./metadata
  gplay preflight --file app.aab --only manifest,permissions
  gplay preflight --file app.aab --skip size --output json | jq .
`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if *listScanners {
				for _, id := range preflightpkg.ScannerIDs() {
					fmt.Println(id)
				}
				return nil
			}
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*file) == "" {
				return fmt.Errorf("--file is required")
			}
			failOn, err := parseSeverity(*severity)
			if err != nil {
				return err
			}

			opts := preflightpkg.Options{
				SkipSecretScan: *skipSecrets,
				ListingsDir:    strings.TrimSpace(*listingsDir),
				MinTargetSDK:   *minTargetSDK,
			}
			if s := strings.TrimSpace(*only); s != "" {
				opts.Only = []string{s}
			}
			if s := strings.TrimSpace(*skip); s != "" {
				opts.Skip = []string{s}
			}
			if strings.TrimSpace(*maxSize) != "" {
				b, err := bundleanalysis.ParseSizeThreshold(*maxSize)
				if err != nil {
					return fmt.Errorf("--max-size: %w", err)
				}
				opts.MaxBundleBytes = b
			}
			if strings.TrimSpace(*maxDex) != "" {
				b, err := bundleanalysis.ParseSizeThreshold(*maxDex)
				if err != nil {
					return fmt.Errorf("--max-dex: %w", err)
				}
				opts.MaxDexBytes = b
			}

			report, err := preflightpkg.Scan(*file, opts)
			if err != nil {
				return err
			}

			if *outputFlag == "text" {
				printTextReport(report)
			} else if err := shared.PrintOutput(report, *outputFlag, *pretty); err != nil {
				return err
			}

			if report.AtOrAbove(failOn) {
				return shared.NewReportedError(fmt.Errorf(
					"preflight: %d error(s), %d warning(s), %d info",
					report.Errors, report.Warnings, report.Infos,
				))
			}
			return nil
		},
	}
}

func parseSeverity(s string) (preflightpkg.Severity, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "info":
		return preflightpkg.SeverityInfo, nil
	case "warn", "warning":
		return preflightpkg.SeverityWarning, nil
	case "error", "err":
		return preflightpkg.SeverityError, nil
	default:
		return "", fmt.Errorf("--fail-on must be one of: info, warning, error")
	}
}

// printTextReport renders a human-readable report grouped by scanner.
func printTextReport(r *preflightpkg.Report) {
	out := os.Stdout

	fmt.Fprintln(out, "gplay preflight")
	fmt.Fprintln(out, "===============")
	fmt.Fprintf(out, "  File:   %s\n", r.Path)
	if r.Format != "" {
		fmt.Fprintf(out, "  Format: %s\n", r.Format)
	}
	if r.Package != "" {
		fmt.Fprintf(out, "  App:    %s", r.Package)
		if r.VersionName != "" || r.VersionCode > 0 {
			fmt.Fprintf(out, " (%s, versionCode %d)", r.VersionName, r.VersionCode)
		}
		fmt.Fprintln(out)
	}
	if r.MinSdk > 0 || r.TargetSdk > 0 {
		fmt.Fprintf(out, "  SDK:    min %d, target %d\n", r.MinSdk, r.TargetSdk)
	}
	fmt.Fprintf(out, "  Size:   %d bytes\n", r.TotalSize)
	fmt.Fprintln(out)

	if len(r.Findings) == 0 {
		fmt.Fprintln(out, "  No findings. Looks clean.")
	}

	for _, run := range r.Scanners {
		findings := r.FindingsFor(run.ID)
		switch {
		case run.Skipped:
			fmt.Fprintf(out, "  %-12s skipped (%s)\n", run.ID, run.Reason)
			continue
		case len(findings) == 0:
			fmt.Fprintf(out, "  %-12s ok\n", run.ID)
			continue
		}

		fmt.Fprintf(out, "  %-12s %d finding(s)\n", run.ID, len(findings))
		sortFindings(findings)
		for _, f := range findings {
			fmt.Fprintf(out, "    [%s] %s: %s\n", strings.ToUpper(string(f.Severity)), f.Check, f.Message)
			if f.Entry != "" {
				fmt.Fprintf(out, "           entry: %s\n", f.Entry)
			}
			if f.Hint != "" {
				fmt.Fprintf(out, "            hint: %s\n", f.Hint)
			}
			if f.Ref != "" {
				fmt.Fprintf(out, "             ref: %s\n", f.Ref)
			}
		}
	}

	fmt.Fprintf(out, "\nSummary: %d error(s), %d warning(s), %d info\n",
		r.Errors, r.Warnings, r.Infos)
}

// sortFindings orders findings most severe first, then by check name so
// output is stable across runs.
func sortFindings(f []preflightpkg.Finding) {
	rank := map[preflightpkg.Severity]int{
		preflightpkg.SeverityError:   0,
		preflightpkg.SeverityWarning: 1,
		preflightpkg.SeverityInfo:    2,
	}
	sort.SliceStable(f, func(i, j int) bool {
		if rank[f[i].Severity] != rank[f[j].Severity] {
			return rank[f[i].Severity] < rank[f[j].Severity]
		}
		if f[i].Check != f[j].Check {
			return f[i].Check < f[j].Check
		}
		return f[i].Entry < f[j].Entry
	})
}
