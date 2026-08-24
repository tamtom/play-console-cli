package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/tamtom/play-console-cli/internal/audit"
	cliruntime "github.com/tamtom/play-console-cli/internal/cli/runtime"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

type recordingAuditSink struct {
	entries []audit.Entry
}

func (s *recordingAuditSink) Enabled() bool { return true }
func (s *recordingAuditSink) Write(entry audit.Entry) error {
	s.entries = append(s.entries, entry)
	return nil
}

func TestRun_VersionFlag(t *testing.T) {
	code := Run([]string{"--version"}, "1.0.0")
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestRun_VersionFlagOutput(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_ = Run([]string{"--version"}, "1.0.0 (commit: abc123, date: 2024-01-01)")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("reading pipe: %v", err)
	}
	output := buf.String()

	if output == "" {
		t.Fatal("expected version output, got empty string")
	}
	if !bytes.Contains([]byte(output), []byte("1.0.0")) {
		t.Errorf("expected output to contain '1.0.0', got %q", output)
	}
}

func TestRunWithRuntimeInjectsIOClockAndAudit(t *testing.T) {
	var stdout, stderr bytes.Buffer
	sink := &recordingAuditSink{}
	wantTime := time.Date(2042, time.March, 4, 5, 6, 7, 0, time.UTC)
	code := RunWithRuntime([]string{"version"}, "1.2.3", func(rt *cliruntime.Runtime) {
		rt.WithIO(&stdout, &stderr).
			WithClock(shared.ClockFunc(func() time.Time { return wantTime })).
			WithAuditSink(sink)
	})
	if code != ExitSuccess {
		t.Fatalf("exit code = %d", code)
	}
	if strings.TrimSpace(stdout.String()) != "1.2.3" || stderr.Len() != 0 {
		t.Fatalf("injected output = %q, %q", stdout.String(), stderr.String())
	}
	if len(sink.entries) != 1 || sink.entries[0].Command != "gplay version" || !sink.entries[0].Timestamp.Equal(wantTime) {
		t.Fatalf("audit entries = %#v", sink.entries)
	}
}

func TestRunWithRuntimeRoutesUsageErrorsToInjectedStderr(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := RunWithRuntime([]string{"workflow", "validate"}, "1.2.3", func(rt *cliruntime.Runtime) {
		rt.WithIO(&stdout, &stderr).WithAuditSink(nil)
	})
	if code != ExitUsage {
		t.Fatalf("exit code = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(stderr.String(), "workflow name or file path is required") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRun_NoArgs(t *testing.T) {
	code := Run([]string{}, "1.0.0")
	if code != ExitUsage {
		t.Errorf("expected exit code %d (ExitUsage), got %d", ExitUsage, code)
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	code := Run([]string{"nonexistent"}, "1.0.0")
	if code != ExitUsage {
		t.Errorf("expected exit code %d (ExitUsage), got %d", ExitUsage, code)
	}
}

func TestIsVersionOnlyInvocation(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "double-dash version", args: []string{"--version"}, want: true},
		{name: "single-dash version", args: []string{"-version"}, want: true},
		{name: "no args", args: []string{}, want: false},
		{name: "version subcommand", args: []string{"version"}, want: false},
		{name: "version with extra args", args: []string{"--version", "extra"}, want: false},
		{name: "other flag", args: []string{"--help"}, want: false},
		{name: "version flag not first", args: []string{"apps", "--version"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isVersionOnlyInvocation(tt.args)
			if got != tt.want {
				t.Errorf("isVersionOnlyInvocation(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestConstructRootCommandForArgs_MaterializesOnlySelectedFamily(t *testing.T) {
	root, _ := constructRootCommandForArgs("test", []string{"--profile", "ci", "rollout", "update", "--help"})

	var selectedFound bool
	for _, command := range root.Subcommands {
		switch command.Name {
		case "rollout":
			selectedFound = true
			if len(command.Subcommands) == 0 {
				t.Fatal("selected rollout command was not materialized")
			}
		case "apps":
			if len(command.Subcommands) != 0 || command.FlagSet != nil {
				t.Fatal("unselected apps command was unexpectedly materialized")
			}
			if !strings.Contains(command.ShortHelp, "service account") {
				t.Fatalf("unselected metadata help was lost: %q", command.ShortHelp)
			}
		}
	}
	if !selectedFound {
		t.Fatal("selected rollout command is missing")
	}
}

func TestScrubArgsRedactsJSONPayloadsAndFiles(t *testing.T) {
	for _, args := range [][]string{
		{"checks", "repo-scans", "generate", "--json", `{"source":"private"}`},
		{"checks", "repo-scans", "generate", "--json=@/private/scan.json"},
	} {
		scrubbed := strings.Join(scrubArgs(args), " ")
		if strings.Contains(scrubbed, "private") || !strings.Contains(scrubbed, "<redacted>") {
			t.Fatalf("scrubbed args disclose JSON input: %q", scrubbed)
		}
	}
}
