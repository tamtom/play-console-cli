package notify

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/config"
)

// SendCommand returns the "notify send" subcommand.
func SendCommand() *ffcli.Command {
	fs := flag.NewFlagSet("notify send", flag.ExitOnError)
	webhookURL := fs.String("webhook-url", "", "Webhook URL (required)")
	message := fs.String("message", "", "Notification message text (required)")
	format := fs.String("format", "slack", "Payload format: slack (default), discord, generic")
	eventType := fs.String("event-type", "", "Event tag (e.g., release, review, rollout)")
	packageName := fs.String("package", "", "Package name for message context")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "send",
		ShortUsage: "gplay notify send --webhook-url <url> --message <text> [flags]",
		ShortHelp:  "Send a notification to a webhook.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			return runSend(ctx, sendOpts{
				webhookURL:  *webhookURL,
				message:     *message,
				format:      *format,
				eventType:   *eventType,
				packageName: *packageName,
				outputFlag:  *outputFlag,
				pretty:      *pretty,
				client:      http.DefaultClient,
			})
		},
	}
}

type sendOpts struct {
	webhookURL  string
	message     string
	format      string
	eventType   string
	packageName string
	outputFlag  string
	pretty      bool
	client      HTTPDoer
}

func runSend(ctx context.Context, opts sendOpts) error {
	if err := shared.ValidateOutputFlags(opts.outputFlag, opts.pretty); err != nil {
		return err
	}

	if strings.TrimSpace(opts.webhookURL) == "" {
		return fmt.Errorf("--webhook-url is required")
	}
	if strings.TrimSpace(opts.message) == "" {
		return fmt.Errorf("--message is required")
	}

	if err := ValidateWebhookURL(opts.webhookURL); err != nil {
		return err
	}

	pf, err := ParseFormat(opts.format)
	if err != nil {
		return err
	}

	payload := BuildPayload(pf, opts.message, opts.eventType, opts.packageName)

	// Apply timeout from config if available.
	cfg, _ := config.Load()
	if cfg == nil {
		cfg = &config.Config{}
	}
	ctx, cancel := shared.ContextWithTimeout(ctx, cfg)
	defer cancel()

	result, err := PostWebhook(ctx, opts.client, opts.webhookURL, payload)
	if err != nil {
		if result != nil {
			result.Format = string(pf)
			return fmt.Errorf("notification failed (HTTP %d): %w", result.StatusCode, err)
		}
		return fmt.Errorf("notification failed: %w", err)
	}

	result.Format = string(pf)
	return shared.PrintOutput(result, opts.outputFlag, opts.pretty)
}
