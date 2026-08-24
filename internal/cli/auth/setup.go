package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/config"
	"github.com/tamtom/play-console-cli/internal/rootfs"
)

const (
	androidPublisherAPI = "androidpublisher.googleapis.com"
	defaultSAName       = "play-console-cli"
	defaultProfileName  = "default"
)

// gcloudRunner is the subset of host functionality the setup flow uses.
// It exists so tests can stub out `gcloud`, the installer, the browser, and
// the clipboard without touching the real system.
type gcloudRunner interface {
	LookPath(string) (string, error)
	// Run executes a command and captures its stdout.
	Run(ctx context.Context, stdin []byte, name string, args ...string) (stdout []byte, err error)
	// RunInteractive executes a command wired to the user's stdio (for
	// browser-based flows like `gcloud auth login`).
	RunInteractive(ctx context.Context, name string, args ...string) error
	// InstallGcloud installs the gcloud CLI for the current platform.
	InstallGcloud(ctx context.Context) error
	// OpenBrowser opens url in the default browser (best-effort).
	OpenBrowser(url string) error
	// Copy places text on the system clipboard (best-effort).
	Copy(text string) error
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

func (realRunner) RunInteractive(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = shared.Stdout(ctx)
	cmd.Stderr = shared.Stderr(ctx)
	return cmd.Run()
}

func (realRunner) InstallGcloud(ctx context.Context) error {
	switch runtime.GOOS {
	case "darwin":
		if _, err := exec.LookPath("brew"); err == nil {
			return runInherit(ctx, "brew", "install", "--cask", "google-cloud-sdk")
		}
		return runInherit(ctx, "bash", "-c", "curl -fsSL https://sdk.cloud.google.com | bash -s -- --disable-prompts")
	case "linux":
		return runInherit(ctx, "bash", "-c", "curl -fsSL https://sdk.cloud.google.com | bash -s -- --disable-prompts")
	default:
		return fmt.Errorf("automatic gcloud install is not supported on %s", runtime.GOOS)
	}
}

func (realRunner) OpenBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return fmt.Errorf("unsupported platform")
	}
}

func (realRunner) Copy(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard")
	case "windows":
		cmd = exec.Command("clip")
	default:
		return fmt.Errorf("unsupported platform")
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func runInherit(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = shared.Stdout(ctx)
	cmd.Stderr = shared.Stderr(ctx)
	return cmd.Run()
}

// SetupCommand wires the top-level `gplay setup [--auto]`.
func SetupCommand() *ffcli.Command {
	return newSetupCommand("setup", "gplay setup --auto [flags]")
}

// AuthSetupCommand wires `gplay auth setup [--auto]`.
func AuthSetupCommand() *ffcli.Command {
	return newSetupCommand("auth setup", "gplay auth setup --auto [--project <id>] [flags]")
}

func newSetupCommand(flagSetName, shortUsage string) *ffcli.Command {
	fs := flag.NewFlagSet(flagSetName, flag.ExitOnError)
	auto := fs.Bool("auto", false, "Automate GCP service-account creation using gcloud")
	project := fs.String("project", "", "GCP project ID (defaults to gcloud default)")
	saName := fs.String("sa-name", defaultSAName, "Service account name")
	profile := fs.String("profile", defaultProfileName, "gplay auth profile to create")
	keyOut := fs.String("key-out", "", "Path to write the service-account JSON (defaults to ~/.gplay/<sa>.json)")
	dryRun := fs.Bool("dry-run", false, "Print the gcloud commands without executing them")
	setDefault := fs.Bool("set-default", true, "Set as default profile in config")
	noBrowser := fs.Bool("no-browser", false, "Do not open a browser (for login or the Play Console grant step)")
	noInstall := fs.Bool("no-install", false, "Do not auto-install gcloud when it is missing")
	output := fs.String("output", "text", "Output format: text (default), json")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "setup",
		ShortUsage: shortUsage,
		ShortHelp:  "Set up Google Play authentication end-to-end.",
		LongHelp: `One-command setup for Google Play authentication.

With --auto, gplay drives the whole flow via gcloud:
  1. Install the gcloud CLI if it is missing (Homebrew on macOS, curl on Linux)
  2. Log you into Google Cloud (gcloud auth login) if needed
  3. Enable the androidpublisher API
  4. Create a service account (--sa-name) and download a JSON key
  5. Store the profile in ~/.gplay/config.json
  6. Open Play Console for the one manual step: granting the account access

Example:
  gplay setup --auto                        # full automated setup
  gplay setup --auto --project my-gcp-project
  gplay setup --auto --dry-run              # preview commands
  gplay setup --auto --no-browser           # CI/agent friendly (no browser)
  gplay setup                                # print manual instructions`,
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
				NoBrowser:  *noBrowser,
				NoInstall:  *noInstall,
				Output:     *output,
				Pretty:     *pretty,
				Runner:     realRunner{},
				SaveConfig: saveProfileToConfig,
				HomeDir:    os.UserHomeDir,
			}
			return RunSetup(ctx, opts, shared.Stdout(ctx))
		},
	}
}

// SetupOptions holds all flags for the setup command, exposed for tests.
type SetupOptions struct {
	Auto       bool
	Project    string
	SAName     string
	Profile    string
	KeyOut     string
	DryRun     bool
	SetDefault bool
	NoBrowser  bool
	NoInstall  bool
	Output     string
	Pretty     bool

	Runner     gcloudRunner
	SaveConfig func(profile config.Profile, setDefault bool) (string, error)
	HomeDir    func() (string, error)
}

