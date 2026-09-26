package updatecmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

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
	if shared.IsDryRun(ctx) {
		fmt.Fprintln(shared.Stderr(ctx), "[DRY RUN] Would check for an update and, unless --check is set, install the verified release; no changes made.")
		return nil
	}
	// Detect installation method
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}
	method := update.DetectInstallMethod(execPath)

	// Check for latest version (force check to bypass cache)
	info, err := update.CheckForUpdate(ctx, update.Options{ForceCheck: true})
	if err != nil {
		return fmt.Errorf("checking for updates: %w", err)
	}
	if info == nil {
		return fmt.Errorf("could not determine latest version")
	}

	currentVersion := version.Version
	if !checkOnly && !force && !update.IsReleaseVersion(info.CurrentVersion) {
		return fmt.Errorf("cannot compare development version %q with release %s; use --check for details or --force to install", currentVersion, info.LatestVersion)
	}
	if !info.IsNewer && !force && !checkOnly {
		fmt.Fprintf(shared.Stderr(ctx), "Already on latest version: %s\n", currentVersion)
		return nil
	}

	if checkOnly {
		fmt.Fprintf(shared.Stderr(ctx), "Current: %s\nLatest:  %s\n", currentVersion, info.LatestVersion)
		if info.IsNewer {
			fmt.Fprintf(shared.Stderr(ctx), "Update available! Run: %s\n", update.InstallHint(execPath))
		}
		return nil
	}

	// Handle based on install method
	switch method {
	case "homebrew", "goinstall":
		fmt.Fprintf(shared.Stderr(ctx), "Update with: %s\n", update.InstallHint(execPath))
		return nil
	case "binary":
		return selfUpdate(ctx, execPath, info)
	default:
		fmt.Fprintf(shared.Stderr(ctx), "Unknown installation method. Download the latest release from:\n  %s\n", info.ReleaseURL)
		return nil
	}
}

func selfUpdate(ctx context.Context, execPath string, info *update.UpdateInfo) error {
	fmt.Fprintf(shared.Stderr(ctx), "Updating %s -> %s...\n", version.Version, info.LatestVersion)

	// Download the new binary
	tmpPath, err := update.DownloadUpdate(ctx, info)
	if err != nil {
		return fmt.Errorf("downloading update: %w", err)
	}
	defer func() { _ = os.Remove(tmpPath) }() // clean up on failure

	// Apply the update (atomic rename)
	if err := update.ApplyUpdate(tmpPath, execPath); err != nil {
		return fmt.Errorf("applying update: %w", err)
	}

	fmt.Fprintf(shared.Stderr(ctx), "Successfully updated to %s!\n", info.LatestVersion)
	return nil
}
