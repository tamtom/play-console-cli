package workflow

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	wf "github.com/tamtom/play-console-cli/internal/workflow"
)

const defaultWorkflowDir = ".gplay/workflows"

// WorkflowCommand returns the top-level workflow command group.
func WorkflowCommand() *ffcli.Command {
	fs := flag.NewFlagSet("workflow", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "workflow",
		ShortUsage: "gplay workflow <subcommand> [flags]",
		ShortHelp:  "Run multi-step automation workflows.",
		LongHelp: `Define named, multi-step automation sequences in .gplay/workflows/*.json.
Each workflow composes existing gplay commands and shell commands.

Example workflow file (.gplay/workflows/deploy.json):

{
  "name": "deploy",
  "description": "Build, test, and deploy the app",
  "params": [
    {"name": "VERSION", "required": true}
  ],
  "env": {
    "PACKAGE": "com.example.app"
  },
  "steps": [
    {"name": "build", "command": "make build"},
    {"name": "test", "command": "make test"},
    {"name": "deploy", "command": "gplay release create --package {{ .PACKAGE }} --version {{ .VERSION }}"}
  ]
}

Security note:
  Workflows execute arbitrary shell commands.
  Only run workflow files you trust.

Tips:
  Use gplay workflow validate before running a new workflow file.
  Preview the plan with gplay workflow run --dry-run <name>.

Examples:
  gplay workflow list
  gplay workflow validate deploy.json
  gplay workflow run deploy --param VERSION=1.0.0
  gplay workflow run --dry-run deploy.json`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			workflowRunCommand(),
			workflowValidateCommand(),
			workflowListCommand(),
		},
		Exec: func(_ context.Context, _ []string) error {
			return flag.ErrHelp
		},
	}
}

func workflowRunCommand() *ffcli.Command {
	fs := flag.NewFlagSet("workflow run", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview steps without executing")
	resume := fs.Bool("resume", false, "Resume from last saved state")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	type paramSlice []string
	var params paramSlice
	fs.Func("param", "Workflow parameter in KEY=VALUE format (repeatable)", func(s string) error {
		params = append(params, s)
		return nil
	})

	return &ffcli.Command{
		Name:       "run",
		ShortUsage: "gplay workflow run <name-or-file> [--param KEY=VALUE ...] [--dry-run] [--resume]",
		ShortHelp:  "Run a named workflow.",
		LongHelp: `Run a workflow from .gplay/workflows/ or a direct file path.

The workflow name is resolved by searching .gplay/workflows/<name>.json first.
If not found, it's treated as a direct file path.

Examples:
  gplay workflow run deploy --param VERSION=1.0.0
  gplay workflow run ./my-workflow.json --param ENV=staging
  gplay workflow run --dry-run deploy
  gplay workflow run deploy --resume`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return shared.UsageError("workflow name or file path is required")
			}

			nameOrFile := args[0]
			if *dryRun && *resume {
				return shared.UsageError("--resume cannot be used with --dry-run")
			}

			// Resolve workflow file path.
			filePath, err := resolveWorkflowPath(nameOrFile)
			if err != nil {
				return fmt.Errorf("workflow run: %w", err)
			}

			w, err := wf.Load(filePath)
			if err != nil {
				return fmt.Errorf("workflow run: %w", err)
			}

			// Parse params.
			paramMap, err := parseParams(params)
			if err != nil {
				return shared.UsageErrorf("invalid parameter: %v", err)
			}

			result, err := wf.Execute(ctx, w, paramMap, wf.ExecuteOptions{
				DryRun: *dryRun,
				Resume: *resume,
				Stdout: os.Stderr,
				Stderr: os.Stderr,
			})
			if err != nil {
				if result != nil {
					_ = printJSON(os.Stdout, result, *pretty)
					return shared.NewReportedError(err)
				}
				return fmt.Errorf("workflow run: %w", err)
			}

			return printJSON(os.Stdout, result, *pretty)
		},
	}
}

func workflowValidateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("workflow validate", flag.ExitOnError)
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "validate",
		ShortUsage: "gplay workflow validate <name-or-file>",
		ShortHelp:  "Validate a workflow definition for errors.",
		LongHelp: `Validate a workflow JSON file for structure, references, and naming.

Examples:
  gplay workflow validate deploy
  gplay workflow validate ./my-workflow.json`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(_ context.Context, args []string) error {
			if len(args) == 0 {
				return shared.UsageError("workflow name or file path is required")
			}

			nameOrFile := args[0]
			filePath, err := resolveWorkflowPath(nameOrFile)
			if err != nil {
				return fmt.Errorf("workflow validate: %w", err)
			}

			w, err := wf.Load(filePath)
			if err != nil {
				return fmt.Errorf("workflow validate: %w", err)
			}

			errs := wf.Validate(w)

			type validationResult struct {
				Valid  bool     `json:"valid"`
				Errors []string `json:"errors,omitempty"`
			}

			errorStrings := make([]string, len(errs))
			for i, e := range errs {
				errorStrings[i] = e.Error()
			}

			result := validationResult{
				Valid:  len(errs) == 0,
				Errors: errorStrings,
			}

			if printErr := printJSON(os.Stdout, result, *pretty); printErr != nil {
				return printErr
			}

			if !result.Valid {
				return shared.NewReportedError(
					fmt.Errorf("workflow validate: found %d error(s)", len(errs)),
				)
			}
			return nil
		},
	}
}

func workflowListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("workflow list", flag.ExitOnError)
	dir := fs.String("dir", defaultWorkflowDir, "Directory containing workflow files")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay workflow list [--dir <path>]",
		ShortHelp:  "List available workflows.",
		LongHelp: `List workflows found in .gplay/workflows/ (or a custom directory).

Examples:
  gplay workflow list
  gplay workflow list --dir ./workflows`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(_ context.Context, args []string) error {
			if len(args) > 0 {
				return shared.UsageErrorf("unexpected argument(s): %s", strings.Join(args, " "))
			}

			entries, err := os.ReadDir(*dir)
			if err != nil {
				if os.IsNotExist(err) {
					return printJSON(os.Stdout, []any{}, *pretty)
				}
				return fmt.Errorf("workflow list: %w", err)
			}

			type workflowInfo struct {
				Name        string `json:"name"`
				Description string `json:"description,omitempty"`
				File        string `json:"file"`
				StepCount   int    `json:"step_count"`
			}

			var workflows []workflowInfo
			for _, entry := range entries {
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
					continue
				}

				filePath := filepath.Join(*dir, entry.Name())
				w, err := wf.Load(filePath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", entry.Name(), err)
					continue
				}

				workflows = append(workflows, workflowInfo{
					Name:        w.Name,
					Description: w.Description,
					File:        entry.Name(),
					StepCount:   len(w.Steps),
				})
			}

			sort.Slice(workflows, func(i, j int) bool {
				return workflows[i].Name < workflows[j].Name
			})

			return printJSON(os.Stdout, workflows, *pretty)
		},
	}
}

// resolveWorkflowPath resolves a workflow name or file path.
// If it's an existing file path, use it directly.
// Otherwise, look in .gplay/workflows/<name>.json.
func resolveWorkflowPath(nameOrFile string) (string, error) {
	// If it looks like a file path (has extension or path separator), use directly.
	if strings.HasSuffix(nameOrFile, ".json") || strings.Contains(nameOrFile, string(os.PathSeparator)) || strings.HasPrefix(nameOrFile, ".") {
		absPath, err := filepath.Abs(nameOrFile)
		if err != nil {
			return "", fmt.Errorf("resolve path: %w", err)
		}
		return absPath, nil
	}

	// Look in default workflows directory.
	dirPath := filepath.Join(defaultWorkflowDir, nameOrFile+".json")
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	return absPath, nil
}

// parseParams converts KEY=VALUE strings to a map.
func parseParams(args []string) (map[string]string, error) {
	params := make(map[string]string, len(args))
	for _, arg := range args {
		idx := strings.Index(arg, "=")
		if idx <= 0 {
			return nil, fmt.Errorf("expected KEY=VALUE format, got %q", arg)
		}
		key := strings.TrimSpace(arg[:idx])
		value := arg[idx+1:]
		if key == "" {
			return nil, fmt.Errorf("parameter key must not be empty in %q", arg)
		}
		params[key] = value
	}
	return params, nil
}

// printJSON encodes data as JSON to the writer.
func printJSON(w io.Writer, data any, pretty bool) error {
	enc := json.NewEncoder(w)
	if pretty {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(data)
}
