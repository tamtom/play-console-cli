package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/config"
)

const (
	androidPublisherAPI = "androidpublisher.googleapis.com"
	defaultSAName       = "play-console-cli"
	defaultProfileName  = "default"
)

// gcloudRunner is the subset of exec functionality the setup flow uses.
// It exists so tests can stub out `gcloud` calls without running the real CLI.
type gcloudRunner interface {
	LookPath(string) (string, error)
	Run(ctx context.Context, stdin []byte, name string, args ...string) (stdout []byte, err error)
}

type realRunner struct{}

func (realRunner) LookPath(name string) (string, error) { return exec.LookPath(name) }
func (realRunner) Run(ctx context.Context, stdin []byte, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// AuthSetupCommand wires `gplay auth setup [--auto]`.
func AuthSetupCommand() *ffcli.Command {
	fs := flag.NewFlagSet("auth setup", flag.ExitOnError)
	auto := fs.Bool("auto", false, "Automate GCP service-account creation using gcloud")
	project := fs.String("project", "", "GCP project ID (defaults to gcloud default)")
	saName := fs.String("sa-name", defaultSAName, "Service account name")
	profile := fs.String("profile", defaultProfileName, "gplay auth profile to create")
	keyOut := fs.String("key-out", "", "Path to write the service-account JSON (defaults to ~/.gplay/<sa>.json)")
	dryRun := fs.Bool("dry-run", false, "Print the gcloud commands without executing them")
	setDefault := fs.Bool("set-default", true, "Set as default profile in config")
	output := fs.String("output", "text", "Output format: text (default), json")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "setup",
		ShortUsage: "gplay auth setup --auto [--project <id>] [flags]",
		ShortHelp:  "Create a Google Cloud service account and link it to this CLI.",
		LongHelp: `Automated setup for Google Play authentication.

With --auto, runs these steps via gcloud:
  1. Detect/confirm GCP project
  2. Enable the androidpublisher API
  3. Create a service account (--sa-name)
  4. Download a JSON key (--key-out)
  5. Store the profile in ~/.gplay/config.json

You still need to link the service account email in Play Console afterwards;
the URL is printed at the end.

Example:
  gplay auth setup --auto --project my-gcp-project
  gplay auth setup --auto --dry-run        # preview commands
  gplay auth setup                          # open a how-to instead (no gcloud)`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			opts := SetupOptions{
				Auto:       *auto,
				Project:    strings.TrimSpace(*project),
				SAName:     strings.TrimSpace(*saName),
				Profile:    strings.TrimSpace(*profile),
				KeyOut:     strings.TrimSpace(*keyOut),
				DryRun:     *dryRun,
				SetDefault: *setDefault,
				Output:     *output,
				Pretty:     *pretty,
				Runner:     realRunner{},
				SaveConfig: saveProfileToConfig,
				HomeDir:    os.UserHomeDir,
			}
			return RunSetup(ctx, opts, os.Stdout)
		},
	}
}

// SetupOptions holds all flags for AuthSetupCommand, exposed for tests.
type SetupOptions struct {
	Auto       bool
	Project    string
	SAName     string
	Profile    string
	KeyOut     string
	DryRun     bool
	SetDefault bool
	Output     string
	Pretty     bool

	Runner     gcloudRunner
	SaveConfig func(profile config.Profile, setDefault bool) (string, error)
	HomeDir    func() (string, error)
}

// SetupResult is what `auth setup --auto` produces.
type SetupResult struct {
	Project       string   `json:"project,omitempty"`
	ServiceAcct   string   `json:"service_account_email"`
	KeyPath       string   `json:"key_path"`
	ConfigPath    string   `json:"config_path,omitempty"`
	ProfileName   string   `json:"profile"`
	PlayLinkURL   string   `json:"play_console_link_url"`
	StepsExecuted []string `json:"steps_executed"`
	DryRun        bool     `json:"dry_run,omitempty"`
}

