package verification

import (
	"context"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androiddeveloperidstatus/v1"
	"google.golang.org/api/option"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

const apiKeyEnv = "GPLAY_ANDROID_DEVELOPER_ID_API_KEY"

var (
	fingerprintPattern    = regexp.MustCompile(`^[0-9a-f]{64}$`)
	packagePattern        = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(\.[A-Za-z][A-Za-z0-9_]*)+$`)
	newDeveloperIDService = func(ctx context.Context, apiKey string) (*androiddeveloperidstatus.Service, error) {
		return androiddeveloperidstatus.NewService(ctx, option.WithAPIKey(apiKey))
	}
)

func Command() *ffcli.Command {
	fs := flag.NewFlagSet("verification", flag.ExitOnError)
	return &ffcli.Command{Name: "verification", ShortUsage: "gplay verification <subcommand> [flags]", ShortHelp: "Check official Android developer package-registration status.", FlagSet: fs, UsageFunc: shared.DefaultUsageFunc, Subcommands: []*ffcli.Command{StatusCommand()}, Exec: func(context.Context, []string) error { return flag.ErrHelp }}
}

func StatusCommand() *ffcli.Command {
	fs := flag.NewFlagSet("verification status", flag.ExitOnError)
	pkg := fs.String("package", "", "Android package name")
	fingerprint := fs.String("certificate-fingerprint", "", "Optional SHA-256 certificate fingerprint: 64 lowercase hex characters without separators")
	apiKey := fs.String("api-key", "", "Android Developer ID Status API key (or GPLAY_ANDROID_DEVELOPER_ID_API_KEY)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")
	return &ffcli.Command{
		Name: "status", ShortUsage: "gplay verification status --package <pkg> [--certificate-fingerprint <sha256>] --api-key <key>", ShortHelp: "Check whether a package/certificate is registered to a verified Android developer.", FlagSet: fs, UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			packageName := strings.TrimSpace(*pkg)
			if !packagePattern.MatchString(packageName) {
				return fmt.Errorf("--package must be a fully qualified Android package name")
			}
			key := strings.TrimSpace(*apiKey)
			if key == "" {
				key = strings.TrimSpace(os.Getenv(apiKeyEnv))
			}
			if key == "" {
				return fmt.Errorf("--api-key is required (or set %s)", apiKeyEnv)
			}
			fingerprintValue := strings.TrimSpace(*fingerprint)
			if fingerprintValue != "" && !fingerprintPattern.MatchString(fingerprintValue) {
				return fmt.Errorf("--certificate-fingerprint must be a 64-character lowercase SHA-256 hex value without separators")
			}
			service, err := newDeveloperIDService(ctx, key)
			if err != nil {
				return err
			}
			call := service.Packages.PackageRegistrationStatus.Check("packages/" + strings.ReplaceAll(packageName, ".", "-") + "/packageRegistrationStatus").Context(ctx)
			if fingerprintValue != "" {
				call.CertificateFingerprint(fingerprintValue)
			}
			response, err := call.Do()
			if err != nil {
				return shared.WrapGoogleAPIError("check Android developer package registration", err)
			}
			return shared.PrintOutput(response, *outputFlag, *pretty)
		},
	}
}
