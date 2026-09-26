package rollout

import (
	"context"
	"flag"
	"fmt"
	"math"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

func RolloutCommand() *ffcli.Command {
	fs := flag.NewFlagSet("rollout", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "rollout",
		ShortUsage: "gplay rollout <subcommand> [flags]",
		ShortHelp:  "Manage staged rollouts.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			HaltCommand(),
			ResumeCommand(),
			UpdateCommand(),
			CompleteCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			return flag.ErrHelp
		},
	}
}

func HaltCommand() *ffcli.Command {
	fs := flag.NewFlagSet("rollout halt", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	track := fs.String("track", "production", "Track name")
	changesNotSent := fs.Bool("changes-not-sent-for-review", false, "Changes not sent for review")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "halt",
		ShortUsage: "gplay rollout halt --package <name> --track <track>",
		ShortHelp:  "Halt a staged or completed rollout.",
		LongHelp: `Halt a staged or completed rollout, preventing new users from getting the update.
An active staged release is selected first. A completed release is selected only
when no staged or halted release exists. Ambiguous tracks require tracks update.
Google requires an eligible previous release to serve as the fallback.
Existing users who received the update are not affected.

Example:
  gplay rollout halt --package com.example.app --track production`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			return updateRolloutStatus(ctx, *packageName, *track, "halt", 0, *changesNotSent, *outputFlag, *pretty)
		},
	}
}

func ResumeCommand() *ffcli.Command {
	fs := flag.NewFlagSet("rollout resume", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	track := fs.String("track", "production", "Track name")
	rolloutFraction := fs.Float64("rollout", 0, "New rollout fraction (0 = keep current)")
	changesNotSent := fs.Bool("changes-not-sent-for-review", false, "Changes not sent for review")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "resume",
		ShortUsage: "gplay rollout resume --package <name> --track <track> [--rollout <fraction>]",
		ShortHelp:  "Resume a halted rollout.",
		LongHelp: `Resume a previously halted staged rollout.
Optionally specify a new rollout fraction.

Example:
  gplay rollout resume --package com.example.app --track production
  gplay rollout resume --package com.example.app --track production --rollout 0.5`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			return updateRolloutStatus(ctx, *packageName, *track, "resume", *rolloutFraction, *changesNotSent, *outputFlag, *pretty)
		},
	}
}

func UpdateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("rollout update", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	track := fs.String("track", "production", "Track name")
	rolloutFraction := fs.Float64("rollout", 0, "New rollout fraction (required)")
	changesNotSent := fs.Bool("changes-not-sent-for-review", false, "Changes not sent for review")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "update",
		ShortUsage: "gplay rollout update --package <name> --track <track> --rollout <fraction>",
		ShortHelp:  "Update rollout percentage.",
		LongHelp: `Update the rollout percentage for an active staged rollout.
Use rollout resume to restart a halted release.
The new fraction must be greater than the current fraction. A fraction of 1 completes the rollout.

Example:
  gplay rollout update --package com.example.app --track production --rollout 0.5`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if *rolloutFraction <= 0 || *rolloutFraction > 1 {
				return fmt.Errorf("--rollout must be between 0.0 and 1.0")
			}
			return updateRolloutStatus(ctx, *packageName, *track, "update", *rolloutFraction, *changesNotSent, *outputFlag, *pretty)
		},
	}
}

func CompleteCommand() *ffcli.Command {
	fs := flag.NewFlagSet("rollout complete", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	track := fs.String("track", "production", "Track name")
	changesNotSent := fs.Bool("changes-not-sent-for-review", false, "Changes not sent for review")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "complete",
		ShortUsage: "gplay rollout complete --package <name> --track <track>",
		ShortHelp:  "Complete a staged rollout to 100%.",
		LongHelp: `Complete a staged rollout, making the release available to all users.

Example:
  gplay rollout complete --package com.example.app --track production`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			return updateRolloutStatus(ctx, *packageName, *track, "complete", 1.0, *changesNotSent, *outputFlag, *pretty)
		},
	}
}

