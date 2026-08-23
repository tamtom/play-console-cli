// Package appsigning exposes the official enterprise self-hosted Cloud KMS
// Play App Signing endpoints. It deliberately excludes standard Play-managed
// signing enrollment, for which Google provides no public API.
package appsigning

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/appsigningclient"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/config"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

const androidPublisherBaseURL = "https://androidpublisher.googleapis.com/"

var newClient = func(ctx context.Context) (*appsigningclient.Client, *config.Config, error) {
	httpClient, cfg, err := playclient.NewAuthenticatedClient(ctx)
	if err != nil {
		return nil, nil, err
	}
	return appsigningclient.New(httpClient, androidPublisherBaseURL), cfg, nil
}

func Command() *ffcli.Command {
	fs := flag.NewFlagSet("app-signing", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "app-signing",
		ShortUsage: "gplay app-signing <subcommand> [flags]",
		ShortHelp:  "Manage enterprise self-hosted Cloud KMS Play App Signing.",
		LongHelp: `Manage the official Android Publisher self-hosted Cloud KMS signing APIs.

These commands are only for enterprise organizations required to retain key
custody in Google Cloud KMS. They do not automate ordinary Google-managed Play
App Signing enrollment or legal agreements.`,
		FlagSet: fs, UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{EnrollCommand(), RotateCommand()},
		Exec:        func(context.Context, []string) error { return flag.ErrHelp },
	}
}

type commandFlags struct {
	packageName    *string
	confirmPackage *string
	jsonArg        *string
	enterprise     *bool
	confirm        *bool
	output         *string
	pretty         *bool
}

func addFlags(fs *flag.FlagSet, confirmation string) commandFlags {
	return commandFlags{
		packageName:    fs.String("package", "", "Package name (applicationId)"),
		confirmPackage: fs.String("confirm-package", "", "Repeat the exact package name being enrolled or rotated"),
		jsonArg:        fs.String("json", "", "Official API request JSON (or @file)"),
		enterprise:     fs.Bool("enterprise-self-hosted-kms", false, "Acknowledge this app uses the enterprise self-hosted Cloud KMS program"),
		confirm:        fs.Bool("confirm", false, confirmation),
		output:         fs.String("output", "json", "Output format: json (default), table, markdown"),
		pretty:         fs.Bool("pretty", false, "Pretty-print JSON output"),
	}
}

func (f commandFlags) validate() error {
	if err := shared.ValidateOutputFlags(*f.output, *f.pretty); err != nil {
		return err
	}
	if !*f.enterprise {
		return fmt.Errorf("--enterprise-self-hosted-kms is required; this API is not standard Play App Signing")
	}
	packageName := strings.TrimSpace(*f.packageName)
	if packageName == "" {
		return fmt.Errorf("--package is required explicitly for enterprise signing operations")
	}
	if strings.TrimSpace(*f.confirmPackage) != packageName {
		return fmt.Errorf("--confirm-package must exactly match --package")
	}
	if strings.TrimSpace(*f.jsonArg) == "" {
		return fmt.Errorf("--json is required")
	}
	if !*f.confirm {
		return fmt.Errorf("--confirm is required")
	}
	return nil
}

func EnrollCommand() *ffcli.Command {
	fs := flag.NewFlagSet("app-signing enroll", flag.ExitOnError)
	f := addFlags(fs, "Confirm enterprise Cloud KMS signing enrollment")
	return &ffcli.Command{
		Name: "enroll", ShortUsage: "gplay app-signing enroll --package <pkg> --confirm-package <pkg> --json @request.json --enterprise-self-hosted-kms --confirm", ShortHelp: "Enroll an app with an enterprise self-hosted Cloud KMS key.", FlagSet: fs, UsageFunc: shared.DefaultUsageFunc,
		LongHelp: `Enroll an enterprise app using a self-hosted Cloud KMS key.

Existing-app JSON example: {"enrollExistingApp":{"cloudKmsKey":{"cryptoKeyVersionResource":"projects/p/locations/l/keyRings/r/cryptoKeys/k/cryptoKeyVersions/1"}}}`,
		Exec: func(ctx context.Context, _ []string) error {
			if err := f.validate(); err != nil {
				return err
			}
			var req appsigningclient.EnrollAppRequest
			if err := shared.LoadJSONArg(*f.jsonArg, &req); err != nil {
				return fmt.Errorf("invalid --json: %w", err)
			}
			if err := validateEnrollRequest(&req); err != nil {
				return err
			}
			client, cfg, err := newClient(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*f.packageName, cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, cfg)
			defer cancel()
			response, err := client.EnrollApp(ctx, pkg, &req)
			if err != nil {
				return err
			}
			return shared.PrintOutput(response, *f.output, *f.pretty)
		},
	}
}