// SetupResult is what `setup --auto` produces.
type SetupResult struct {
	Project       string   `json:"project,omitempty"`
	ServiceAcct   string   `json:"service_account_email"`
	KeyPath       string   `json:"key_path"`
	ConfigPath    string   `json:"config_path,omitempty"`
	ProfileName   string   `json:"profile"`
	PlayLinkURL   string   `json:"play_console_link_url"`
	BrowserOpened bool     `json:"browser_opened,omitempty"`
	StepsExecuted []string `json:"steps_executed"`
	DryRun        bool     `json:"dry_run,omitempty"`
}

// RunSetup performs the setup flow; stdout is where text-mode messages go.
// Test entry point.
func RunSetup(ctx context.Context, opts SetupOptions, stdout io.Writer) error {
	if stdout == nil {
		stdout = shared.Stdout(ctx)
	}
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
				"or re-run with --auto to automate via gcloud",
		))
	}

	var preSteps []string

	// 1. Ensure gcloud is installed.
	if err := ensureGcloud(ctx, opts, &preSteps); err != nil {
		return err
	}

	// 2. Ensure we are logged into gcloud.
	if err := ensureGcloudAuth(ctx, opts, &preSteps); err != nil {
		return err
	}

	// 3. Resolve the project.
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
		Project:       project,
		ServiceAcct:   saEmail,
		KeyPath:       keyPath,
		ProfileName:   opts.Profile,
		PlayLinkURL:   linkURL,
		DryRun:        opts.DryRun,
		StepsExecuted: preSteps,
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

	keyRoot, err := rootfs.OpenOrCreate(filepath.Dir(keyPath), 0o700)
	if err != nil {
		return fmt.Errorf("prepare key output dir: %w", err)
	}
	if err := keyRoot.Close(); err != nil {
		return fmt.Errorf("close key output dir: %w", err)
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

		// Final step: open Play Console for the manual grant (best-effort).
		if !opts.NoBrowser {
			_ = opts.Runner.Copy(saEmail)
			if err := opts.Runner.OpenBrowser(linkURL); err == nil {
				result.BrowserOpened = true
			}
		}
	}

	if strings.ToLower(opts.Output) == "json" {
		return shared.PrintOutputContext(ctx, result, "json", opts.Pretty)
	}
	printSetupText(stdout, result)
	return nil
}

// ensureGcloud makes sure the gcloud CLI is available, installing it when
// missing (unless --no-install or --dry-run).
func ensureGcloud(ctx context.Context, opts SetupOptions, steps *[]string) error {
	if _, err := opts.Runner.LookPath("gcloud"); err == nil {
		return nil
	}

	if opts.DryRun {
		*steps = append(*steps, "install gcloud CLI (skipped: dry-run)")
		return nil
	}
	if opts.NoInstall {
		return shared.NewReportedError(fmt.Errorf(
			"gcloud CLI not found; install it from https://cloud.google.com/sdk or omit --no-install to auto-install",
		))
	}

	if err := opts.Runner.InstallGcloud(ctx); err != nil {
		return shared.NewReportedError(fmt.Errorf(
			"install gcloud: %w; install manually from https://cloud.google.com/sdk", err,
		))
	}
	if _, err := opts.Runner.LookPath("gcloud"); err != nil {
		return shared.NewReportedError(fmt.Errorf(
			"gcloud was installed but is not on PATH yet; restart your shell and re-run `gplay setup --auto`",
		))
	}
	*steps = append(*steps, "installed gcloud CLI")
	return nil
}

// ensureGcloudAuth makes sure there is an active gcloud account, launching
// `gcloud auth login` when there is not (unless --no-browser or --dry-run).
func ensureGcloudAuth(ctx context.Context, opts SetupOptions, steps *[]string) error {
	if opts.DryRun {
		*steps = append(*steps, "gcloud auth login (if not already authenticated)")
		return nil
	}

	if activeGcloudAccount(ctx, opts.Runner) != "" {
		*steps = append(*steps, "gcloud account: "+activeGcloudAccount(ctx, opts.Runner))
		return nil
	}

	if opts.NoBrowser {
		return shared.NewReportedError(fmt.Errorf(
			"not logged into gcloud; run `gcloud auth login` first (or omit --no-browser to launch it)",
		))
	}

	if err := opts.Runner.RunInteractive(ctx, "gcloud", "auth", "login"); err != nil {
		return shared.NewReportedError(fmt.Errorf("gcloud auth login: %w", err))
	}
	account := activeGcloudAccount(ctx, opts.Runner)
	if account == "" {
		return shared.NewReportedError(fmt.Errorf("gcloud login did not complete; re-run `gplay setup --auto`"))
	}
	*steps = append(*steps, "logged into gcloud: "+account)
	return nil
}

func activeGcloudAccount(ctx context.Context, runner gcloudRunner) string {
	out, err := runner.Run(ctx, nil, "gcloud", "auth", "list", "--filter=status:ACTIVE", "--format=value(account)")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
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

func printSetupText(w io.Writer, r SetupResult) {
	fmt.Fprintln(w, "gplay setup")
	fmt.Fprintln(w, "===========")
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
	fmt.Fprintln(w, "\nLast step — grant access in Play Console:")
	if r.BrowserOpened {
		fmt.Fprintln(w, "  Opened Play Console in your browser (service account email copied to clipboard).")
	}
	fmt.Fprintf(w, "  %s\n", r.PlayLinkURL)
	fmt.Fprintf(w, "  Grant %s access, then verify with `gplay auth doctor`.\n", r.ServiceAcct)
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

// saveProfileToConfig is the real SaveConfig hook for the setup command.
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
