package validate

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// SubmissionCommand returns the "validate submission" subcommand which runs
// all validation checks against the current app state.
func SubmissionCommand() *ffcli.Command {
	fs := flag.NewFlagSet("validate submission", flag.ExitOnError)
	pkg := fs.String("package", "", "Application package name")
	dir := fs.String("dir", "./metadata", "Directory containing listing metadata")
	track := fs.String("track", "production", "Target track to validate")
	releaseNotes := fs.String("release-notes", "", "Release notes input: plain text, JSON array, or @file")
	format := fs.String("format", "fastlane", "Metadata format: fastlane (default), json")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")
	strict := fs.Bool("strict", false, "Treat warnings as failures")

	return &ffcli.Command{
		Name:       "submission",
		ShortUsage: "gplay validate submission --package <name> [--dir <path>] [--output json|table]",
		ShortHelp:  "Compatibility alias for the canonical readiness command.",
		LongHelp: `Run the canonical Google Play release-readiness report using the
legacy validate submission entry point.

This command remains for compatibility, but new docs and workflows should use:
  gplay validate --package <name> [flags]`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}

			pkgName, err := shared.RequirePackageName(*pkg, nil)
			if err != nil {
				return err
			}

			_ = *format

			return runReadinessCommand(ctx, readinessOptions{
				PackageName:  pkgName,
				Track:        *track,
				MetadataDir:  *dir,
				ReleaseNotes: *releaseNotes,
				Strict:       *strict,
				Output:       *outputFlag,
				Pretty:       *pretty,
			})
		},
	}
}
