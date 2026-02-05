package internalsharing

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/googleapi"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

func InternalSharingCommand() *ffcli.Command {
	fs := flag.NewFlagSet("internal-sharing", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "internal-sharing",
		ShortUsage: "gplay internal-sharing <subcommand> [flags]",
		ShortHelp:  "Quick internal testing without review.",
		LongHelp: `Upload APKs or bundles for internal sharing.

Internal app sharing allows you to quickly share builds with
internal testers without going through the review process.

Shared artifacts:
  - Are only accessible to users in your organization
  - Don't require publishing to a track
  - Generate shareable URLs for direct installation`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			UploadAPKCommand(),
			UploadBundleCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			return flag.ErrHelp
		},
	}
}

func UploadAPKCommand() *ffcli.Command {
	fs := flag.NewFlagSet("internal-sharing upload-apk", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	filePath := fs.String("file", "", "Path to .apk file")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "upload-apk",
		ShortUsage: "gplay internal-sharing upload-apk --package <name> --file <path>",
		ShortHelp:  "Upload an APK for internal sharing.",
		LongHelp: `Upload an APK for internal app sharing.

After upload, you'll receive a download URL that can be shared
with internal testers for direct installation.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*filePath) == "" {
				return fmt.Errorf("--file is required")
			}
			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			file, err := os.Open(*filePath)
			if err != nil {
				return shared.WrapActionable(err, "failed to open APK file", "Check that the file exists and is readable.")
			}
			defer file.Close()

			ctx, cancel := shared.ContextWithUploadTimeout(ctx, service.Cfg)
			defer cancel()

			call := service.API.Internalappsharingartifacts.Uploadapk(pkg)
			call.Media(file, googleapi.ContentType("application/octet-stream"))
			resp, err := call.Context(ctx).Do()
			if err != nil {
				return shared.WrapGoogleAPIError("failed to upload APK for internal sharing", err)
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func UploadBundleCommand() *ffcli.Command {
	fs := flag.NewFlagSet("internal-sharing upload-bundle", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	filePath := fs.String("file", "", "Path to .aab bundle file")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "upload-bundle",
		ShortUsage: "gplay internal-sharing upload-bundle --package <name> --file <path>",
		ShortHelp:  "Upload a bundle for internal sharing.",
		LongHelp: `Upload an app bundle for internal app sharing.

After upload, you'll receive a download URL that can be shared
with internal testers for direct installation.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*filePath) == "" {
				return fmt.Errorf("--file is required")
			}
			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			file, err := os.Open(*filePath)
			if err != nil {
				return shared.WrapActionable(err, "failed to open bundle file", "Check that the file exists and is readable.")
			}
			defer file.Close()

			ctx, cancel := shared.ContextWithUploadTimeout(ctx, service.Cfg)
			defer cancel()

			call := service.API.Internalappsharingartifacts.Uploadbundle(pkg)
			call.Media(file, googleapi.ContentType("application/octet-stream"))
			resp, err := call.Context(ctx).Do()
			if err != nil {
				return shared.WrapGoogleAPIError("failed to upload bundle for internal sharing", err)
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}
