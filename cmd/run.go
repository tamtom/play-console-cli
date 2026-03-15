package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/cli/shared/errfmt"
)

// Run is the main entry point. It returns an exit code.
func Run(args []string, versionInfo string) int {
	// Fast path: --version flag
	if isVersionOnlyInvocation(args) {
		fmt.Fprintln(os.Stdout, versionInfo)
		return 0
	}

	// Build command tree
	root := RootCommand(versionInfo)

	// Signal handling for graceful Ctrl+C
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Parse flags and subcommands
	if err := root.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	// Apply dry-run context
	if rootDryRun != nil && *rootDryRun {
		ctx = shared.ContextWithDryRun(ctx, true)
	}

	// Execute
	if err := root.Run(ctx); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 1
		}
		if !shared.IsReportedError(err) {
			fmt.Fprintln(os.Stderr, errfmt.FormatStderr(err))
		}
		return 1
	}

	return 0
}

// isVersionOnlyInvocation returns true if the args are exactly ["--version"].
func isVersionOnlyInvocation(args []string) bool {
	return len(args) == 1 && (args[0] == "--version" || args[0] == "-version")
}
