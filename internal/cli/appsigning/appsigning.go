// Package appsigning exposes the official enterprise self-hosted Cloud KMS
// Play App Signing endpoints behind a sealed plan/apply/receipt contract. It
// deliberately excludes ordinary Google-managed Play App Signing enrollment.
package appsigning

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"regexp"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/appsigningclient"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

var signingPackagePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(\.[A-Za-z][A-Za-z0-9_]*)+$`)

// AppSigningCommand returns the enterprise App Signing transaction surface.
func AppSigningCommand() *ffcli.Command {
	fs := flag.NewFlagSet("app-signing", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "app-signing",
		ShortUsage: "gplay app-signing <plan-enroll|plan-rotation|apply> [flags]",
		ShortHelp:  "Plan and apply enterprise self-hosted Cloud KMS Play App Signing operations.",
		LongHelp: `These commands are only for enterprise organizations required to retain key
custody in Google Cloud KMS. They do not automate ordinary Google-managed Play
App Signing enrollment or legal agreements.

Planning is offline. Apply requires the exact SHA-256 plan ID and writes a
sealed receipt before the irreversible request. Because Google exposes no
readback endpoint for these operations, an ambiguous receipt is never replayed.`,
		FlagSet:     fs,
		UsageFunc:   shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{EnrollPlanCommand(), RotatePlanCommand(), ApplyCommand()},
		Exec:        func(context.Context, []string) error { return flag.ErrHelp },
	}
}

type planFlags struct {
	packageName *string
	jsonArg     *string
	planFile    *string
	enterprise  *bool
	output      *string
	pretty      *bool
}

func addPlanFlags(fs *flag.FlagSet) planFlags {
	return planFlags{
		packageName: fs.String("package", "", "Package name (applicationId)"),
		jsonArg:     fs.String("json", "", "Official API request JSON (or @file)"),
		planFile:    fs.String("plan-file", "", "Path for the sealed operation plan"),
		enterprise:  fs.Bool("enterprise-self-hosted-kms", false, "Acknowledge this app uses the enterprise self-hosted Cloud KMS program"),
		output:      fs.String("output", "json", "Output format: json, table, markdown"),
		pretty:      fs.Bool("pretty", false, "Pretty-print JSON output"),
	}
}

func (f planFlags) validate() error {
	if err := shared.ValidateOutputFlags(*f.output, *f.pretty); err != nil {
		return err
	}
	if !*f.enterprise {
		return shared.UsageError("--enterprise-self-hosted-kms is required; this API is not standard Play App Signing")
	}
	if !signingPackagePattern.MatchString(strings.TrimSpace(*f.packageName)) {
		return shared.UsageError("--package must be an explicit valid applicationId")
	}
	if strings.TrimSpace(*f.jsonArg) == "" {
		return shared.UsageError("--json is required")
	}
	if strings.TrimSpace(*f.planFile) == "" {
		return shared.UsageError("--plan-file is required")
	}
	return nil
}

// EnrollPlanCommand creates an offline enrollment plan.
func EnrollPlanCommand() *ffcli.Command {
	fs := flag.NewFlagSet("app-signing plan-enroll", flag.ExitOnError)
	f := addPlanFlags(fs)
	return &ffcli.Command{
		Name:       "plan-enroll",
		ShortUsage: "gplay app-signing plan-enroll --package <pkg> --json @request.json --plan-file <path> --enterprise-self-hosted-kms",
		ShortHelp:  "Create a sealed enterprise Cloud KMS enrollment plan offline.",
		LongHelp:   `Existing-app JSON example: {"enrollExistingApp":{"cloudKmsKey":{"cryptoKeyVersionResource":"projects/p/locations/l/keyRings/r/cryptoKeys/k/cryptoKeyVersions/1"}}}`,
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 0 {
				return shared.UsageErrorf("unexpected arguments: %s", strings.Join(args, " "))
			}
			if err := f.validate(); err != nil {
				return err
			}
			var request appsigningclient.EnrollAppRequest
			if err := loadSigningRequest(ctx, *f.jsonArg, &request); err != nil {
				return fmt.Errorf("invalid --json: %w", err)
			}
			if err := validateEnrollRequest(&request); err != nil {
				return err
			}
			plan, err := createSigningPlan(ctx, operationEnroll, strings.TrimSpace(*f.packageName), request)
			if err != nil {
				return err
			}
			if err := writeSigningPlan(shared.FilesystemFrom(ctx), *f.planFile, plan); err != nil {
				return err
			}
			return shared.PrintOutputContext(ctx, plan, *f.output, *f.pretty)
		},
	}
}

// RotatePlanCommand creates an offline signing-key rotation plan.
func RotatePlanCommand() *ffcli.Command {
	fs := flag.NewFlagSet("app-signing plan-rotation", flag.ExitOnError)
	f := addPlanFlags(fs)
	return &ffcli.Command{
		Name:       "plan-rotation",
		ShortUsage: "gplay app-signing plan-rotation --package <pkg> --json @request.json --plan-file <path> --enterprise-self-hosted-kms",
		ShortHelp:  "Create a sealed enterprise Cloud KMS key-rotation plan offline.",
		LongHelp:   `JSON example: {"keyRotationReason":"ROUTINE_KEY_UPGRADE","rotatedCloudKmsKey":{"cloudKmsKeyAndCert":{"cloudKmsKey":{"cryptoKeyVersionResource":"projects/p/locations/l/keyRings/r/cryptoKeys/k/cryptoKeyVersions/2"},"pemCertificate":"BASE64_PEM"},"signingCertificateLineage":"BASE64_LINEAGE"}}`,
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 0 {
				return shared.UsageErrorf("unexpected arguments: %s", strings.Join(args, " "))
			}
			if err := f.validate(); err != nil {
				return err
			}
			var request appsigningclient.RotateAppSigningKeyRequest
			if err := loadSigningRequest(ctx, *f.jsonArg, &request); err != nil {
				return fmt.Errorf("invalid --json: %w", err)
			}
			if err := validateRotateRequest(&request); err != nil {
				return err
			}
			plan, err := createSigningPlan(ctx, operationRotate, strings.TrimSpace(*f.packageName), request)
			if err != nil {
				return err
			}
			if err := writeSigningPlan(shared.FilesystemFrom(ctx), *f.planFile, plan); err != nil {
				return err
			}
			return shared.PrintOutputContext(ctx, plan, *f.output, *f.pretty)
		},
	}
}

func loadSigningRequest(ctx context.Context, input string, output any) error {
	trimmed := strings.TrimSpace(input)
	var data []byte
	var err error
	if strings.HasPrefix(trimmed, "@") {
		path := strings.TrimSpace(strings.TrimPrefix(trimmed, "@"))
		if path == "" {
			return fmt.Errorf("invalid @file path")
		}
		data, err = shared.FilesystemFrom(ctx).ReadFile(path)
		if err != nil {
			return fmt.Errorf("read request file: %w", err)
		}
	} else {
		data = []byte(trimmed)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return fmt.Errorf("decode request: %w", err)
	}
	return nil
}

func validateEnrollRequest(request *appsigningclient.EnrollAppRequest) error {
	if (request.EnrollNewApp == nil) == (request.EnrollExistingApp == nil) {
		return shared.UsageError("exactly one of enrollNewApp or enrollExistingApp is required")
	}
	if request.EnrollNewApp != nil {
		keyAndCert := request.EnrollNewApp.CloudKMSKeyAndCert
		if keyAndCert == nil || keyAndCert.CloudKMSKey == nil || strings.TrimSpace(keyAndCert.CloudKMSKey.CryptoKeyVersionResource) == "" || strings.TrimSpace(keyAndCert.PEMCertificate) == "" {
			return shared.UsageError("enrollNewApp requires cloudKmsKeyAndCert with cryptoKeyVersionResource and pemCertificate")
		}
	}
	if request.EnrollExistingApp != nil && (request.EnrollExistingApp.CloudKMSKey == nil || strings.TrimSpace(request.EnrollExistingApp.CloudKMSKey.CryptoKeyVersionResource) == "") {
		return shared.UsageError("enrollExistingApp requires cloudKmsKey.cryptoKeyVersionResource")
	}
	return nil
}

func validateRotateRequest(request *appsigningclient.RotateAppSigningKeyRequest) error {
	validReasons := map[string]bool{"COMPROMISED_KEY": true, "USE_STRONGER_KEY": true, "USE_SAME_KEY_FOR_MULTIPLE_APPS": true, "ROUTINE_KEY_UPGRADE": true, "OTHER": true}
	if !validReasons[request.KeyRotationReason] {
		return shared.UsageError("keyRotationReason must be a documented non-unspecified value")
	}
	rotated := request.RotatedCloudKMSKey
	if rotated == nil || rotated.CloudKMSKeyAndCert == nil || rotated.CloudKMSKeyAndCert.CloudKMSKey == nil || strings.TrimSpace(rotated.CloudKMSKeyAndCert.CloudKMSKey.CryptoKeyVersionResource) == "" || strings.TrimSpace(rotated.CloudKMSKeyAndCert.PEMCertificate) == "" || strings.TrimSpace(rotated.SigningCertificateLineage) == "" {
		return shared.UsageError("rotatedCloudKmsKey requires key resource, PEM certificate, and signingCertificateLineage")
	}
	return nil
}
