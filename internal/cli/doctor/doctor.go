// Package doctor implements the `gplay doctor` command: a comprehensive
// setup and environment health check.
package doctor

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// DoctorCommand returns the `gplay doctor` command.
func DoctorCommand() *ffcli.Command {
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	outputFlag := fs.String("output", "text", "Output format: text (default), json, markdown, table")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "doctor",
		ShortUsage: "gplay doctor [flags]",
		ShortHelp:  "Diagnose CLI setup, network, credentials, and configuration.",
		LongHelp: `Run 15+ checks across config, credentials, network, disk, and environment.

Examples:
  gplay doctor
  gplay doctor --output json --pretty
  gplay doctor --output markdown`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			report := Run(ctx, DefaultEnv())
			if *outputFlag == "text" {
				printTextReport(shared.Stdout(ctx), report)
				if report.Failures > 0 {
					return shared.NewReportedError(fmt.Errorf("doctor: %d failure(s), %d warning(s)", report.Failures, report.Warnings))
				}
				return nil
			}
			if err := shared.PrintOutputContext(ctx, report, *outputFlag, *pretty); err != nil {
				return err
			}
			if report.Failures > 0 {
				return shared.NewReportedError(fmt.Errorf("doctor: %d failure(s)", report.Failures))
			}
			return nil
		},
	}
}

func printTextReport(out io.Writer, r Report) {
	fmt.Fprintln(out, "gplay doctor")
	fmt.Fprintln(out, "============")
	for _, c := range r.Checks {
		fmt.Fprintf(out, "  [%s] %s", symbol(c.Severity), c.Name)
		if c.Detail != "" {
			fmt.Fprintf(out, " — %s", c.Detail)
		}
		fmt.Fprintln(out)
		if c.Hint != "" && (c.Severity == SeverityWarn || c.Severity == SeverityFail) {
			fmt.Fprintf(out, "         hint: %s\n", c.Hint)
		}
	}
	fmt.Fprintf(out, "\nSummary: %d ok / %d warn / %d fail / %d skip\n",
		r.Passed, r.Warnings, r.Failures, r.Skipped)
}

func symbol(s Severity) string {
	switch s {
	case SeverityOK:
		return "OK  "
	case SeverityWarn:
		return "WARN"
	case SeverityFail:
		return "FAIL"
	case SeveritySkip:
		return "SKIP"
	default:
		return "????"
	}
}
