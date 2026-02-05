package release

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/googleapi"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

func ReleaseCommand() *ffcli.Command {
	fs := flag.NewFlagSet("release", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	track := fs.String("track", "internal", "Target track (production, beta, alpha, internal)")
	bundlePath := fs.String("bundle", "", "Path to .aab bundle file")
	apkPath := fs.String("apk", "", "Path to .apk file (use --bundle or --apk, not both)")
	releaseNotesJSON := fs.String("release-notes", "", "Release notes JSON (or @file). Format: [{\"language\": \"en-US\", \"text\": \"...\"}]")
	rolloutFraction := fs.Float64("rollout", 1.0, "Staged rollout fraction (0.0-1.0, default: 1.0 for full rollout)")
	status := fs.String("status", "completed", "Release status: draft, inProgress, halted, completed")
	versionName := fs.String("version-name", "", "Version name (optional, defaults to versionName from bundle/apk)")
	wait := fs.Bool("wait", false, "Wait for processing to complete")
	pollInterval := fs.Duration("poll-interval", 10*time.Second, "Polling interval when waiting")
	changesNotSent := fs.Bool("changes-not-sent-for-review", false, "Changes not sent for review")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "release",
		ShortUsage: "gplay release --package <name> --track <track> --bundle <path> [--release-notes <json>] [--rollout <fraction>] [--wait]",
		ShortHelp:  "Create a complete release: create edit, upload bundle/apk, configure track, commit.",
		LongHelp: `The release command is a high-level workflow that combines:
  1. Create a new edit
  2. Upload app bundle or APK
  3. Configure track with release notes and rollout
  4. Validate and commit the edit

This replaces the manual workflow of:
  gplay edits create
  gplay bundles upload
  gplay tracks update
  gplay edits commit

Example:
  gplay release --package com.example.app --track production --bundle app.aab --release-notes @notes.json --rollout 0.1`,
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
			if *rolloutFraction < 0 || *rolloutFraction > 1 {
				return fmt.Errorf("--rollout must be between 0.0 and 1.0")
			}

			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			// Step 1: Create edit
			fmt.Fprintf(os.Stderr, "Creating edit...\n")
			editCtx, editCancel := shared.ContextWithTimeout(ctx, service.Cfg)
			edit, err := service.API.Edits.Insert(pkg, &androidpublisher.AppEdit{}).Context(editCtx).Do()
			editCancel()
			if err != nil {
				return fmt.Errorf("failed to create edit: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Edit created: %s\n", edit.Id)

			// Step 2: Upload bundle or APK
			var versionCode int64
			uploadCtx, uploadCancel := shared.ContextWithUploadTimeout(ctx, service.Cfg)
			defer uploadCancel()

			if strings.TrimSpace(*bundlePath) != "" {
				fmt.Fprintf(os.Stderr, "Uploading bundle: %s\n", *bundlePath)
				file, err := os.Open(*bundlePath)
				if err != nil {
					return shared.WrapActionable(err, "failed to open bundle", "Check that the file exists and is readable.")
				}
				defer file.Close()

				call := service.API.Edits.Bundles.Upload(pkg, edit.Id)
				call.Media(file, googleapi.ContentType("application/octet-stream"))
				bundle, err := call.Context(uploadCtx).Do()
				if err != nil {
					return shared.WrapGoogleAPIError("failed to upload bundle", err)
				}
				versionCode = bundle.VersionCode
				fmt.Fprintf(os.Stderr, "Bundle uploaded: version code %d\n", versionCode)
			} else {
				fmt.Fprintf(os.Stderr, "Uploading APK: %s\n", *apkPath)
				file, err := os.Open(*apkPath)
				if err != nil {
					return shared.WrapActionable(err, "failed to open APK", "Check that the file exists and is readable.")
				}
				defer file.Close()

				call := service.API.Edits.Apks.Upload(pkg, edit.Id)
				call.Media(file, googleapi.ContentType("application/octet-stream"))
				apk, err := call.Context(uploadCtx).Do()
				if err != nil {
					return shared.WrapGoogleAPIError("failed to upload APK", err)
				}
				versionCode = int64(apk.VersionCode)
				fmt.Fprintf(os.Stderr, "APK uploaded: version code %d\n", versionCode)
			}

			// Step 3: Configure track
			fmt.Fprintf(os.Stderr, "Configuring track: %s\n", *track)
			release := &androidpublisher.TrackRelease{
				Status:       *status,
				VersionCodes: []int64{versionCode},
			}

			if strings.TrimSpace(*versionName) != "" {
				release.Name = *versionName
			}

			if *rolloutFraction < 1.0 && *status == "inProgress" {
				release.UserFraction = *rolloutFraction
			} else if *rolloutFraction < 1.0 && *status == "completed" {
				// For staged rollout with completed status, set the fraction
				release.UserFraction = *rolloutFraction
				release.Status = "inProgress"
			}

			if strings.TrimSpace(*releaseNotesJSON) != "" {
				var releaseNotes []*androidpublisher.LocalizedText
				if err := shared.LoadJSONArg(*releaseNotesJSON, &releaseNotes); err != nil {
					return fmt.Errorf("invalid release notes JSON: %w", err)
				}
				release.ReleaseNotes = releaseNotes
			}

			trackObj := &androidpublisher.Track{
				Track:    *track,
				Releases: []*androidpublisher.TrackRelease{release},
			}

			trackCtx, trackCancel := shared.ContextWithTimeout(ctx, service.Cfg)
			_, err = service.API.Edits.Tracks.Update(pkg, edit.Id, *track, trackObj).Context(trackCtx).Do()
			trackCancel()
			if err != nil {
				return fmt.Errorf("failed to update track: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Track configured\n")

			// Step 4: Validate edit
			fmt.Fprintf(os.Stderr, "Validating edit...\n")
			validateCtx, validateCancel := shared.ContextWithTimeout(ctx, service.Cfg)
			_, err = service.API.Edits.Validate(pkg, edit.Id).Context(validateCtx).Do()
			validateCancel()
			if err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Edit validated\n")

			// Step 5: Commit edit
			fmt.Fprintf(os.Stderr, "Committing edit...\n")
			commitCtx, commitCancel := shared.ContextWithTimeout(ctx, service.Cfg)
			commitCall := service.API.Edits.Commit(pkg, edit.Id).Context(commitCtx)
			if *changesNotSent {
				commitCall = commitCall.ChangesNotSentForReview(true)
			}
			committed, err := commitCall.Do()
			commitCancel()
			if err != nil {
				return fmt.Errorf("commit failed: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Edit committed successfully\n")

			// Step 6: Wait for processing if requested
			if *wait {
				fmt.Fprintf(os.Stderr, "Waiting for processing to complete (poll interval: %v)...\n", *pollInterval)
				// Note: Google Play API doesn't provide a direct processing status endpoint
				// The commit is synchronous, so if it succeeds, the release is queued
				// We'll poll the track to see when the release becomes available
				for {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(*pollInterval):
						checkCtx, checkCancel := shared.ContextWithTimeout(ctx, service.Cfg)
						// Create a new edit to check track status
						checkEdit, err := service.API.Edits.Insert(pkg, &androidpublisher.AppEdit{}).Context(checkCtx).Do()
						if err != nil {
							checkCancel()
							fmt.Fprintf(os.Stderr, "Warning: failed to check status: %v\n", err)
							continue
						}
						trackStatus, err := service.API.Edits.Tracks.Get(pkg, checkEdit.Id, *track).Context(checkCtx).Do()
						service.API.Edits.Delete(pkg, checkEdit.Id).Context(checkCtx).Do()
						checkCancel()
						if err != nil {
							fmt.Fprintf(os.Stderr, "Warning: failed to get track status: %v\n", err)
							continue
						}
						// Check if our version is in the track
						for _, rel := range trackStatus.Releases {
							for _, vc := range rel.VersionCodes {
								if vc == versionCode {
									fmt.Fprintf(os.Stderr, "Release is live with status: %s\n", rel.Status)
									goto done
								}
							}
						}
						fmt.Fprintf(os.Stderr, ".")
					}
				}
			done:
			}

			// Output result
			result := map[string]interface{}{
				"editId":      committed.Id,
				"packageName": pkg,
				"track":       *track,
				"versionCode": versionCode,
				"status":      release.Status,
			}
			if release.UserFraction > 0 && release.UserFraction < 1 {
				result["rolloutFraction"] = release.UserFraction
			}

			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}
