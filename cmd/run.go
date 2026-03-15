package cmd

import (
	"context"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/cli/shared/errfmt"
)

// Run is the main entry point. It returns an exit code.
func Run(args []string, versionInfo string) int {
	// Fast path: --version flag
	if isVersionOnlyInvocation(args) {
		fmt.Fprintln(os.Stdout, versionInfo)
		return ExitSuccess
	}

	// Build command tree
	root := RootCommand(versionInfo)

	// Signal handling for graceful Ctrl+C
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Parse flags and subcommands
	if err := root.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return ExitCodeFromError(err)
	}

	// Apply root flags to environment
	rootFlags.Apply()

	// Validate report flags
	if err := rootFlags.ValidateReportFlags(); err != nil {
		return ExitUsage
	}

	// Apply dry-run context
	if rootFlags.DryRun != nil && *rootFlags.DryRun {
		ctx = shared.ContextWithDryRun(ctx, true)
	}

	// Record start time for JUnit reporting
	startTime := time.Now()

	// Determine command name for reporting
	commandName := getCommandName(args)

	// Execute
	runErr := root.Run(ctx)

	elapsed := time.Since(startTime)

	// Write JUnit report if requested
	if rootFlags.Report != nil && strings.ToLower(strings.TrimSpace(*rootFlags.Report)) == "junit" &&
		rootFlags.ReportFile != nil && strings.TrimSpace(*rootFlags.ReportFile) != "" {
		if reportErr := writeJUnitReport(*rootFlags.ReportFile, commandName, runErr, elapsed); reportErr != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to write JUnit report: %v\n", reportErr)
			if runErr == nil {
				return ExitError
			}
		}
	}

	if runErr != nil {
		if errors.Is(runErr, flag.ErrHelp) {
			return ExitUsage
		}
		if !shared.IsReportedError(runErr) {
			fmt.Fprintln(os.Stderr, errfmt.FormatStderr(runErr))
		}
		return ExitCodeFromError(runErr)
	}

	return ExitSuccess
}

// isVersionOnlyInvocation returns true if the args are exactly ["--version"].
func isVersionOnlyInvocation(args []string) bool {
	return len(args) == 1 && (args[0] == "--version" || args[0] == "-version")
}

// getCommandName extracts a human-readable command name from the args.
func getCommandName(args []string) string {
	var parts []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			break
		}
		parts = append(parts, arg)
	}
	if len(parts) == 0 {
		return "gplay"
	}
	return "gplay " + strings.Join(parts, " ")
}

// writeJUnitReport writes a JUnit XML report for CI integration.
func writeJUnitReport(reportFile, commandName string, runErr error, elapsed time.Duration) error {
	type junitTestCase struct {
		XMLName   xml.Name `xml:"testcase"`
		Name      string   `xml:"name,attr"`
		ClassName string   `xml:"classname,attr"`
		Time      string   `xml:"time,attr"`
		Failure   *struct {
			Message string `xml:"message,attr"`
			Text    string `xml:",chardata"`
		} `xml:"failure,omitempty"`
	}
	type junitTestSuite struct {
		XMLName  xml.Name        `xml:"testsuite"`
		Name     string          `xml:"name,attr"`
		Tests    int             `xml:"tests,attr"`
		Failures int             `xml:"failures,attr"`
		Time     string          `xml:"time,attr"`
		Cases    []junitTestCase `xml:"testcase"`
	}

	tc := junitTestCase{
		Name:      commandName,
		ClassName: commandName,
		Time:      fmt.Sprintf("%.3f", elapsed.Seconds()),
	}

	failures := 0
	if runErr != nil {
		failures = 1
		tc.Failure = &struct {
			Message string `xml:"message,attr"`
			Text    string `xml:",chardata"`
		}{
			Message: runErr.Error(),
			Text:    runErr.Error(),
		}
	}

	suite := junitTestSuite{
		Name:     "gplay",
		Tests:    1,
		Failures: failures,
		Time:     fmt.Sprintf("%.3f", elapsed.Seconds()),
		Cases:    []junitTestCase{tc},
	}

	data, err := xml.MarshalIndent(suite, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JUnit XML: %w", err)
	}

	content := []byte(xml.Header + string(data) + "\n")
	return os.WriteFile(reportFile, content, 0o644)
}
