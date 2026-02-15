package auth

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/config"
	"github.com/tamtom/play-console-cli/internal/output"
)

// AuthCommand builds the auth root command.
func AuthCommand() *ffcli.Command {
	fs := flag.NewFlagSet("auth", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "auth",
		ShortUsage: "gplay auth <subcommand> [flags]",
		ShortHelp:  "Manage Google Play authentication.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			AuthInitCommand(),
			AuthLoginCommand(),
			AuthSwitchCommand(),
			AuthLogoutCommand(),
			AuthStatusCommand(),
			AuthDoctorCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n\n", args[0])
			return flag.ErrHelp
		},
	}
}

func AuthInitCommand() *ffcli.Command {
	fs := flag.NewFlagSet("auth init", flag.ExitOnError)
	force := fs.Bool("force", false, "Overwrite existing config.json")
	local := fs.Bool("local", false, "Write config.json to ./.gplay in the current repo")

	return &ffcli.Command{
		Name:       "init",
		ShortUsage: "gplay auth init [flags]",
		ShortHelp:  "Create a template config.json for authentication.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			var path string
			var err error
			if *local {
				path, err = config.LocalPath()
			} else {
				path, err = config.GlobalPath()
			}
			if err != nil {
				return err
			}

			if !*force {
				if _, err := os.Stat(path); err == nil {
					return fmt.Errorf("auth init: config already exists at %s (use --force to overwrite)", path)
				} else if !os.IsNotExist(err) {
					return err
				}
			}

			template := &config.Config{}
			if err := config.SaveAt(path, template); err != nil {
				return err
			}

			result := struct {
				ConfigPath string         `json:"config_path"`
				Created    bool           `json:"created"`
				Config     *config.Config `json:"config"`
			}{
				ConfigPath: path,
				Created:    true,
				Config:     template,
			}
			return output.PrintJSON(result)
		},
	}
}

func AuthLoginCommand() *ffcli.Command {
	fs := flag.NewFlagSet("auth login", flag.ExitOnError)
	profile := fs.String("profile", "default", "Profile name")
	serviceAccount := fs.String("service-account", "", "Path to service account JSON (required)")
	setDefault := fs.Bool("set-default", true, "Set as default profile")
	local := fs.Bool("local", false, "Write to local repo config")

	return &ffcli.Command{
		Name:       "login",
		ShortUsage: "gplay auth login --service-account <path> [flags]",
		ShortHelp:  "Authenticate with Google Play Console using a service account.",
		LongHelp: `Authenticate with Google Play Console using a service account.

Service accounts are required for the Google Play Android Developer API.
See README.md for setup instructions.

Examples:
  gplay auth login --service-account /path/to/key.json
  gplay auth login --service-account key.json --profile work
  gplay auth login --service-account key.json --local`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if strings.TrimSpace(*profile) == "" {
				return fmt.Errorf("--profile is required")
			}
			if strings.TrimSpace(*serviceAccount) == "" {
				return fmt.Errorf("--service-account is required")
			}

			newProfile := config.Profile{
				Name:    *profile,
				Type:    "service_account",
				KeyPath: *serviceAccount,
			}

			cfg, _ := config.Load()
			if cfg == nil {
				cfg = &config.Config{}
			}

			cfg.Profiles = upsertProfile(cfg.Profiles, newProfile)
			if *setDefault {
				cfg.DefaultProfile = newProfile.Name
			}

			path, err := resolveConfigPath(*local)
			if err != nil {
				return err
			}
			if err := config.SaveAt(path, cfg); err != nil {
				return err
			}

			result := struct {
				ConfigPath string         `json:"config_path"`
				Profile    config.Profile `json:"profile"`
			}{
				ConfigPath: path,
				Profile:    newProfile,
			}
			return output.PrintJSON(result)
		},
	}
}

func AuthSwitchCommand() *ffcli.Command {
	fs := flag.NewFlagSet("auth switch", flag.ExitOnError)
	profile := fs.String("profile", "", "Profile name")

	return &ffcli.Command{
		Name:       "switch",
		ShortUsage: "gplay auth switch --profile <name>",
		ShortHelp:  "Switch the default auth profile.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if strings.TrimSpace(*profile) == "" {
				return fmt.Errorf("--profile is required")
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if _, ok := findProfile(cfg.Profiles, *profile); !ok {
				return fmt.Errorf("profile not found: %s", *profile)
			}
			cfg.DefaultProfile = *profile
			path, err := config.Path()
			if err != nil {
				return err
			}
			if err := config.SaveAt(path, cfg); err != nil {
				return err
			}
			result := struct {
				ConfigPath string `json:"config_path"`
				Default    string `json:"default_profile"`
			}{
				ConfigPath: path,
				Default:    *profile,
			}
			return output.PrintJSON(result)
		},
	}
}

func AuthLogoutCommand() *ffcli.Command {
	fs := flag.NewFlagSet("auth logout", flag.ExitOnError)
	profile := fs.String("profile", "", "Profile name")
	confirm := fs.Bool("confirm", false, "Confirm removal")

	return &ffcli.Command{
		Name:       "logout",
		ShortUsage: "gplay auth logout --profile <name> --confirm",
		ShortHelp:  "Remove a stored auth profile.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if strings.TrimSpace(*profile) == "" {
				return fmt.Errorf("--profile is required")
			}
			if !*confirm {
				return fmt.Errorf("--confirm is required")
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			profiles, removed := removeProfile(cfg.Profiles, *profile)
			if !removed {
				return fmt.Errorf("profile not found: %s", *profile)
			}
			cfg.Profiles = profiles
			if cfg.DefaultProfile == *profile {
				cfg.DefaultProfile = ""
			}
			path, err := config.Path()
			if err != nil {
				return err
			}
			if err := config.SaveAt(path, cfg); err != nil {
				return err
			}
			result := struct {
				ConfigPath string `json:"config_path"`
				Removed    string `json:"removed_profile"`
			}{
				ConfigPath: path,
				Removed:    *profile,
			}
			return output.PrintJSON(result)
		},
	}
}

