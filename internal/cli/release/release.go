package release

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

func ReleaseCommand() *ffcli.Command {
	fs := flag.NewFlagSet("release", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	track := fs.String("track", "internal", "Target track (production, beta, alpha, internal)")
	bundlePath := fs.String("bundle", "", "Path to .aab bundle file")
	apkPath := fs.String("apk", "", "Path to .apk file (use --bundle or --apk, not both)")
	releaseNotesJSON := fs.String("release-notes", "", "Release notes: plain text (en-US), JSON array, or @file path")
	rolloutFraction := fs.Float64("rollout", 1.0, "Staged rollout fraction (0.0-1.0, default: 1.0 for full rollout)")
	status := fs.String("status", "completed", "Release status: draft, inProgress, halted, completed")
	versionName := fs.String("version-name", "", "Version name (optional, defaults to versionName from bundle/apk)")
	wait := fs.Bool("wait", false, "Wait for processing to complete")
	pollInterval := fs.Duration("poll-interval", 10*time.Second, "Polling interval when waiting")
	changesNotSent := fs.Bool("changes-not-sent-for-review", false, "Changes not sent for review")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	// Metadata and screenshots flags
	listingsDir := fs.String("listings-dir", "", "Path to listings metadata directory (locale/title.txt, short_description.txt, etc.); a missing file keeps the field, an empty file clears it")
	screenshotsDir := fs.String("screenshots-dir", "", "Path to screenshots directory (locale/deviceType/files); skips screenshots already on Play and never deletes screenshots")
	skipMetadata := fs.Bool("skip-metadata", false, "Skip listings metadata update even if --listings-dir is set")
	skipScreenshots := fs.Bool("skip-screenshots", false, "Skip screenshot uploads even if --screenshots-dir is set")

	return &ffcli.Command{
		Name:       "release",
		ShortUsage: "gplay release --package <name> --track <track> --bundle <path> [--release-notes <json>] [--rollout <fraction>] [--wait]",
		ShortHelp:  "Create a complete release: create edit, upload bundle/apk, configure track, commit.",
		LongHelp: `The release command is a high-level workflow that combines:
  1. Check the bundle or APK, the listings, and the screenshots locally
  2. Create a new edit
  3. Update store listings (if --listings-dir is provided)
  4. Compare the screenshots with the images on Play (if --screenshots-dir is provided)
  5. Upload app bundle or APK
  6. Upload the new screenshots
  7. Configure track with release notes and rollout
  8. Validate and commit the edit

Screenshots:
  Release never deletes screenshots. It skips a local screenshot that has the
  same SHA-256 as an image already on Play, so you can run the same release
  again. If a type would have more than 8 screenshots, release stops before it
  uploads the bundle. Use "gplay images delete" or "gplay images delete-all"
  to remove old screenshots.

This replaces the manual workflow of:
  gplay edits create
  gplay bundles upload
  gplay listings update
  gplay images upload
  gplay tracks update
  gplay edits commit

Example:
  gplay release --package com.example.app --track production --bundle app.aab --release-notes @notes.json --rollout 0.1
  gplay release --package com.example.app --track internal --bundle app.aab --listings-dir ./metadata --screenshots-dir ./screenshots`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}

			// Validate inputs
			if strings.TrimSpace(*bundlePath) == "" && strings.TrimSpace(*apkPath) == "" {
				return fmt.Errorf("either --bundle or --apk is required")
			}
			if strings.TrimSpace(*bundlePath) != "" && strings.TrimSpace(*apkPath) != "" {
				return fmt.Errorf("use either --bundle or --apk, not both")
			}
			if err := shared.ValidateRolloutFraction(*rolloutFraction); err != nil {
				return err
			}

			// Validate listings directory if provided and not skipped
			if strings.TrimSpace(*listingsDir) != "" && !*skipMetadata {
				if _, err := ParseListingsDir(*listingsDir); err != nil {
					return fmt.Errorf("--listings-dir: %w", err)
				}
			}

			// Validate screenshots directory if provided and not skipped
			if strings.TrimSpace(*screenshotsDir) != "" && !*skipScreenshots {
				if _, err := ParseScreenshotsDir(*screenshotsDir); err != nil {
					return fmt.Errorf("--screenshots-dir: %w", err)
				}
			}

			// Validate release notes if provided (supports plain text, JSON, and @file)
			if strings.TrimSpace(*releaseNotesJSON) != "" {
				if _, err := ParseReleaseNotes(*releaseNotesJSON); err != nil {
					return fmt.Errorf("--release-notes: %w", err)
				}
			}

			result, err := Execute(ctx, Options{
				PackageName:     *packageName,
				Track:           *track,
				BundlePath:      *bundlePath,
				APKPath:         *apkPath,
				ReleaseNotes:    *releaseNotesJSON,
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
			return shared.PrintOutputContext(ctx, result, *outputFlag, *pretty)
		},
	}
}
