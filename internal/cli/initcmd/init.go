package initcmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/output"
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
		ShortHelp:  "Initialize a .gplay/config.yaml in the current directory.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			configDir := ".gplay"
			configPath := filepath.Join(configDir, "config.yaml")

			// Check if config already exists
			if !*force {
				if _, err := os.Stat(configPath); err == nil {
					return fmt.Errorf("config already exists at %s (use --force to overwrite)", configPath)
				}
			}

			// Validate service account path if provided
			if *serviceAccount != "" {
				if _, err := os.Stat(*serviceAccount); os.IsNotExist(err) {
					fmt.Fprintf(os.Stderr, "Warning: service account file not found at %s\n", *serviceAccount)
				}
			}

			// Create directory
			if err := os.MkdirAll(configDir, 0o700); err != nil {
				return fmt.Errorf("creating config directory: %w", err)
			}

			// Generate config content
			pkg := *packageName
			if pkg == "" {
				pkg = "com.example.app"
			}
			content := generateConfig(pkg, *serviceAccount, *timeout)

			if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
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

			if err := output.PrintJSON(result); err != nil {
				return err
			}

			fmt.Fprintln(os.Stderr, "\nNext steps:")
			fmt.Fprintln(os.Stderr, "  gplay auth login --service-account /path/to/key.json --local")
			fmt.Fprintln(os.Stderr, "  gplay auth doctor")

			return nil
		},
	}
}

func generateConfig(packageName, serviceAccount, timeout string) string {
	cfg := "# gplay local configuration\n"
	cfg += "# See: gplay --help for all available options\n\n"
	cfg += fmt.Sprintf("default_package: %s\n", packageName)
	if serviceAccount != "" {
		cfg += fmt.Sprintf("service_account: %s\n", serviceAccount)
	} else {
		cfg += "# service_account: /path/to/service-account.json\n"
	}
	cfg += fmt.Sprintf("\n# timeout: %s\n", timeout)
	cfg += "# upload_timeout: 5m\n"
	cfg += "# max_retries: 3\n"
	cfg += "# debug: false\n"
	return cfg
}
