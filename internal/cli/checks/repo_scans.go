package checks

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	checksapi "google.golang.org/api/checks/v1alpha"

	"github.com/tamtom/play-console-cli/internal/checksclient"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

func RepoScansCommand() *ffcli.Command {
	fs := flag.NewFlagSet("checks repo-scans", flag.ExitOnError)
	return &ffcli.Command{Name: "repo-scans", ShortUsage: "gplay checks repo-scans <subcommand> [flags]", ShortHelp: "Generate and inspect official Checks repository scans.", LongHelp: "Repository scan generation sends the explicitly supplied CLI analysis and SCM metadata JSON to the Google Checks API. It does not discover or upload source files itself.", FlagSet: fs, UsageFunc: shared.DefaultUsageFunc, Subcommands: []*ffcli.Command{RepoScanGenerateCommand(), RepoScanGetCommand(), RepoScanListCommand(), RepoOperationGetCommand()}, Exec: func(context.Context, []string) error { return flag.ErrHelp }}
}

type repoFlags struct {
	account, repo, output *string
	pretty                *bool
}

func addRepoFlags(fs *flag.FlagSet) repoFlags {
	return repoFlags{
		account: fs.String("account", "", "Checks account ID"), repo: fs.String("repo", "", "Checks repository ID or resource name"),
		output: fs.String("output", "json", "Output format: json (default), table, markdown"), pretty: fs.Bool("pretty", false, "Pretty-print JSON output"),
	}
}

func (f repoFlags) validate() (string, error) {
	if err := shared.ValidateOutputFlags(*f.output, *f.pretty); err != nil {
		return "", err
	}
	if strings.TrimSpace(*f.repo) == "" {
		return "", shared.UsageError("--repo is required")
	}
	account, err := resolveAccountForCommand(*f.account)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(account) == "" {
		return "", shared.UsageError("--account is required (or set GPLAY_CHECKS_ACCOUNT/checks_account)")
	}
	return account, nil
}

func RepoScanGenerateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("checks repo-scans generate", flag.ExitOnError)
	f := addRepoFlags(fs)
	jsonArg := fs.String("json", "", "GenerateScanRequest JSON containing the explicit CLI analysis and SCM metadata (or @file)")
	confirm := fs.Bool("confirm", false, "Confirm upload of the supplied analysis and repository metadata")
	confirmManifest := fs.String("confirm-manifest", "", "Exact upload-manifest SHA-256 from a --dry-run")
	manifestFile := fs.String("manifest-file", "", "Optional path for the redacted upload manifest")
	return &ffcli.Command{
		Name: "generate", ShortUsage: "gplay checks repo-scans generate --account <id> --repo <id> --json @scan.json --confirm", ShortHelp: "Generate a Checks repository scan from explicit analysis metadata.", FlagSet: fs, UsageFunc: shared.DefaultUsageFunc,
		LongHelp: `Generate a repository scan from data produced by a local Checks-compatible analyzer.

Only the supplied JSON is sent. Source snippets are checked for binary content
and credential-shaped data first. The request must use @file so source never
enters process arguments or audit logs. Run with global --dry-run to print the
redacted upload manifest and its confirmation hash without authentication.

Example: {"cliVersion":"1.0.0","localScanPath":".","cliAnalysis":{},"scmMetadata":{}}`,
		Exec: func(ctx context.Context, _ []string) error {
			if !*confirm {
				return fmt.Errorf("--confirm is required")
			}
			if strings.TrimSpace(*jsonArg) == "" {
				return fmt.Errorf("--json is required")
			}
			if !strings.HasPrefix(strings.TrimSpace(*jsonArg), "@") {
				return shared.UsageError("--json must use @file so repository source and metadata never enter process arguments")
			}
			account, err := f.validate()
			if err != nil {
				return err
			}
			var req checksapi.GoogleChecksRepoScanV1alphaGenerateScanRequest
			if err := shared.LoadJSONArg(*jsonArg, &req); err != nil {
				return fmt.Errorf("invalid --json: %w", err)
			}
			if strings.TrimSpace(req.CliVersion) == "" || strings.TrimSpace(req.LocalScanPath) == "" || req.CliAnalysis == nil || req.ScmMetadata == nil {
				return shared.UsageError("--json requires cliVersion, localScanPath, cliAnalysis, and scmMetadata")
			}
			manifest, err := buildRepoUploadManifest(account, *f.repo, &req)
			if err != nil {
				return err
			}
			if shared.IsDryRun(ctx) {
				return shared.PrintOutputContext(ctx, manifest, *f.output, *f.pretty)
			}
			if strings.TrimSpace(*confirmManifest) != manifest.ManifestHash {
				return fmt.Errorf("--confirm-manifest must exactly match upload manifest %s; run with global --dry-run first", manifest.ManifestHash)
			}
			if strings.TrimSpace(*manifestFile) != "" {
				if err := writeRepoUploadManifest(shared.FilesystemFrom(ctx), *manifestFile, manifest); err != nil {
					return err
				}
			}
			service, err := checksclient.NewService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()
			resp, err := service.API.Accounts.Repos.Scans.Generate(repoResource(account, *f.repo), &req).Context(ctx).Do()
			if err != nil {
				return shared.WrapGoogleAPIError("generate Checks repository scan", err)
			}
			return shared.PrintOutputContext(ctx, map[string]any{"uploadManifest": manifest, "operation": resp}, *f.output, *f.pretty)
		},
	}
}

func RepoScanGetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("checks repo-scans get", flag.ExitOnError)
	f := addRepoFlags(fs)
	scan := fs.String("scan", "", "Scan ID or resource name")
	return &ffcli.Command{
		Name: "get", ShortUsage: "gplay checks repo-scans get --account <id> --repo <id> --scan <id>", ShortHelp: "Get a repository scan.", FlagSet: fs, UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			if strings.TrimSpace(*scan) == "" {
				return shared.UsageError("--scan is required")
			}
			account, err := f.validate()
			if err != nil {
				return err
			}
			service, err := checksclient.NewService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()
			resp, err := service.API.Accounts.Repos.Scans.Get(repoScanResource(account, *f.repo, *scan)).Context(ctx).Do()
			if err != nil {
				return shared.WrapGoogleAPIError("get Checks repository scan", err)
			}
			return shared.PrintOutputContext(ctx, resp, *f.output, *f.pretty)
		},
	}
}

func RepoScanListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("checks repo-scans list", flag.ExitOnError)
	f := addRepoFlags(fs)
	filter := fs.String("filter", "", "Checks API filter")
	pageSize := fs.Int("page-size", 10, "Results per page (1-50)")
	paginate := fs.Bool("paginate", false, "Fetch all pages")
	return &ffcli.Command{
		Name: "list", ShortUsage: "gplay checks repo-scans list --account <id> --repo <id> [flags]", ShortHelp: "List repository scans.", FlagSet: fs, UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			if err := validatePageSize(*pageSize); err != nil {
				return err
			}
			account, err := f.validate()
			if err != nil {
				return err
			}
			service, err := checksclient.NewService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()
			call := service.API.Accounts.Repos.Scans.List(repoResource(account, *f.repo)).PageSize(int64(*pageSize)).Context(ctx)
			if *filter != "" {
				call.Filter(*filter)
			}
			if !*paginate {
				resp, err := call.Do()
				if err != nil {
					return shared.WrapGoogleAPIError("list Checks repository scans", err)
				}
				return shared.PrintOutputContext(ctx, resp, *f.output, *f.pretty)
			}
			var scans []*checksapi.GoogleChecksRepoScanV1alphaRepoScan
			err = call.Pages(ctx, func(resp *checksapi.GoogleChecksRepoScanV1alphaListRepoScansResponse) error {
				scans = append(scans, resp.RepoScans...)
				return nil
			})
			if err != nil {
				return shared.WrapGoogleAPIError("list Checks repository scans", err)
			}
			return shared.PrintOutputContext(ctx, scans, *f.output, *f.pretty)
		},
	}
}

func RepoOperationGetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("checks repo-scans operation", flag.ExitOnError)
	f := addRepoFlags(fs)
	operation := fs.String("operation", "", "Repository operation ID or resource name")
	return &ffcli.Command{
		Name: "operation", ShortUsage: "gplay checks repo-scans operation --account <id> --repo <id> --operation <id>", ShortHelp: "Get a repository-scan long-running operation.", FlagSet: fs, UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			if strings.TrimSpace(*operation) == "" {
				return shared.UsageError("--operation is required")
			}
			account, err := f.validate()
			if err != nil {
				return err
			}
			service, err := checksclient.NewService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()
			resp, err := service.API.Accounts.Repos.Operations.Get(repoOperationResource(account, *f.repo, *operation)).Context(ctx).Do()
			if err != nil {
				return shared.WrapGoogleAPIError("get Checks repository operation", err)
			}
			return shared.PrintOutputContext(ctx, resp, *f.output, *f.pretty)
		},
	}
}
