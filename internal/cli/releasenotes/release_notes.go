package releasenotes

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	rn "github.com/tamtom/play-console-cli/internal/releasenotes"
)

// ReleaseNotesCommand returns the parent "release-notes" command group.
func ReleaseNotesCommand() *ffcli.Command {
	fs := flag.NewFlagSet("release-notes", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "release-notes",
		ShortUsage: "gplay release-notes <subcommand> [flags]",
		ShortHelp:  "Generate release notes from git history.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			GenerateCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// GenerateCommand returns the "release-notes generate" subcommand.
func GenerateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("release-notes generate", flag.ExitOnError)
	sinceTag := fs.String("since-tag", "", "Start from this git tag (exclusive)")
	sinceRef := fs.String("since-ref", "", "Start from this git ref (exclusive, alternative to --since-tag)")
	untilRef := fs.String("until-ref", "HEAD", "End at this ref (inclusive, default: HEAD)")
	maxChars := fs.Int("max-chars", 500, "Maximum character count (Google Play limit: 500)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")

	return &ffcli.Command{
		Name:       "generate",
		ShortUsage: "gplay release-notes generate --since-tag <tag> [flags]",
		ShortHelp:  "Generate release notes from git commit history.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			return runGenerate(ctx, generateOpts{
				sinceTag:   *sinceTag,
				sinceRef:   *sinceRef,
				untilRef:   *untilRef,
				maxChars:   *maxChars,
				outputFlag: *outputFlag,
			})
		},
	}
}

type generateOpts struct {
	sinceTag   string
	sinceRef   string
	untilRef   string
	maxChars   int
	outputFlag string
}

// generateResult is the JSON-serializable output.
type generateResult struct {
	ReleaseNotes string         `json:"release_notes"`
	CommitCount  int            `json:"commit_count"`
	Truncated    bool           `json:"truncated"`
	Commits      []rn.GitCommit `json:"commits"`
}

func runGenerate(ctx context.Context, opts generateOpts) error {
	// Validate flags
	sinceTag := strings.TrimSpace(opts.sinceTag)
	sinceRef := strings.TrimSpace(opts.sinceRef)

	if sinceTag != "" && sinceRef != "" {
		fmt.Fprintln(os.Stderr, "Error: --since-tag and --since-ref are mutually exclusive")
		return flag.ErrHelp
	}
	if sinceTag == "" && sinceRef == "" {
		fmt.Fprintln(os.Stderr, "Error: one of --since-tag or --since-ref is required")
		return flag.ErrHelp
	}

	ref := sinceRef
	if sinceTag != "" {
		ref = sinceTag
	}

	commits, err := rn.GitLog(ctx, ref, opts.untilRef)
	if err != nil {
		return fmt.Errorf("generating release notes: %w", err)
	}

	notes := rn.Format(commits, opts.maxChars)
	fullNotes := rn.Format(commits, 0)

	result := generateResult{
		ReleaseNotes: notes,
		CommitCount:  len(commits),
		Truncated:    len(notes) < len(fullNotes),
		Commits:      commits,
	}

	return shared.PrintOutput(result, opts.outputFlag, false)
}
