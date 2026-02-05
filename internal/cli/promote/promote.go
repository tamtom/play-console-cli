package promote

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

func PromoteCommand() *ffcli.Command {
	fs := flag.NewFlagSet("promote", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	fromTrack := fs.String("from", "", "Source track (e.g., internal, alpha, beta)")
	toTrack := fs.String("to", "", "Destination track (e.g., beta, production)")
	rolloutFraction := fs.Float64("rollout", 1.0, "Staged rollout fraction for destination (0.0-1.0)")
	status := fs.String("status", "completed", "Release status: draft, inProgress, halted, completed")
	releaseNotesJSON := fs.String("release-notes", "", "Release notes JSON (or @file) - if not provided, copies from source")
	changesNotSent := fs.Bool("changes-not-sent-for-review", false, "Changes not sent for review")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "promote",
		ShortUsage: "gplay promote --package <name> --from <track> --to <track> [--rollout <fraction>]",
		ShortHelp:  "Promote a release from one track to another.",
		LongHelp: `Promote a release from one track to another.

This command:
  1. Creates a new edit
  2. Gets the current release from the source track
  3. Configures the destination track with the same version codes
  4. Commits the edit

Example:
  gplay promote --package com.example.app --from internal --to beta
  gplay promote --package com.example.app --from beta --to production --rollout 0.1`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*fromTrack) == "" {
				return fmt.Errorf("--from is required")
			}
			if strings.TrimSpace(*toTrack) == "" {
				return fmt.Errorf("--to is required")
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

			// Step 2: Get source track
			fmt.Fprintf(os.Stderr, "Getting source track: %s\n", *fromTrack)
			getCtx, getCancel := shared.ContextWithTimeout(ctx, service.Cfg)
			sourceTrack, err := service.API.Edits.Tracks.Get(pkg, edit.Id, *fromTrack).Context(getCtx).Do()
			getCancel()
			if err != nil {
				return fmt.Errorf("failed to get source track: %w", err)
			}

			// Find the active release (completed or inProgress)
			var sourceRelease *androidpublisher.TrackRelease
			for _, rel := range sourceTrack.Releases {
				if rel.Status == "completed" || rel.Status == "inProgress" {
					sourceRelease = rel
					break
				}
			}
			if sourceRelease == nil {
				return fmt.Errorf("no active release found in %s track", *fromTrack)
			}

			fmt.Fprintf(os.Stderr, "Found release with version codes: %v\n", sourceRelease.VersionCodes)

			// Step 3: Configure destination track
			fmt.Fprintf(os.Stderr, "Configuring destination track: %s\n", *toTrack)

			newRelease := &androidpublisher.TrackRelease{
				Status:       *status,
				VersionCodes: sourceRelease.VersionCodes,
				Name:         sourceRelease.Name,
			}

			// Handle rollout
			if *rolloutFraction < 1.0 {
				newRelease.UserFraction = *rolloutFraction
				if *status == "completed" {
					newRelease.Status = "inProgress"
				}
			}

			// Handle release notes
			if strings.TrimSpace(*releaseNotesJSON) != "" {
				var releaseNotes []*androidpublisher.LocalizedText
				if err := shared.LoadJSONArg(*releaseNotesJSON, &releaseNotes); err != nil {
					return fmt.Errorf("invalid release notes JSON: %w", err)
				}
				newRelease.ReleaseNotes = releaseNotes
			} else if sourceRelease.ReleaseNotes != nil {
				// Copy release notes from source
				newRelease.ReleaseNotes = sourceRelease.ReleaseNotes
			}

			trackObj := &androidpublisher.Track{
				Track:    *toTrack,
				Releases: []*androidpublisher.TrackRelease{newRelease},
			}

			trackCtx, trackCancel := shared.ContextWithTimeout(ctx, service.Cfg)
			_, err = service.API.Edits.Tracks.Update(pkg, edit.Id, *toTrack, trackObj).Context(trackCtx).Do()
			trackCancel()
			if err != nil {
				return fmt.Errorf("failed to update destination track: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Destination track configured\n")

			// Step 4: Validate
			fmt.Fprintf(os.Stderr, "Validating edit...\n")
			validateCtx, validateCancel := shared.ContextWithTimeout(ctx, service.Cfg)
			_, err = service.API.Edits.Validate(pkg, edit.Id).Context(validateCtx).Do()
			validateCancel()
			if err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Edit validated\n")

			// Step 5: Commit
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

			// Output result
			result := map[string]interface{}{
				"editId":       committed.Id,
				"packageName":  pkg,
				"fromTrack":    *fromTrack,
				"toTrack":      *toTrack,
				"versionCodes": sourceRelease.VersionCodes,
				"status":       newRelease.Status,
			}
			if newRelease.UserFraction > 0 && newRelease.UserFraction < 1 {
				result["rolloutFraction"] = newRelease.UserFraction
			}

			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}