func updateRolloutStatus(ctx context.Context, packageName, track, action string, rolloutFraction float64, changesNotSent bool, outputFlag string, pretty bool) error {
	if err := shared.ValidateOutputFlags(outputFlag, pretty); err != nil {
		return err
	}

	if math.IsNaN(rolloutFraction) || math.IsInf(rolloutFraction, 0) || rolloutFraction < 0 || rolloutFraction > 1 {
		return fmt.Errorf("--rollout must be finite and between 0 and 1; use rollout complete for 100%%")
	}
	status := "inProgress"
	switch action {
	case "halt":
		status = "halted"
	case "complete":
		status = "completed"
	}
	if rolloutFraction == 1 {
		status = "completed"
	}

	service, err := playclient.NewService(ctx)
	if err != nil {
		return err
	}
	pkg := shared.ResolvePackageName(packageName, service.Cfg)
	if strings.TrimSpace(pkg) == "" {
		return fmt.Errorf("--package is required")
	}

	// Step 1: Create edit
	fmt.Fprintf(shared.Stderr(ctx), "Creating edit...\n")
	editCtx, editCancel := shared.ContextWithTimeout(ctx, service.Cfg)
	edit, err := service.API.Edits.Insert(pkg, &androidpublisher.AppEdit{}).Context(editCtx).Do()
	editCancel()
	if err != nil {
		return fmt.Errorf("failed to create edit: %w", err)
	}

	// Step 2: Get current track
	fmt.Fprintf(shared.Stderr(ctx), "Getting current track state...\n")
	getCtx, getCancel := shared.ContextWithTimeout(ctx, service.Cfg)
	currentTrack, err := service.API.Edits.Tracks.Get(pkg, edit.Id, track).Context(getCtx).Do()
	getCancel()
	if err != nil {
		return fmt.Errorf("failed to get track: %w", err)
	}

	var candidates []*androidpublisher.TrackRelease
	for _, rel := range currentTrack.Releases {
		if action == "update" && rel.Status != "inProgress" {
			continue
		}
		if rel.Status == "inProgress" || rel.Status == "halted" {
			if action != "resume" || rel.Status == "halted" {
				candidates = append(candidates, rel)
			}
		}
	}
	if len(candidates) == 0 && action == "halt" {
		for _, rel := range currentTrack.Releases {
			if rel.Status == "completed" {
				candidates = append(candidates, rel)
			}
		}
	}
	if len(candidates) == 0 {
		if action == "update" {
			return fmt.Errorf("no active staged release found in %s track; use rollout resume to restart a halted release", track)
		}
		return fmt.Errorf("no eligible release found in %s track", track)
	}
	if len(candidates) != 1 {
		return fmt.Errorf("multiple eligible releases in %s; use tracks update to select the release explicitly", track)
	}
	targetRelease := candidates[0]
	if rolloutFraction > 0 && rolloutFraction < targetRelease.UserFraction {
		return fmt.Errorf("--rollout cannot decrease the current fraction %g", targetRelease.UserFraction)
	}
	if action == "resume" && rolloutFraction == 0 && targetRelease.UserFraction == 0 {
		status = "completed"
	}
	if status == "inProgress" && rolloutFraction == 0 && (targetRelease.UserFraction <= 0 || targetRelease.UserFraction >= 1) {
		return fmt.Errorf("--rollout must specify a fraction strictly between 0 and 1 for a staged release")
	}

	fmt.Fprintf(shared.Stderr(ctx), "Updating rollout status to: %s\n", status)
	targetRelease.Status = status
	if status == "completed" {
		targetRelease.UserFraction = 0
	} else if rolloutFraction > 0 {
		targetRelease.UserFraction = rolloutFraction
	}
	if status != "inProgress" {
		targetRelease.CountryTargeting = nil
	}

	trackObj := &androidpublisher.Track{
		Track:    track,
		Releases: []*androidpublisher.TrackRelease{targetRelease},
	}

	trackCtx, trackCancel := shared.ContextWithTimeout(ctx, service.Cfg)
	_, err = service.API.Edits.Tracks.Update(pkg, edit.Id, track, trackObj).Context(trackCtx).Do()
	trackCancel()
	if err != nil {
		return fmt.Errorf("failed to update track: %w", err)
	}

	// Step 4: Validate
	fmt.Fprintf(shared.Stderr(ctx), "Validating edit...\n")
	validateCtx, validateCancel := shared.ContextWithTimeout(ctx, service.Cfg)
	_, err = service.API.Edits.Validate(pkg, edit.Id).Context(validateCtx).Do()
	validateCancel()
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Step 5: Commit
	fmt.Fprintf(shared.Stderr(ctx), "Committing edit...\n")
	commitCtx, commitCancel := shared.ContextWithTimeout(ctx, service.Cfg)
	commitCall := service.API.Edits.Commit(pkg, edit.Id).Context(commitCtx)
	if changesNotSent {
		commitCall = commitCall.ChangesNotSentForReview(true)
	}
	committed, err := commitCall.Do()
	commitCancel()
	if err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}
	fmt.Fprintf(shared.Stderr(ctx), "Rollout updated successfully\n")

	// Output result
	result := map[string]interface{}{
		"editId":       committed.Id,
		"packageName":  pkg,
		"track":        track,
		"status":       targetRelease.Status,
		"versionCodes": targetRelease.VersionCodes,
	}
	if targetRelease.UserFraction > 0 && targetRelease.UserFraction < 1 {
		result["rolloutFraction"] = targetRelease.UserFraction
	}

	return shared.PrintOutputContext(ctx, result, outputFlag, pretty)
}
