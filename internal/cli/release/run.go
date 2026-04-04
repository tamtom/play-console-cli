package release

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/googleapi"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

// Options describes the high-level release workflow inputs.
type Options struct {
	PackageName     string
	Track           string
	BundlePath      string
	APKPath         string
	ReleaseNotes    string
	RolloutFraction float64
	Status          string
	VersionName     string
	Wait            bool
	PollInterval    time.Duration
	ChangesNotSent  bool
	ListingsDir     string
	ScreenshotsDir  string
	SkipMetadata    bool
	SkipScreenshots bool
}

// Execute runs the high-level release workflow and returns the resulting payload.
func Execute(ctx context.Context, opts Options) (map[string]interface{}, error) {
	service, err := playclient.NewService(ctx)
	if err != nil {
		return nil, err
	}
	pkg := shared.ResolvePackageName(opts.PackageName, service.Cfg)
	if strings.TrimSpace(pkg) == "" {
		return nil, fmt.Errorf("--package is required")
	}

	fmt.Fprintf(os.Stderr, "Creating edit...\n")
	editCtx, editCancel := shared.ContextWithTimeout(ctx, service.Cfg)
	edit, err := service.API.Edits.Insert(pkg, &androidpublisher.AppEdit{}).Context(editCtx).Do()
	editCancel()
	if err != nil {
		return nil, fmt.Errorf("failed to create edit: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Edit created: %s\n", edit.Id)

	var versionCode int64
	uploadCtx, uploadCancel := shared.ContextWithUploadTimeout(ctx, service.Cfg)
	defer uploadCancel()

	if strings.TrimSpace(opts.BundlePath) != "" {
		fmt.Fprintf(os.Stderr, "Uploading bundle: %s\n", opts.BundlePath)
		file, err := os.Open(opts.BundlePath)
		if err != nil {
			return nil, shared.WrapActionable(err, "failed to open bundle", "Check that the file exists and is readable.")
		}
		defer file.Close()

		call := service.API.Edits.Bundles.Upload(pkg, edit.Id)
		call.Media(file, googleapi.ContentType("application/octet-stream"))
		bundle, err := call.Context(uploadCtx).Do()
		if err != nil {
			return nil, shared.WrapGoogleAPIError("failed to upload bundle", err)
		}
		versionCode = bundle.VersionCode
		fmt.Fprintf(os.Stderr, "Bundle uploaded: version code %d\n", versionCode)
	} else {
		fmt.Fprintf(os.Stderr, "Uploading APK: %s\n", opts.APKPath)
		file, err := os.Open(opts.APKPath)
		if err != nil {
			return nil, shared.WrapActionable(err, "failed to open APK", "Check that the file exists and is readable.")
		}
		defer file.Close()

		call := service.API.Edits.Apks.Upload(pkg, edit.Id)
		call.Media(file, googleapi.ContentType("application/octet-stream"))
		apk, err := call.Context(uploadCtx).Do()
		if err != nil {
			return nil, shared.WrapGoogleAPIError("failed to upload APK", err)
		}
		versionCode = int64(apk.VersionCode)
		fmt.Fprintf(os.Stderr, "APK uploaded: version code %d\n", versionCode)
	}

	fmt.Fprintf(os.Stderr, "Configuring track: %s\n", opts.Track)
	release := &androidpublisher.TrackRelease{
		Status:       opts.Status,
		VersionCodes: []int64{versionCode},
	}

	if strings.TrimSpace(opts.VersionName) != "" {
		release.Name = opts.VersionName
	}
	if opts.RolloutFraction < 1.0 && opts.Status == "inProgress" {
		release.UserFraction = opts.RolloutFraction
	} else if opts.RolloutFraction < 1.0 && opts.Status == "completed" {
		release.UserFraction = opts.RolloutFraction
		release.Status = "inProgress"
	}
	if strings.TrimSpace(opts.ReleaseNotes) != "" {
		notes, _ := ParseReleaseNotes(opts.ReleaseNotes)
		var releaseNotes []*androidpublisher.LocalizedText
		for _, note := range notes {
			releaseNotes = append(releaseNotes, &androidpublisher.LocalizedText{
				Language: note.Language,
				Text:     note.Text,
			})
		}
		release.ReleaseNotes = releaseNotes
	}

	trackObj := &androidpublisher.Track{
		Track:    opts.Track,
		Releases: []*androidpublisher.TrackRelease{release},
	}

	trackCtx, trackCancel := shared.ContextWithTimeout(ctx, service.Cfg)
	_, err = service.API.Edits.Tracks.Update(pkg, edit.Id, opts.Track, trackObj).Context(trackCtx).Do()
	trackCancel()
	if err != nil {
		return nil, fmt.Errorf("failed to update track: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Track configured\n")

	fmt.Fprintf(os.Stderr, "Validating edit...\n")
	validateCtx, validateCancel := shared.ContextWithTimeout(ctx, service.Cfg)
	_, err = service.API.Edits.Validate(pkg, edit.Id).Context(validateCtx).Do()
	validateCancel()
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Edit validated\n")

	fmt.Fprintf(os.Stderr, "Committing edit...\n")
	commitCtx, commitCancel := shared.ContextWithTimeout(ctx, service.Cfg)
	commitCall := service.API.Edits.Commit(pkg, edit.Id).Context(commitCtx)
	if opts.ChangesNotSent {
		commitCall = commitCall.ChangesNotSentForReview(true)
	}
	committed, err := commitCall.Do()
	commitCancel()
	if err != nil {
		return nil, fmt.Errorf("commit failed: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Edit committed successfully\n")

	if opts.Wait {
		fmt.Fprintf(os.Stderr, "Waiting for processing to complete (poll interval: %v)...\n", opts.PollInterval)
		for {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(opts.PollInterval):
				checkCtx, checkCancel := shared.ContextWithTimeout(ctx, service.Cfg)
				checkEdit, err := service.API.Edits.Insert(pkg, &androidpublisher.AppEdit{}).Context(checkCtx).Do()
				if err != nil {
					checkCancel()
					fmt.Fprintf(os.Stderr, "Warning: failed to check status: %v\n", err)
					continue
				}
				trackStatus, err := service.API.Edits.Tracks.Get(pkg, checkEdit.Id, opts.Track).Context(checkCtx).Do()
				_ = service.API.Edits.Delete(pkg, checkEdit.Id).Context(checkCtx).Do()
				checkCancel()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to get track status: %v\n", err)
					continue
				}
				for _, current := range trackStatus.Releases {
					for _, vc := range current.VersionCodes {
						if vc == versionCode {
							fmt.Fprintf(os.Stderr, "Release is live with status: %s\n", current.Status)
							goto done
						}
					}
				}
				fmt.Fprintf(os.Stderr, ".")
			}
		}
	}

done:
	result := map[string]interface{}{
		"editId":      committed.Id,
		"packageName": pkg,
		"track":       opts.Track,
		"versionCode": versionCode,
		"status":      release.Status,
	}
	if release.UserFraction > 0 && release.UserFraction < 1 {
		result["rolloutFraction"] = release.UserFraction
	}

	// These flags are validated up-front and reserved for future release-media wiring.
	_ = opts.ListingsDir
	_ = opts.ScreenshotsDir
	_ = opts.SkipMetadata
	_ = opts.SkipScreenshots

	return result, nil
}
