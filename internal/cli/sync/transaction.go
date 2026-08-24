package sync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/googleapi"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
	"github.com/tamtom/play-console-cli/internal/rootfs"
)

const syncPlanVersion = 1

type syncPlan struct {
	Version           int          `json:"version"`
	Provider          string       `json:"provider"`
	API               string       `json:"api"`
	Package           string       `json:"package"`
	EditID            string       `json:"editId"`
	SourceDir         string       `json:"sourceDir"`
	SourceFingerprint string       `json:"sourceFingerprint"`
	ExpiresAt         string       `json:"expiresAt"`
	CapabilityIDs     []string     `json:"capabilityIds"`
	DestructiveCount  int          `json:"destructiveCount"`
	PlanHash          string       `json:"planHash"`
	Effects           []syncEffect `json:"effects"`
}

type syncEffect struct {
	ID           string          `json:"id"`
	Kind         string          `json:"kind"`
	Locale       string          `json:"locale"`
	ImageType    string          `json:"imageType,omitempty"`
	RelativePath string          `json:"relativePath,omitempty"`
	OldSHA256    string          `json:"oldSha256"`
	NewSHA256    string          `json:"newSha256"`
	Listing      *listingPayload `json:"listing,omitempty"`
}

type listingPayload struct {
	Title            string `json:"title,omitempty"`
	ShortDescription string `json:"shortDescription,omitempty"`
	FullDescription  string `json:"fullDescription,omitempty"`
	Video            string `json:"video,omitempty"`
}

type sourceRecord struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type localSyncSnapshot struct {
	SourceDir   string
	Fingerprint string
	Listings    map[string]listingPayload
	Images      []localSyncImage
	Records     []sourceRecord
}

type localSyncImage struct {
	Locale       string
	ImageType    string
	RelativePath string
	SHA256       string
}

type syncReceipt struct {
	Version  int             `json:"version"`
	Provider string          `json:"provider"`
	PlanHash string          `json:"planHash"`
	Package  string          `json:"package"`
	EditID   string          `json:"editId"`
	Status   string          `json:"status"`
	Effects  []receiptEffect `json:"effects"`
}

type syncRunResult struct {
	PlanFile    string       `json:"planFile"`
	ReceiptFile string       `json:"receiptFile"`
	Receipt     *syncReceipt `json:"receipt"`
}

// RunResult is the durable output of a one-shot official listing/image sync.
type RunResult = syncRunResult

type receiptEffect struct {
	ID           string `json:"id"`
	Kind         string `json:"kind"`
	Locale       string `json:"locale"`
	ImageType    string `json:"imageType,omitempty"`
	RelativePath string `json:"relativePath,omitempty"`
	Status       string `json:"status"`
	ResultSHA256 string `json:"resultSha256,omitempty"`
}

// PlanCommand builds a content-addressed listing and image synchronization plan.
func PlanCommand() *ffcli.Command {
	fs := flag.NewFlagSet("sync plan", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	inputDir := fs.String("dir", "./metadata", "Fastlane-compatible metadata directory")
	planFile := fs.String("plan-file", "", "Path for the generated plan JSON")
	outputFlags := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "plan",
		ShortUsage: "gplay sync plan --package <name> --edit <id> --dir <path> --plan-file <path>",
		ShortHelp:  "Create a deterministic listings and images synchronization plan.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(outputFlags.Format(), outputFlags.IsPretty()); err != nil {
				return err
			}
			if strings.TrimSpace(*packageName) == "" {
				return fmt.Errorf("--package is required")
			}
			if strings.TrimSpace(*editID) == "" {
				return fmt.Errorf("--edit is required")
			}
			if strings.TrimSpace(*inputDir) == "" {
				return fmt.Errorf("--dir is required")
			}
			if strings.TrimSpace(*planFile) == "" {
				return fmt.Errorf("--plan-file is required")
			}
			plan, err := createSyncPlan(ctx, *packageName, *editID, *inputDir)
			if err != nil {
				return err
			}
			if err := writePlan(shared.FilesystemFrom(ctx), *planFile, plan); err != nil {
				return err
			}
			return shared.PrintOutputContext(ctx, plan, outputFlags.Format(), outputFlags.IsPretty())
		},
	}
}

