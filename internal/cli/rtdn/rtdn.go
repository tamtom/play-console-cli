// Package rtdn wires the `gplay rtdn` command family for Real-Time Developer
// Notifications (Pub/Sub + Play Billing).
package rtdn

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	rtdnpkg "github.com/tamtom/play-console-cli/internal/rtdn"
)

// playPublisherPrincipal is the Google-owned service account that must have
// Pub/Sub Publisher role on the topic for RTDN to work.
const playPublisherPrincipal = "google-play-developer-notifications@system.gserviceaccount.com"

// gcloudRunner is the minimal interface the setup/status commands need.
type gcloudRunner interface {
	LookPath(string) (string, error)
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type realRunner struct{}

func (realRunner) LookPath(name string) (string, error) { return exec.LookPath(name) }
func (realRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

// RtdnCommand builds the root `gplay rtdn`.
func RtdnCommand() *ffcli.Command {
	fs := flag.NewFlagSet("rtdn", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "rtdn",
		ShortUsage: "gplay rtdn <subcommand> [flags]",
		ShortHelp:  "Real-Time Developer Notifications: setup, status, decode.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			setupCommand(),
			statusCommand(),
			decodeCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", args[0])
			return flag.ErrHelp
		},
	}
}

// setupCommand creates the Pub/Sub topic and grants Google Play publisher access.
func setupCommand() *ffcli.Command {
	fs := flag.NewFlagSet("rtdn setup", flag.ExitOnError)
	project := fs.String("project", "", "GCP project ID (required)")
	topic := fs.String("topic", "play-rtdn", "Pub/Sub topic name")
	dryRun := fs.Bool("dry-run", false, "Print gcloud commands without executing")
	outputFlag := fs.String("output", "text", "Output format: text (default), json")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "setup",
		ShortUsage: "gplay rtdn setup --project <id> [--topic <name>] [--dry-run]",
		ShortHelp:  "Create the Pub/Sub topic and grant Play publisher access.",
		LongHelp: `Create a Pub/Sub topic and grant Google Play's system service account
the Pub/Sub Publisher role so it can push notifications.

You still need to paste the topic name (projects/<id>/topics/<topic>) into the
Play Console -> Monetization setup page afterwards. The topic name is printed
at the end.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			return runSetup(ctx, SetupOptions{
				Project: strings.TrimSpace(*project),
				Topic:   strings.TrimSpace(*topic),
				DryRun:  *dryRun,
				Output:  *outputFlag,
				Pretty:  *pretty,
				Runner:  realRunner{},
				Stdout:  os.Stdout,
			})
		},
	}
}

// SetupOptions is exposed for tests.
type SetupOptions struct {
	Project string
	Topic   string
	DryRun  bool
	Output  string
	Pretty  bool
	Runner  gcloudRunner
	Stdout  io.Writer
}

// SetupResult is the JSON payload.
type SetupResult struct {
	Project       string   `json:"project"`
	Topic         string   `json:"topic"`
	TopicResource string   `json:"topic_resource"`
	DryRun        bool     `json:"dry_run,omitempty"`
	StepsExecuted []string `json:"steps_executed"`
	NextStepURL   string   `json:"next_step_url"`
}

func runSetup(ctx context.Context, opts SetupOptions) error {
	if opts.Project == "" {
		return errors.New("--project is required")
	}
	if opts.Topic == "" {
		opts.Topic = "play-rtdn"
	}
	if opts.Runner == nil {
		opts.Runner = realRunner{}
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if _, err := opts.Runner.LookPath("gcloud"); err != nil {
		return shared.NewReportedError(errors.New("gcloud CLI is required for rtdn setup"))
	}

	topicResource := fmt.Sprintf("projects/%s/topics/%s", opts.Project, opts.Topic)
	result := SetupResult{
		Project:       opts.Project,
		Topic:         opts.Topic,
		TopicResource: topicResource,
		DryRun:        opts.DryRun,
		NextStepURL:   "https://play.google.com/console -> Monetization setup",
	}

	steps := [][]string{
		{"pubsub", "topics", "create", opts.Topic, "--project", opts.Project, "--quiet"},
		{
			"pubsub", "topics", "add-iam-policy-binding", opts.Topic,
			"--member", "serviceAccount:" + playPublisherPrincipal,
			"--role", "roles/pubsub.publisher",
			"--project", opts.Project,
			"--quiet",
		},
	}

	for _, s := range steps {
		cmdLine := "gcloud " + strings.Join(s, " ")
		if opts.DryRun {
			result.StepsExecuted = append(result.StepsExecuted, cmdLine)
			continue
		}
		if _, err := opts.Runner.Run(ctx, "gcloud", s...); err != nil {
			// topics create is idempotent-ish: if it already exists, continue.
			if !strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("step failed: %s: %w", cmdLine, err)
			}
		}
		result.StepsExecuted = append(result.StepsExecuted, cmdLine)
	}

	if strings.ToLower(opts.Output) == "json" {
		return shared.PrintOutput(result, "json", opts.Pretty)
	}
	fmt.Fprintln(opts.Stdout, "gplay rtdn setup")
	fmt.Fprintln(opts.Stdout, "================")
	fmt.Fprintf(opts.Stdout, "  Project: %s\n", result.Project)
	fmt.Fprintf(opts.Stdout, "  Topic:   %s\n", result.TopicResource)
	fmt.Fprintln(opts.Stdout, "\nSteps:")
	for _, s := range result.StepsExecuted {
		fmt.Fprintf(opts.Stdout, "  - %s\n", s)
	}
	fmt.Fprintf(opts.Stdout, "\nNext: paste %s into %s\n", result.TopicResource, result.NextStepURL)
	return nil
}

// statusCommand prints the current IAM policy of the topic.
func statusCommand() *ffcli.Command {
	fs := flag.NewFlagSet("rtdn status", flag.ExitOnError)
	project := fs.String("project", "", "GCP project ID (required)")
	topic := fs.String("topic", "play-rtdn", "Pub/Sub topic name")
	outputFlag := fs.String("output", "text", "Output format: text (default), json")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "status",
		ShortUsage: "gplay rtdn status --project <id> [--topic <name>]",
		ShortHelp:  "Show Pub/Sub topic configuration and Play publisher binding.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*project) == "" {
				return errors.New("--project is required")
			}
			return runStatus(ctx, StatusOptions{
				Project: *project,
				Topic:   *topic,
				Output:  *outputFlag,
				Pretty:  *pretty,
				Runner:  realRunner{},
				Stdout:  os.Stdout,
			})
		},
	}
}

// StatusOptions is exposed for tests.
type StatusOptions struct {
	Project string
	Topic   string
	Output  string
	Pretty  bool
	Runner  gcloudRunner
	Stdout  io.Writer
}

// StatusResult is the JSON shape.
type StatusResult struct {
	Project        string `json:"project"`
	TopicResource  string `json:"topic_resource"`
	PublisherBound bool   `json:"publisher_bound"`
	RawPolicy      string `json:"raw_policy,omitempty"`
}

func runStatus(ctx context.Context, opts StatusOptions) error {
	if opts.Runner == nil {
		opts.Runner = realRunner{}
	}
	if _, err := opts.Runner.LookPath("gcloud"); err != nil {
		return shared.NewReportedError(errors.New("gcloud CLI is required"))
	}
	out, err := opts.Runner.Run(ctx, "gcloud", "pubsub", "topics", "get-iam-policy",
		opts.Topic, "--project", opts.Project, "--format", "json", "--quiet")
	if err != nil {
		return fmt.Errorf("get-iam-policy: %w", err)
	}

	bound := strings.Contains(string(out), playPublisherPrincipal)
	result := StatusResult{
		Project:        opts.Project,
		TopicResource:  fmt.Sprintf("projects/%s/topics/%s", opts.Project, opts.Topic),
		PublisherBound: bound,
		RawPolicy:      strings.TrimSpace(string(out)),
	}
	if strings.ToLower(opts.Output) == "json" {
		return shared.PrintOutput(result, "json", opts.Pretty)
	}
	fmt.Fprintln(opts.Stdout, "gplay rtdn status")
	fmt.Fprintf(opts.Stdout, "  Project: %s\n", result.Project)
	fmt.Fprintf(opts.Stdout, "  Topic:   %s\n", result.TopicResource)
	if bound {
		fmt.Fprintln(opts.Stdout, "  [OK] Play publisher role bound")
	} else {
		fmt.Fprintln(opts.Stdout, "  [WARN] Play publisher role NOT bound; run `gplay rtdn setup`")
	}
	return nil
}

// decodeCommand parses a Pub/Sub envelope (or inner payload) from --file, --data, or stdin.
func decodeCommand() *ffcli.Command {
	fs := flag.NewFlagSet("rtdn decode", flag.ExitOnError)
	file := fs.String("file", "", "Path to payload JSON file ('-' for stdin)")
	data := fs.String("data", "", "Inline payload JSON (overrides --file)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "decode",
		ShortUsage: "gplay rtdn decode --file <payload.json>",
		ShortHelp:  "Decode a Pub/Sub RTDN payload into a typed notification.",
		LongHelp: `Decode a Pub/Sub RTDN payload into a typed notification.

Accepts either a full Pub/Sub envelope (message.data is base64) or the raw
inner JSON (packageName/subscriptionNotification/etc.).

Examples:
  gplay rtdn decode --file payload.json
  cat payload.json | gplay rtdn decode --file -
  gplay rtdn decode --data '{"message":{"data":"..."}}'
`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			raw, err := loadPayload(*file, *data, os.Stdin)
			if err != nil {
				return err
			}
			decoded, err := rtdnpkg.Decode(raw)
			if err != nil {
				return err
			}
			return shared.PrintOutput(decoded, *outputFlag, *pretty)
		},
	}
}

func loadPayload(file, inline string, stdin io.Reader) ([]byte, error) {
	if strings.TrimSpace(inline) != "" {
		return []byte(inline), nil
	}
	if strings.TrimSpace(file) == "" {
		return nil, errors.New("one of --file or --data is required")
	}
	if file == "-" {
		return io.ReadAll(stdin)
	}
	return os.ReadFile(file) // #nosec G304 -- user-supplied path
}
