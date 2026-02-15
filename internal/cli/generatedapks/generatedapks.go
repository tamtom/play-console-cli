package generatedapks

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

func GeneratedAPKsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("generated-apks", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "generated-apks",
		ShortUsage: "gplay generated-apks <subcommand> [flags]",
		ShortHelp:  "Download device-specific APKs generated from app bundles.",
		LongHelp: `Download device-specific APKs that Google Play generates from app bundles.

When you upload an app bundle, Google Play generates optimized APKs
for each device configuration. Use this command to download these
APKs for testing or distribution outside of the Play Store.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			ListCommand(),
			DownloadCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			return flag.ErrHelp
		},
	}
}

func ListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("generated-apks list", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	versionCode := fs.Int64("version-code", 0, "Version code of the app bundle")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay generated-apks list --package <name> --version-code <code>",
		ShortHelp:  "List generated APKs for a version.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if *versionCode == 0 {
				return fmt.Errorf("--version-code is required")
			}
			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.API.Generatedapks.List(pkg, *versionCode).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func DownloadCommand() *ffcli.Command {
	fs := flag.NewFlagSet("generated-apks download", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	versionCode := fs.Int64("version-code", 0, "Version code of the app bundle")
	downloadID := fs.String("download-id", "", "Download ID from list command")
	outputDir := fs.String("output", ".", "Output directory for downloaded APK")
	outputFlag := fs.String("format", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "download",
		ShortUsage: "gplay generated-apks download --package <name> --version-code <code> --download-id <id> --output <dir>",
		ShortHelp:  "Download a generated APK.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if *versionCode == 0 {
				return fmt.Errorf("--version-code is required")
			}
			if strings.TrimSpace(*downloadID) == "" {
				return fmt.Errorf("--download-id is required")
			}
			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			ctx, cancel := shared.ContextWithUploadTimeout(ctx, service.Cfg)
			defer cancel()

			// Get download URL
			resp, err := service.API.Generatedapks.Download(pkg, *versionCode, *downloadID).Context(ctx).Download()
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("download failed with status: %s", resp.Status)
			}

			// Create output directory
			if err := os.MkdirAll(*outputDir, 0o755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}

			// Write file
			fileName := fmt.Sprintf("%s_%d_%s.apk", pkg, *versionCode, *downloadID)
			filePath := filepath.Join(*outputDir, fileName)
			file, err := os.Create(filePath)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			defer file.Close()

			written, err := io.Copy(file, resp.Body)
			if err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}

			result := map[string]interface{}{
				"downloaded": true,
				"path":       filePath,
				"size":       written,
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}