// ApplyCommand executes a previously generated synchronization plan and writes
// an atomic receipt that is reused to resume interrupted executions.
func ApplyCommand() *ffcli.Command {
	fs := flag.NewFlagSet("sync apply", flag.ExitOnError)
	planFile := fs.String("plan-file", "", "Path to a synchronization plan JSON")
	receiptFile := fs.String("receipt-file", "", "Path for the execution receipt (defaults beside the plan)")
	outputFlags := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "apply",
		ShortUsage: "gplay sync apply --plan-file <path> [--receipt-file <path>]",
		ShortHelp:  "Apply or safely resume a synchronization plan.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(outputFlags.Format(), outputFlags.IsPretty()); err != nil {
				return err
			}
			if strings.TrimSpace(*planFile) == "" {
				return fmt.Errorf("--plan-file is required")
			}
			path := strings.TrimSpace(*receiptFile)
			if path == "" {
				path = *planFile + ".receipt.json"
			}
			receipt, err := applySyncPlan(ctx, *planFile, path)
			if err != nil {
				return err
			}
			return shared.PrintOutputContext(ctx, receipt, outputFlags.Format(), outputFlags.IsPretty())
		},
	}
}

func createSyncPlan(ctx context.Context, packageName, editID, inputDir string) (*syncPlan, error) {
	snapshot, err := scanSyncSource(inputDir)
	if err != nil {
		return nil, err
	}
	service, err := playclient.NewService(ctx)
	if err != nil {
		return nil, err
	}
	ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
	defer cancel()
	edit, err := service.API.Edits.Get(packageName, editID).Context(ctx).Do()
	if err != nil {
		return nil, shared.WrapGoogleAPIError("read edit expiry", err)
	}
	expirySeconds, err := strconv.ParseInt(strings.TrimSpace(edit.ExpiryTimeSeconds), 10, 64)
	if err != nil || expirySeconds <= 0 {
		return nil, fmt.Errorf("edit %s did not return a valid expiry time", editID)
	}
	response, err := service.API.Edits.Listings.List(packageName, editID).Context(ctx).Do()
	if err != nil {
		return nil, shared.WrapGoogleAPIError("list remote store listings", err)
	}
	remote := make(map[string]listingPayload, len(response.Listings))
	for _, listing := range response.Listings {
		if listing == nil || strings.TrimSpace(listing.Language) == "" {
			continue
		}
		remote[listing.Language] = listingPayloadFromAPI(listing)
	}

	plan := &syncPlan{
		Version:           syncPlanVersion,
		Provider:          "official-api",
		API:               "android-publisher-v3",
		Package:           packageName,
		EditID:            editID,
		SourceDir:         snapshot.SourceDir,
		SourceFingerprint: snapshot.Fingerprint,
		ExpiresAt:         time.Unix(expirySeconds, 0).UTC().Format(time.RFC3339),
		CapabilityIDs:     []string{"app.store_listing", "app.store_media"},
		DestructiveCount:  0,
	}
	locales := make([]string, 0, len(snapshot.Listings))
	for locale := range snapshot.Listings {
		locales = append(locales, locale)
	}
	sort.Strings(locales)
	for _, locale := range locales {
		local := snapshot.Listings[locale]
		oldHash, err := hashCanonical(remote[locale])
		if err != nil {
			return nil, err
		}
		newHash, err := hashCanonical(local)
		if err != nil {
			return nil, err
		}
		if oldHash == newHash {
			continue
		}
		effect := syncEffect{
			Kind:      "listing.update",
			Locale:    locale,
			OldSHA256: oldHash,
			NewSHA256: newHash,
			Listing:   &local,
		}
		effect.ID = effectID(effect)
		plan.Effects = append(plan.Effects, effect)
	}
	for start := 0; start < len(snapshot.Images); {
		end := start + 1
		for end < len(snapshot.Images) && snapshot.Images[end].Locale == snapshot.Images[start].Locale && snapshot.Images[end].ImageType == snapshot.Images[start].ImageType {
			end++
		}
		group := snapshot.Images[start:end]
		response, listErr := service.API.Edits.Images.List(packageName, editID, group[0].Locale, group[0].ImageType).Context(ctx).Do()
		if listErr != nil {
			return nil, shared.WrapGoogleAPIError("list remote store images", listErr)
		}
		remoteHashes := make([]string, 0, len(response.Images))
		for _, image := range response.Images {
			if image == nil {
				continue
			}
			value := strings.TrimSpace(image.Sha256)
			if value == "" {
				value = strings.TrimSpace(image.Sha1)
			}
			if value != "" {
				remoteHashes = append(remoteHashes, value)
			}
		}
		sort.Strings(remoteHashes)
		for _, local := range group {
			if containsString(remoteHashes, local.SHA256) {
				continue
			}
			oldHash, hashErr := hashCanonical(remoteHashes)
			if hashErr != nil {
				return nil, hashErr
			}
			effect := syncEffect{
				Kind:         "image.upload",
				Locale:       local.Locale,
				ImageType:    local.ImageType,
				RelativePath: local.RelativePath,
				OldSHA256:    oldHash,
				NewSHA256:    local.SHA256,
			}
			effect.ID = effectID(effect)
			plan.Effects = append(plan.Effects, effect)
			remoteHashes = append(remoteHashes, local.SHA256)
			sort.Strings(remoteHashes)
		}
		start = end
	}
	plan.PlanHash, err = planHash(plan)
	if err != nil {
		return nil, err
	}
	return plan, nil
}

