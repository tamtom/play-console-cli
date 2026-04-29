package release

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/googleapi"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

var newServiceFn = playclient.NewService

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
	if shared.IsDryRun(ctx) {
		return dryRunReleasePlan(opts)
	}
	if err := validateArtifactPaths(opts); err != nil {
		return nil, err
	}

	service, err := newServiceFn(ctx)
	if err != nil {
		return nil, err
	}
	pkg := shared.ResolvePackageName(opts.PackageName, service.Cfg)
	if strings.TrimSpace(pkg) == "" {
		return nil, fmt.Errorf("--package is required")
	}
	opts.Track = strings.TrimSpace(opts.Track)
	if opts.Track == "" {
		return nil, fmt.Errorf("--track is required")
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

	metadataLocales := 0
	if strings.TrimSpace(opts.ListingsDir) != "" && !opts.SkipMetadata {
		fmt.Fprintf(os.Stderr, "Updating listing metadata: %s\n", opts.ListingsDir)
		metadataLocales, err = updateListings(ctx, service, pkg, edit.Id, opts.ListingsDir)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(os.Stderr, "Listing metadata updated: %d locale(s)\n", metadataLocales)
	}

	screenshotReplacements := 0
	screenshotImages := 0
	if strings.TrimSpace(opts.ScreenshotsDir) != "" && !opts.SkipScreenshots {
		fmt.Fprintf(os.Stderr, "Replacing screenshots: %s\n", opts.ScreenshotsDir)
		screenshotReplacements, screenshotImages, err = replaceScreenshots(ctx, service, pkg, edit.Id, opts.ScreenshotsDir)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(os.Stderr, "Screenshots replaced: %d type(s), %d image(s)\n", screenshotReplacements, screenshotImages)
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
	if metadataLocales > 0 {
		result["metadataLocales"] = metadataLocales
	}
	if screenshotReplacements > 0 {
		result["screenshotReplacements"] = screenshotReplacements
		result["screenshotImages"] = screenshotImages
	}

	return result, nil
}

func dryRunReleasePlan(opts Options) (map[string]interface{}, error) {
	pkg, err := shared.RequirePackageName(opts.PackageName, nil)
	if err != nil {
		return nil, err
	}
	opts.Track = strings.TrimSpace(opts.Track)
	if opts.Track == "" {
		return nil, fmt.Errorf("--track is required")
	}
	if err := validateArtifactPaths(opts); err != nil {
		return nil, err
	}
	if strings.TrimSpace(opts.ReleaseNotes) != "" {
		if _, err := ParseReleaseNotes(opts.ReleaseNotes); err != nil {
			return nil, fmt.Errorf("--release-notes: %w", err)
		}
	}

	artifactType := "bundle"
	artifactPath := strings.TrimSpace(opts.BundlePath)
	if artifactPath == "" {
		artifactType = "apk"
		artifactPath = strings.TrimSpace(opts.APKPath)
	}

	metadataLocales := 0
	if strings.TrimSpace(opts.ListingsDir) != "" && !opts.SkipMetadata {
		listings, err := ParseListingsDir(opts.ListingsDir)
		if err != nil {
			return nil, fmt.Errorf("--listings-dir: %w", err)
		}
		metadataLocales = len(listings)
	}

	screenshotReplacements := 0
	screenshotImages := 0
	if strings.TrimSpace(opts.ScreenshotsDir) != "" && !opts.SkipScreenshots {
		screenshots, err := ParseScreenshotsDir(opts.ScreenshotsDir)
		if err != nil {
			return nil, fmt.Errorf("--screenshots-dir: %w", err)
		}
		for _, deviceTypes := range screenshots {
			screenshotReplacements += len(deviceTypes)
			for _, files := range deviceTypes {
				screenshotImages += len(files)
			}
		}
	}

	steps := []string{
		"create edit",
		"upload " + artifactType,
	}
	if metadataLocales > 0 {
		steps = append(steps, "update listings")
	}
	if screenshotReplacements > 0 {
		steps = append(steps, "replace screenshots")
	}
	steps = append(steps, "update track", "validate edit", "commit edit")
	if opts.Wait {
		steps = append(steps, "wait for track")
	}

	result := map[string]interface{}{
		"dryRun":      true,
		"packageName": pkg,
		"track":       opts.Track,
		"artifact": map[string]string{
			"type": artifactType,
			"path": artifactPath,
		},
		"status": opts.Status,
		"steps":  steps,
	}
	if opts.RolloutFraction > 0 && opts.RolloutFraction < 1 {
		result["rolloutFraction"] = opts.RolloutFraction
	}
	if metadataLocales > 0 {
		result["metadataLocales"] = metadataLocales
	}
	if screenshotReplacements > 0 {
		result["screenshotReplacements"] = screenshotReplacements
		result["screenshotImages"] = screenshotImages
	}
	return result, nil
}

func validateArtifactPaths(opts Options) error {
	bundlePath := strings.TrimSpace(opts.BundlePath)
	apkPath := strings.TrimSpace(opts.APKPath)
	if bundlePath == "" && apkPath == "" {
		return fmt.Errorf("either --bundle or --apk is required")
	}
	if bundlePath != "" && apkPath != "" {
		return fmt.Errorf("use either --bundle or --apk, not both")
	}
	if bundlePath != "" {
		if _, err := os.Stat(bundlePath); err != nil {
			return shared.WrapActionable(err, "failed to open bundle", "Check that the file exists and is readable.")
		}
	}
	if apkPath != "" {
		if _, err := os.Stat(apkPath); err != nil {
			return shared.WrapActionable(err, "failed to open APK", "Check that the file exists and is readable.")
		}
	}
	return nil
}

func updateListings(ctx context.Context, service *playclient.Service, pkg, editID, dir string) (int, error) {
	listings, err := ParseListingsDir(dir)
	if err != nil {
		return 0, fmt.Errorf("--listings-dir: %w", err)
	}

	locales := make([]string, 0, len(listings))
	for locale := range listings {
		locales = append(locales, locale)
	}
	sort.Strings(locales)

	for _, locale := range locales {
		data := listings[locale]
		listing := &androidpublisher.Listing{
			Title:            data.Title,
			ShortDescription: data.ShortDescription,
			FullDescription:  data.FullDescription,
			Video:            data.Video,
		}
		reqCtx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
		_, err := service.API.Edits.Listings.Update(pkg, editID, locale, listing).Context(reqCtx).Do()
		cancel()
		if err != nil {
			return 0, fmt.Errorf("failed to update listing for %s: %w", locale, err)
		}
	}
	return len(locales), nil
}

func replaceScreenshots(ctx context.Context, service *playclient.Service, pkg, editID, dir string) (int, int, error) {
	screenshots, err := ParseScreenshotsDir(dir)
	if err != nil {
		return 0, 0, fmt.Errorf("--screenshots-dir: %w", err)
	}

	locales := make([]string, 0, len(screenshots))
	for locale := range screenshots {
		locales = append(locales, locale)
	}
	sort.Strings(locales)

	replacements := 0
	uploads := 0
	for _, locale := range locales {
		deviceTypes := screenshots[locale]
		types := make([]string, 0, len(deviceTypes))
		for imageType := range deviceTypes {
			types = append(types, imageType)
		}
		sort.Strings(types)

		for _, imageType := range types {
			deleteCtx, deleteCancel := shared.ContextWithTimeout(ctx, service.Cfg)
			_, err := service.API.Edits.Images.Deleteall(pkg, editID, locale, imageType).Context(deleteCtx).Do()
			deleteCancel()
			if err != nil {
				return replacements, uploads, fmt.Errorf("failed to clear screenshots for %s/%s: %w", locale, imageType, err)
			}
			replacements++

			for _, filePath := range deviceTypes[imageType] {
				file, err := os.Open(filePath)
				if err != nil {
					return replacements, uploads, shared.WrapActionable(err, "failed to open screenshot", "Check that the file exists and is readable.")
				}
				uploadCtx, uploadCancel := shared.ContextWithUploadTimeout(ctx, service.Cfg)
				call := service.API.Edits.Images.Upload(pkg, editID, locale, imageType)
				call.Media(file, googleapi.ContentType(mimeTypeForUpload(filePath)))
				_, err = call.Context(uploadCtx).Do()
				uploadCancel()
				closeErr := file.Close()
				if err != nil {
					return replacements, uploads, shared.WrapGoogleAPIError(fmt.Sprintf("failed to upload screenshot %s", filepath.Base(filePath)), err)
				}
				if closeErr != nil {
					return replacements, uploads, closeErr
				}
				uploads++
			}
		}
	}

	return replacements, uploads, nil
}

func mimeTypeForUpload(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return "application/octet-stream"
	}
}
