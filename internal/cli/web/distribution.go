package web

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/websession"
)

// distributionResult is the `web apps distribution` output.
type distributionResult struct {
	PackageName string   `json:"packageName"`
	FormFactors []string `json:"formFactors"`
	Tasks       []string `json:"tasks,omitempty"`
	Added       string   `json:"added,omitempty"`
	Changed     bool     `json:"changed"`
}

// WebAppsDistributionCommand returns the `gplay web apps distribution` subcommand.
func WebAppsDistributionCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web apps distribution", flag.ExitOnError)
	pkg := fs.String("package", "", "Package name (applicationId)")
	add := fs.String("add", "", "Form factor to add, e.g. \"Android TV\" or \"Wear OS\"")
	developerID := fs.String("developer", "", "Developer account ID (numeric; default: auto-discover)")
	account := fs.String("account", "", "Web session account email (default: last used session)")
	confirm := fs.Bool("confirm", false, "Confirm adding the form factor")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "distribution",
		ShortUsage: "gplay web apps distribution --package <id> [--add \"<factor>\" --confirm]",
		ShortHelp:  "Read or add app form factors (Android TV, Wear OS, and more).",
		LongHelp: `Read the app's form factors, or add one.

Without --add this lists the managed form factors and any outstanding
per-factor requirements (screenshots, opt-ins). Adding requires --confirm;
the factor list is re-read to verify. Product-policy opt-ins and other
requirements reported in "tasks" remain explicit follow-up work.

Available factors depend on the app; the menu typically offers Android TV,
Wear OS, Android Automotive OS, Android Auto, and Google Play Games on PC.

This drives the console's Advanced settings because the official Android
Publisher API has no form-factor endpoints. Always confirm with the user.

Examples:
  gplay web apps distribution --package com.example.app
  gplay web apps distribution --package com.example.app --add "Android TV" --confirm`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			packageName := strings.TrimSpace(*pkg)
			if packageName == "" {
				return fmt.Errorf("--package is required")
			}
			factor := strings.TrimSpace(*add)
			write := factor != ""
			if write && !*confirm {
				return fmt.Errorf("--confirm is required to add a form factor")
			}

			if shared.IsDryRun(ctx) {
				fmt.Fprintf(os.Stderr, "[DRY RUN] would %s: package=%s factor=%q\n",
					map[bool]string{true: "add form factor", false: "read form factors"}[write], packageName, factor)
				fmt.Fprintln(os.Stderr, "[DRY RUN] No changes were made.")
				return nil
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, nil)
			defer cancel()
			sess, err := sessionLoad(strings.TrimSpace(*account))
			if err != nil {
				return err
			}
			target, err := resolveWebApp(ctx, newWebClient(sess), *developerID, packageName)
			if err != nil {
				return err
			}
			pb, err := newPublishBrowser(ctx, websession.BrowserProfileDir(), 90*time.Second)
			if err != nil {
				return err
			}
			defer func() { _ = pb.Close() }() //nolint:errcheck // best-effort cleanup

			before, err := pb.ReadDistribution(ctx, target.DeveloperID, target.AppID, sess.UserEmail)
			if err != nil {
				return err
			}
			result := &distributionResult{
				PackageName: packageName,
				FormFactors: before.FormFactors,
				Tasks:       before.Tasks,
			}
			if !write {
				return shared.PrintOutput(result, *outputFlag, *pretty)
			}
			for _, f := range before.FormFactors {
				if strings.EqualFold(f, factor) || strings.EqualFold("Manage "+f, factor) {
					return shared.PrintOutput(result, *outputFlag, *pretty) // already present
				}
			}
			if err := pb.AddFormFactor(ctx, factor); err != nil {
				return err
			}
			after, err := pb.ReadDistribution(ctx, target.DeveloperID, target.AppID, sess.UserEmail)
			if err != nil {
				return err
			}
			found := false
			for _, f := range after.FormFactors {
				if strings.EqualFold(strings.TrimSpace(f), factor) {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("the form factor %q does not appear after adding (current: %s)", factor, strings.Join(after.FormFactors, ", "))
			}
			result.FormFactors = after.FormFactors
			result.Tasks = after.Tasks
			result.Added = factor
			result.Changed = true
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}