func applySyncPlan(ctx context.Context, planPath, receiptPath string) (*syncReceipt, error) {
	filesystem := shared.FilesystemFrom(ctx)
	plan, err := readAndValidateSyncPlan(filesystem, planPath)
	if err != nil {
		return nil, err
	}
	snapshot, err := scanSyncSource(plan.SourceDir)
	if err != nil {
		return nil, err
	}
	if snapshot.Fingerprint != plan.SourceFingerprint {
		return nil, fmt.Errorf("sync source changed after planning: fingerprint %s does not match plan %s; create a new plan", snapshot.Fingerprint, plan.SourceFingerprint)
	}
	expiresAt, err := time.Parse(time.RFC3339, plan.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("sync plan has invalid expiry %q: %w", plan.ExpiresAt, err)
	}
	if !shared.Now(ctx).Before(expiresAt) {
		return nil, fmt.Errorf("sync plan expired at %s; create a new plan", plan.ExpiresAt)
	}
	if shared.IsDryRun(ctx) {
		return newReceipt(plan, "dry-run", "planned"), nil
	}

	receipt, err := loadOrCreateReceipt(filesystem, receiptPath, plan)
	if err != nil {
		return nil, err
	}
	if receipt.Status == "complete" {
		return receipt, nil
	}
	if err := writeReceipt(filesystem, receiptPath, receipt); err != nil {
		return nil, err
	}

	service, err := playclient.NewService(ctx)
	if err != nil {
		return nil, err
	}
	for index := range plan.Effects {
		effect := &plan.Effects[index]
		result := &receipt.Effects[index]
		if result.Status == "applied" || result.Status == "reconciled" {
			continue
		}
		switch effect.Kind {
		case "listing.update":
			status, applyErr := applyListingEffect(ctx, service, plan, effect)
			if applyErr != nil {
				result.Status = "conflict"
				receipt.Status = "incomplete"
				if writeErr := writeReceipt(filesystem, receiptPath, receipt); writeErr != nil {
					return nil, errors.Join(applyErr, writeErr)
				}
				return nil, applyErr
			}
			result.Status = status
			result.ResultSHA256 = effect.NewSHA256
		case "image.upload":
			status, applyErr := applyImageEffect(ctx, service, plan, effect)
			if applyErr != nil {
				result.Status = "conflict"
				receipt.Status = "incomplete"
				if writeErr := writeReceipt(filesystem, receiptPath, receipt); writeErr != nil {
					return nil, errors.Join(applyErr, writeErr)
				}
				return nil, applyErr
			}
			result.Status = status
			result.ResultSHA256 = effect.NewSHA256
		default:
			return nil, fmt.Errorf("unsupported sync effect kind %q", effect.Kind)
		}
		if err := writeReceipt(filesystem, receiptPath, receipt); err != nil {
			return nil, err
		}
	}
	receipt.Status = "complete"
	if err := writeReceipt(filesystem, receiptPath, receipt); err != nil {
		return nil, err
	}
	return receipt, nil
}