func AuthStatusCommand() *ffcli.Command {
	fs := flag.NewFlagSet("auth status", flag.ExitOnError)
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "status",
		ShortUsage: "gplay auth status [flags]",
		ShortHelp:  "Show authentication status.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			cfg, _ := config.Load()
			configPath, _ := config.Path()
			profileName := shared.ResolveProfileName(cfg)
			result := struct {
				ConfigPath  string         `json:"config_path"`
				Profile     string         `json:"profile"`
				Profiles    []config.Profile `json:"profiles"`
				EnvPresent  bool           `json:"env_present"`
			}{
				ConfigPath: configPath,
				Profile:    profileName,
				Profiles:   nil,
				EnvPresent: envAuthPresent(),
			}
			if cfg != nil {
				result.Profiles = cfg.Profiles
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}

func AuthDoctorCommand() *ffcli.Command {
	fs := flag.NewFlagSet("auth doctor", flag.ExitOnError)
	outputFlag := fs.String("output", "text", "Output format: text (default), json")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")
	fix := fs.Bool("fix", false, "Attempt to auto-fix detected issues")
	confirm := fs.Bool("confirm", false, "Required with --fix to apply changes (without it, --fix does a dry run)")

	return &ffcli.Command{
		Name:       "doctor",
		ShortUsage: "gplay auth doctor [flags]",
		ShortHelp:  "Diagnose authentication configuration issues.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			normalized := strings.ToLower(strings.TrimSpace(*outputFlag))
			if normalized != "text" && normalized != "json" {
				return fmt.Errorf("unsupported format: %s", *outputFlag)
			}
			if normalized != "json" && *pretty {
				return fmt.Errorf("--pretty is only valid with JSON output")
			}
			if *confirm && !*fix {
				return fmt.Errorf("--confirm requires --fix")
			}

			report := buildAuthReport()
			if normalized == "json" {
				if *fix {
					fixes := attemptFixes(report, *confirm)
					result := struct {
						Report authReport  `json:"report"`
						Fixes  []fixResult `json:"fixes"`
					}{Report: report, Fixes: fixes}
					if *pretty {
						return output.PrintPrettyJSON(result)
					}
					return output.PrintJSON(result)
				}
				if *pretty {
					return output.PrintPrettyJSON(report)
				}
				return output.PrintJSON(report)
			}

			printAuthReport(report)

			if *fix {
				fixes := attemptFixes(report, *confirm)
				printFixes(fixes)
			}

			if report.Errors > 0 {
				return shared.NewReportedError(fmt.Errorf("auth doctor: found %d error(s)", report.Errors))
			}
			return nil
		},
	}
}

type authReport struct {
	Errors int      `json:"errors"`
	Warnings int    `json:"warnings"`
	Checks []string `json:"checks"`
}

func buildAuthReport() authReport {
	var report authReport
	cfg, err := config.Load()
	if err != nil && err != config.ErrNotFound {
		report.Errors++
		report.Checks = append(report.Checks, fmt.Sprintf("failed to load config: %v", err))
		return report
	}

	if cfg == nil || len(cfg.Profiles) == 0 {
		report.Warnings++
		report.Checks = append(report.Checks, "no profiles configured")
	} else {
		report.Checks = append(report.Checks, fmt.Sprintf("profiles configured: %d", len(cfg.Profiles)))
	}

	if envAuthPresent() {
		report.Checks = append(report.Checks, "environment credentials detected")
	}

	if cfg != nil {
		profile := shared.ResolveProfileName(cfg)
		if profile == "" {
			report.Warnings++
			report.Checks = append(report.Checks, "no default profile selected")
		} else {
			report.Checks = append(report.Checks, fmt.Sprintf("default profile: %s", profile))
		}
	}
	return report
}

func printAuthReport(report authReport) {
	fmt.Println("Auth Doctor")
	for _, check := range report.Checks {
		fmt.Printf("  - %s\n", check)
	}
	if report.Errors == 0 && report.Warnings == 0 {
		fmt.Println("No issues found.")
	} else {
		fmt.Printf("Found %d warning(s) and %d error(s).\n", report.Warnings, report.Errors)
	}
}

func resolveConfigPath(local bool) (string, error) {
	if local {
		return config.LocalPath()
	}
	return config.Path()
}

func upsertProfile(existing []config.Profile, profile config.Profile) []config.Profile {
	var out []config.Profile
	updated := false
	for _, p := range existing {
		if p.Name == profile.Name {
			out = append(out, profile)
			updated = true
			continue
		}
		out = append(out, p)
	}
	if !updated {
		out = append(out, profile)
	}
	return out
}

func removeProfile(existing []config.Profile, name string) ([]config.Profile, bool) {
	var out []config.Profile
	removed := false
	for _, p := range existing {
		if p.Name == name {
			removed = true
			continue
		}
		out = append(out, p)
	}
	return out, removed
}

func findProfile(existing []config.Profile, name string) (config.Profile, bool) {
	for _, p := range existing {
		if p.Name == name {
			return p, true
		}
	}
	return config.Profile{}, false
}

func envAuthPresent() bool {
	if strings.TrimSpace(os.Getenv("GPLAY_SERVICE_ACCOUNT_JSON")) != "" {
		return true
	}
	if strings.TrimSpace(os.Getenv("GPLAY_OAUTH_TOKEN_PATH")) != "" {
		return true
	}
	return false
}

