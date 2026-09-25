package release

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/googleapi"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
	"github.com/tamtom/play-console-cli/internal/preflight"
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
	bundlePath := strings.TrimSpace(opts.BundlePath)
	apkPath := strings.TrimSpace(opts.APKPath)
	if bundlePath == "" && apkPath == "" {
		return nil, shared.UsageError("either --bundle or --apk is required")
	}
	if bundlePath != "" && apkPath != "" {
		return nil, shared.UsageError("use either --bundle or --apk, not both")
	}
	artifactPath := bundlePath
	artifactDescription := "bundle file"
	if artifactPath == "" {
		artifactPath = apkPath
		artifactDescription = "APK file"
	}

	var listings map[string]ListingData
	if strings.TrimSpace(opts.ListingsDir) != "" && !opts.SkipMetadata {
		var err error
		listings, err = ParseListingsDir(opts.ListingsDir)
		if err != nil {
			return nil, fmt.Errorf("--listings-dir: %w", err)
		}
	}

	var screenshots map[string]map[string][]string
	if strings.TrimSpace(opts.ScreenshotsDir) != "" && !opts.SkipScreenshots {
		var err error
		screenshots, err = ParseScreenshotsDir(opts.ScreenshotsDir)
		if err != nil {
			return nil, fmt.Errorf("--screenshots-dir: %w", err)
		}
	}
	screenshotUploads, err := openScreenshotUploads(screenshots)
	if err != nil {
		return nil, err
	}
	defer closeScreenshotUploads(screenshotUploads)

	artifact, err := shared.OpenUploadFile(artifactPath, artifactDescription)
	if err != nil {
		return nil, err
	}
	defer artifact.Close()

	service, err := playclient.NewService(ctx)
	if err != nil {
		return nil, err
	}
	pkg := shared.ResolvePackageName(opts.PackageName, service.Cfg)
	if strings.TrimSpace(pkg) == "" {
		return nil, fmt.Errorf("--package is required")
	}

	fmt.Fprintf(shared.Stderr(ctx), "Creating edit...\n")
	editCtx, editCancel := shared.ContextWithTimeout(ctx, service.Cfg)
	edit, err := service.API.Edits.Insert(pkg, &androidpublisher.AppEdit{}).Context(editCtx).Do()
	editCancel()
	if err != nil {
		return nil, fmt.Errorf("failed to create edit: %w", err)
	}
	fmt.Fprintf(shared.Stderr(ctx), "Edit created: %s\n", edit.Id)

	updatedLocales, err := applyListings(ctx, service, pkg, edit.Id, listings)
	if err != nil {
		return nil, err
	}
	// Plan the screenshots before the artifact upload, so that a limit error
	// stops the release before the large upload.
	plan, err := planScreenshotUploads(ctx, service, pkg, edit.Id, screenshotUploads)
	if err != nil {
		return nil, err
	}

	var versionCode int64
	uploadCtx, uploadCancel := shared.ContextWithUploadTimeout(ctx, service.Cfg)
	defer uploadCancel()

	if bundlePath != "" {
		fmt.Fprintf(shared.Stderr(ctx), "Uploading bundle: %s\n", opts.BundlePath)
		call := service.API.Edits.Bundles.Upload(pkg, edit.Id)
		call.Media(artifact, googleapi.ContentType("application/octet-stream"))
		bundle, err := call.Context(uploadCtx).Do()
		if err != nil {
			return nil, shared.WrapGoogleAPIError("failed to upload bundle", err)
		}
		versionCode = bundle.VersionCode
		fmt.Fprintf(shared.Stderr(ctx), "Bundle uploaded: version code %d\n", versionCode)
	} else {
		fmt.Fprintf(shared.Stderr(ctx), "Uploading APK: %s\n", opts.APKPath)
		call := service.API.Edits.Apks.Upload(pkg, edit.Id)
		call.Media(artifact, googleapi.ContentType("application/octet-stream"))
		apk, err := call.Context(uploadCtx).Do()
		if err != nil {
			return nil, shared.WrapGoogleAPIError("failed to upload APK", err)
		}
		versionCode = int64(apk.VersionCode)
		fmt.Fprintf(shared.Stderr(ctx), "APK uploaded: version code %d\n", versionCode)
	}

	if err := uploadScreenshots(ctx, service, pkg, edit.Id, plan.uploads); err != nil {
		return nil, err
	}

	fmt.Fprintf(shared.Stderr(ctx), "Configuring track: %s\n", opts.Track)
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
	fmt.Fprintf(shared.Stderr(ctx), "Track configured\n")

	fmt.Fprintf(shared.Stderr(ctx), "Validating edit...\n")
	validateCtx, validateCancel := shared.ContextWithTimeout(ctx, service.Cfg)
	_, err = service.API.Edits.Validate(pkg, edit.Id).Context(validateCtx).Do()
	validateCancel()
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	fmt.Fprintf(shared.Stderr(ctx), "Edit validated\n")

	fmt.Fprintf(shared.Stderr(ctx), "Committing edit...\n")
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
	fmt.Fprintf(shared.Stderr(ctx), "Edit committed successfully\n")

	if opts.Wait {
		fmt.Fprintf(shared.Stderr(ctx), "Waiting for processing to complete (poll interval: %v)...\n", opts.PollInterval)
		for {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(opts.PollInterval):
				checkCtx, checkCancel := shared.ContextWithTimeout(ctx, service.Cfg)
				checkEdit, err := service.API.Edits.Insert(pkg, &androidpublisher.AppEdit{}).Context(checkCtx).Do()
				if err != nil {
					checkCancel()
					fmt.Fprintf(shared.Stderr(ctx), "Warning: failed to check status: %v\n", err)
					continue
				}
				trackStatus, err := service.API.Edits.Tracks.Get(pkg, checkEdit.Id, opts.Track).Context(checkCtx).Do()
				_ = service.API.Edits.Delete(pkg, checkEdit.Id).Context(checkCtx).Do()
				checkCancel()
				if err != nil {
					fmt.Fprintf(shared.Stderr(ctx), "Warning: failed to get track status: %v\n", err)
					continue
				}
				for _, current := range trackStatus.Releases {
					for _, vc := range current.VersionCodes {
						if vc == versionCode {
							fmt.Fprintf(shared.Stderr(ctx), "Release is live with status: %s\n", current.Status)
							goto done
						}
					}
				}
				fmt.Fprintf(shared.Stderr(ctx), ".")
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
	if listings != nil {
		result["listingsUpdated"] = updatedLocales
	}
	if screenshots != nil {
		result["screenshotsUploaded"] = len(plan.uploads)
		result["screenshotsSkipped"] = plan.skipped
	}

	shared.SuggestGitHubStar(ctx)
	return result, nil
}

// maxScreenshotsPerType is the Play limit for each screenshot type.
const maxScreenshotsPerType = 8

func applyListings(ctx context.Context, service *playclient.Service, pkg, editID string, listings map[string]ListingData) ([]string, error) {
	locales := sortedKeys(listings)
	for _, locale := range locales {
		data := listings[locale]
		listing := &androidpublisher.Listing{
			Title:            data.Title,
			ShortDescription: data.ShortDescription,
			FullDescription:  data.FullDescription,
			Video:            data.Video,
			ForceSendFields:  data.forceSendFields(),
		}
		requestCtx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
		_, err := service.API.Edits.Listings.Patch(pkg, editID, locale, listing).Context(requestCtx).Do()
		cancel()
		if err != nil {
			return nil, shared.WrapGoogleAPIError(fmt.Sprintf("failed to update listing for %s", locale), err)
		}
		fmt.Fprintf(shared.Stderr(ctx), "Listing updated: %s\n", locale)
	}
	return locales, nil
}

type screenshotUpload struct {
	locale    string
	imageType string
	path      string
	sha256    string
	file      *os.File
}

func (u *screenshotUpload) key() string {
	return u.locale + "/" + u.imageType
}

// openScreenshotUploads opens, validates, and hashes each screenshot before
// the service exists. The upload later sends the same descriptors.
func openScreenshotUploads(screenshots map[string]map[string][]string) ([]*screenshotUpload, error) {
	var uploads []*screenshotUpload
	fail := func(err error) ([]*screenshotUpload, error) {
		closeScreenshotUploads(uploads)
		return nil, err
	}
	for _, locale := range sortedKeys(screenshots) {
		for _, imageType := range sortedKeys(screenshots[locale]) {
			unique := make(map[string]struct{})
			for _, path := range screenshots[locale][imageType] {
				file, err := shared.OpenUploadFile(path, "screenshot")
				if err != nil {
					return fail(err)
				}
				upload := &screenshotUpload{locale: locale, imageType: imageType, path: path, file: file}
				uploads = append(uploads, upload)

				info, err := file.Stat()
				if err != nil {
					return fail(fmt.Errorf("inspect screenshot %s: %w", path, err))
				}
				for _, finding := range preflight.ValidateScreenshotReader(locale, path, file, info.Size()) {
					if finding.Severity == preflight.SeverityError {
						return fail(fmt.Errorf("invalid screenshot: %s", finding.Message))
					}
				}
				hasher := sha256.New()
				if _, err := io.Copy(hasher, file); err != nil {
					return fail(fmt.Errorf("hash screenshot %s: %w", path, err))
				}
				if _, err := file.Seek(0, io.SeekStart); err != nil {
					return fail(fmt.Errorf("rewind screenshot %s: %w", path, err))
				}
				upload.sha256 = hex.EncodeToString(hasher.Sum(nil))
				unique[upload.sha256] = struct{}{}
			}
			if len(unique) > maxScreenshotsPerType {
				return fail(fmt.Errorf("[%s] %d different %s exceed the %d that Play accepts", locale, len(unique), imageType, maxScreenshotsPerType))
			}
		}
	}
	return uploads, nil
}

func closeScreenshotUploads(uploads []*screenshotUpload) {
	for _, upload := range uploads {
		if upload.file != nil {
			_ = upload.file.Close()
			upload.file = nil
		}
	}
}

type screenshotPlan struct {
	uploads []*screenshotUpload
	skipped int
}

// planScreenshotUploads compares the local screenshots with the images that
// are already in the edit. Release never deletes screenshots. It skips a
// local file that has the same SHA-256 as an image on Play, and it skips a
// local file that has the same content as an earlier local file. It then
// checks the Play limits on the final count, before any upload starts.
func planScreenshotUploads(ctx context.Context, service *playclient.Service, pkg, editID string, uploads []*screenshotUpload) (*screenshotPlan, error) {
	plan := &screenshotPlan{}
	if len(uploads) == 0 {
		return plan, nil
	}

	remote := make(map[string]*remoteScreenshots)
	remoteKnown := !shared.IsDryRun(ctx)
	if !remoteKnown {
		fmt.Fprintf(shared.Stderr(ctx), "[DRY RUN] Screenshots already on Play are not checked.\n")
	} else {
		var err error
		remote, err = listRemoteScreenshots(ctx, service, pkg, editID, uploads)
		if err != nil {
			return nil, err
		}
	}

	var keys []string
	firstLocal := make(map[string]map[string]string)
	newCount := make(map[string]int)
	for _, upload := range uploads {
		key := upload.key()
		if _, ok := firstLocal[key]; !ok {
			firstLocal[key] = make(map[string]string)
			keys = append(keys, key)
		}
		if remote[key].has(upload.sha256) {
			fmt.Fprintf(shared.Stderr(ctx), "Screenshot already on Play, skipped: %s/%s\n", key, filepath.Base(upload.path))
			plan.skipped++
			continue
		}
		if first, ok := firstLocal[key][upload.sha256]; ok {
			fmt.Fprintf(shared.Stderr(ctx), "Screenshot has the same content as %s, skipped: %s/%s\n", filepath.Base(first), key, filepath.Base(upload.path))
			plan.skipped++
			continue
		}
		firstLocal[key][upload.sha256] = upload.path
		newCount[key]++
		plan.uploads = append(plan.uploads, upload)
	}

	for _, key := range keys {
		onPlay := remote[key].len()
		total := onPlay + newCount[key]
		if total > maxScreenshotsPerType {
			return nil, fmt.Errorf("[%s] would have %d screenshots (%d already on Play, %d new); Play accepts at most %d. "+
				"Release does not delete screenshots. Remove old screenshots with \"gplay images delete\" or \"gplay images delete-all\" in a separate edit, then run the release again",
				key, total, onPlay, newCount[key], maxScreenshotsPerType)
		}
		if remoteKnown && strings.HasSuffix(key, "/phoneScreenshots") && total < 2 {
			return nil, fmt.Errorf("[%s] would have %d screenshot(s) (%d already on Play, %d new); Play requires at least 2 phone screenshots",
				key, total, onPlay, newCount[key])
		}
	}
	return plan, nil
}

// remoteScreenshots holds the images of one locale and image type in the edit.
type remoteScreenshots struct {
	sums  map[string]struct{}
	count int
}

func (r *remoteScreenshots) has(sum string) bool {
	if r == nil {
		return false
	}
	_, ok := r.sums[sum]
	return ok
}

// len returns the number of images. Two identical images count as two.
func (r *remoteScreenshots) len() int {
	if r == nil {
		return 0
	}
	return r.count
}

// listRemoteScreenshots returns the images in the edit for each locale and
// image type that has local screenshots. A locale that has no listing in the
// edit has no images.
func listRemoteScreenshots(ctx context.Context, service *playclient.Service, pkg, editID string, uploads []*screenshotUpload) (map[string]*remoteScreenshots, error) {
	requestCtx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
	resp, err := service.API.Edits.Listings.List(pkg, editID).Context(requestCtx).Do()
	cancel()
	if err != nil {
		return nil, shared.WrapGoogleAPIError("failed to list listings", err)
	}
	locales := make(map[string]struct{}, len(resp.Listings))
	for _, listing := range resp.Listings {
		if listing != nil {
			locales[listing.Language] = struct{}{}
		}
	}

	remote := make(map[string]*remoteScreenshots)
	for _, upload := range uploads {
		key := upload.key()
		if _, done := remote[key]; done {
			continue
		}
		remote[key] = &remoteScreenshots{sums: make(map[string]struct{})}
		if _, ok := locales[upload.locale]; !ok {
			continue
		}
		requestCtx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
		images, err := service.API.Edits.Images.List(pkg, editID, upload.locale, upload.imageType).Context(requestCtx).Do()
		cancel()
		if err != nil {
			return nil, shared.WrapGoogleAPIError(fmt.Sprintf("failed to list screenshots for %s", key), err)
		}
		for _, image := range images.Images {
			if image == nil {
				continue
			}
			remote[key].count++
			if sum := strings.ToLower(strings.TrimSpace(image.Sha256)); sum != "" {
				remote[key].sums[sum] = struct{}{}
			}
		}
	}
	return remote, nil
}

func uploadScreenshots(ctx context.Context, service *playclient.Service, pkg, editID string, uploads []*screenshotUpload) error {
	for _, upload := range uploads {
		requestCtx, cancel := shared.ContextWithUploadTimeout(ctx, service.Cfg)
		call := service.API.Edits.Images.Upload(pkg, editID, upload.locale, upload.imageType)
		call.Media(upload.file, googleapi.ContentType(releaseImageContentType(upload.path)))
		_, uploadErr := call.Context(requestCtx).Do()
		cancel()
		closeErr := upload.file.Close()
		upload.file = nil
		if uploadErr != nil {
			return shared.WrapGoogleAPIError(fmt.Sprintf("failed to upload screenshot %s", upload.path), uploadErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close screenshot %s: %w", upload.path, closeErr)
		}
		fmt.Fprintf(shared.Stderr(ctx), "Screenshot uploaded: %s/%s\n", upload.key(), filepath.Base(upload.path))
	}
	return nil
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func releaseImageContentType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	default:
		return "application/octet-stream"
	}
}
