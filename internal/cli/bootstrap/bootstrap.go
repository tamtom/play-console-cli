package bootstrap

import (
	"context"
	"flag"
	"fmt"

	"github.com/peterbourgon/ff/v3/ffcli"

	bootstrapplan "github.com/tamtom/play-console-cli/internal/bootstrap"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/output"
)

func init() {
	output.RegisterType(&bootstrapplan.Plan{}, []string{"STEP", "MODE", "STATUS", "TITLE", "COMMAND/URL"}, func(data any) [][]string {
		plan := data.(*bootstrapplan.Plan)
		rows := make([][]string, 0, len(plan.Steps))
		for _, step := range plan.Steps {
			action := step.Command
			if action == "" {
				action = step.URL
			}
			rows = append(rows, []string{step.ID, step.Mode, step.Status, step.Title, action})
		}
		return rows
	})
}

// Command returns the offline initial-app bootstrap command group.
func Command() *ffcli.Command {
	fs := flag.NewFlagSet("bootstrap", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "bootstrap",
		ShortUsage: "gplay bootstrap <subcommand> [flags]",
		ShortHelp:  "Plan policy-safe initial app setup.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			PlanCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			fmt.Fprintf(shared.Stderr(ctx), "Unknown subcommand: %s\n", args[0])
			return flag.ErrHelp
		},
	}
}

// PlanCommand returns the deterministic, offline bootstrap planner.
func PlanCommand() *ffcli.Command {
	fs := flag.NewFlagSet("bootstrap plan", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	appName := fs.String("name", "", "App name shown in Play Console")
	aabPath := fs.String("aab", "", "Path to the first Android App Bundle")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "plan",
		ShortUsage: "gplay bootstrap plan --package <name> --name <app-name> --aab <path> [flags]",
		ShortHelp:  "Build an offline manual-handoff plan for initial app setup.",
		LongHelp: `Validate local inputs and build a deterministic initial-app setup plan.

This command does not authenticate, contact Google, open a browser, accept any
agreement, upload an artifact, or change a Play Console account. Unsupported
initial setup actions are emitted as explicit steps for you to complete manually.

Examples:
  gplay bootstrap plan --package dev.example.app --name "Example App" --aab ./app.aab
  gplay bootstrap plan --package dev.example.app --name "Example App" --aab ./app.aab --output table`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("unexpected arguments: %v", args)
			}
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			plan, err := bootstrapplan.BuildPlan(bootstrapplan.Request{
				PackageName:  *packageName,
				AppName:      *appName,
				ArtifactPath: *aabPath,
			})
			if err != nil {
				return err
			}
			return shared.PrintOutputContext(ctx, plan, *outputFlag, *pretty)
		},
	}
}
