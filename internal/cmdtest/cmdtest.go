package cmdtest

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

// Result holds the output of a CLI command execution.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// BinaryPath is set by TestMain to point to the compiled gplay binary.
var BinaryPath string

var (
	buildOnce sync.Once
	buildErr  error
	buildDir  string
)

// Run executes the gplay binary with the given arguments and returns the result.
func Run(t *testing.T, args ...string) Result {
	t.Helper()
	if BinaryPath == "" {
		t.Fatal("cmdtest.BinaryPath not set; call cmdtest.Build in TestMain")
	}

	cmd := exec.Command(BinaryPath, args...) // #nosec G204 -- BinaryPath is set by test infrastructure, not user input
	cmd.Env = append(cmd.Environ(), "GPLAY_NO_UPDATE=1")

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		exitCode = -1
	}

	return Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}
}

// RunJSON runs the binary and parses stdout as JSON.
func RunJSON(t *testing.T, args ...string) (map[string]interface{}, Result) {
	t.Helper()
	r := Run(t, args...)
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(r.Stdout), &parsed); err != nil {
		t.Fatalf("failed to parse JSON from stdout: %v\nstdout: %s\nstderr: %s", err, r.Stdout, r.Stderr)
	}
	return parsed, r
}

// Build compiles the gplay binary into a temp directory and sets BinaryPath.
func Build(t *testing.T) {
	t.Helper()
	buildOnce.Do(func() {
		buildDir, buildErr = os.MkdirTemp("", "gplay-cmdtest-binary-*")
		if buildErr != nil {
			return
		}
		binary := buildDir + "/gplay"
		if os.PathSeparator == '\\' {
			binary += ".exe"
		}
		cmd := exec.Command("go", "build", "-o", binary, ".") // #nosec G204 -- building our own project binary for tests
		cmd.Dir = findProjectRoot()
		var out []byte
		out, buildErr = cmd.CombinedOutput()
		if buildErr != nil {
			buildErr = fmt.Errorf("failed to build gplay binary: %w\n%s", buildErr, out)
			return
		}
		BinaryPath = binary
	})
	if buildErr != nil {
		t.Fatal(buildErr)
	}
}

// Cleanup removes the shared black-box binary directory after a package test
// run. The binary intentionally outlives each individual test.
func Cleanup() {
	if buildDir != "" {
		_ = os.RemoveAll(buildDir)
	}
}

// findProjectRoot walks up from cwd to find go.mod
func findProjectRoot() string {
	cmd := exec.Command("go", "env", "GOMOD")
	out, err := cmd.Output()
	if err != nil {
		return "."
	}
	modPath := strings.TrimSpace(string(out))
	if modPath == "" {
		return "."
	}
	idx := strings.LastIndex(modPath, "/")
	if idx < 0 {
		idx = strings.LastIndex(modPath, "\\")
	}
	if idx >= 0 {
		return modPath[:idx]
	}
	return "."
}

// AssertExitCode asserts that the exit code matches expected.
func AssertExitCode(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("exit code = %d, want %d", got, want)
	}
}

// AssertStderrContains asserts that stderr contains the given substring.
func AssertStderrContains(t *testing.T, stderr, substring string) {
	t.Helper()
	if !strings.Contains(stderr, substring) {
		t.Errorf("stderr does not contain %q\nstderr: %s", substring, stderr)
	}
}

// AssertStdoutContains asserts that stdout contains the given substring.
func AssertStdoutContains(t *testing.T, stdout, substring string) {
	t.Helper()
	if !strings.Contains(stdout, substring) {
		t.Errorf("stdout does not contain %q\nstdout: %s", substring, stdout)
	}
}
