package shared

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v3/ffcli"
)

// PaginatedListCommandConfig configures a standard paginated list command.
type PaginatedListCommandConfig struct {
	Name       string
	ShortUsage string
	ShortHelp  string
	LongHelp   string
	// ExtraFlags registers additional command-specific flags.
	ExtraFlags func(fs *flag.FlagSet)
	// Exec is called with the parsed pagination and output settings.
	Exec func(ctx context.Context, pageSize int, pageToken string, paginate bool, output *OutputFlags) error
}

// BuildPaginatedListCommand creates a standard list command with --page-size,
// --paginate, --next, --output, and --pretty flags.
func BuildPaginatedListCommand(cfg PaginatedListCommandConfig) *ffcli.Command {
	fs := flag.NewFlagSet(cfg.Name, flag.ExitOnError)
	pageSize := fs.Int("page-size", 25, "Number of items per page")
	paginate := fs.Bool("paginate", false, "Automatically fetch all pages")
	next := fs.String("next", "", "Page token for the next page of results")
	output := BindOutputFlags(fs)

	if cfg.ExtraFlags != nil {
		cfg.ExtraFlags(fs)
	}

	return &ffcli.Command{
		Name:       cfg.Name,
		ShortUsage: cfg.ShortUsage,
		ShortHelp:  cfg.ShortHelp,
		LongHelp:   cfg.LongHelp,
		FlagSet:    fs,
		UsageFunc:  DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			token := ""
			if next != nil {
				token = *next
			}
			return cfg.Exec(ctx, *pageSize, token, *paginate, output)
		},
	}
}

// ConfirmDeleteCommandConfig configures a standard delete command with --confirm.
type ConfirmDeleteCommandConfig struct {
	Name       string
	ShortUsage string
	ShortHelp  string
	LongHelp   string
	// ExtraFlags registers additional command-specific flags.
	ExtraFlags func(fs *flag.FlagSet)
	// Exec is called only if --confirm is true.
	Exec func(ctx context.Context) error
}

// BuildConfirmDeleteCommand creates a standard delete command requiring --confirm.
func BuildConfirmDeleteCommand(cfg ConfirmDeleteCommandConfig) *ffcli.Command {
	fs := flag.NewFlagSet(cfg.Name, flag.ExitOnError)
	confirm := fs.Bool("confirm", false, "Confirm the deletion")

	if cfg.ExtraFlags != nil {
		cfg.ExtraFlags(fs)
	}

	return &ffcli.Command{
		Name:       cfg.Name,
		ShortUsage: cfg.ShortUsage,
		ShortHelp:  cfg.ShortHelp,
		LongHelp:   cfg.LongHelp,
		FlagSet:    fs,
		UsageFunc:  DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if !*confirm {
				fmt.Fprintln(os.Stderr, "Error: --confirm is required for destructive operations")
				return flag.ErrHelp
			}
			return cfg.Exec(ctx)
		},
	}
}
