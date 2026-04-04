package images

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tamtom/play-console-cli/internal/config"
	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/googleapi"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

var (
	newMediaBackend = newPlayMediaBackend
	downloadRemote  = defaultDownloadRemote
)

var supportedScreenshotTypes = map[string]struct{}{
	"phoneScreenshots":     {},
	"sevenInchScreenshots": {},
	"tenInchScreenshots":   {},
	"tvScreenshots":        {},
	"wearScreenshots":      {},
}

var singleImageTypes = map[string]string{
	"featureGraphic.png": "featureGraphic",
	"icon.png":           "icon",
	"promoGraphic.png":   "promoGraphic",
	"tvBanner.png":       "tvBanner",
}

var imageTypeOrder = []string{
	"phoneScreenshots",
	"sevenInchScreenshots",
	"tenInchScreenshots",
	"tvScreenshots",
	"wearScreenshots",
	"featureGraphic",
	"icon",
	"promoGraphic",
	"tvBanner",
}

type mediaBackend interface {
	ListLocales(ctx context.Context, packageName, editID string) ([]string, error)
	ListImages(ctx context.Context, packageName, editID, locale, imageType string) ([]remoteImage, error)
	UploadImage(ctx context.Context, packageName, editID, locale, imageType, filePath string) (*androidpublisher.Image, error)
	Config() *config.Config
}

type remoteImage struct {
	ID     string `json:"id,omitempty"`
	Sha1   string `json:"sha1,omitempty"`
	Sha256 string `json:"sha256,omitempty"`
	URL    string `json:"url,omitempty"`
}

type localAsset struct {
	Locale   string `json:"locale"`
	Type     string `json:"type"`
	Path     string `json:"path"`
	Sha256   string `json:"sha256"`
	FileName string `json:"fileName"`
}

type mediaPlanSummary struct {
	Locales    int `json:"locales"`
	Keep       int `json:"keep"`
	Upload     int `json:"upload"`
	RemoteOnly int `json:"remoteOnly"`
	Errors     int `json:"errors"`
}

type mediaPlanAsset struct {
	Type         string `json:"type"`
	Action       string `json:"action"`
	LocalPath    string `json:"localPath,omitempty"`
	LocalSHA256  string `json:"localSha256,omitempty"`
	RemoteID     string `json:"remoteId,omitempty"`
	RemoteSHA256 string `json:"remoteSha256,omitempty"`
	RemoteURL    string `json:"remoteUrl,omitempty"`
}

type mediaPlanLocale struct {
	Locale string           `json:"locale"`
	Assets []mediaPlanAsset `json:"assets"`
}

type mediaPlan struct {
	Package string            `json:"package"`
	EditID  string            `json:"editId"`
	Dir     string            `json:"dir"`
	Summary mediaPlanSummary  `json:"summary"`
	Locales []mediaPlanLocale `json:"locales"`
	Errors  []string          `json:"errors,omitempty"`
}

type pullResult struct {
	Package string   `json:"package"`
	EditID  string   `json:"editId"`
	Dir     string   `json:"dir"`
	Files   []string `json:"files"`
	DryRun  bool     `json:"dryRun"`
}

type syncResult struct {
	Package    string   `json:"package"`
	EditID     string   `json:"editId"`
	Dir        string   `json:"dir"`
	Uploaded   int      `json:"uploaded"`
	Kept       int      `json:"kept"`
	RemoteOnly int      `json:"remoteOnly"`
	DryRun     bool     `json:"dryRun"`
	Errors     []string `json:"errors,omitempty"`
}

