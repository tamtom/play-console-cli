package validate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/api/androidpublisher/v3"

	"github.com/tamtom/play-console-cli/internal/cli/release"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
	"github.com/tamtom/play-console-cli/internal/validation"
)

type readinessOptions struct {
	PackageName    string
	Track          string
	BundlePath     string
	APKPath        string
	MetadataDir    string
	ListingsDir    string
	ScreenshotsDir string
	ReleaseNotes   string
	Strict         bool
	Output         string
	Pretty         bool
}

// ReadinessOptions is the exported shape used by other command families that
// need the canonical release-readiness report.
type ReadinessOptions = readinessOptions

type remoteReadinessState struct {
	Tracks      []*androidpublisher.Track
	TargetTrack *androidpublisher.Track
	Listings    []*androidpublisher.Listing
}

var fetchRemoteReadinessStateFn = fetchRemoteReadinessState

func runReadinessCommand(ctx context.Context, opts readinessOptions) error {
	report := buildReadinessReport(ctx, opts)

	fmt.Fprintln(os.Stderr, report.SummaryLine())
	if err := shared.PrintOutput(report, opts.Output, opts.Pretty); err != nil {
		return err
	}

	if report.Summary.Blocking > 0 {
		return shared.NewReportedError(fmt.Errorf("validate: found %d blocking issue(s)", report.Summary.Blocking))
	}
	if opts.Strict && report.Summary.Warnings > 0 {
		return shared.NewReportedError(fmt.Errorf("validate: strict mode found %d warning(s)", report.Summary.Warnings))
	}

	return nil
}

// BuildReadinessReport constructs the canonical Play release-readiness report.
func BuildReadinessReport(ctx context.Context, opts ReadinessOptions) *validation.ReadinessReport {
	return buildReadinessReport(ctx, opts)
}

func buildReadinessReport(ctx context.Context, opts readinessOptions) *validation.ReadinessReport {
	report := &validation.ReadinessReport{
		PackageName: opts.PackageName,
		Track:       normalizedTrack(opts.Track),
	}

	addArtifactChecks(report, opts)
	addLocalListingChecks(report, opts)
	addLocalScreenshotChecks(report, opts)
	addReleaseNotesChecks(report, opts.ReleaseNotes)
	addRemoteChecks(ctx, report, opts)
	addManualChecks(report)

	return report
}

func addArtifactChecks(report *validation.ReadinessReport, opts readinessOptions) {
	switch {
	case strings.TrimSpace(opts.BundlePath) != "":
		report.Artifact = filepath.Base(opts.BundlePath)
		result := validateBundle(opts.BundlePath)
		addValidationResultChecks(report, "artifact", result)
	case strings.TrimSpace(opts.APKPath) != "":
		report.Artifact = filepath.Base(opts.APKPath)
		info, err := os.Stat(opts.APKPath)
		if err != nil {
			report.AddCheck(validation.ReadinessCheck{
				ID:          "apk-missing",
				Section:     "artifact",
				State:       validation.ReadinessBlocking,
				Message:     fmt.Sprintf("APK file could not be read: %v", err),
				Remediation: "Provide a readable .apk file with --apk.",
			})
			return
		}
		report.AddCheck(validation.ReadinessCheck{
			ID:      "apk-present",
			Section: "artifact",
			State:   validation.ReadinessInfo,
			Message: fmt.Sprintf("APK is present (%d bytes).", info.Size()),
			Details: map[string]interface{}{
				"path": opts.APKPath,
				"size": info.Size(),
			},
		})
	default:
		report.AddCheck(validation.ReadinessCheck{
			ID:          "artifact-not-provided",
			Section:     "artifact",
			State:       validation.ReadinessWarning,
			Message:     "No local artifact was provided for validation.",
			Remediation: "Provide --bundle or --apk to verify the artifact before publishing.",
		})
	}
}

