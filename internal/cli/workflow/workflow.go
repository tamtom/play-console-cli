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
		LongHelp: `Define reusable Google Play release workflows in .gplay/workflows/*.json.
Workflow files can contain one legacy workflow or multiple named workflows.

Example workflow file (.gplay/workflows/release.json):

{
  "workflows": {
    "preflight": {
      "steps": [
        {
          "name": "readiness",
          "run": "gplay validate --package {{ .PACKAGE }} --track {{ .TRACK }} --bundle {{ .BUNDLE }}",
          "outputs": {
            "status": "$.summary.status"
          }
        }
      ]
    },
    "publish": {
      "params": [
        {"name": "PACKAGE", "required": true},
        {"name": "TRACK", "required": true},
        {"name": "BUNDLE", "required": true}
      ],
      "steps": [
        {"name": "preflight", "workflow": "preflight", "with": {"PACKAGE": "{{ .PACKAGE }}", "TRACK": "{{ .TRACK }}", "BUNDLE": "{{ .BUNDLE }}"}},
        {"name": "release", "run": "gplay publish track --package {{ .PACKAGE }} --track {{ .TRACK }} --bundle {{ .BUNDLE }}"}
      ]
    }
  }
}

Security note:
  Workflows execute arbitrary shell commands.
  Only run workflow files you trust.

Tips:
  Use gplay workflow validate before running a new workflow file.
  Preview the plan with gplay workflow run --dry-run release --workflow publish.

Examples:
  gplay workflow list
  gplay workflow validate release.json
  gplay workflow run release --workflow publish --param PACKAGE=com.example.app --param TRACK=internal --param BUNDLE=app.aab
  gplay workflow run --dry-run ./release.json --workflow publish`,
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
	workflowName := fs.String("workflow", "", "Workflow name when the file contains multiple workflows")

	type paramSlice []string
	var params paramSlice
	fs.Func("param", "Workflow parameter in KEY=VALUE format (repeatable)", func(s string) error {
		params = append(params, s)
		return nil
	})

	return &ffcli.Command{
		Name:       "run",
		ShortUsage: "gplay workflow run <name-or-file> [--workflow <name>] [--param KEY=VALUE ...] [--dry-run] [--resume]",
		ShortHelp:  "Run a named workflow.",
		LongHelp: `Run a workflow from .gplay/workflows/ or a direct file path.

The workflow file is resolved from .gplay/workflows/<name>.json first unless
you pass an explicit path. Legacy single-workflow files still work.

Examples:
  gplay workflow run release --workflow publish --param PACKAGE=com.example.app --param TRACK=internal --param BUNDLE=app.aab
  gplay workflow run ./release.json --workflow publish --param TRACK=production
  gplay workflow run --dry-run release --workflow publish
  gplay workflow run release --workflow publish --resume`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return shared.UsageError("workflow name or file path is required")
			}
			if *dryRun && *resume {
				return shared.UsageError("--resume cannot be used with --dry-run")
			}

			nameOrFile := args[0]
			filePath, err := resolveWorkflowPath(nameOrFile)
			if err != nil {
				return fmt.Errorf("workflow run: %w", err)
			}

			def, err := wf.LoadDefinition(filePath)
			if err != nil {
				return fmt.Errorf("workflow run: %w", err)
			}

			selectedName, _, err := wf.SelectWorkflow(def, workflowImplicitName(nameOrFile, filePath), *workflowName)
			if err != nil {
				return fmt.Errorf("workflow run: %w", err)
			}

			paramMap, err := parseParams(params)
			if err != nil {
				return shared.UsageErrorf("invalid parameter: %v", err)
			}

			result, err := wf.ExecuteDefinition(ctx, def, selectedName, paramMap, wf.ExecuteOptions{
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
		LongHelp: `Validate a workflow file for structure, references, output declarations, and cycles.

Examples:
  gplay workflow validate release
  gplay workflow validate ./release.json`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(_ context.Context, args []string) error {
			if len(args) == 0 {
				return shared.UsageError("workflow name or file path is required")
			}

			filePath, err := resolveWorkflowPath(args[0])
			if err != nil {
				return fmt.Errorf("workflow validate: %w", err)
			}

			def, err := wf.LoadDefinition(filePath)
			if err != nil {
				return fmt.Errorf("workflow validate: %w", err)
			}

			errs := wf.Validate(def)
			result := struct {
				Valid  bool                  `json:"valid"`
				Errors []*wf.ValidationError `json:"errors,omitempty"`
			}{
				Valid:  len(errs) == 0,
				Errors: errs,
			}

			if printErr := printJSON(os.Stdout, result, *pretty); printErr != nil {
				return printErr
			}
			if !result.Valid {
				return shared.NewReportedError(fmt.Errorf("workflow validate: found %d error(s)", len(errs)))
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
				Private     bool   `json:"private,omitempty"`
				StepCount   int    `json:"step_count"`
			}

			var workflows []workflowInfo
			for _, entry := range entries {
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
					continue
				}

				filePath := filepath.Join(*dir, entry.Name())
				def, err := wf.LoadDefinition(filePath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", entry.Name(), err)
					continue
				}

				for _, name := range sortedWorkflowNames(def) {
					workflow := def.Workflows[name]
					workflows = append(workflows, workflowInfo{
						Name:        name,
						Description: workflow.Description,
						File:        entry.Name(),
						Private:     workflow.Private,
						StepCount:   len(workflow.Steps),
					})
				}
			}

			sort.Slice(workflows, func(i, j int) bool {
				if workflows[i].Name == workflows[j].Name {
					return workflows[i].File < workflows[j].File
				}
				return workflows[i].Name < workflows[j].Name
			})

			return printJSON(os.Stdout, workflows, *pretty)
		},
	}
}

func workflowImplicitName(nameOrFile, filePath string) string {
	if !strings.Contains(nameOrFile, string(os.PathSeparator)) && filepath.Ext(nameOrFile) == "" && !strings.HasPrefix(nameOrFile, ".") {
		return nameOrFile
	}
	return strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
}

// resolveWorkflowPath resolves a workflow name or file path.
// If it's an existing file path, use it directly.
// Otherwise, look in .gplay/workflows/<name>.json.
func resolveWorkflowPath(nameOrFile string) (string, error) {
	if strings.HasSuffix(nameOrFile, ".json") || strings.Contains(nameOrFile, string(os.PathSeparator)) || strings.HasPrefix(nameOrFile, ".") {
		absPath, err := filepath.Abs(nameOrFile)
		if err != nil {
			return "", fmt.Errorf("resolve path: %w", err)
		}
		return absPath, nil
	}

	absPath, err := filepath.Abs(filepath.Join(defaultWorkflowDir, nameOrFile+".json"))
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	return absPath, nil
}

func parseParams(args []string) (map[string]string, error) {
	params := make(map[string]string, len(args))
	for _, arg := range args {
		idx := strings.Index(arg, "=")
		if idx <= 0 {
			return nil, fmt.Errorf("expected KEY=VALUE format, got %q", arg)
		}
		key := strings.TrimSpace(arg[:idx])
		if key == "" {
			return nil, fmt.Errorf("parameter key must not be empty in %q", arg)
		}
		params[key] = arg[idx+1:]
	}
	return params, nil
}

func sortedWorkflowNames(def *wf.Definition) []string {
	names := make([]string, 0, len(def.Workflows))
	for name := range def.Workflows {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func printJSON(writer io.Writer, data any, pretty bool) error {
	enc := json.NewEncoder(writer)
	if pretty {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(data)
}