func applyListingEffect(ctx context.Context, service *playclient.Service, plan *syncPlan, effect *syncEffect) (string, error) {
	ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
	defer cancel()
	remote := listingPayload{}
	current, err := service.API.Edits.Listings.Get(plan.Package, plan.EditID, effect.Locale).Context(ctx).Do()
	if err != nil {
		var apiErr *googleapi.Error
		if !errors.As(err, &apiErr) || apiErr.Code != httpStatusNotFound {
			return "", shared.WrapGoogleAPIError("read listing precondition", err)
		}
	} else {
		remote = listingPayloadFromAPI(current)
	}
	remoteHash, err := hashCanonical(remote)
	if err != nil {
		return "", err
	}
	if remoteHash == effect.NewSHA256 {
		return "reconciled", nil
	}
	if remoteHash != effect.OldSHA256 {
		return "", fmt.Errorf("listing %s changed remotely after planning: current fingerprint %s, expected %s", effect.Locale, remoteHash, effect.OldSHA256)
	}
	if effect.Listing == nil {
		return "", fmt.Errorf("listing effect %s has no desired listing", effect.ID)
	}
	if _, err := service.API.Edits.Listings.Update(plan.Package, plan.EditID, effect.Locale, effect.Listing.apiListing()).Context(ctx).Do(); err != nil {
		return "", shared.WrapGoogleAPIError("update store listing", err)
	}
	return "applied", nil
}

func applyImageEffect(ctx context.Context, service *playclient.Service, plan *syncPlan, effect *syncEffect) (string, error) {
	ctx, cancel := shared.ContextWithUploadTimeout(ctx, service.Cfg)
	defer cancel()
	response, err := service.API.Edits.Images.List(plan.Package, plan.EditID, effect.Locale, effect.ImageType).Context(ctx).Do()
	if err != nil {
		return "", shared.WrapGoogleAPIError("read image precondition", err)
	}
	remoteHashes := make([]string, 0, len(response.Images))
	for _, image := range response.Images {
		if image == nil {
			continue
		}
		value := strings.TrimSpace(image.Sha256)
		if value == "" {
			value = strings.TrimSpace(image.Sha1)
		}
		if value != "" {
			remoteHashes = append(remoteHashes, value)
		}
	}
	sort.Strings(remoteHashes)
	if containsString(remoteHashes, effect.NewSHA256) {
		return "reconciled", nil
	}
	remoteHash, err := hashCanonical(remoteHashes)
	if err != nil {
		return "", err
	}
	if remoteHash != effect.OldSHA256 {
		return "", fmt.Errorf("images %s/%s changed remotely after planning: current fingerprint %s, expected %s", effect.Locale, effect.ImageType, remoteHash, effect.OldSHA256)
	}

	sourceRoot, err := rootfs.Open(plan.SourceDir)
	if err != nil {
		return "", fmt.Errorf("open sync source: %w", err)
	}
	defer func() { _ = sourceRoot.Close() }()
	file, err := sourceRoot.OpenRead(filepath.FromSlash(effect.RelativePath))
	if err != nil {
		return "", fmt.Errorf("open planned image %q: %w", effect.RelativePath, err)
	}
	defer func() { _ = file.Close() }()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("hash planned image %q: %w", effect.RelativePath, err)
	}
	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if actualHash != effect.NewSHA256 {
		return "", fmt.Errorf("planned image %q changed: fingerprint %s does not match %s", effect.RelativePath, actualHash, effect.NewSHA256)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("rewind planned image %q: %w", effect.RelativePath, err)
	}
	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(effect.RelativePath)))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	call := service.API.Edits.Images.Upload(plan.Package, plan.EditID, effect.Locale, effect.ImageType)
	call.Media(file, googleapi.ContentType(contentType))
	if _, err := call.Context(ctx).Do(); err != nil {
		return "", shared.WrapGoogleAPIError("upload store image", err)
	}
	return "applied", nil
}

const httpStatusNotFound = 404

func readAndValidateSyncPlan(filesystem shared.Filesystem, path string) (*syncPlan, error) {
	data, err := filesystem.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read sync plan: %w", err)
	}
	var plan syncPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("decode sync plan: %w", err)
	}
	if plan.Version != syncPlanVersion {
		return nil, fmt.Errorf("unsupported sync plan version %d", plan.Version)
	}
	if plan.Provider != "official-api" || plan.API != "android-publisher-v3" {
		return nil, fmt.Errorf("unsupported sync plan provider %q", plan.Provider)
	}
	if strings.TrimSpace(plan.Package) == "" || strings.TrimSpace(plan.EditID) == "" || strings.TrimSpace(plan.SourceDir) == "" {
		return nil, fmt.Errorf("sync plan target and source are required")
	}
	expectedHash, err := planHash(&plan)
	if err != nil {
		return nil, err
	}
	if plan.PlanHash != expectedHash {
		return nil, fmt.Errorf("sync plan hash mismatch: got %s, expected %s", plan.PlanHash, expectedHash)
	}
	for index := range plan.Effects {
		if plan.Effects[index].ID != effectID(plan.Effects[index]) {
			return nil, fmt.Errorf("sync plan effect %d hash mismatch", index)
		}
	}
	return &plan, nil
}

