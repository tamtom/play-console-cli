package integrity

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/playintegrity/v1"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/integrityclient"
)

var newIntegrityService = integrityclient.NewService

func Command() *ffcli.Command {
	fs := flag.NewFlagSet("integrity", flag.ExitOnError)
	return &ffcli.Command{Name: "integrity", ShortUsage: "gplay integrity <subcommand> [flags]", ShortHelp: "Decode Play Integrity tokens and manage restricted Device Recall state.", FlagSet: fs, UsageFunc: shared.DefaultUsageFunc, Subcommands: []*ffcli.Command{DecodeCommand(false), DecodeCommand(true), DeviceRecallWriteCommand()}, Exec: func(context.Context, []string) error { return flag.ErrHelp }}
}

func DecodeCommand(pc bool) *ffcli.Command {
	name := "decode"
	if pc {
		name = "decode-pc"
	}
	fs := flag.NewFlagSet("integrity "+name, flag.ExitOnError)
	pkgFlag := fs.String("package", "", "Package name (applicationId)")
	tokenFile := fs.String("token-file", "", "File containing the encoded integrity token; use - for stdin")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")
	return &ffcli.Command{
		Name: name, ShortUsage: "gplay integrity " + name + " --package <pkg> --token-file <path|->", ShortHelp: "Decode an official Play Integrity token without placing it in process arguments.", FlagSet: fs, UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*tokenFile) == "" {
				return fmt.Errorf("--token-file is required (use - for stdin); raw --token arguments are intentionally unsupported")
			}
			token, err := readToken(*tokenFile)
			if err != nil {
				return err
			}
			service, err := newIntegrityService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*pkgFlag, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()
			if pc {
				response, err := service.API.V1.DecodePcIntegrityToken(pkg, &playintegrity.DecodePcIntegrityTokenRequest{IntegrityToken: token}).Context(ctx).Do()
				if err != nil {
					return shared.WrapGoogleAPIError("decode PC integrity token", err)
				}
				return shared.PrintOutputContext(ctx, response, *outputFlag, *pretty)
			}
			response, err := service.API.V1.DecodeIntegrityToken(pkg, &playintegrity.DecodeIntegrityTokenRequest{IntegrityToken: token}).Context(ctx).Do()
			if err != nil {
				return shared.WrapGoogleAPIError("decode integrity token", err)
			}
			return shared.PrintOutputContext(ctx, response, *outputFlag, *pretty)
		},
	}
}

func readToken(path string) (string, error) {
	var reader io.Reader
	var file *os.File
	if path == "-" {
		reader = os.Stdin
	} else {
		var err error
		file, err = os.Open(path)
		if err != nil {
			return "", fmt.Errorf("open token file: %w", err)
		}
		defer file.Close()
		reader = file
	}
	data, err := io.ReadAll(io.LimitReader(reader, 4<<20))
	if err != nil {
		return "", fmt.Errorf("read token: %w", err)
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", fmt.Errorf("token file is empty")
	}
	return token, nil
}

func DeviceRecallWriteCommand() *ffcli.Command {
	fs := flag.NewFlagSet("integrity device-recall-write", flag.ExitOnError)
	pkgFlag := fs.String("package", "", "Package name (applicationId)")
	jsonArg := fs.String("json", "", "@file containing WriteDeviceRecallRequest JSON")
	restrictedUse := fs.Bool("security-fraud-abuse-use", false, "Acknowledge Device Recall is used only for security, fraud, or abuse prevention")
	confirm := fs.Bool("confirm", false, "Confirm mutation of per-device recall state")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")
	return &ffcli.Command{
		Name: "device-recall-write", ShortUsage: "gplay integrity device-recall-write --package <pkg> --json @request.json --security-fraud-abuse-use --confirm", ShortHelp: "Write restricted Device Recall bits for anti-abuse use.", FlagSet: fs, UsageFunc: shared.DefaultUsageFunc,
		LongHelp: `Write restricted Device Recall bits. Google permits this only for security, fraud, and abuse prevention.

The token-bearing request must be read from a file. JSON example: {"integrityToken":"TOKEN","newValues":{"bitFirst":true}}`,
		Exec: func(ctx context.Context, _ []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if !*restrictedUse {
				return fmt.Errorf("--security-fraud-abuse-use is required")
			}
			if !*confirm {
				return fmt.Errorf("--confirm is required")
			}
			if !strings.HasPrefix(strings.TrimSpace(*jsonArg), "@") {
				return fmt.Errorf("--json must use @file so the integrity token is not exposed in process arguments")
			}
			var req playintegrity.WriteDeviceRecallRequest
			if err := shared.LoadJSONArg(*jsonArg, &req); err != nil {
				return fmt.Errorf("invalid --json: %w", err)
			}
			if strings.TrimSpace(req.IntegrityToken) == "" || req.NewValues == nil {
				return fmt.Errorf("integrityToken and newValues are required")
			}
			service, err := newIntegrityService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*pkgFlag, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()
			response, err := service.API.DeviceRecall.Write(pkg, &req).Context(ctx).Do()
			if err != nil {
				return shared.WrapGoogleAPIError("write Device Recall state", err)
			}
			return shared.PrintOutputContext(ctx, response, *outputFlag, *pretty)
		},
	}
}
