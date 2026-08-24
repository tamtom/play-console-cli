package checks

import (
	"context"
	"flag"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	checksapi "google.golang.org/api/checks/v1alpha"

	"github.com/tamtom/play-console-cli/internal/checksclient"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

func ContentCommand() *ffcli.Command {
	fs := flag.NewFlagSet("checks content", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "content",
		ShortUsage: "gplay checks content <subcommand> [flags]",
		ShortHelp:  "Classify app content with Checks AI safety policies.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			ClassifyCommand("classify", "gplay checks content classify --text <content> [flags]"),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

func ClassifyCommand(name string, usageOverride ...string) *ffcli.Command {
	fs := flag.NewFlagSet("checks "+name, flag.ExitOnError)
	text := fs.String("text", "", "Text content to classify, or @file")
	language := fs.String("language", "", "ISO 639-1 language code")
	policies := fs.String("policies", "all", "Comma-separated policies, or all")
	contextPrompt := fs.String("context", "", "Prompt or context that generated the content")
	classifierVersion := fs.String("classifier-version", "", "Classifier version: STABLE or LATEST")
	severityThreshold := fs.Bool("severity-threshold", false, "Fail if any policy result is VIOLATIVE")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	usage := "gplay checks content classify --text <content> [flags]"
	if name == "classify" {
		usage = "gplay checks classify --text <content> [flags]"
	}
	if len(usageOverride) > 0 && strings.TrimSpace(usageOverride[0]) != "" {
		usage = usageOverride[0]
	}

	return &ffcli.Command{
		Name:       name,
		ShortUsage: usage,
		ShortHelp:  "Classify text content against Checks AI safety policies.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*text) == "" {
				return shared.UsageError("--text is required")
			}
			if v := strings.ToUpper(strings.TrimSpace(*classifierVersion)); v != "" && v != "STABLE" && v != "LATEST" {
				return shared.UsageError("--classifier-version must be STABLE or LATEST")
			}
			policyConfigs, err := parsePolicies(*policies)
			if err != nil {
				return err
			}
			inputText, err := readTextArg(*text)
			if err != nil {
				return err
			}
			service, err := checksclient.NewService(ctx)
			if err != nil {
				return err
			}
			reqCtx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			req := &checksapi.GoogleChecksAisafetyV1alphaClassifyContentRequest{
				Input: &checksapi.GoogleChecksAisafetyV1alphaClassifyContentRequestInputContent{
					TextInput: &checksapi.GoogleChecksAisafetyV1alphaTextInput{
						Content:      inputText,
						LanguageCode: strings.TrimSpace(*language),
					},
				},
				Policies:          policyConfigs,
				ClassifierVersion: strings.ToUpper(strings.TrimSpace(*classifierVersion)),
			}
			if strings.TrimSpace(*contextPrompt) != "" {
				req.Context = &checksapi.GoogleChecksAisafetyV1alphaClassifyContentRequestContext{
					Prompt: strings.TrimSpace(*contextPrompt),
				}
			}
			resp, err := service.API.Aisafety.ClassifyContent(req).Context(reqCtx).Do()
			if err != nil {
				return shared.WrapGoogleAPIError("classify content with Checks", err)
			}
			if *severityThreshold && classificationViolates(resp) {
				if printErr := shared.PrintOutputContext(ctx, resp, *outputFlag, *pretty); printErr != nil {
					return printErr
				}
				return errThresholdExceeded
			}
			return shared.PrintOutputContext(ctx, resp, *outputFlag, *pretty)
		},
	}
}