func loadOrCreateReceipt(filesystem shared.Filesystem, path string, plan *syncPlan) (*syncReceipt, error) {
	data, err := filesystem.ReadFile(path)
	if err == nil {
		var receipt syncReceipt
		if err := json.Unmarshal(data, &receipt); err != nil {
			return nil, fmt.Errorf("decode sync receipt: %w", err)
		}
		if receipt.Version != syncPlanVersion || receipt.Provider != plan.Provider || receipt.PlanHash != plan.PlanHash || receipt.Package != plan.Package || receipt.EditID != plan.EditID {
			return nil, fmt.Errorf("sync receipt does not belong to plan %s", plan.PlanHash)
		}
		if len(receipt.Effects) != len(plan.Effects) {
			return nil, fmt.Errorf("sync receipt effect count does not match plan")
		}
		for index := range receipt.Effects {
			if receipt.Effects[index].ID != plan.Effects[index].ID {
				return nil, fmt.Errorf("sync receipt effect %d does not match plan", index)
			}
		}
		return &receipt, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read sync receipt: %w", err)
	}
	return newReceipt(plan, "incomplete", "pending"), nil
}

func newReceipt(plan *syncPlan, status, effectStatus string) *syncReceipt {
	receipt := &syncReceipt{
		Version:  syncPlanVersion,
		Provider: plan.Provider,
		PlanHash: plan.PlanHash,
		Package:  plan.Package,
		EditID:   plan.EditID,
		Status:   status,
		Effects:  make([]receiptEffect, 0, len(plan.Effects)),
	}
	for _, effect := range plan.Effects {
		receipt.Effects = append(receipt.Effects, receiptEffect{
			ID:           effect.ID,
			Kind:         effect.Kind,
			Locale:       effect.Locale,
			ImageType:    effect.ImageType,
			RelativePath: effect.RelativePath,
			Status:       effectStatus,
		})
	}
	return receipt
}

func writeReceipt(filesystem shared.Filesystem, path string, receipt *syncReceipt) error {
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return fmt.Errorf("encode sync receipt: %w", err)
	}
	if err := filesystem.AtomicWriteFile(path, data, 0o600, 0o700); err != nil {
		return fmt.Errorf("write sync receipt: %w", err)
	}
	return nil
}

func writePlan(filesystem shared.Filesystem, path string, plan *syncPlan) error {
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("encode sync plan: %w", err)
	}
	if err := filesystem.AtomicWriteFile(path, data, 0o600, 0o700); err != nil {
		return fmt.Errorf("write sync plan: %w", err)
	}
	return nil
}

