// Package experiments exposes the policy-safe boundary for Google Play store
// listing experiments. The public API can apply a manually selected winner,
// but currently has no lifecycle or results resource.
package experiments

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/apischema"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/cli/sync"
)

var runWinnerSync = sync.RunTransaction

// Command returns the store-listing experiments boundary.
func Command() *ffcli.Command {
	fs := flag.NewFlagSet("experiments", flag.ExitOnError)
	return &ffcli.Command{
		Name:        "experiments",
		ShortUsage:  "gplay experiments <support|apply-winner> [flags]",
		ShortHelp:   "Inspect experiment API support and apply a manually selected winner.",
		LongHelp:    "Experiment lifecycle and results remain manual until Google publishes an official API. Applying a selected winner uses only Android Publisher edits.listings and edits.images.",
		FlagSet:     fs,
		UsageFunc:   shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{SupportCommand(), ApplyWinnerCommand()},
		Exec:        func(context.Context, []string) error { return flag.ErrHelp },
	}
}

type supportResult struct {
	OfficialLifecycleAPI    bool     `json:"officialLifecycleApi"`
	OfficialResultsAPI      bool     `json:"officialResultsApi"`
	OfficialApplyWinnerAPI  bool     `json:"officialApplyWinnerApi"`
	ApplyWinnerResources    []string `json:"applyWinnerResources"`
	DiscoveryMethodsFound   []string `json:"discoveryMethodsFound"`
	DiscoveryRevision       string   `json:"discoveryRevision,omitempty"`
	ManualConsoleRequired   bool     `json:"manualConsoleRequired"`
	PrivateInterfacesUsed   bool     `json:"privateInterfacesUsed"`
	RecommendedApplyCommand string   `json:"recommendedApplyCommand"`
}

// SupportCommand reports the reviewed embedded public-API boundary offline.
func SupportCommand() *ffcli.Command {
	fs := flag.NewFlagSet("experiments support", flag.ExitOnError)
	output := fs.String("output", "json", "Output format: json, table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")
	return &ffcli.Command{
		Name:       "support",
		ShortUsage: "gplay experiments support",
		ShortHelp:  "Report official API support for store-listing experiments.",
		LongHelp:   "Reads the embedded reviewed Google discovery index only. It never authenticates or contacts Google.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 0 {
				return shared.UsageErrorf("unexpected arguments: %s", strings.Join(args, " "))
			}
			if err := shared.ValidateOutputFlags(*output, *pretty); err != nil {
				return err
			}
			result, err := currentSupport()
			if err != nil {
				return err
			}
			return shared.PrintOutputContext(ctx, result, *output, *pretty)
		},
	}
}

func currentSupport() (supportResult, error) {
	index, err := apischema.Load()
	if err != nil {
		return supportResult{}, err
	}
	endpoints, err := index.FindEndpoints(apischema.Filter{API: "androidpublisher", Query: "experiment"})
	if err != nil {
		return supportResult{}, err
	}
	methods := make([]string, 0, len(endpoints))
	for _, endpoint := range endpoints {
		methods = append(methods, endpoint.ID)
	}
	revision := ""
	for _, api := range index.APIs {
		if api.Name == "androidpublisher" {
			revision = api.Revision
			break
		}
	}
	return supportResult{
		OfficialLifecycleAPI:    false,
		OfficialResultsAPI:      false,
		OfficialApplyWinnerAPI:  true,
		ApplyWinnerResources:    []string{"edits.listings", "edits.images"},
		DiscoveryMethodsFound:   methods,
		DiscoveryRevision:       revision,
		ManualConsoleRequired:   true,
		PrivateInterfacesUsed:   false,
		RecommendedApplyCommand: "gplay experiments apply-winner --package <name> --edit <id> --winner <name> --confirm-winner <name> --dir <metadata>",
	}, nil
}

type applyWinnerResult struct {
	Provider        string          `json:"provider"`
	SelectionSource string          `json:"selectionSource"`
	Winner          string          `json:"winner"`
	Lifecycle       string          `json:"lifecycle"`
	Sync            *sync.RunResult `json:"sync"`
}

// ApplyWinnerCommand applies metadata/images after a person selected the
// experiment winner in Play Console.
func ApplyWinnerCommand() *ffcli.Command {
	fs := flag.NewFlagSet("experiments apply-winner", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Existing Android Publisher edit ID")
	winner := fs.String("winner", "", "Human-selected experiment winner name")
	confirmWinner := fs.String("confirm-winner", "", "Repeat the exact winner name to authorize application")
	dir := fs.String("dir", "", "Winner metadata/images directory")
	stateDir := fs.String("state-dir", ".gplay/experiments", "Directory for official sync plans and receipts")
	output := fs.String("output", "json", "Output format: json, table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")
	return &ffcli.Command{
		Name:       "apply-winner",
		ShortUsage: "gplay experiments apply-winner --package <name> --edit <id> --winner <name> --confirm-winner <name> --dir <path>",
		ShortHelp:  "Apply a manually selected winner through official listing/image APIs.",
		LongHelp: `A human must first review experiment results and select the winner in Play
Console. This command cannot read or infer experiment results. It requires the
winner name twice, then delegates to the official, resumable sync transaction
with remote preconditions, a content-addressed plan, and an atomic receipt.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 0 {
				return shared.UsageErrorf("unexpected arguments: %s", strings.Join(args, " "))
			}
			if err := shared.ValidateOutputFlags(*output, *pretty); err != nil {
				return err
			}
			for name, value := range map[string]string{"--package": *packageName, "--edit": *editID, "--winner": *winner, "--confirm-winner": *confirmWinner, "--dir": *dir} {
				if strings.TrimSpace(value) == "" {
					return shared.UsageError(name + " is required")
				}
			}
			selected := strings.TrimSpace(*winner)
			if strings.TrimSpace(*confirmWinner) != selected {
				return fmt.Errorf("--confirm-winner must exactly match --winner %q", selected)
			}
			result, err := runWinnerSync(ctx, strings.TrimSpace(*packageName), strings.TrimSpace(*editID), strings.TrimSpace(*dir), strings.TrimSpace(*stateDir))
			if err != nil {
				return err
			}
			return shared.PrintOutputContext(ctx, applyWinnerResult{
				Provider: "official-api", SelectionSource: "manual", Winner: selected,
				Lifecycle: "manual-play-console", Sync: result,
			}, *output, *pretty)
		},
	}
}