func addLocalListingChecks(report *validation.ReadinessReport, opts readinessOptions) {
	dir := strings.TrimSpace(opts.ListingsDir)
	if dir == "" {
		dir = strings.TrimSpace(opts.MetadataDir)
	}
	if dir == "" {
		report.AddCheck(validation.ReadinessCheck{
			ID:          "local-listings-skipped",
			Section:     "metadata",
			State:       validation.ReadinessInfo,
			Message:     "No local listings directory was provided.",
			Remediation: "Provide --dir or --listings-dir to validate local listing metadata.",
		})
		return
	}

	listings, err := release.ParseListingsDir(dir)
	if err != nil {
		report.AddCheck(validation.ReadinessCheck{
			ID:          "local-listings-unreadable",
			Section:     "metadata",
			State:       validation.ReadinessBlocking,
			Message:     fmt.Sprintf("Local listings could not be parsed: %v", err),
			Remediation: "Fix the metadata directory structure and required text files.",
		})
		return
	}

	report.AddCheck(validation.ReadinessCheck{
		ID:      "local-listings-found",
		Section: "metadata",
		State:   validation.ReadinessInfo,
		Message: fmt.Sprintf("Found local listing data for %d locale(s).", len(listings)),
		Details: map[string]interface{}{
			"localeCount": len(listings),
			"path":        dir,
		},
	})

	for locale, listing := range listings {
		fields := map[string]string{
			"title":             listing.Title,
			"short_description": listing.ShortDescription,
			"full_description":  listing.FullDescription,
		}
		for _, result := range validation.ValidateRequiredListingFields(locale, fields) {
			report.AddCheck(readinessCheckFromValidation("metadata", result))
		}
		if result := validation.ValidateTitle(locale, listing.Title); result != nil {
			report.AddCheck(readinessCheckFromValidation("metadata", *result))
		}
		if result := validation.ValidateShortDescription(locale, listing.ShortDescription); result != nil {
			report.AddCheck(readinessCheckFromValidation("metadata", *result))
		}
		if result := validation.ValidateFullDescription(locale, listing.FullDescription); result != nil {
			report.AddCheck(readinessCheckFromValidation("metadata", *result))
		}
	}
}

func addLocalScreenshotChecks(report *validation.ReadinessReport, opts readinessOptions) {
	if dir := strings.TrimSpace(opts.ScreenshotsDir); dir != "" {
		screenshots, err := release.ParseScreenshotsDir(dir)
		if err != nil {
			report.AddCheck(validation.ReadinessCheck{
				ID:          "local-screenshots-unreadable",
				Section:     "media",
				State:       validation.ReadinessBlocking,
				Message:     fmt.Sprintf("Local screenshots could not be parsed: %v", err),
				Remediation: "Fix the screenshots directory structure before publishing.",
			})
			return
		}
		for locale, deviceTypes := range screenshots {
			for deviceType, files := range deviceTypes {
				count := len(files)
				state := validation.ReadinessInfo
				message := fmt.Sprintf("%s has %d %s file(s).", locale, count, deviceType)
				remediation := ""
				switch {
				case count > 0 && count < minScreenshots:
					state = validation.ReadinessWarning
					message = fmt.Sprintf("%s has only %d %s file(s).", locale, count, deviceType)
					remediation = fmt.Sprintf("Provide at least %d screenshots for %s.", minScreenshots, deviceType)
				case count > maxPhoneScreenshots:
					state = validation.ReadinessBlocking
					message = fmt.Sprintf("%s has %d %s file(s), above the supported maximum.", locale, count, deviceType)
					remediation = fmt.Sprintf("Reduce %s to %d files or fewer.", deviceType, maxPhoneScreenshots)
				}
				report.AddCheck(validation.ReadinessCheck{
					ID:          "local-screenshots-count",
					Section:     "media",
					State:       state,
					Locale:      locale,
					Field:       deviceType,
					Message:     message,
					Remediation: remediation,
					Details: map[string]interface{}{
						"path":  dir,
						"count": count,
					},
				})
			}
		}
		return
	}

	if dir := strings.TrimSpace(opts.MetadataDir); dir != "" {
		result := validateScreenshots(dir, "")
		addValidationResultChecks(report, "media", result)
		return
	}

	report.AddCheck(validation.ReadinessCheck{
		ID:          "local-screenshots-skipped",
		Section:     "media",
		State:       validation.ReadinessInfo,
		Message:     "No local screenshots directory was provided.",
		Remediation: "Provide --screenshots-dir or a metadata directory to validate local store media.",
	})
}

func addReleaseNotesChecks(report *validation.ReadinessReport, input string) {
	if strings.TrimSpace(input) == "" {
		report.AddCheck(validation.ReadinessCheck{
			ID:          "release-notes-not-provided",
			Section:     "release",
			State:       validation.ReadinessWarning,
			Message:     "No local release notes were provided.",
			Remediation: "Provide --release-notes so this release can be validated before publishing.",
		})
		return
	}

	notes, err := release.ParseReleaseNotes(input)
	if err != nil {
		report.AddCheck(validation.ReadinessCheck{
			ID:          "release-notes-invalid",
			Section:     "release",
			State:       validation.ReadinessBlocking,
			Message:     fmt.Sprintf("Release notes could not be parsed: %v", err),
			Remediation: "Fix the --release-notes input or referenced file.",
		})
		return
	}

	report.AddCheck(validation.ReadinessCheck{
		ID:      "release-notes-found",
		Section: "release",
		State:   validation.ReadinessInfo,
		Message: fmt.Sprintf("Found release notes for %d locale(s).", len(notes)),
	})

	for _, note := range notes {
		if result := validation.ValidateReleaseNotes(note.Language, note.Text); result != nil {
			report.AddCheck(readinessCheckFromValidation("release", *result))
		}
	}
}