func scanSyncSource(inputDir string) (*localSyncSnapshot, error) {
	abs, err := filepath.Abs(inputDir)
	if err != nil {
		return nil, fmt.Errorf("resolve metadata directory: %w", err)
	}
	sourceRoot, err := rootfs.Open(abs)
	if err != nil {
		return nil, fmt.Errorf("open metadata directory: %w", err)
	}
	defer func() { _ = sourceRoot.Close() }()
	entries, err := sourceRoot.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("read metadata directory: %w", err)
	}
	snapshot := &localSyncSnapshot{SourceDir: abs, Listings: map[string]listingPayload{}}
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("%w: %s", rootfs.ErrSymlink, entry.Name())
		}
		if !entry.IsDir() {
			continue
		}
		locale := entry.Name()
		payload := listingPayload{}
		found := false
		for _, field := range []struct {
			name string
			set  func(string)
		}{
			{name: titleFile, set: func(value string) { payload.Title = value }},
			{name: shortDescFile, set: func(value string) { payload.ShortDescription = value }},
			{name: fullDescFile, set: func(value string) { payload.FullDescription = value }},
			{name: videoFile, set: func(value string) { payload.Video = value }},
		} {
			relative := filepath.Join(locale, field.name)
			data, readErr := sourceRoot.ReadFile(relative)
			if errors.Is(readErr, os.ErrNotExist) {
				continue
			}
			if readErr != nil {
				return nil, fmt.Errorf("read metadata source %q: %w", relative, readErr)
			}
			found = true
			field.set(strings.TrimSpace(string(data)))
			snapshot.Records = append(snapshot.Records, sourceRecord{Path: filepath.ToSlash(relative), SHA256: sha256Hex(data)})
		}
		if found {
			snapshot.Listings[locale] = payload
		}
		if err := scanLocaleImages(sourceRoot, locale, snapshot); err != nil {
			return nil, err
		}
	}
	sort.Slice(snapshot.Images, func(i, j int) bool {
		if snapshot.Images[i].Locale != snapshot.Images[j].Locale {
			return snapshot.Images[i].Locale < snapshot.Images[j].Locale
		}
		if snapshot.Images[i].ImageType != snapshot.Images[j].ImageType {
			return snapshot.Images[i].ImageType < snapshot.Images[j].ImageType
		}
		return snapshot.Images[i].RelativePath < snapshot.Images[j].RelativePath
	})
	sort.Slice(snapshot.Records, func(i, j int) bool { return snapshot.Records[i].Path < snapshot.Records[j].Path })
	snapshot.Fingerprint, err = hashCanonical(snapshot.Records)
	if err != nil {
		return nil, fmt.Errorf("fingerprint metadata source: %w", err)
	}
	return snapshot, nil
}

func scanLocaleImages(sourceRoot *rootfs.Root, locale string, snapshot *localSyncSnapshot) error {
	relativeImagesDir := filepath.Join(locale, imagesDir)
	entries, err := sourceRoot.ReadDir(relativeImagesDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read image directory for %s: %w", locale, err)
	}
	imageDirectories := map[string]string{
		phoneScreenshotsDir: "phoneScreenshots",
		tablet7ScreensDir:   "sevenInchScreenshots",
		tablet10ScreensDir:  "tenInchScreenshots",
		tvScreenshotsDir:    "tvScreenshots",
		wearScreenshotsDir:  "wearScreenshots",
	}
	singleImages := map[string]string{
		featureGraphicFile: "featureGraphic",
		iconFile:           "icon",
		promoGraphicFile:   "promoGraphic",
		tvBannerFile:       "tvBanner",
	}
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: %s", rootfs.ErrSymlink, filepath.Join(relativeImagesDir, entry.Name()))
		}
		if entry.IsDir() {
			imageType, ok := imageDirectories[entry.Name()]
			if !ok {
				continue
			}
			directory := filepath.Join(relativeImagesDir, entry.Name())
			files, readErr := sourceRoot.ReadDir(directory)
			if readErr != nil {
				return fmt.Errorf("read screenshot directory %q: %w", directory, readErr)
			}
			for _, file := range files {
				if file.Type()&os.ModeSymlink != 0 {
					return fmt.Errorf("%w: %s", rootfs.ErrSymlink, filepath.Join(directory, file.Name()))
				}
				if file.IsDir() || !isSupportedImageFile(file.Name()) {
					continue
				}
				if err := addLocalImage(sourceRoot, filepath.Join(directory, file.Name()), locale, imageType, snapshot); err != nil {
					return err
				}
			}
			continue
		}
		imageType, ok := singleImages[entry.Name()]
		if !ok || !isSupportedImageFile(entry.Name()) {
			continue
		}
		if err := addLocalImage(sourceRoot, filepath.Join(relativeImagesDir, entry.Name()), locale, imageType, snapshot); err != nil {
			return err
		}
	}
	return nil
}

func addLocalImage(sourceRoot *rootfs.Root, relativePath, locale, imageType string, snapshot *localSyncSnapshot) error {
	data, err := sourceRoot.ReadFile(relativePath)
	if err != nil {
		return fmt.Errorf("read image source %q: %w", relativePath, err)
	}
	value := sha256Hex(data)
	relativePath = filepath.ToSlash(relativePath)
	snapshot.Images = append(snapshot.Images, localSyncImage{Locale: locale, ImageType: imageType, RelativePath: relativePath, SHA256: value})
	snapshot.Records = append(snapshot.Records, sourceRecord{Path: relativePath, SHA256: value})
	return nil
}

