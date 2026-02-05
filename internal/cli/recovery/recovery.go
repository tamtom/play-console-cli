package recovery

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

func RecoveryCommand() *ffcli.Command {
	fs := flag.NewFlagSet("recovery", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "recovery",
		ShortUsage: "gplay recovery <subcommand> [flags]",
		ShortHelp:  "Manage app recovery actions.",
		LongHelp: `Manage remote app recovery actions.

App recovery allows you to remotely trigger actions on user devices
for crash mitigation, such as forcing an update or clearing app data.

Use with caution - these actions directly affect users.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			ListCommand(),
			CreateCommand(),
			DeployCommand(),
			CancelCommand(),
			AddTargetingCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			return flag.ErrHelp
		},
	}
}

func ListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("recovery list", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	versionCode := fs.Int64("version-code", 0, "Version code (optional, filters by version)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay recovery list --package <name> [--version-code <code>]",
		ShortHelp:  "List recovery actions.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			call := service.API.Apprecovery.List(pkg).Context(ctx)
			if *versionCode > 0 {
				call = call.VersionCode(*versionCode)
			}
			resp, err := call.Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func CreateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("recovery create", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	jsonFlag := fs.String("json", "", "CreateDraftAppRecoveryRequest JSON (or @file)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "create",
		ShortUsage: "gplay recovery create --package <name> --json <json>",
		ShortHelp:  "Create a draft recovery action.",
		LongHelp: `Create a draft app recovery action.

JSON format:
{
  "targeting": {
    "versionList": {
      "versionCodes": ["100", "101", "102"]
    }
  },
  "remoteInAppUpdate": {
    "isRemoteInAppUpdateRequested": true
  }
}

Or for data deletion:
{
  "targeting": {
    "allUsers": {}
  },
  "appDataDeletion": {}
}`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*jsonFlag) == "" {
				return fmt.Errorf("--json is required")
			}
			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			var req androidpublisher.CreateDraftAppRecoveryRequest
			if err := shared.LoadJSONArg(*jsonFlag, &req); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.API.Apprecovery.Create(pkg, &req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func DeployCommand() *ffcli.Command {
	fs := flag.NewFlagSet("recovery deploy", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	recoveryID := fs.Int64("recovery-id", 0, "Recovery action ID")
	confirm := fs.Bool("confirm", false, "Confirm deployment")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "deploy",
		ShortUsage: "gplay recovery deploy --package <name> --recovery-id <id> --confirm",
		ShortHelp:  "Deploy a recovery action to users.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if *recoveryID == 0 {
				return fmt.Errorf("--recovery-id is required")
			}
			if !*confirm {
				return fmt.Errorf("--confirm is required (this action affects users)")
			}
			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			req := &androidpublisher.DeployAppRecoveryRequest{}
			resp, err := service.API.Apprecovery.Deploy(pkg, *recoveryID, req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func CancelCommand() *ffcli.Command {
	fs := flag.NewFlagSet("recovery cancel", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	recoveryID := fs.Int64("recovery-id", 0, "Recovery action ID")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "cancel",
		ShortUsage: "gplay recovery cancel --package <name> --recovery-id <id>",
		ShortHelp:  "Cancel a recovery action.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if *recoveryID == 0 {
				return fmt.Errorf("--recovery-id is required")
			}
			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			req := &androidpublisher.CancelAppRecoveryRequest{}
			resp, err := service.API.Apprecovery.Cancel(pkg, *recoveryID, req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func AddTargetingCommand() *ffcli.Command {
	fs := flag.NewFlagSet("recovery add-targeting", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	recoveryID := fs.Int64("recovery-id", 0, "Recovery action ID")
	jsonFlag := fs.String("json", "", "AddTargetingRequest JSON (or @file)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "add-targeting",
		ShortUsage: "gplay recovery add-targeting --package <name> --recovery-id <id> --json <json>",
		ShortHelp:  "Add targeting criteria to a recovery action.",
		LongHelp: `Add additional targeting criteria to a draft recovery action.

JSON format:
{
  "targetingUpdate": {
    "versionList": {
      "versionCodes": ["103", "104"]
    }
  }
}`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if *recoveryID == 0 {
				return fmt.Errorf("--recovery-id is required")
			}
			if strings.TrimSpace(*jsonFlag) == "" {
				return fmt.Errorf("--json is required")
			}
			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			var req androidpublisher.AddTargetingRequest
			if err := shared.LoadJSONArg(*jsonFlag, &req); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			resp, err := service.API.Apprecovery.AddTargeting(pkg, *recoveryID, &req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}
