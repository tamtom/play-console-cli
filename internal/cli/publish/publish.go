package publish

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/release"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	validatecli "github.com/tamtom/play-console-cli/internal/cli/validate"
	"github.com/tamtom/play-console-cli/internal/validation"
)

var (
	buildReadinessReportFn = validatecli.BuildReadinessReport
	executeReleaseFn       = release.Execute
)

// PublishCommand returns the top-level canonical publish family.
func PublishCommand() *ffcli.Command {
	fs := flag.NewFlagSet("publish", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "publish",
		ShortUsage: "gplay publish <subcommand> [flags]",
		ShortHelp:  "Canonical Google Play release workflows.",
		LongHelp: `Publish is the canonical high-level release entry point.

Use gplay publish track for the standard artifact -> track workflow.
Lower-level commands such as release, promote, and rollout remain available for
advanced control and debugging.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			TrackCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// TrackCommand publishes an artifact directly to a target Play track.
func TrackCommand() *ffcli.Command {
	fs := flag.NewFlagSet("publish track", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	track := fs.String("track", "", "Target track")
	bundlePath := fs.String("bundle", "", "Path to .aab bundle file")
	apkPath := fs.String("apk", "", "Path to .apk file")
	releaseNotes := fs.String("release-notes", "", "Release notes: plain text, JSON array, or @file")
	rolloutFraction := fs.Float64("rollout", 1.0, "Staged rollout fraction (0.0-1.0)")
	status := fs.String("status", "completed", "Release status: draft, inProgress, halted, completed")
	versionName := fs.String("version-name", "", "Optional version name override")
	wait := fs.Bool("wait", false, "Wait for the published release to appear in the target track")
	pollInterval := fs.Duration("poll-interval", 10*time.Second, "Polling interval when waiting")
	changesNotSent := fs.Bool("changes-not-sent-for-review", false, "Changes not sent for review")
	listingsDir := fs.String("listings-dir", "", "Path to listings metadata directory")
	screenshotsDir := fs.String("screenshots-dir", "", "Path to screenshots directory")
	skipMetadata := fs.Bool("skip-metadata", false, "Skip metadata sync even if --listings-dir is set")
	skipScreenshots := fs.Bool("skip-screenshots", false, "Skip screenshot sync even if --screenshots-dir is set")
	strict := fs.Bool("strict", false, "Treat readiness warnings as publish blockers")
	dryRun := fs.Bool("dry-run", false, "Preview the canonical publish workflow without making changes")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "track",
		ShortUsage: "gplay publish track --package <name> --track <track> (--bundle <path> | --apk <path>) [flags]",
		ShortHelp:  "Run preflight and publish to a Play track.",
		LongHelp: `Publish an artifact to a Play track using the canonical API-driven flow.

This command:
  1. Builds the canonical readiness report
  2. Stops on blocking issues (and optionally warnings with --strict)
  3. Executes the lower-level release workflow if preflight passes`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if *dryRun {
				ctx = shared.ContextWithDryRun(ctx, true)
			}
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			trackName := strings.TrimSpace(*track)
			if trackName == "" {
				return fmt.Errorf("--track is required")
			}
			if strings.TrimSpace(*bundlePath) == "" && strings.TrimSpace(*apkPath) == "" {
				return fmt.Errorf("either --bundle or --apk is required")
			}
			if strings.TrimSpace(*bundlePath) != "" && strings.TrimSpace(*apkPath) != "" {
				return fmt.Errorf("use either --bundle or --apk, not both")
			}
			if *rolloutFraction < 0 || *rolloutFraction > 1 {
				return fmt.Errorf("--rollout must be between 0.0 and 1.0")
			}

			pkgName, err := shared.RequirePackageName(*packageName, nil)
			if err != nil {
				return err
			}

			readinessListingsDir := *listingsDir
			if *skipMetadata {
				readinessListingsDir = ""
			}
			readinessScreenshotsDir := *screenshotsDir
			if *skipScreenshots {
				readinessScreenshotsDir = ""
			}

			preflight := buildReadinessReportFn(ctx, validatecli.ReadinessOptions{
				PackageName:    pkgName,
				Track:          trackName,
				BundlePath:     *bundlePath,
				APKPath:        *apkPath,
				ListingsDir:    readinessListingsDir,
				ScreenshotsDir: readinessScreenshotsDir,
				ReleaseNotes:   *releaseNotes,
				Strict:         *strict,
			})

			result := map[string]interface{}{
				"packageName": pkgName,
				"track":       trackName,
				"published":   false,
				"preflight":   preflight,
			}
			if shared.IsDryRun(ctx) {
				result["dryRun"] = true
			}

			if preflight.Summary.Blocking > 0 {
				if err := shared.PrintOutput(result, *outputFlag, *pretty); err != nil {
					return err
				}
				return shared.NewReportedError(fmt.Errorf("publish track: found %d blocking readiness issue(s)", preflight.Summary.Blocking))
			}
			if *strict && preflight.Summary.Warnings > 0 {
				if err := shared.PrintOutput(result, *outputFlag, *pretty); err != nil {
					return err
				}
				return shared.NewReportedError(fmt.Errorf("publish track: strict mode found %d warning(s)", preflight.Summary.Warnings))
			}

			releaseResult, err := executeReleaseFn(ctx, release.Options{
				PackageName:     pkgName,
				Track:           trackName,
				BundlePath:      *bundlePath,
				APKPath:         *apkPath,
				ReleaseNotes:    *releaseNotes,
				RolloutFraction: *rolloutFraction,
				Status:          *status,
				VersionName:     *versionName,
				Wait:            *wait,
				PollInterval:    *pollInterval,
				ChangesNotSent:  *changesNotSent,
				ListingsDir:     *listingsDir,
				ScreenshotsDir:  *screenshotsDir,
				SkipMetadata:    *skipMetadata,
				SkipScreenshots: *skipScreenshots,
			})
			if err != nil {
				return err
			}

			result["published"] = !shared.IsDryRun(ctx)
			result["release"] = releaseResult

			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}

func readyReport() *validation.ReadinessReport {
	report := &validation.ReadinessReport{}
	report.AddCheck(validation.ReadinessCheck{
		ID:      "ready",
		Section: "test",
		State:   validation.ReadinessInfo,
		Message: "ready",
	})
	return report
}
