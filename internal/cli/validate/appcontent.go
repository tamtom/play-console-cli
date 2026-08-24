package validate

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/validation"
)

// AppContentCommand validates the local inventory of Console-only app-content
// declarations without authenticating or contacting Google.
func AppContentCommand() *ffcli.Command {
	fs := flag.NewFlagSet("validate app-content", flag.ExitOnError)
	input := fs.String("json", "", "App-content inventory JSON or @file")
	packageName := fs.String("package", "", "Optional package name for the report")
	strict := fs.Bool("strict", false, "Treat warnings as failures")
	output := fs.String("output", "json", "Output format: json, table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")
	return &ffcli.Command{
		Name:       "app-content",
		ShortUsage: "gplay validate app-content --json @app-content.json [flags]",
		ShortHelp:  "Validate a local inventory of Play Console app-content declarations.",
		LongHelp: `Validate privacy/contact, ads, reviewer access, target audience,
content rating, Data Safety, category, launch countries, policy declarations,
and sensitive-permission declarations from a local JSON inventory.

This command is fully offline. It does not authenticate, contact Google, or
claim that Console-only state has been read.

Example JSON:
  {"privacyPolicyUrl":"https://example.com/privacy","supportEmail":"support@example.com","ads":"no","appAccess":"all-accessible","targetAudience":["18+"],"contentRatingStatus":"complete","dataSafetyStatus":"complete","policyDeclarationsReviewed":true,"declarations":{"financial-features":"not-applicable","health":"not-applicable","news":"not-applicable"},"sensitivePermissionsReviewed":true}`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 0 {
				return shared.UsageErrorf("unexpected arguments: %s", strings.Join(args, " "))
			}
			if strings.TrimSpace(*input) == "" {
				return shared.UsageError("--json is required")
			}
			if err := shared.ValidateOutputFlags(*output, *pretty); err != nil {
				return err
			}
			report := &validation.ReadinessReport{PackageName: strings.TrimSpace(*packageName), Offline: true}
			addAppContentChecks(ctx, report, *input)
			fmt.Fprintln(shared.Stderr(ctx), report.SummaryLine())
			if err := shared.PrintOutputContext(ctx, report, *output, *pretty); err != nil {
				return err
			}
			if report.Summary.Blocking > 0 {
				return shared.NewReportedError(fmt.Errorf("validate app-content: found %d blocking issue(s)", report.Summary.Blocking))
			}
			if *strict && report.Summary.Warnings > 0 {
				return shared.NewReportedError(fmt.Errorf("validate app-content: strict mode found %d warning(s)", report.Summary.Warnings))
			}
			return nil
		},
	}
}
