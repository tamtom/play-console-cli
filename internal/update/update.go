package update

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/mod/semver"

	"github.com/tamtom/play-console-cli/internal/rootfs"
	"github.com/tamtom/play-console-cli/internal/version"
)

const (
	// GitHubRepo is the repository for releases
	GitHubRepo = "tamtom/play-console-cli"

	// BinaryName is the name of the binary
	BinaryName = "gplay"

	// CheckInterval is how often to check for updates
	CheckInterval = 24 * time.Hour
)

// Release represents a GitHub release
type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []Asset   `json:"assets"`
	HTMLURL     string    `json:"html_url"`
}

// Asset represents a release asset
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// UpdateInfo contains information about an available update
type UpdateInfo struct {
	CurrentVersion string
	LatestVersion  string
	ReleaseURL     string
	DownloadURL    string
	AssetName      string
	ChecksumURL    string
	IsNewer        bool
}

// Options configures update behavior
type Options struct {
	CurrentVersion string

	// SkipCheck disables update checking
	SkipCheck bool

	// ForceCheck ignores the check interval cache
	ForceCheck bool
}

type cachedRelease struct {
	CheckedAt time.Time `json:"checked_at"`
	Release   Release   `json:"release"`
}

func cachePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".cache", "gplay", "update-release.json"), nil
}

// CheckForUpdate caches only successful stable-release lookups. A cached
// release is compared with the running version again after every upgrade.
func CheckForUpdate(ctx context.Context, opts Options) (*UpdateInfo, error) {
	if opts.SkipCheck {
		return nil, nil
	}
	current := opts.CurrentVersion
	if current == "" {
		current = version.Version
	}
	current = strings.TrimPrefix(current, "v")
	if !opts.ForceCheck && (!semver.IsValid("v"+current) || os.Getenv("GPLAY_NO_UPDATE") == "1") {
		return nil, nil
	}

	var release *Release
	path, pathErr := cachePath()
	if !opts.ForceCheck && pathErr == nil {
		if data, err := os.ReadFile(path); err == nil {
			var cached cachedRelease
			if json.Unmarshal(data, &cached) == nil && time.Since(cached.CheckedAt) >= 0 && time.Since(cached.CheckedAt) < CheckInterval && validStableRelease(&cached.Release) {
				release = &cached.Release
			}
		}
	}
	if release == nil {
		var err error
		release, err = getLatestRelease(ctx)
		if err != nil {
			return nil, err
		}
		if !validStableRelease(release) {
			return nil, fmt.Errorf("GitHub did not return a valid stable release")
		}
		if pathErr == nil {
			data, err := json.Marshal(cachedRelease{CheckedAt: time.Now().UTC(), Release: *release})
			if err == nil {
				_ = rootfs.AtomicWriteFile(path, data, 0o600, 0o700)
			}
		}
	}
	latest := strings.TrimPrefix(release.TagName, "v")
	info := &UpdateInfo{
		CurrentVersion: current,
		LatestVersion:  latest,
		ReleaseURL:     release.HTMLURL,
		IsNewer:        semver.IsValid("v"+current) && compareVersions(latest, current) > 0,
	}
	for _, asset := range release.Assets {
		if asset.Name == getBinaryName() {
			info.DownloadURL, info.AssetName = asset.BrowserDownloadURL, asset.Name
		}
		if asset.Name == "checksums.txt" {
			info.ChecksumURL = asset.BrowserDownloadURL
		}
	}
	return info, nil
}

func validStableRelease(release *Release) bool {
	v := "v" + strings.TrimPrefix(release.TagName, "v")
	return !release.Draft && !release.Prerelease && semver.IsValid(v) && semver.Prerelease(v) == ""
}

