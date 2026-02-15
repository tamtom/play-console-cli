package updatecmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/update"
	"github.com/tamtom/play-console-cli/internal/version"
)

// UpdateCommand returns the "gplay update" command.
func UpdateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	check := fs.Bool("check", false, "Only check for updates, don't install")
	force := fs.Bool("force", false, "Force update even if already on latest")

	return &ffcli.Command{
		Name:       "update",
		ShortUsage: "gplay update [--check] [--force]",
		ShortHelp:  "Update gplay to the latest version.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			return runUpdate(ctx, *check, *force)
		},
	}
}

func runUpdate(ctx context.Context, checkOnly bool, force bool) error {
	// Detect installation method
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %w", err)
	}
	execPath, _ = filepath.EvalSymlinks(execPath)
	method := detectInstallMethod(execPath)

	// Check for latest version (force check to bypass cache)
	info, err := update.CheckForUpdate(ctx, update.Options{ForceCheck: true})
	if err != nil {
		return fmt.Errorf("checking for updates: %w", err)
	}
	if info == nil {
		fmt.Fprintf(os.Stderr, "Could not determine latest version.\n")
		return nil
	}

	currentVersion := version.Version
	if !info.IsNewer && !force {
		fmt.Fprintf(os.Stderr, "Already on latest version: %s\n", currentVersion)
		return nil
	}

	if checkOnly {
		fmt.Fprintf(os.Stderr, "Current: %s\nLatest:  %s\n", currentVersion, info.LatestVersion)
		if info.IsNewer {
			fmt.Fprintf(os.Stderr, "Update available! Run 'gplay update' to install.\n")
		}
		return nil
	}

	// Handle based on install method
	switch method {
	case "homebrew":
		fmt.Fprintf(os.Stderr, "Installed via Homebrew. Run:\n  brew upgrade gplay\n")
		return nil
	case "goinstall":
		fmt.Fprintf(os.Stderr, "Installed via go install. Run:\n  go install github.com/tamtom/play-console-cli@latest\n")
		return nil
	case "binary":
		return selfUpdate(ctx, execPath, info)
	default:
		fmt.Fprintf(os.Stderr, "Unknown installation method. Download the latest release from:\n  %s\n", info.ReleaseURL)
		return nil
	}
}

// detectInstallMethod determines how gplay was installed based on the executable path.
func detectInstallMethod(path string) string {
	if strings.Contains(path, "homebrew") || strings.Contains(path, "Cellar") || strings.Contains(path, "linuxbrew") {
		return "homebrew"
	}
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = filepath.Join(os.Getenv("HOME"), "go")
	}
	if strings.HasPrefix(path, filepath.Join(gopath, "bin")) {
		return "goinstall"
	}
	return "binary"
}

func selfUpdate(ctx context.Context, execPath string, info *update.UpdateInfo) error {
	fmt.Fprintf(os.Stderr, "Updating %s -> %s...\n", version.Version, info.LatestVersion)

	if info.DownloadURL == "" {
		assetName := fmt.Sprintf("gplay-%s-%s", runtime.GOOS, runtime.GOARCH)
		fmt.Fprintf(os.Stderr, "No matching asset (%s) found in the release.\n", assetName)
		fmt.Fprintf(os.Stderr, "Download the latest release manually:\n  %s\n", info.ReleaseURL)
		return nil
	}

	// Download the new binary
	tmpPath, err := update.DownloadUpdate(ctx, info)
	if err != nil {
		return fmt.Errorf("downloading update: %w", err)
	}
	defer os.Remove(tmpPath) // clean up on failure

	// Apply the update (atomic rename)
	if err := update.ApplyUpdate(tmpPath); err != nil {
		return fmt.Errorf("applying update: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Successfully updated to %s!\n", info.LatestVersion)
	return nil
}
