package cmd

import (
	"context"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/tamtom/play-console-cli/internal/audit"
	cliruntime "github.com/tamtom/play-console-cli/internal/cli/runtime"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/cli/shared/errfmt"
	"github.com/tamtom/play-console-cli/internal/config"
	"github.com/tamtom/play-console-cli/internal/update"
	"golang.org/x/term"
)

// Run is the main entry point. It returns an exit code.
func Run(args []string, versionInfo string) int {
	return RunWithRuntime(args, versionInfo, nil)
}

// RunWithRuntime executes the CLI after applying optional runtime dependency
// overrides. Production calls Run; tests use this seam to fail closed on I/O,
// time, audit, and client boundaries.
func RunWithRuntime(args []string, versionInfo string, configure func(*cliruntime.Runtime)) int {
	// Build root metadata and materialize only the selected command family.
	root, rt := constructRootCommandForArgs(versionInfo, args)
	if configure != nil {
		configure(rt)
	}

	// Signal handling for graceful Ctrl+C
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	var err error
	ctx, err = rt.ApplyRootContext(ctx)
	if err != nil {
		fmt.Fprintln(shared.Stderr(ctx), err)
		return ExitUsage
	}
	setCommandOutput(root, shared.Stderr(ctx))

	// Fast path: --version flag. Runtime dependencies are still honored so the
	// root execution seam remains fully testable.
	if isVersionOnlyInvocation(args) {
		fmt.Fprintln(shared.Stdout(ctx), versionInfo)
		return ExitSuccess
	}

	// Parse flags and subcommands
	if err := root.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitSuccess
		}
		// Every flag set uses ContinueOnError (see setCommandOutput), so the
		// flag package already printed the error and the usage to stderr.
		return ExitUsage
	}

	ctx, err = rt.ApplyRootContext(ctx)
	if err != nil {
		fmt.Fprintln(shared.Stderr(ctx), err)
		return ExitUsage
	}

	if legacy := config.IgnoredLegacyGlobalPath(); legacy != "" {
		fmt.Fprintf(shared.Stderr(ctx), "Warning: %s is ignored. gplay reads config.json in the same directory. Run gplay auth login --service-account <path> to create it.\n", legacy)
	}

	// Record start time for JUnit reporting
	startTime := shared.Now(ctx)

	// Determine command name for reporting
	commandName := getCommandName(args)

	// Execute
	runErr := root.Run(ctx)

	elapsed := shared.Now(ctx).Sub(startTime)

	logAudit(ctx, rt, commandName, args, runErr, elapsed)

	// Write JUnit report if requested
	if rt.RootFlags != nil &&
		rt.RootFlags.Report != nil && strings.ToLower(strings.TrimSpace(*rt.RootFlags.Report)) == "junit" &&
		rt.RootFlags.ReportFile != nil && strings.TrimSpace(*rt.RootFlags.ReportFile) != "" {
		if reportErr := writeJUnitReport(shared.FilesystemFrom(ctx), *rt.RootFlags.ReportFile, commandName, runErr, elapsed); reportErr != nil {
			fmt.Fprintf(shared.Stderr(ctx), "Error: failed to write JUnit report: %v\n", reportErr)
			if runErr == nil {
				return ExitError
			}
		}
	}

	if runErr != nil {
		if errors.Is(runErr, flag.ErrHelp) {
			var usageErr *shared.CommandUsageError
			if errors.As(runErr, &usageErr) {
				fmt.Fprintln(shared.Stderr(ctx), usageErr.Error())
			}
			return ExitUsage
		}
		if !shared.IsReportedError(runErr) {
			fmt.Fprintln(shared.Stderr(ctx), shared.RedactURLsInText(errfmt.FormatStderr(runErr)))
		}
		return ExitCodeFromError(runErr)
	}

	maybeSuggestUpdate(ctx, args, versionInfo)
	return ExitSuccess
}

func maybeSuggestUpdate(ctx context.Context, args []string, versionInfo string) {
	terminal, ok := shared.Stderr(ctx).(*os.File)
	if shared.IsDryRun(ctx) || update.Disabled() || os.Getenv("CI") != "" || (!ok || !term.IsTerminal(int(terminal.Fd()))) {
		return
	}
	switch selectedRootCommand(args) {
	case "", "version", "completion", "update":
		return
	}
	fields := strings.Fields(versionInfo)
	if len(fields) == 0 {
		return
	}
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	info, err := update.CheckForUpdate(checkCtx, update.Options{CurrentVersion: fields[0]})
	if err == nil {
		update.PrintUpdateMessageTo(shared.Stderr(ctx), info)
	}
}

