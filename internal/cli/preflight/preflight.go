// Package preflight wires the `gplay preflight` command.
package preflight

import (
	"context"
	"flag"
	"fmt"
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
	maxSize := fs.String("max-size", "", "Max allowed bundle size (e.g. 150M)")
	maxDex := fs.String("max-dex", "", "Max allowed size per dex file (e.g. 64M)")
	skipSecrets := fs.Bool("skip-secrets", false, "Skip secret-pattern scan (faster)")
	severity := fs.String("fail-on", "error", "Exit non-zero when findings reach this severity: info, warning, error")
	outputFlag := fs.String("output", "text", "Output format: text (default), json, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "preflight",
		ShortUsage: "gplay preflight --file <app.aab> [flags]",
		ShortHelp:  "Run offline compliance and hygiene checks against an AAB/APK.",
		LongHelp: `Run offline checks against an AAB or APK without any API calls.

Checks include: manifest presence, bundle size, native lib coverage,
dex count/size, debuggable flag, testOnly flag, cleartext traffic,
dangerous permissions, secret scan (API keys/private keys/etc.),
and developer-environment artifacts.

Exit codes:
  0   clean
  1   findings at or above --fail-on severity

Example:
  gplay preflight --file app.aab
  gplay preflight --file app.aab --max-size 100M --fail-on warning
  gplay preflight --file app.aab --output json | jq .
`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
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
			opts := preflightpkg.Options{SkipSecretScan: *skipSecrets}
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

			if shouldFail(report, failOn) {
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

func severityRank(s preflightpkg.Severity) int {
	switch s {
	case preflightpkg.SeverityError:
		return 3
	case preflightpkg.SeverityWarning:
		return 2
	case preflightpkg.SeverityInfo:
		return 1
	}
	return 0
}

func shouldFail(r *preflightpkg.Report, min preflightpkg.Severity) bool {
	minR := severityRank(min)
	for _, f := range r.Findings {
		if severityRank(f.Severity) >= minR {
			return true
		}
	}
	return false
}

func printTextReport(r *preflightpkg.Report) {
	fmt.Println("gplay preflight")
	fmt.Println("===============")
	fmt.Printf("  File: %s\n", r.Path)
	fmt.Printf("  Size: %d bytes\n\n", r.TotalSize)
	if len(r.Findings) == 0 {
		fmt.Println("  No findings. Looks clean.")
	} else {
		for _, f := range r.Findings {
			fmt.Printf("  [%s] %s: %s\n", strings.ToUpper(string(f.Severity)), f.Check, f.Message)
			if f.Entry != "" {
				fmt.Printf("         entry: %s\n", f.Entry)
			}
			if f.Hint != "" {
				fmt.Printf("          hint: %s\n", f.Hint)
			}
		}
	}
	fmt.Printf("\nSummary: %d error(s), %d warning(s), %d info\n",
		r.Errors, r.Warnings, r.Infos)
}
