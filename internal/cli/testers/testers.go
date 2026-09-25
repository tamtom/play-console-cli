package testers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

func TestersCommand() *ffcli.Command {
	fs := flag.NewFlagSet("testers", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "testers",
		ShortUsage: "gplay testers <subcommand> [flags]",
		ShortHelp:  "Manage testers for closed testing tracks.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			GetCommand(),
			UpdateCommand(),
			PatchCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			return flag.ErrHelp
		},
	}
}

func GetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("testers get", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	track := fs.String("track", "", "Track name (e.g., internal, alpha, beta, or custom track name)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "gplay testers get --package <name> --edit <id> --track <track>",
		ShortHelp:  "Get testers for a track.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*editID) == "" {
				return fmt.Errorf("--edit is required")
			}
			if strings.TrimSpace(*track) == "" {
				return fmt.Errorf("--track is required")
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
			resp, err := service.API.Edits.Testers.Get(pkg, *editID, *track).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutputContext(ctx, resp, *outputFlag, *pretty)
		},
	}
}

func UpdateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("testers update", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	track := fs.String("track", "", "Track name")
	emails := fs.String("emails", "", "Deprecated: individual tester emails are not supported by the Google Play API; use --google-groups")
	googleGroups := fs.String("google-groups", "", "Comma-separated list of Google Group email addresses")
	jsonFlag := fs.String("json", "", "Full Testers JSON (or @file) - overrides other flags")
	confirm := fs.Bool("confirm", false, "Confirm replacement of the entire tester resource")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "update",
		ShortUsage: "gplay testers update --package <name> --edit <id> --track <track> [--google-groups <list>] [--json <json>] --confirm",
		ShortHelp:  "Update testers for a track (replaces entire resource).",
		LongHelp: `Update testers for a track. This replaces the entire tester resource.

Any existing testers not included in the request will be removed.
For partial updates that preserve existing testers, use "patch" instead.

JSON format (via --json):
{
  "googleGroups": [
    "beta-testers@example.com",
    "qa-team@example.com"
  ]
}

Alternatively, use the --google-groups flag:
  --google-groups "beta-testers@example.com,qa-team@example.com" --confirm`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			return updateTesters(ctx, *packageName, *editID, *track, *emails, *googleGroups, *jsonFlag, *outputFlag, *pretty, false, *confirm, flagWasSet(fs, "google-groups"))
		},
	}
}

func PatchCommand() *ffcli.Command {
	fs := flag.NewFlagSet("testers patch", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID")
	track := fs.String("track", "", "Track name")
	emails := fs.String("emails", "", "Deprecated: individual tester emails are not supported by the Google Play API; use --google-groups")
	googleGroups := fs.String("google-groups", "", "Comma-separated list of Google Group email addresses")
	jsonFlag := fs.String("json", "", "Partial Testers JSON (or @file) - overrides other flags")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "patch",
		ShortUsage: "gplay testers patch --package <name> --edit <id> --track <track> [--google-groups <list>] [--json <json>]",
		ShortHelp:  "Patch testers for a track (partial update).",
		LongHelp: `Patch testers for a track. This performs a partial update.

Unlike "update", patch merges the provided fields with the
existing resource, preserving any fields not included in the request.

JSON format (via --json):
{
  "googleGroups": [
    "new-testers@example.com"
  ]
}

Alternatively, use the --google-groups flag:
  --google-groups "new-testers@example.com"`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			return updateTesters(ctx, *packageName, *editID, *track, *emails, *googleGroups, *jsonFlag, *outputFlag, *pretty, true, false, flagWasSet(fs, "google-groups"))
		},
	}
}

func updateTesters(ctx context.Context, packageName, editID, track, emails, googleGroups, jsonFlag, outputFlag string, pretty, patch, confirm, googleGroupsSet bool) error {
	if err := shared.ValidateOutputFlags(outputFlag, pretty); err != nil {
		return err
	}
	if strings.TrimSpace(editID) == "" {
		return fmt.Errorf("--edit is required")
	}
	if strings.TrimSpace(track) == "" {
		return fmt.Errorf("--track is required")
	}
	if strings.TrimSpace(emails) != "" {
		return shared.UsageError("--emails is not supported by the Google Play Android Publisher API; create a Google Group and pass its address with --google-groups")
	}

	var testers androidpublisher.Testers
	googleGroupsPresent := googleGroupsSet
	if strings.TrimSpace(jsonFlag) != "" {
		var err error
		testers, googleGroupsPresent, err = parseTestersJSON(jsonFlag)
		if err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
	} else {
		if googleGroupsSet {
			testers.GoogleGroups = shared.SplitUniqueCSV(googleGroups)
		}
	}
	if !patch && !confirm {
		return shared.UsageError("--confirm is required because testers update replaces the entire tester resource")
	}
	if !patch || googleGroupsPresent {
		testers.ForceSendFields = []string{"GoogleGroups"}
	}

	service, err := playclient.NewService(ctx)
	if err != nil {
		return err
	}
	pkg := shared.ResolvePackageName(packageName, service.Cfg)
	if strings.TrimSpace(pkg) == "" {
		return fmt.Errorf("--package is required")
	}

	ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
	defer cancel()

	if patch {
		resp, err := service.API.Edits.Testers.Patch(pkg, editID, track, &testers).Context(ctx).Do()
		if err != nil {
			return err
		}
		return shared.PrintOutputContext(ctx, resp, outputFlag, pretty)
	}

	resp, err := service.API.Edits.Testers.Update(pkg, editID, track, &testers).Context(ctx).Do()
	if err != nil {
		return err
	}
	return shared.PrintOutputContext(ctx, resp, outputFlag, pretty)
}

func parseTestersJSON(value string) (androidpublisher.Testers, bool, error) {
	raw, err := shared.LoadJSONArgRaw(value)
	if err != nil {
		return androidpublisher.Testers{}, false, err
	}
	var input struct {
		GoogleGroups []string `json:"googleGroups"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return androidpublisher.Testers{}, false, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return androidpublisher.Testers{}, false, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return androidpublisher.Testers{}, false, err
	}
	_, present := fields["googleGroups"]
	return androidpublisher.Testers{GoogleGroups: input.GoogleGroups}, present, nil
}

func flagWasSet(fs *flag.FlagSet, name string) bool {
	set := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			set = true
		}
	})
	return set
}
