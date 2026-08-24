package checks

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
	checksapi "google.golang.org/api/checks/v1alpha"
	"google.golang.org/api/googleapi"

	"github.com/tamtom/play-console-cli/internal/checksclient"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

func UploadCommand(name string) *ffcli.Command {
	fs := flag.NewFlagSet("checks "+name, flag.ExitOnError)
	account := fs.String("account", "", "Checks account ID (overrides GPLAY_CHECKS_ACCOUNT/checks_account)")
	app := fs.String("app", "", "Checks app ID or resource name")
	binaryPath := fs.String("binary", "", "Path to APK, AAB, or IPA file")
	binaryType := fs.String("binary-type", "ANDROID_APK", "App binary file type: ANDROID_APK, ANDROID_AAB, IOS_IPA")
	codeRef := fs.String("code-ref", "", "Git commit hash or changelist number for the upload")
	checksFilter := fs.String("checks-filter", "", "AIP-160 filter for checks in the resulting report (e.g. state = FAILED)")
	wait := fs.Bool("wait", true, "Wait for analysis to complete and print the report")
	waitTimeout := fs.Duration("wait-timeout", 10*time.Minute, "Maximum time to wait for analysis")
	severityThreshold := fs.String("severity-threshold", "", "Fail if a failed check meets severity: PRIORITY, POTENTIAL, OPPORTUNITY")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	shortUsage := "gplay checks " + name + " --account <account-id> --app <app-id> --binary <path> [flags]"
	shortHelp := "Upload an app binary for Checks compliance analysis."
	if name == "analyze" {
		shortHelp = "Alias for `gplay checks upload`."
	}

	return &ffcli.Command{
		Name:       name,
		ShortUsage: shortUsage,
		ShortHelp:  shortHelp,
		LongHelp: `Upload an APK, AAB, or IPA to the Checks media upload endpoint and start
an asynchronous compliance analysis.

Use --severity-threshold PRIORITY in CI before release commands to fail the
pipeline when high-priority failed checks are present. The command polls
operations.get when --wait is true and prints progress to stderr.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*app) == "" {
				return shared.UsageError("--app is required")
			}
			if strings.TrimSpace(*binaryPath) == "" {
				return shared.UsageError("--binary is required")
			}
			if err := validateAppBinaryFileType(*binaryType); err != nil {
				return err
			}
			if err := validateSeverityThreshold(*severityThreshold); err != nil {
				return err
			}
			if *waitTimeout <= 0 {
				return fmt.Errorf("--wait-timeout must be positive")
			}
			resolvedAccount, err := resolveAccountForCommand(*account)
			if err != nil {
				return err
			}
			if strings.TrimSpace(resolvedAccount) == "" {
				return shared.UsageError("--account is required (or set GPLAY_CHECKS_ACCOUNT/checks_account)")
			}
			file, err := os.Open(*binaryPath)
			if err != nil {
				return fmt.Errorf("open binary: %w", err)
			}
			defer file.Close()

			service, err := checksclient.NewService(ctx)
			if err != nil {
				return err
			}

			uploadCtx, cancel := shared.ContextWithUploadTimeout(ctx, service.Cfg)
			defer cancel()

			req := &checksapi.GoogleChecksReportV1alphaAnalyzeUploadRequest{
				AppBinaryFileType: strings.TrimSpace(*binaryType),
				CodeReferenceId:   strings.TrimSpace(*codeRef),
			}
			parent := appResource(resolvedAccount, *app)
			op, err := service.API.Media.Upload(parent, req).
				Media(file, googleapi.ContentType(mediaTypeForPath(*binaryPath))).
				Context(uploadCtx).
				Do()
			if err != nil {
				return shared.WrapGoogleAPIError("upload app binary to Checks", err)
			}
			if shared.IsDryRun(ctx) || !*wait {
				return shared.PrintOutputContext(ctx, op, *outputFlag, *pretty)
			}

			report, err := pollOperation(ctx, service, op, *waitTimeout)
			if err != nil {
				return err
			}
			if filter := strings.TrimSpace(*checksFilter); filter != "" && strings.TrimSpace(report.Name) != "" {
				reqCtx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
				filtered, err := service.API.Accounts.Apps.Reports.Get(report.Name).ChecksFilter(filter).Context(reqCtx).Do()
				cancel()
				if err != nil {
					return shared.WrapGoogleAPIError("get filtered Checks report", err)
				}
				report = filtered
			}
			if reportExceedsThreshold(report, *severityThreshold) {
				if printErr := shared.PrintOutputContext(ctx, report, *outputFlag, *pretty); printErr != nil {
					return printErr
				}
				return errThresholdExceeded
			}
			return shared.PrintOutputContext(ctx, report, *outputFlag, *pretty)
		},
	}
}

func pollOperation(ctx context.Context, service *checksclient.Service, op *checksapi.Operation, timeout time.Duration) (*checksapi.GoogleChecksReportV1alphaReport, error) {
	if op == nil || strings.TrimSpace(op.Name) == "" {
		return nil, fmt.Errorf("upload did not return an operation name")
	}
	deadline := time.Now().Add(timeout)
	delay := 5 * time.Second
	for {
		if op.Done {
			if err := requireOperationDone(op); err != nil {
				return nil, err
			}
			report, err := reportFromOperation(op)
			if err != nil {
				return nil, err
			}
			if report == nil {
				return nil, fmt.Errorf("operation completed without a report response")
			}
			return report, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for Checks analysis after %s", timeout)
		}
		fmt.Fprintf(shared.Stderr(ctx), "Analyzing... (operation: %s)\n", op.Name)
		sleep := delay
		if remaining := time.Until(deadline); sleep > remaining {
			sleep = remaining
		}
		timer := time.NewTimer(sleep)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
		if delay < 30*time.Second {
			delay *= 2
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}
		}
		reqCtx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
		next, err := service.API.Accounts.Apps.Operations.Get(op.Name).Context(reqCtx).Do()
		cancel()
		if err != nil {
			return nil, shared.WrapGoogleAPIError("poll Checks analysis operation", err)
		}
		op = next
	}
}
