package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

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
	IsNewer        bool
}

// Options configures update behavior
type Options struct {
	// SkipCheck disables update checking
	SkipCheck bool

	// ForceCheck ignores the check interval cache
	ForceCheck bool

	// AutoUpdate enables automatic updating
	AutoUpdate bool
}

// getCacheDir returns the cache directory for update checks
func getCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cacheDir := filepath.Join(homeDir, ".cache", "gplay")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}
	return cacheDir, nil
}

// shouldCheck returns true if enough time has passed since the last check
func shouldCheck(forceCheck bool) bool {
	if forceCheck {
		return true
	}

	cacheDir, err := getCacheDir()
	if err != nil {
		return true
	}

	lastCheckFile := filepath.Join(cacheDir, "last_update_check")
	info, err := os.Stat(lastCheckFile)
	if err != nil {
		return true
	}

	return time.Since(info.ModTime()) > CheckInterval
}

// recordCheck updates the last check timestamp
func recordCheck() {
	cacheDir, err := getCacheDir()
	if err != nil {
		return
	}

	lastCheckFile := filepath.Join(cacheDir, "last_update_check")
	_ = os.WriteFile(lastCheckFile, []byte(time.Now().Format(time.RFC3339)), 0644)
}

// CheckForUpdate checks if a newer version is available
func CheckForUpdate(ctx context.Context, opts Options) (*UpdateInfo, error) {
	if opts.SkipCheck {
		return nil, nil
	}

	if !shouldCheck(opts.ForceCheck) {
		return nil, nil
	}

	defer recordCheck()

	release, err := getLatestRelease(ctx)
	if err != nil {
		return nil, err
	}

	currentVersion := strings.TrimPrefix(version.Version, "v")
	latestVersion := strings.TrimPrefix(release.TagName, "v")

	info := &UpdateInfo{
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		ReleaseURL:     release.HTMLURL,
		IsNewer:        compareVersions(latestVersion, currentVersion) > 0,
	}

	// Find the appropriate asset for this platform
	assetName := getBinaryName()
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			info.DownloadURL = asset.BrowserDownloadURL
			break
		}
	}

	return info, nil
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
	partsA := strings.Split(strings.TrimPrefix(a, "v"), ".")
	partsB := strings.Split(strings.TrimPrefix(b, "v"), ".")

	for i := 0; i < 3; i++ {
		var numA, numB int
		if i < len(partsA) {
			fmt.Sscanf(partsA[i], "%d", &numA)
		}
		if i < len(partsB) {
			fmt.Sscanf(partsB[i], "%d", &numB)
		}

		if numA > numB {
			return 1
		}
		if numA < numB {
			return -1
		}
	}

	return 0
}

// DownloadUpdate downloads the latest binary
func DownloadUpdate(ctx context.Context, info *UpdateInfo) (string, error) {
	if info.DownloadURL == "" {
		return "", fmt.Errorf("no download URL available for this platform")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", info.DownloadURL, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 5 * time.Minute}
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

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", err
	}

	tmpFile.Close()
	return tmpFile.Name(), nil
}

// ApplyUpdate replaces the current binary with the new one
func ApplyUpdate(newBinaryPath string) error {
	currentBinary, err := os.Executable()
	if err != nil {
		return err
	}

	// Make the new binary executable
	if err := os.Chmod(newBinaryPath, 0755); err != nil {
		return err
	}

	// Backup current binary
	backupPath := currentBinary + ".backup"
	if err := os.Rename(currentBinary, backupPath); err != nil {
		return err
	}

	// Move new binary into place
	if err := os.Rename(newBinaryPath, currentBinary); err != nil {
		// Try to restore backup
		os.Rename(backupPath, currentBinary)
		return err
	}

	// Remove backup
	os.Remove(backupPath)

	return nil
}

// PrintUpdateMessage prints an update notification if one is available
func PrintUpdateMessage(info *UpdateInfo) {
	if info == nil || !info.IsNewer {
		return
	}

	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "A new version of gplay is available: %s → %s\n", info.CurrentVersion, info.LatestVersion)

	// Check installation method and provide appropriate instructions
	if isHomebrew() {
		fmt.Fprintf(os.Stderr, "Update with: brew upgrade tamtom/tap/gplay\n")
	} else {
		fmt.Fprintf(os.Stderr, "Update with: curl -fsSL https://raw.githubusercontent.com/%s/main/install.sh | bash\n", GitHubRepo)
	}

	fmt.Fprintf(os.Stderr, "Release notes: %s\n", info.ReleaseURL)
	fmt.Fprintf(os.Stderr, "\n")
}

// isHomebrew checks if gplay was installed via Homebrew
func isHomebrew() bool {
	executable, err := os.Executable()
	if err != nil {
		return false
	}

	// Check if the executable is in a Homebrew cellar
	return strings.Contains(executable, "/Cellar/") || strings.Contains(executable, "/homebrew/")
}