func validateEnrollRequest(req *appsigningclient.EnrollAppRequest) error {
	if (req.EnrollNewApp == nil) == (req.EnrollExistingApp == nil) {
		return fmt.Errorf("exactly one of enrollNewApp or enrollExistingApp is required")
	}
	if req.EnrollNewApp != nil {
		keyAndCert := req.EnrollNewApp.CloudKMSKeyAndCert
		if keyAndCert == nil || keyAndCert.CloudKMSKey == nil || strings.TrimSpace(keyAndCert.CloudKMSKey.CryptoKeyVersionResource) == "" || strings.TrimSpace(keyAndCert.PEMCertificate) == "" {
			return fmt.Errorf("enrollNewApp requires cloudKmsKeyAndCert with cryptoKeyVersionResource and pemCertificate")
		}
	}
	if req.EnrollExistingApp != nil && (req.EnrollExistingApp.CloudKMSKey == nil || strings.TrimSpace(req.EnrollExistingApp.CloudKMSKey.CryptoKeyVersionResource) == "") {
		return fmt.Errorf("enrollExistingApp requires cloudKmsKey.cryptoKeyVersionResource")
	}
	return nil
}

func RotateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("app-signing rotate-key", flag.ExitOnError)
	f := addFlags(fs, "Confirm irreversible enterprise Cloud KMS key rotation")
	return &ffcli.Command{
		Name: "rotate-key", ShortUsage: "gplay app-signing rotate-key --package <pkg> --confirm-package <pkg> --json @request.json --enterprise-self-hosted-kms --confirm", ShortHelp: "Rotate an enterprise self-hosted Cloud KMS signing key.", FlagSet: fs, UsageFunc: shared.DefaultUsageFunc,
		LongHelp: `Rotate an enterprise self-hosted Cloud KMS app-signing key.

JSON example: {"keyRotationReason":"ROUTINE_KEY_UPGRADE","rotatedCloudKmsKey":{"cloudKmsKeyAndCert":{"cloudKmsKey":{"cryptoKeyVersionResource":"projects/p/locations/l/keyRings/r/cryptoKeys/k/cryptoKeyVersions/2"},"pemCertificate":"BASE64_PEM"},"signingCertificateLineage":"BASE64_LINEAGE"}}`,
		Exec: func(ctx context.Context, _ []string) error {
			if err := f.validate(); err != nil {
				return err
			}
			var req appsigningclient.RotateAppSigningKeyRequest
			if err := shared.LoadJSONArg(*f.jsonArg, &req); err != nil {
				return fmt.Errorf("invalid --json: %w", err)
			}
			if err := validateRotateRequest(&req); err != nil {
				return err
			}
			client, cfg, err := newClient(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*f.packageName, cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, cfg)
			defer cancel()
			response, err := client.RotateAppSigningKey(ctx, pkg, &req)
			if err != nil {
				return err
			}
			return shared.PrintOutput(response, *f.output, *f.pretty)
		},
	}
}

func validateRotateRequest(req *appsigningclient.RotateAppSigningKeyRequest) error {
	validReasons := map[string]bool{"COMPROMISED_KEY": true, "USE_STRONGER_KEY": true, "USE_SAME_KEY_FOR_MULTIPLE_APPS": true, "ROUTINE_KEY_UPGRADE": true, "OTHER": true}
	if !validReasons[req.KeyRotationReason] {
		return fmt.Errorf("keyRotationReason must be a documented non-unspecified value")
	}
	rotated := req.RotatedCloudKMSKey
	if rotated == nil || rotated.CloudKMSKeyAndCert == nil || rotated.CloudKMSKeyAndCert.CloudKMSKey == nil || strings.TrimSpace(rotated.CloudKMSKeyAndCert.CloudKMSKey.CryptoKeyVersionResource) == "" || strings.TrimSpace(rotated.CloudKMSKeyAndCert.PEMCertificate) == "" || strings.TrimSpace(rotated.SigningCertificateLineage) == "" {
		return fmt.Errorf("rotatedCloudKmsKey requires key resource, PEM certificate, and signingCertificateLineage")
	}
	return nil
}
