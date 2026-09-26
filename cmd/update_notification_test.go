package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	cliruntime "github.com/tamtom/play-console-cli/internal/cli/runtime"
)

// A terminal subprocess exercises the real startup path, including terminal
// detection and release lookup. Only the external HTTP boundary is replaced.
func TestNormalCommandUpdateSuggestion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("PTY integration runs on macOS and Linux")
	}
	script, err := exec.LookPath("script")
	if err != nil {
		t.Skip("script is required for a real PTY")
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"interactive", "offline", "opt_out", "ci", "noninteractive", "version", "completion", "help", "dry_run"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			args := []string{"-test.run=^TestUpdateNotificationProcess$"}
			var child *exec.Cmd
			if mode == "noninteractive" {
				child = exec.CommandContext(ctx, executable, args...)
			} else if runtime.GOOS == "darwin" {
				child = exec.CommandContext(ctx, script, append([]string{"-q", "/dev/null", executable}, args...)...)
			} else {
				quoted := "'" + strings.ReplaceAll(executable, "'", "'\\''") + "'"
				child = exec.CommandContext(ctx, script, "-q", "-e", "-c", quoted+" "+args[0], "/dev/null")
			}
			home := t.TempDir()
			child.Env = append(os.Environ(), "HOME="+home, "USERPROFILE="+home,
				"GPLAY_CONFIG_PATH="+filepath.Join(home, "config.json"), "GPLAY_NO_UPDATE=", "CI=", "GPLAY_AUDIT=0", "GPLAY_TEST_UPDATE_CASE="+mode)
			out, err := child.CombinedOutput()
			if err != nil || !bytes.Contains(out, []byte("UPDATE_NOTIFICATION_CHECK_OK")) {
				t.Fatalf("child: %v\n%s", err, out)
			}
			hasAdvice := bytes.Contains(out, []byte("A new version of gplay is available: 0.9.0")) && bytes.Contains(out, []byte("gplay update"))
			if hasAdvice != (mode == "interactive") {
				t.Fatalf("unexpected notification: %s", out)
			}
		})
	}
}

type updateHTTPTransport func(*http.Request) (*http.Response, error)

func (f updateHTTPTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestUpdateNotificationProcess(t *testing.T) {
	mode := os.Getenv("GPLAY_TEST_UPDATE_CASE")
	if mode == "" {
		t.Skip("subprocess helper")
	}
	if mode == "opt_out" {
		t.Setenv("GPLAY_NO_UPDATE", "1")
	}
	if mode == "ci" {
		t.Setenv("CI", "true")
	}
	calls := 0
	original := http.DefaultTransport
	defer func() { http.DefaultTransport = original }()
	http.DefaultTransport = updateHTTPTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "GET" || r.URL.Host != "api.github.com" || !strings.HasSuffix(r.URL.Path, "/releases/latest") {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		deadline, ok := r.Context().Deadline()
		if !ok || time.Until(deadline) > 3*time.Second {
			t.Error("unbounded startup lookup")
		}
		if mode == "offline" {
			return nil, fmt.Errorf("offline")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"tag_name":"v1.0.0","html_url":"https://example.test/release"}`))}, nil
	})
	args := []string{"experiments", "support"}
	switch mode {
	case "version":
		args = []string{"version"}
	case "completion":
		args = []string{"completion", "bash"}
	case "help":
		args = []string{"--help"}
	case "dry_run":
		args = append([]string{"--dry-run"}, args...)
	}
	var expected, baselineErr bytes.Buffer
	baseline := RunWithRuntime(args, "0.9.0", func(rt *cliruntime.Runtime) { rt.WithIO(&expected, &baselineErr).WithAuditSink(nil) })
	var actual bytes.Buffer
	code := RunWithRuntime(args, "0.9.0", func(rt *cliruntime.Runtime) { rt.WithIO(&actual, os.Stderr).WithAuditSink(nil) })
	wantCalls := 0
	if mode == "interactive" || mode == "offline" {
		wantCalls = 1
	}
	if code != 0 || code != baseline || calls != wantCalls || actual.String() != expected.String() {
		t.Fatalf("code=%d baseline=%d HTTP calls=%d want=%d stdout changed=%v", code, baseline, calls, wantCalls, actual.String() != expected.String())
	}
	fmt.Println("UPDATE_NOTIFICATION_CHECK_OK")
}