func addRemoteChecks(ctx context.Context, report *validation.ReadinessReport, opts readinessOptions) {
	state, err := fetchRemoteReadinessStateFn(ctx, opts.PackageName, normalizedTrack(opts.Track))
	if err != nil {
		report.AddCheck(validation.ReadinessCheck{
			ID:          "remote-play-state-unavailable",
			Section:     "remote",
			State:       validation.ReadinessBlocking,
			Message:     fmt.Sprintf("Play state could not be fetched: %v", err),
			Remediation: "Check Play credentials, package access, and network connectivity.",
		})
		return
	}

	report.AddCheck(validation.ReadinessCheck{
		ID:      "remote-play-state-fetched",
		Section: "remote",
		State:   validation.ReadinessInfo,
		Message: fmt.Sprintf("Fetched Play state for %d track(s) and %d listing(s).", len(state.Tracks), len(state.Listings)),
	})

	if state.TargetTrack == nil {
		report.AddCheck(validation.ReadinessCheck{
			ID:          "target-track-missing",
			Section:     "track",
			State:       validation.ReadinessWarning,
			Message:     fmt.Sprintf("Target track %q does not currently exist in Play state.", normalizedTrack(opts.Track)),
			Remediation: "Create the track or publish to an existing track before rollout.",
		})
	} else {
		activeRelease := activeReleaseForTrack(state.TargetTrack)
		if activeRelease == nil {
			report.AddCheck(validation.ReadinessCheck{
				ID:          "target-track-empty",
				Section:     "track",
				State:       validation.ReadinessWarning,
				Message:     fmt.Sprintf("Track %q has no active release.", state.TargetTrack.Track),
				Remediation: "Upload an artifact and attach a release before publishing.",
			})
		} else {
			stateValue := validation.ReadinessInfo
			remediation := ""
			switch activeRelease.Status {
			case "halted", "draft":
				stateValue = validation.ReadinessWarning
				remediation = "Resolve the halted or draft release state before shipping broadly."
			case "inProgress":
				stateValue = validation.ReadinessWarning
				remediation = "Review staged rollout safety before treating this track as fully ready."
			}
			report.AddCheck(validation.ReadinessCheck{
				ID:          "target-track-active-release",
				Section:     "track",
				State:       stateValue,
				Message:     fmt.Sprintf("Track %q has an active release with status %q.", state.TargetTrack.Track, activeRelease.Status),
				Remediation: remediation,
				Details: map[string]interface{}{
					"versionCodes": activeRelease.VersionCodes,
					"userFraction": activeRelease.UserFraction,
				},
			})

			if strings.TrimSpace(opts.ReleaseNotes) == "" {
				if len(activeRelease.ReleaseNotes) == 0 {
					report.AddCheck(validation.ReadinessCheck{
						ID:          "remote-release-notes-missing",
						Section:     "release",
						State:       validation.ReadinessWarning,
						Message:     "The active remote release does not expose localized release notes.",
						Remediation: "Provide --release-notes or update the release notes before publishing.",
					})
				} else {
					for _, note := range activeRelease.ReleaseNotes {
						if result := validation.ValidateReleaseNotes(note.Language, note.Text); result != nil {
							report.AddCheck(readinessCheckFromValidation("release", *result))
						}
					}
				}
			}
		}
	}

	if len(state.Listings) == 0 {
		report.AddCheck(validation.ReadinessCheck{
			ID:          "remote-listings-missing",
			Section:     "metadata",
			State:       validation.ReadinessWarning,
			Message:     "No remote Play listings were found for this app.",
			Remediation: "Create listings in Play Console or provide local metadata to be synced before release.",
		})
		return
	}

	report.AddCheck(validation.ReadinessCheck{
		ID:      "remote-listings-found",
		Section: "metadata",
		State:   validation.ReadinessInfo,
		Message: fmt.Sprintf("Found %d remote Play listing(s).", len(state.Listings)),
	})
}