// getLatestRelease fetches the latest release from GitHub
func getLatestRelease(ctx context.Context) (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", GitHubRepo)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

// getBinaryName returns the expected binary name for the current platform
func getBinaryName() string {
	name := fmt.Sprintf("%s-%s-%s", BinaryName, runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// compareVersions compares two semver versions
// Returns: 1 if a > b, -1 if a < b, 0 if equal
func compareVersions(a, b string) int {
	return semver.Compare("v"+strings.TrimPrefix(a, "v"), "v"+strings.TrimPrefix(b, "v"))
}

// DownloadUpdate downloads the latest binary
func DownloadUpdate(ctx context.Context, info *UpdateInfo) (string, error) {
	if info == nil || info.DownloadURL == "" {
		return "", fmt.Errorf("no download URL available for this platform")
	}
	if info.ChecksumURL == "" || info.AssetName != getBinaryName() {
		return "", fmt.Errorf("release is missing checksum metadata for this platform")
	}
	client := &http.Client{Timeout: 5 * time.Minute}
	expected, err := releaseChecksum(ctx, client, info.ChecksumURL, info.AssetName)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", info.DownloadURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "gplay-update-*")
	if err != nil {
		return "", err
	}
	success := false
	defer func() {
		_ = tmpFile.Close()
		if !success {
			_ = os.Remove(tmpFile.Name())
		}
	}()

	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmpFile, hash), resp.Body); err != nil {
		return "", err
	}
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("close update download: %w", err)
	}
	if fmt.Sprintf("%x", hash.Sum(nil)) != expected {
		return "", fmt.Errorf("SHA-256 checksum mismatch for %s", info.AssetName)
	}
	success = true
	return tmpFile.Name(), nil
}

// ApplyUpdate replaces the current binary with the new one
func ApplyUpdate(newBinaryPath, currentBinary string) error {
	sourceRoot, err := rootfs.Open(filepath.Dir(newBinaryPath))
	if err != nil {
		return err
	}
	defer func() { _ = sourceRoot.Close() }()
	source, err := sourceRoot.OpenRead(filepath.Base(newBinaryPath))
	if err != nil {
		return err
	}
	defer func() { _ = source.Close() }()
	if err := replaceExecutable(currentBinary, source); err != nil {
		return err
	}
	_ = os.Remove(newBinaryPath)

	return nil
}

func PrintUpdateMessageTo(w io.Writer, info *UpdateInfo) {
	if info == nil || !info.IsNewer {
		return
	}

	fmt.Fprintf(w, "\nA new version of gplay is available: %s → %s\n", info.CurrentVersion, info.LatestVersion)

	// Check installation method and provide appropriate instructions
	executable, _ := os.Executable()
	if resolved, err := filepath.EvalSymlinks(executable); err == nil {
		executable = resolved
	}
	fmt.Fprintf(w, "Update with: %s\n", InstallHint(executable))
	fmt.Fprintf(w, "Release notes: %s\n\n", info.ReleaseURL)
}

// DetectInstallMethod selects update instructions for the resolved executable path.
func DetectInstallMethod(path string) string {
	if strings.Contains(path, "homebrew") || strings.Contains(path, "Cellar") || strings.Contains(path, "linuxbrew") {
		return "homebrew"
	}
	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		if home, err := os.UserHomeDir(); err == nil {
			goPath = filepath.Join(home, "go")
		}
	}
	binDirs := []string{os.Getenv("GOBIN")}
	for _, dir := range filepath.SplitList(goPath) {
		binDirs = append(binDirs, filepath.Join(dir, "bin"))
	}
	for _, dir := range binDirs {
		if dir != "" && filepath.Clean(filepath.Dir(path)) == filepath.Clean(dir) {
			return "goinstall"
		}
	}
	return "binary"
}

func InstallHint(path string) string {
	switch DetectInstallMethod(path) {
	case "homebrew":
		return "brew upgrade tamtom/tap/gplay"
	case "goinstall":
		return "go install github.com/tamtom/play-console-cli@latest (installs play-console-cli; rename it to gplay if desired)"
	default:
		return "gplay update"
	}
}
