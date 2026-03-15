package cmdtest_test

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"sync"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/registry"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// RootCommand constructs the full gplay command tree for in-process testing.
// This avoids needing to compile a binary.
func RootCommand(version string) *ffcli.Command {
	rootFS := flag.NewFlagSet("gplay", flag.ContinueOnError)
	// Use ContinueOnError so tests can check parse errors without os.Exit

	var root *ffcli.Command
	root = &ffcli.Command{
		Name:        "gplay",
		ShortUsage:  "gplay <command> [flags]",
		ShortHelp:   "A CLI for Google Play Console.",
		FlagSet:     rootFS,
		Subcommands: registry.Subcommands(version),
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			var names []string
			for _, sub := range root.Subcommands {
				names = append(names, sub.Name)
			}
			fmt.Fprintln(os.Stderr, shared.FormatUnknownCommand(args[0], names))
			return flag.ErrHelp
		},
	}
	return root
}

// captureOutput redirects os.Stdout and os.Stderr during fn() execution,
// returning the captured output. This enables testing command output without
// running a subprocess.
func captureOutput(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()

	// Save originals
	origStdout := os.Stdout
	origStderr := os.Stderr

	// Create pipes
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating stdout pipe: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating stderr pipe: %v", err)
	}

	// Redirect
	os.Stdout = wOut
	os.Stderr = wErr

	// Capture in goroutines
	var outBuf, errBuf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(&outBuf, rOut)
	}()
	go func() {
		defer wg.Done()
		io.Copy(&errBuf, rErr)
	}()

	// Run the function
	fn()

	// Close writers and restore
	wOut.Close()
	wErr.Close()
	os.Stdout = origStdout
	os.Stderr = origStderr

	// Wait for goroutines to finish reading
	wg.Wait()
	rOut.Close()
	rErr.Close()

	return outBuf.String(), errBuf.String()
}
