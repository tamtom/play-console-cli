package initcmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/config"
)

// InitCommand returns the init command.
func InitCommand() *ffcli.Command {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	packageName := fs.String("package", "", "Default package name (applicationId)")
	serviceAccount := fs.String("service-account", "", "Path to service account JSON file")
	force := fs.Bool("force", false, "Overwrite existing config")
	timeout := fs.String("timeout", "30s", "Default request timeout")

	return &ffcli.Command{
		Name:       "init",
		ShortUsage: "gplay init [--package <name>] [--service-account <path>] [flags]",
		ShortHelp:  "Initialize a .gplay/config.json in the current directory.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			configPath := filepath.Join(".gplay", "config.json")
			duration, err := config.ParseDurationValue(*timeout)
			if err != nil || duration.Duration <= 0 {
				return fmt.Errorf("--timeout must be a positive duration")
			}
			if !*force {
				for _, path := range []string{configPath, filepath.Join(".gplay", "config.yaml")} {
					if _, err := os.Stat(path); err == nil {
						return fmt.Errorf("config already exists at %s (use --force to overwrite)", path)
					} else if !errors.Is(err, os.ErrNotExist) {
						return fmt.Errorf("inspect existing config: %w", err)
					}
				}
			}

			// Validate service account path if provided
			if *serviceAccount != "" {
				if _, err := os.Stat(*serviceAccount); os.IsNotExist(err) {
					fmt.Fprintf(shared.Stderr(ctx), "Warning: service account file not found at %s\n", *serviceAccount)
				}
			}

			// Generate config content
			pkg := *packageName
			if pkg == "" {
				pkg = "com.example.app"
			}
			cfg := &config.Config{PackageName: pkg, Timeout: duration}
			if *serviceAccount != "" {
				cfg.DefaultProfile = "default"
				cfg.Profiles = []config.Profile{{Name: "default", Type: "service_account", KeyPath: *serviceAccount}}
			}
			if shared.IsDryRun(ctx) {
				fmt.Fprintf(shared.Stderr(ctx), "[DRY RUN] Would write %s. No changes were made.\n", configPath)
				return shared.PrintOutputContext(ctx, map[string]any{"config_path": configPath, "created": false, "dry_run": true}, "json", false)
			}
			if err := config.SaveAt(configPath, cfg); err != nil {
				return fmt.Errorf("writing config: %w", err)
			}

			result := struct {
				ConfigPath string `json:"config_path"`
				Created    bool   `json:"created"`
				Package    string `json:"package,omitempty"`
			}{
				ConfigPath: configPath,
				Created:    true,
				Package:    pkg,
			}

			if err := shared.PrintOutputContext(ctx, result, "json", false); err != nil {
				return err
			}

			fmt.Fprintln(shared.Stderr(ctx), "\nNext steps:")
			fmt.Fprintln(shared.Stderr(ctx), "  gplay auth login --service-account /path/to/key.json --local")
			fmt.Fprintln(shared.Stderr(ctx), "  gplay auth doctor")

			return nil
		},
	}
}
