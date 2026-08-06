package deobfuscation

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/googleapi"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

// newPlayService is the test seam for injecting a mock Play API service.
var newPlayService = playclient.NewService

// deobfuscationTypes maps a lowercased --type value to the exact casing the
// Play API expects. The type is interpolated into the request URL as the
// {deobfuscationFileType} path segment, so casing is load-bearing: sending
// "nativecode" fails with HTTP 400 "Invalid value at 'deobfuscation_file_type'".
var deobfuscationTypes = map[string]string{
	"proguard":   "proguard",
	"nativecode": "nativeCode",
}

// canonicalDeobfuscationType validates --type case-insensitively and returns
// the value in the casing the API requires.
func canonicalDeobfuscationType(raw string) (string, error) {
	apiType, ok := deobfuscationTypes[strings.ToLower(strings.TrimSpace(raw))]
	if !ok {
		return "", fmt.Errorf("--type must be 'proguard' or 'nativeCode'")
	}
	return apiType, nil
}

func DeobfuscationCommand() *ffcli.Command {
	fs := flag.NewFlagSet("deobfuscation", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "deobfuscation",
		ShortUsage: "gplay deobfuscation <subcommand> [flags]",
		ShortHelp:  "Manage deobfuscation files (ProGuard/R8 mapping files).",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			UploadCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			return flag.ErrHelp
		},
	}
}

func UploadCommand() *ffcli.Command {
	fs := flag.NewFlagSet("deobfuscation upload", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	apkVersionCode := fs.String("apk-version", "", "APK version code")
	deobfuscationType := fs.String("type", "proguard", "Deobfuscation file type: proguard (default), nativeCode")
	filePath := fs.String("file", "", "Path to mapping file (e.g., mapping.txt)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "upload",
		ShortUsage: "gplay deobfuscation upload --package <name> --edit <id> --apk-version <code> --file <path> [--type proguard|nativeCode]",
		ShortHelp:  "Upload a deobfuscation file for crash symbolication.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*editID) == "" {
				return fmt.Errorf("--edit is required")
			}
			if strings.TrimSpace(*apkVersionCode) == "" {
				return fmt.Errorf("--apk-version is required")
			}
			if strings.TrimSpace(*filePath) == "" {
				return fmt.Errorf("--file is required")
			}

			// Validate deobfuscation type. Accept any casing from the user, but
			// send the exact casing the API defines: the value becomes a URL
			// path segment and the API rejects "nativecode" with HTTP 400.
			deobType, err := canonicalDeobfuscationType(*deobfuscationType)
			if err != nil {
				return err
			}

			// Parse version code
			versionCode, err := strconv.ParseInt(*apkVersionCode, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid --apk-version: %w", err)
			}

			service, err := newPlayService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			file, err := os.Open(*filePath)
			if err != nil {
				return shared.WrapActionable(err, "failed to open deobfuscation file", "Check that the file exists and is readable.")
			}
			defer file.Close()

			ctx, cancel := shared.ContextWithUploadTimeout(ctx, service.Cfg)
			defer cancel()

			call := service.API.Edits.Deobfuscationfiles.Upload(pkg, *editID, int64(versionCode), deobType)
			call.Media(file, googleapi.ContentType("application/octet-stream"))
			resp, err := call.Context(ctx).Do()
			if err != nil {
				return shared.WrapGoogleAPIError("failed to upload deobfuscation file", err)
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}