func setCommandOutput(command *ffcli.Command, writer io.Writer) {
	if command == nil {
		return
	}
	if command.FlagSet == nil {
		command.FlagSet = flag.NewFlagSet(command.Name, flag.ContinueOnError)
	}
	command.FlagSet.Init(command.FlagSet.Name(), flag.ContinueOnError)
	command.FlagSet.SetOutput(writer)
	for _, subcommand := range command.Subcommands {
		setCommandOutput(subcommand, writer)
	}
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

// logAudit writes an audit entry for the completed command invocation.
// Errors are swallowed so the audit log never breaks a user command.
func logAudit(ctx context.Context, rt *cliruntime.Runtime, commandName string, args []string, runErr error, elapsed time.Duration) {
	sink := rt.AuditSink()
	if sink == nil || !sink.Enabled() {
		return
	}
	// Skip logging the audit command itself to avoid self-noise.
	if strings.HasPrefix(commandName, "gplay audit") {
		return
	}
	entry := audit.Entry{
		Timestamp: shared.Now(ctx).UTC(),
		Command:   commandName,
		Args:      scrubArgs(args),
		Status:    "ok",
		DurationM: elapsed.Milliseconds(),
	}
	if runErr != nil && !errors.Is(runErr, flag.ErrHelp) {
		entry.Status = "error"
		// API errors can echo credentials from files or responses, not just flags.
		// Persist only the classification; keep detailed diagnostics on stderr.
		entry.Error = string(errfmt.Classify(runErr).Category)
	}
	_ = sink.Write(entry)
}

// scrubArgs removes flag values that might contain secrets.
func scrubArgs(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	sensitive := func(name string) bool {
		name = strings.ToLower(strings.TrimLeft(name, "-"))
		switch name {
		case "service-account", "service-account-json", "webhook-url", "json", "data", "body", "payload", "authorization":
			return true
		}
		for _, part := range []string{"token", "secret", "password", "key", "credential"} {
			if strings.Contains(name, part) {
				return true
			}
		}
		return false
	}
	// keyValue reports flags that take KEY=VALUE pairs. The key stays in the
	// log; the value is redacted, because it can hold a secret.
	keyValue := func(name string) bool {
		return strings.ToLower(strings.TrimLeft(name, "-")) == "param"
	}
	redactPair := func(pair string) string {
		if eq := strings.IndexByte(pair, '='); eq >= 0 {
			return pair[:eq] + "=<redacted>"
		}
		return "<redacted>"
	}
	out := make([]string, 0, len(args))
	skipNext, pairNext := false, false
	for _, a := range args {
		if skipNext {
			out = append(out, "<redacted>")
			skipNext = false
			continue
		}
		if pairNext {
			out = append(out, redactPair(a))
			pairNext = false
			continue
		}
		if eq := strings.IndexByte(a, '='); eq > 0 {
			if strings.HasPrefix(a, "-") && sensitive(a[:eq]) {
				out = append(out, a[:eq]+"=<redacted>")
				continue
			}
			if strings.HasPrefix(a, "-") && keyValue(a[:eq]) {
				out = append(out, a[:eq]+"="+redactPair(a[eq+1:]))
				continue
			}
		} else if strings.HasPrefix(a, "-") && sensitive(a) {
			out = append(out, a)
			skipNext = true
			continue
		} else if strings.HasPrefix(a, "-") && keyValue(a) {
			out = append(out, a)
			pairNext = true
			continue
		}
		out = append(out, a)
	}
	return out
}

// writeJUnitReport writes a JUnit XML report for CI integration.
func writeJUnitReport(filesystem shared.Filesystem, reportFile, commandName string, runErr error, elapsed time.Duration) error {
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
			Message: shared.RedactURLsInText(runErr.Error()),
			Text:    shared.RedactURLsInText(runErr.Error()),
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
	return filesystem.AtomicWriteFile(reportFile, content, 0o644, 0o755)
}