func PlanCommand() *ffcli.Command {
	fs := defaultImagesFlagSet("images plan")
	packageName, editID, dir, locale, outputFlag, pretty := bindImagesSyncFlags(fs)

	return &ffcli.Command{
		Name:       "plan",
		ShortUsage: "gplay images plan --package <name> --edit <id> [--dir <path>] [--locale <lang>]",
		ShortHelp:  "Plan deterministic Play media sync operations.",
		LongHelp: `Compare local Play media files against the current edit state.

Local directory layout:
  <dir>/<locale>/images/
    phoneScreenshots/
    sevenInchScreenshots/
    tenInchScreenshots/
    tvScreenshots/
    wearScreenshots/
    featureGraphic.png
    icon.png
    promoGraphic.png
    tvBanner.png

The plan is based on SHA-256 hashes and does not mutate remote state.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			return runMediaPlan(ctx, *packageName, *editID, *dir, *locale, *outputFlag, *pretty)
		},
	}
}

func PullCommand() *ffcli.Command {
	fs := defaultImagesFlagSet("images pull")
	packageName, editID, dir, locale, outputFlag, pretty := bindImagesSyncFlags(fs)

	return &ffcli.Command{
		Name:       "pull",
		ShortUsage: "gplay images pull --package <name> --edit <id> [--dir <path>] [--locale <lang>]",
		ShortHelp:  "Pull remote Play media into the local directory layout.",
		LongHelp: `Download the current edit's media into the local Play directory layout.

Screenshot files are saved as deterministic numbered files under each device
type directory. Single-image assets are saved by their conventional names.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			return runMediaPull(ctx, *packageName, *editID, *dir, *locale, *outputFlag, *pretty)
		},
	}
}