func addManualChecks(report *validation.ReadinessReport) {
	report.AddCheck(validation.ReadinessCheck{
		ID:          "manual-data-safety",
		Section:     "manual",
		State:       validation.ReadinessManual,
		Message:     "Data safety completeness cannot be fully verified from the current API surface.",
		Remediation: "Confirm the Play Console data safety form is complete for this release.",
	})
	report.AddCheck(validation.ReadinessCheck{
		ID:          "manual-policy-review",
		Section:     "manual",
		State:       validation.ReadinessManual,
		Message:     "Policy review and Console-only release gates may still require manual confirmation.",
		Remediation: "Review Play Console warnings, policy messages, and publishing state before rollout.",
	})
}

func fetchRemoteReadinessState(ctx context.Context, packageName, track string) (*remoteReadinessState, error) {
	service, err := playclient.NewService(ctx)
	if err != nil {
		return nil, err
	}
	pkg, err := shared.RequirePackageName(packageName, service.Cfg)
	if err != nil {
		return nil, err
	}

	editCtx, editCancel := shared.ContextWithTimeout(ctx, service.Cfg)
	edit, err := service.API.Edits.Insert(pkg, &androidpublisher.AppEdit{}).Context(editCtx).Do()
	editCancel()
	if err != nil {
		return nil, err
	}

	defer func() {
		deleteCtx, deleteCancel := shared.ContextWithTimeout(context.Background(), service.Cfg)
		_ = service.API.Edits.Delete(pkg, edit.Id).Context(deleteCtx).Do()
		deleteCancel()
	}()

	tracksCtx, tracksCancel := shared.ContextWithTimeout(ctx, service.Cfg)
	tracksResp, err := service.API.Edits.Tracks.List(pkg, edit.Id).Context(tracksCtx).Do()
	tracksCancel()
	if err != nil {
		return nil, err
	}

	listingsCtx, listingsCancel := shared.ContextWithTimeout(ctx, service.Cfg)
	listingsResp, err := service.API.Edits.Listings.List(pkg, edit.Id).Context(listingsCtx).Do()
	listingsCancel()
	if err != nil {
		return nil, err
	}

	state := &remoteReadinessState{
		Tracks:   tracksResp.Tracks,
		Listings: listingsResp.Listings,
	}
	for _, candidate := range state.Tracks {
		if candidate.Track == track {
			state.TargetTrack = candidate
			break
		}
	}

	return state, nil
}

func addValidationResultChecks(report *validation.ReadinessReport, section string, result *ValidationResult) {
	if result == nil {
		return
	}
	for _, message := range result.Errors {
		report.AddCheck(validation.ReadinessCheck{
			ID:          "legacy-validation-error",
			Section:     section,
			State:       validation.ReadinessBlocking,
			Message:     message,
			Remediation: "Fix the local validation error before publishing.",
		})
	}
	for _, message := range result.Warnings {
		report.AddCheck(validation.ReadinessCheck{
			ID:          "legacy-validation-warning",
			Section:     section,
			State:       validation.ReadinessWarning,
			Message:     message,
			Remediation: "Review the warning before publishing.",
		})
	}
	if result.Valid {
		report.AddCheck(validation.ReadinessCheck{
			ID:      "legacy-validation-passed",
			Section: section,
			State:   validation.ReadinessInfo,
			Message: fmt.Sprintf("%s checks passed.", section),
		})
	}
}

func readinessCheckFromValidation(section string, result validation.CheckResult) validation.ReadinessCheck {
	state := validation.ReadinessInfo
	switch result.Severity {
	case validation.SeverityError:
		state = validation.ReadinessBlocking
	case validation.SeverityWarning:
		state = validation.ReadinessWarning
	case validation.SeverityInfo:
		state = validation.ReadinessInfo
	}

	return validation.ReadinessCheck{
		ID:          result.ID,
		Section:     section,
		State:       state,
		Locale:      result.Locale,
		Field:       result.Field,
		Message:     result.Message,
		Remediation: result.Remediation,
	}
}

func activeReleaseForTrack(track *androidpublisher.Track) *androidpublisher.TrackRelease {
	if track == nil {
		return nil
	}
	for _, release := range track.Releases {
		if release == nil {
			continue
		}
		switch release.Status {
		case "completed", "inProgress", "halted", "draft":
			return release
		}
	}
	return nil
}

func normalizedTrack(track string) string {
	trimmed := strings.TrimSpace(track)
	if trimmed == "" {
		return "production"
	}
	return trimmed
}