func isSupportedImageFile(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png", ".jpg", ".jpeg", ".webp":
		return true
	default:
		return false
	}
}

func containsString(values []string, target string) bool {
	index := sort.SearchStrings(values, target)
	return index < len(values) && values[index] == target
}

func listingPayloadFromAPI(listing *androidpublisher.Listing) listingPayload {
	return listingPayload{
		Title:            listing.Title,
		ShortDescription: listing.ShortDescription,
		FullDescription:  listing.FullDescription,
		Video:            listing.Video,
	}
}

func (p listingPayload) apiListing() *androidpublisher.Listing {
	return &androidpublisher.Listing{
		Title:            p.Title,
		ShortDescription: p.ShortDescription,
		FullDescription:  p.FullDescription,
		Video:            p.Video,
	}
}

func effectID(effect syncEffect) string {
	copyEffect := effect
	copyEffect.ID = ""
	value, _ := hashCanonical(copyEffect)
	return value
}

func planHash(plan *syncPlan) (string, error) {
	copyPlan := *plan
	copyPlan.PlanHash = ""
	return hashCanonical(copyPlan)
}

func hashCanonical(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode canonical value: %w", err)
	}
	return sha256Hex(data), nil
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// RunCommand creates and applies a plan in one invocation.
func RunCommand() *ffcli.Command {
	fs := flag.NewFlagSet("sync run", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	inputDir := fs.String("dir", "./metadata", "Fastlane-compatible metadata directory")
	stateDir := fs.String("state-dir", ".gplay/sync", "Directory for plans and receipts")
	outputFlags := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "run",
		ShortUsage: "gplay sync run --package <name> --edit <id> --dir <path> [--state-dir <path>]",
		ShortHelp:  "Plan, apply, and persist a resumable receipt in one command.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(outputFlags.Format(), outputFlags.IsPretty()); err != nil {
				return err
			}
			if strings.TrimSpace(*packageName) == "" {
				return fmt.Errorf("--package is required")
			}
			if strings.TrimSpace(*editID) == "" {
				return fmt.Errorf("--edit is required")
			}
			if strings.TrimSpace(*inputDir) == "" {
				return fmt.Errorf("--dir is required")
			}
			if strings.TrimSpace(*stateDir) == "" {
				return fmt.Errorf("--state-dir is required")
			}
			result, err := RunTransaction(ctx, *packageName, *editID, *inputDir, *stateDir)
			if err != nil {
				return err
			}
			return shared.PrintOutputContext(ctx, result, outputFlags.Format(), outputFlags.IsPretty())
		},
	}
}

// RunTransaction plans, applies, and persists a resumable official-API sync.
// Other policy-safe workflows use this seam instead of reimplementing listing
// or image mutations.
func RunTransaction(ctx context.Context, packageName, editID, inputDir, stateDir string) (*RunResult, error) {
	if strings.TrimSpace(packageName) == "" || strings.TrimSpace(editID) == "" || strings.TrimSpace(inputDir) == "" || strings.TrimSpace(stateDir) == "" {
		return nil, fmt.Errorf("package, edit, metadata directory, and state directory are required")
	}
	plan, err := createSyncPlan(ctx, packageName, editID, inputDir)
	if err != nil {
		return nil, err
	}
	stateRoot, err := rootfs.OpenOrCreate(stateDir, 0o700)
	if err != nil {
		return nil, fmt.Errorf("open sync state directory: %w", err)
	}
	planRelative := filepath.Join("plans", plan.PlanHash+".json")
	planData, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		_ = stateRoot.Close()
		return nil, fmt.Errorf("encode sync plan: %w", err)
	}
	if err := stateRoot.AtomicWrite(planRelative, planData, 0o600); err != nil {
		_ = stateRoot.Close()
		return nil, fmt.Errorf("write sync plan: %w", err)
	}
	if err := stateRoot.Close(); err != nil {
		return nil, fmt.Errorf("close sync state directory: %w", err)
	}
	planPath := filepath.Join(stateDir, planRelative)
	receiptPath := filepath.Join(stateDir, "receipts", plan.PlanHash+".json")
	receipt, err := applySyncPlan(ctx, planPath, receiptPath)
	if err != nil {
		return nil, err
	}
	return &RunResult{PlanFile: planPath, ReceiptFile: receiptPath, Receipt: receipt}, nil
}