func SyncCommand() *ffcli.Command {
	fs := defaultImagesFlagSet("images sync")
	packageName, editID, dir, locale, outputFlag, pretty := bindImagesSyncFlags(fs)

	return &ffcli.Command{
		Name:       "sync",
		ShortUsage: "gplay images sync --package <name> --edit <id> [--dir <path>] [--locale <lang>]",
		ShortHelp:  "Upload local Play media to the current edit.",
		LongHelp: `Upload local Play media using the deterministic local directory layout.

The command compares local files to remote SHA-256 hashes and only uploads
assets that are missing or changed. Use ` + "`--dry-run`" + ` for a transport-level
preview of the upload calls.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			return runMediaSync(ctx, *packageName, *editID, *dir, *locale, *outputFlag, *pretty)
		},
	}
}

func defaultImagesFlagSet(name string) *flag.FlagSet {
	return flag.NewFlagSet(name, flag.ExitOnError)
}

func bindImagesSyncFlags(fs *flag.FlagSet) (*string, *string, *string, *string, *string, *bool) {
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	dir := fs.String("dir", "./metadata", "Directory containing Play media files")
	locale := fs.String("locale", "", "Specific locale to sync (optional)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")
	return packageName, editID, dir, locale, outputFlag, pretty
}

func runMediaPlan(ctx context.Context, packageName, editID, dir, locale, outputFlag string, pretty bool) error {
	backend, err := newMediaBackend(ctx)
	if err != nil {
		return err
	}
	pkg := shared.ResolvePackageName(packageName, backend.Config())
	if strings.TrimSpace(pkg) == "" {
		return fmt.Errorf("--package is required")
	}
	if strings.TrimSpace(editID) == "" {
		return fmt.Errorf("--edit is required")
	}

	plan, err := buildMediaPlan(ctx, backend, pkg, editID, dir, locale)
	if err != nil {
		return err
	}
	return shared.PrintOutput(plan, outputFlag, pretty)
}

func runMediaPull(ctx context.Context, packageName, editID, dir, locale, outputFlag string, pretty bool) error {
	backend, err := newMediaBackend(ctx)
	if err != nil {
		return err
	}
	pkg := shared.ResolvePackageName(packageName, backend.Config())
	if strings.TrimSpace(pkg) == "" {
		return fmt.Errorf("--package is required")
	}
	if strings.TrimSpace(editID) == "" {
		return fmt.Errorf("--edit is required")
	}

	result, err := pullMedia(ctx, backend, pkg, editID, dir, locale)
	if err != nil {
		return err
	}
	return shared.PrintOutput(result, outputFlag, pretty)
}

func runMediaSync(ctx context.Context, packageName, editID, dir, locale, outputFlag string, pretty bool) error {
	backend, err := newMediaBackend(ctx)
	if err != nil {
		return err
	}
	pkg := shared.ResolvePackageName(packageName, backend.Config())
	if strings.TrimSpace(pkg) == "" {
		return fmt.Errorf("--package is required")
	}
	if strings.TrimSpace(editID) == "" {
		return fmt.Errorf("--edit is required")
	}

	result, err := syncMedia(ctx, backend, pkg, editID, dir, locale)
	if err != nil {
		return err
	}
	return shared.PrintOutput(result, outputFlag, pretty)
}

type mediaBackendAdapter struct {
	service *playclient.Service
}

func (a *mediaBackendAdapter) Config() *config.Config { return a.service.Cfg }

func newPlayMediaBackend(ctx context.Context) (mediaBackend, error) {
	service, err := playclient.NewService(ctx)
	if err != nil {
		return nil, err
	}
	return &mediaBackendAdapter{service: service}, nil
}

func (a *mediaBackendAdapter) ListLocales(ctx context.Context, packageName, editID string) ([]string, error) {
	resp, err := a.service.API.Edits.Listings.List(packageName, editID).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	locales := make([]string, 0, len(resp.Listings))
	for _, listing := range resp.Listings {
		if strings.TrimSpace(listing.Language) != "" {
			locales = append(locales, listing.Language)
		}
	}
	sort.Strings(locales)
	return uniqueStrings(locales), nil
}

func (a *mediaBackendAdapter) ListImages(ctx context.Context, packageName, editID, locale, imageType string) ([]remoteImage, error) {
	resp, err := a.service.API.Edits.Images.List(packageName, editID, locale, imageType).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	out := make([]remoteImage, 0, len(resp.Images))
	for _, img := range resp.Images {
		if img == nil {
			continue
		}
		out = append(out, remoteImage{
			ID:     img.Id,
			Sha1:   img.Sha1,
			Sha256: img.Sha256,
			URL:    img.Url,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Sha256 < out[j].Sha256 || (out[i].Sha256 == out[j].Sha256 && out[i].ID < out[j].ID)
	})
	return out, nil
}

func (a *mediaBackendAdapter) UploadImage(ctx context.Context, packageName, editID, locale, imageType, filePath string) (*androidpublisher.Image, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, shared.WrapActionable(err, "failed to open image file", "Check that the file exists and is readable.")
	}
	defer file.Close()

	call := a.service.API.Edits.Images.Upload(packageName, editID, locale, imageType)
	call.Media(file, googleapi.ContentType(mimeTypeForImage(filePath)))
	ctx, cancel := shared.ContextWithUploadTimeout(ctx, a.service.Cfg)
	defer cancel()
	resp, err := call.Context(ctx).Do()
	if err != nil {
		return nil, shared.WrapGoogleAPIError("failed to upload image", err)
	}
	return resp.Image, nil
}

func buildMediaPlan(ctx context.Context, backend mediaBackend, packageName, editID, rootDir, localeFilter string) (*mediaPlan, error) {
	localMedia, err := scanLocalMedia(rootDir, localeFilter)
	if err != nil {
		return nil, err
	}

	remoteLocales, err := backend.ListLocales(ctx, packageName, editID)
	if err != nil {
		return nil, err
	}

	locales := sortedLocaleUnion(localMedia, remoteLocales, localeFilter)
	plan := &mediaPlan{
		Package: packageName,
		EditID:  editID,
		Dir:     rootDir,
	}

	for _, locale := range locales {
		plan.Summary.Locales++
		localePlan := mediaPlanLocale{Locale: locale}
		localAssets := localMedia[locale]
		remoteAssets := map[string][]remoteImage{}
		for _, imageType := range imageTypeOrder {
			remote, err := backend.ListImages(ctx, packageName, editID, locale, imageType)
			if err != nil {
				plan.Errors = append(plan.Errors, fmt.Sprintf("[%s/%s] %v", locale, imageType, err))
				plan.Summary.Errors++
				continue
			}
			remoteAssets[imageType] = remote
		}

		for _, imageType := range imageTypeOrder {
			assets := localAssets[imageType]
			remotes := remoteAssets[imageType]
			localePlan.Assets = append(localePlan.Assets, compareMediaAssets(imageType, assets, remotes, &plan.Summary)...)
		}

		if len(localePlan.Assets) > 0 {
			plan.Locales = append(plan.Locales, localePlan)
		}
	}
	return plan, nil
}

func compareMediaAssets(imageType string, localAssets []localAsset, remoteAssets []remoteImage, summary *mediaPlanSummary) []mediaPlanAsset {
	remoteBySHA := make(map[string]remoteImage, len(remoteAssets))
	for _, remote := range remoteAssets {
		key := strings.TrimSpace(remote.Sha256)
		if key == "" {
			key = strings.TrimSpace(remote.Sha1)
		}
		if key != "" {
			remoteBySHA[key] = remote
		}
	}

	var out []mediaPlanAsset
	for _, asset := range localAssets {
		remote, ok := remoteBySHA[asset.Sha256]
		if ok {
			out = append(out, mediaPlanAsset{
				Type:         imageType,
				Action:       "keep",
				LocalPath:    asset.Path,
				LocalSHA256:  asset.Sha256,
				RemoteID:     remote.ID,
				RemoteSHA256: remote.Sha256,
				RemoteURL:    remote.URL,
			})
			summary.Keep++
			continue
		}
		out = append(out, mediaPlanAsset{
			Type:        imageType,
			Action:      "upload",
			LocalPath:   asset.Path,
			LocalSHA256: asset.Sha256,
		})
		summary.Upload++
	}

	localBySHA := make(map[string]struct{}, len(localAssets))
	for _, asset := range localAssets {
		localBySHA[asset.Sha256] = struct{}{}
	}
	for _, remote := range remoteAssets {
		key := strings.TrimSpace(remote.Sha256)
		if key == "" {
			key = strings.TrimSpace(remote.Sha1)
		}
		if key == "" {
			continue
		}
		if _, ok := localBySHA[key]; ok {
			continue
		}
		out = append(out, mediaPlanAsset{
			Type:         imageType,
			Action:       "remote-only",
			RemoteID:     remote.ID,
			RemoteSHA256: remote.Sha256,
			RemoteURL:    remote.URL,
		})
		summary.RemoteOnly++
	}
	return out
}

func pullMedia(ctx context.Context, backend mediaBackend, packageName, editID, rootDir, localeFilter string) (*pullResult, error) {
	locales, err := backend.ListLocales(ctx, packageName, editID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(localeFilter) != "" {
		locales = []string{localeFilter}
	}
	sort.Strings(locales)

	result := &pullResult{
		Package: packageName,
		EditID:  editID,
		Dir:     rootDir,
		DryRun:  shared.IsDryRun(ctx),
	}
	for _, locale := range locales {
		for _, imageType := range imageTypeOrder {
			remoteAssets, err := backend.ListImages(ctx, packageName, editID, locale, imageType)
			if err != nil || len(remoteAssets) == 0 {
				continue
			}
			targetDir := filepath.Join(rootDir, locale, "images")
			if isScreenshotType(imageType) {
				targetDir = filepath.Join(targetDir, imageType)
			}
			for i, remote := range remoteAssets {
				if strings.TrimSpace(remote.URL) == "" {
					continue
				}
				fileName := imageType
				if isScreenshotType(imageType) {
					fileName = fmt.Sprintf("%02d", i+1)
				}
				ext := extFromRemote(remote.URL)
				targetPath := filepath.Join(targetDir, fileName+ext)
				if shared.IsDryRun(ctx) {
					result.Files = append(result.Files, targetPath)
					continue
				}
				if err := os.MkdirAll(targetDir, 0o755); err != nil {
					return nil, err
				}
				if err := downloadRemote(ctx, remote.URL, targetPath); err != nil {
					return nil, err
				}
				result.Files = append(result.Files, targetPath)
			}
		}
	}
	return result, nil
}

func syncMedia(ctx context.Context, backend mediaBackend, packageName, editID, rootDir, localeFilter string) (*syncResult, error) {
	plan, err := buildMediaPlan(ctx, backend, packageName, editID, rootDir, localeFilter)
	if err != nil {
		return nil, err
	}
	result := &syncResult{
		Package:    packageName,
		EditID:     editID,
		Dir:        rootDir,
		DryRun:     shared.IsDryRun(ctx),
		RemoteOnly: plan.Summary.RemoteOnly,
		Errors:     append([]string(nil), plan.Errors...),
	}

	localMedia, err := scanLocalMedia(rootDir, localeFilter)
	if err != nil {
		return nil, err
	}

	for _, localePlan := range plan.Locales {
		for _, asset := range localePlan.Assets {
			switch asset.Action {
			case "keep":
				result.Kept++
			case "upload":
				localAsset := findLocalAsset(localMedia, localePlan.Locale, asset.Type, asset.LocalPath)
				if localAsset == nil {
					continue
				}
				if _, err := backend.UploadImage(ctx, packageName, editID, localePlan.Locale, asset.Type, localAsset.Path); err != nil {
					result.Errors = append(result.Errors, err.Error())
					continue
				}
				result.Uploaded++
			}
		}
	}
	return result, nil
}

func scanLocalMedia(rootDir, localeFilter string) (map[string]map[string][]localAsset, error) {
	info, err := os.Stat(rootDir)
	if err != nil {
		return nil, fmt.Errorf("media directory not found: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("media path is not a directory: %s", rootDir)
	}

	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read media directory: %w", err)
	}

	locales := make(map[string]map[string][]localAsset)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		locale := entry.Name()
		if strings.TrimSpace(localeFilter) != "" && locale != localeFilter {
			continue
		}
		localeAssets, err := scanLocaleMedia(filepath.Join(rootDir, locale))
		if err != nil {
			return nil, err
		}
		if len(localeAssets) > 0 {
			locales[locale] = localeAssets
		}
	}
	return locales, nil
}

func scanLocaleMedia(localeDir string) (map[string][]localAsset, error) {
	imagesDir := filepath.Join(localeDir, "images")
	if _, err := os.Stat(imagesDir); err != nil {
		if os.IsNotExist(err) {
			return map[string][]localAsset{}, nil
		}
		return nil, err
	}

	entries, err := os.ReadDir(imagesDir)
	if err != nil {
		return nil, err
	}

	assets := make(map[string][]localAsset)
	for _, entry := range entries {
		path := filepath.Join(imagesDir, entry.Name())
		if entry.IsDir() {
			if !isScreenshotType(entry.Name()) {
				return nil, fmt.Errorf("unknown screenshot type %q in %s; valid types: %s", entry.Name(), localeDir, validScreenshotTypesString())
			}
			files, err := os.ReadDir(path)
			if err != nil {
				return nil, err
			}
			for _, file := range files {
				if file.IsDir() {
					continue
				}
				ext := strings.ToLower(filepath.Ext(file.Name()))
				if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".webp" {
					continue
				}
				filePath := filepath.Join(path, file.Name())
				sha, err := sha256OfFile(filePath)
				if err != nil {
					return nil, err
				}
				assets[entry.Name()] = append(assets[entry.Name()], localAsset{
					Locale:   filepath.Base(localeDir),
					Type:     entry.Name(),
					Path:     filePath,
					Sha256:   sha,
					FileName: file.Name(),
				})
			}
			continue
		}

		imageType, ok := singleImageTypes[entry.Name()]
		if !ok {
			continue
		}
		if ext := strings.ToLower(filepath.Ext(entry.Name())); ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".webp" {
			continue
		}
		sha, err := sha256OfFile(path)
		if err != nil {
			return nil, err
		}
		assets[imageType] = append(assets[imageType], localAsset{
			Locale:   filepath.Base(localeDir),
			Type:     imageType,
			Path:     path,
			Sha256:   sha,
			FileName: entry.Name(),
		})
	}

	for _, assetsForType := range assets {
		sort.Slice(assetsForType, func(i, j int) bool { return assetsForType[i].Path < assetsForType[j].Path })
	}
	return assets, nil
}

func findLocalAsset(localMedia map[string]map[string][]localAsset, locale, imageType, path string) *localAsset {
	byType := localMedia[locale]
	for _, asset := range byType[imageType] {
		if asset.Path == path {
			return &asset
		}
	}
	return nil
}

func sortedLocaleUnion(localMedia map[string]map[string][]localAsset, remoteLocales []string, localeFilter string) []string {
	set := make(map[string]struct{})
	if strings.TrimSpace(localeFilter) != "" {
		set[localeFilter] = struct{}{}
	}
	for locale := range localMedia {
		set[locale] = struct{}{}
	}
	for _, locale := range remoteLocales {
		set[locale] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for locale := range set {
		out = append(out, locale)
	}
	sort.Strings(out)
	return out
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}
	out := values[:0]
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func isScreenshotType(imageType string) bool {
	_, ok := supportedScreenshotTypes[imageType]
	return ok
}

func validScreenshotTypesString() string {
	values := make([]string, 0, len(supportedScreenshotTypes))
	for key := range supportedScreenshotTypes {
		values = append(values, key)
	}
	sort.Strings(values)
	return strings.Join(values, ", ")
}

func sha256OfFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

func extFromRemote(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err == nil {
		if ext := filepath.Ext(parsed.Path); ext != "" {
			return ext
		}
	}
	if ext := filepath.Ext(rawURL); ext != "" {
		return ext
	}
	return ".png"
}

func defaultDownloadRemote(ctx context.Context, rawURL, targetPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download failed: %s", resp.Status)
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}
	file, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, resp.Body)
	return err
}
