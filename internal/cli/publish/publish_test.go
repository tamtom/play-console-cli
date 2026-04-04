package publish

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/cli/release"
	validatecli "github.com/tamtom/play-console-cli/internal/cli/validate"
	"github.com/tamtom/play-console-cli/internal/validation"
)

func TestPublishCommand_HasTrackSubcommand(t *testing.T) {
	cmd := PublishCommand()
	if cmd.Name != "publish" {
		t.Fatalf("Name = %q, want publish", cmd.Name)
	}
	if len(cmd.Subcommands) != 1 || cmd.Subcommands[0].Name != "track" {
		t.Fatalf("expected publish track subcommand, got %#v", cmd.Subcommands)
	}
}

func TestTrackCommand_PreflightBlockingStopsPublish(t *testing.T) {
	origBuild := buildReadinessReportFn
	origExecute := executeReleaseFn
	buildReadinessReportFn = func(context.Context, validatecli.ReadinessOptions) *validation.ReadinessReport {
		report := &validation.ReadinessReport{}
		report.AddCheck(validation.ReadinessCheck{
			ID:      "blocking",
			Section: "artifact",
			State:   validation.ReadinessBlocking,
			Message: "broken",
		})
		return report
	}
	executeCalled := false
	executeReleaseFn = func(context.Context, release.Options) (map[string]interface{}, error) {
		executeCalled = true
		return nil, nil
	}
	t.Cleanup(func() {
		buildReadinessReportFn = origBuild
		executeReleaseFn = origExecute
	})

	cmd := TrackCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--bundle", "app.aab"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected blocking preflight error")
	}
	if executeCalled {
		t.Fatal("release execution should not run when preflight blocks publish")
	}
}

func TestTrackCommand_Success(t *testing.T) {
	origBuild := buildReadinessReportFn
	origExecute := executeReleaseFn
	buildReadinessReportFn = func(context.Context, validatecli.ReadinessOptions) *validation.ReadinessReport {
		return readyReport()
	}
	executeReleaseFn = func(context.Context, release.Options) (map[string]interface{}, error) {
		return map[string]interface{}{"track": "production", "versionCode": int64(42)}, nil
	}
	t.Cleanup(func() {
		buildReadinessReportFn = origBuild
		executeReleaseFn = origExecute
	})

	cmd := TrackCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--bundle", "app.aab"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	stdout, err := captureOutput(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}
	if !strings.Contains(stdout, "\"published\":true") {
		t.Fatalf("expected published output, got %s", stdout)
	}
	if !strings.Contains(stdout, "\"preflight\"") {
		t.Fatalf("expected preflight payload, got %s", stdout)
	}
}

func captureOutput(fn func() error) (string, error) {
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	defer r.Close()

	os.Stdout = w
	runErr := fn()
	_ = w.Close()
	os.Stdout = origStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String(), runErr
}
