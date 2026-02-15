package listings

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

// localesResponse is the JSON output for the locales command.
type localesResponse struct {
	Locales []string `json:"locales"`
	Total   int      `json:"total"`
}

// LocalesCommand returns the listings locales subcommand.
func LocalesCommand() *ffcli.Command {
	fs := flag.NewFlagSet("listings locales", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID (if omitted, creates a temporary edit)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "locales",
		ShortUsage: "gplay listings locales --package <name> [flags]",
		ShortHelp:  "List supported locales for an app's store listings.",
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

			eid := strings.TrimSpace(*editID)
			tempEdit := false
			if eid == "" {
				// Create a temporary edit to list locales.
				edit, err := service.API.Edits.Insert(pkg, nil).Context(ctx).Do()
				if err != nil {
					return fmt.Errorf("creating temporary edit: %w", err)
				}
				eid = edit.Id
				tempEdit = true
			}

			call := service.API.Edits.Listings.List(pkg, eid).Context(ctx)
			resp, err := call.Do()
			if err != nil {
				return err
			}

			// Clean up temporary edit (best-effort, ignore errors).
			if tempEdit {
				_ = service.API.Edits.Delete(pkg, eid).Context(ctx).Do()
			}

			locales := make([]string, 0, len(resp.Listings))
			for _, listing := range resp.Listings {
				locales = append(locales, listing.Language)
			}

			result := localesResponse{
				Locales: locales,
				Total:   len(locales),
			}

			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}
