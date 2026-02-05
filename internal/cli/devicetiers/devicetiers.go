package devicetiers

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

func DeviceTiersCommand() *ffcli.Command {
	fs := flag.NewFlagSet("device-tiers", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "device-tiers",
		ShortUsage: "gplay device-tiers <subcommand> [flags]",
		ShortHelp:  "Manage device tier configurations.",
		LongHelp: `Manage device tier configurations for Play Asset Delivery.

Device tiers allow you to customize asset delivery based on
device characteristics like RAM, texture compression support,
or device model.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			ListCommand(),
			GetCommand(),
			CreateCommand(),
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
	fs := flag.NewFlagSet("device-tiers list", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay device-tiers list --package <name>",
		ShortHelp:  "List device tier configurations.",
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

			resp, err := service.API.Applications.DeviceTierConfigs.List(pkg).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func GetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("device-tiers get", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	configID := fs.Int64("config-id", 0, "Device tier config ID")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "gplay device-tiers get --package <name> --config-id <id>",
		ShortHelp:  "Get a device tier configuration.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if *configID == 0 {
				return fmt.Errorf("--config-id is required")
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

			resp, err := service.API.Applications.DeviceTierConfigs.Get(pkg, *configID).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func CreateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("device-tiers create", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	jsonFlag := fs.String("json", "", "DeviceTierConfig JSON (or @file)")
	allowUnknownDevices := fs.Bool("allow-unknown-devices", false, "Allow unknown devices in tiers")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "create",
		ShortUsage: "gplay device-tiers create --package <name> --json <json>",
		ShortHelp:  "Create a device tier configuration.",
		LongHelp: `Create a device tier configuration for asset delivery.

JSON format for RAM-based tiers:
{
  "deviceGroups": [
    {
      "name": "low_ram",
      "deviceSelectors": [
        {
          "deviceRam": {
            "minBytes": "0",
            "maxBytes": "2147483648"
          }
        }
      ]
    },
    {
      "name": "high_ram",
      "deviceSelectors": [
        {
          "deviceRam": {
            "minBytes": "2147483648"
          }
        }
      ]
    }
  ],
  "deviceTierSet": {
    "deviceTiers": [
      {
        "deviceGroupNames": ["low_ram"],
        "level": 0
      },
      {
        "deviceGroupNames": ["high_ram"],
        "level": 1
      }
    ]
  }
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

			var config androidpublisher.DeviceTierConfig
			if err := shared.LoadJSONArg(*jsonFlag, &config); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			call := service.API.Applications.DeviceTierConfigs.Create(pkg, &config).Context(ctx)
			if *allowUnknownDevices {
				call = call.AllowUnknownDevices(true)
			}
			resp, err := call.Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}