// RunSetup performs the setup flow; stdout is where text-mode messages go.
// Test entry point.
func RunSetup(ctx context.Context, opts SetupOptions, stdout *os.File) error {
	if opts.SAName == "" {
		opts.SAName = defaultSAName
	}
	if opts.Profile == "" {
		opts.Profile = defaultProfileName
	}
	if opts.Runner == nil {
		opts.Runner = realRunner{}
	}
	if opts.HomeDir == nil {
		opts.HomeDir = os.UserHomeDir
	}

	if !opts.Auto {
		return shared.NewReportedError(fmt.Errorf(
			"manual setup: see https://developers.google.com/android-publisher/getting_started — " +
				"or re-run with --auto to automate via gcloud"))
	}

	if _, err := opts.Runner.LookPath("gcloud"); err != nil {
		return shared.NewReportedError(fmt.Errorf(
			"gcloud CLI is required for --auto; install from https://cloud.google.com/sdk"))
	}

	project := opts.Project
	if project == "" {
		out, err := opts.Runner.Run(ctx, nil, "gcloud", "config", "get-value", "project", "--quiet")
		if err != nil {
			return shared.NewReportedError(fmt.Errorf("resolve project: %w", err))
		}
		project = strings.TrimSpace(string(out))
		if project == "" || project == "(unset)" {
			return shared.NewReportedError(errors.New("no GCP project set; pass --project or run `gcloud config set project <id>`"))
		}
	}

	saEmail := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", opts.SAName, project)
	keyPath := opts.KeyOut
	if keyPath == "" {
		home, err := opts.HomeDir()
		if err != nil {
			return err
		}
		keyPath = filepath.Join(home, ".gplay", opts.SAName+".json")
	}

	linkURL := fmt.Sprintf(
		"https://play.google.com/console/u/0/developers/_/users-and-permissions?invite=%s",
		saEmail,
	)

	result := SetupResult{
		Project:     project,
		ServiceAcct: saEmail,
		KeyPath:     keyPath,
		ProfileName: opts.Profile,
		PlayLinkURL: linkURL,
		DryRun:      opts.DryRun,
	}

	steps := [][]string{
		{"gcloud", "services", "enable", androidPublisherAPI, "--project", project, "--quiet"},
		{"gcloud", "iam", "service-accounts", "describe", saEmail, "--project", project, "--quiet"},
		{
			"gcloud", "iam", "service-accounts", "create", opts.SAName,
			"--display-name", "Play Console CLI",
			"--project", project,
			"--quiet",
		},
		{
			"gcloud", "iam", "service-accounts", "keys", "create", keyPath,
			"--iam-account", saEmail,
			"--project", project,
			"--quiet",
		},
	}

	if err := os.MkdirAll(filepath.Dir(keyPath), 0o755); err != nil {
		return fmt.Errorf("prepare key output dir: %w", err)
	}

	// Enable API.
	if err := maybeRun(ctx, opts, steps[0], &result); err != nil {
		return err
	}

	// Create SA only if describe fails. In dry-run, always show create step.
	if opts.DryRun {
		result.StepsExecuted = append(result.StepsExecuted, cmdString(steps[1]), cmdString(steps[2]))
	} else {
		if _, err := opts.Runner.Run(ctx, nil, steps[1][0], steps[1][1:]...); err != nil {
			// Not found -> create it.
			if err := maybeRun(ctx, opts, steps[2], &result); err != nil {
				return err
			}
		} else {
			result.StepsExecuted = append(result.StepsExecuted, "service account already exists: "+saEmail)
		}
	}

	// Key download.
	if err := maybeRun(ctx, opts, steps[3], &result); err != nil {
		return err
	}

	if !opts.DryRun {
		if _, err := os.Stat(keyPath); err != nil {
			return fmt.Errorf("expected key at %s: %w", keyPath, err)
		}
		// Validate the key is valid JSON and looks like a service account.
		if err := validateServiceAccountKey(keyPath); err != nil {
			return err
		}

		profile := config.Profile{
			Name:    opts.Profile,
			Type:    "service_account",
			KeyPath: keyPath,
		}
		if opts.SaveConfig != nil {
			cfgPath, err := opts.SaveConfig(profile, opts.SetDefault)
			if err != nil {
				return err
			}
			result.ConfigPath = cfgPath
		}
	}

	if strings.ToLower(opts.Output) == "json" {
		return shared.PrintOutput(result, "json", opts.Pretty)
	}
	printSetupText(stdout, result)
	return nil
}

func maybeRun(ctx context.Context, opts SetupOptions, args []string, result *SetupResult) error {
	label := cmdString(args)
	if opts.DryRun {
		result.StepsExecuted = append(result.StepsExecuted, label)
		return nil
	}
	if _, err := opts.Runner.Run(ctx, nil, args[0], args[1:]...); err != nil {
		return fmt.Errorf("step failed: %s: %w", label, err)
	}
	result.StepsExecuted = append(result.StepsExecuted, label)
	return nil
}

func cmdString(argv []string) string {
	return strings.Join(argv, " ")
}

func printSetupText(w *os.File, r SetupResult) {
	if w == nil {
		w = os.Stdout
	}
	fmt.Fprintln(w, "gplay auth setup")
	fmt.Fprintln(w, "================")
	fmt.Fprintf(w, "  Project:          %s\n", r.Project)
	fmt.Fprintf(w, "  Service account:  %s\n", r.ServiceAcct)
	fmt.Fprintf(w, "  Key path:         %s\n", r.KeyPath)
	fmt.Fprintf(w, "  Profile:          %s\n", r.ProfileName)
	if r.ConfigPath != "" {
		fmt.Fprintf(w, "  Config:           %s\n", r.ConfigPath)
	}
	fmt.Fprintln(w, "\nSteps:")
	for _, s := range r.StepsExecuted {
		fmt.Fprintf(w, "  - %s\n", s)
	}
	fmt.Fprintln(w, "\nNext step (manual):")
	fmt.Fprintf(w, "  Open %s and grant the service account access in Play Console.\n", r.PlayLinkURL)
}

// validateServiceAccountKey sanity-checks that the downloaded file is a
// service-account JSON key.
func validateServiceAccountKey(path string) error {
	data, err := os.ReadFile(path) // #nosec G304 -- path just created by gcloud
	if err != nil {
		return fmt.Errorf("read key: %w", err)
	}
	var payload struct {
		Type        string `json:"type"`
		ClientEmail string `json:"client_email"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("invalid JSON key at %s: %w", path, err)
	}
	if payload.Type != "service_account" {
		return fmt.Errorf("unexpected key type %q at %s", payload.Type, path)
	}
	if payload.ClientEmail == "" {
		return fmt.Errorf("missing client_email in key at %s", path)
	}
	return nil
}

// saveProfileToConfig is the real SaveConfig hook for AuthSetupCommand.
func saveProfileToConfig(profile config.Profile, setDefault bool) (string, error) {
	cfg, err := config.Load()
	if err != nil && !errors.Is(err, config.ErrNotFound) {
		return "", err
	}
	if cfg == nil {
		cfg = &config.Config{}
	}
	cfg.Profiles = upsertProfile(cfg.Profiles, profile)
	if setDefault {
		cfg.DefaultProfile = profile.Name
	}
	path, err := config.GlobalPath()
	if err != nil {
		return "", err
	}
	if err := config.SaveAt(path, cfg); err != nil {
		return "", err
	}
	return path, nil
}
