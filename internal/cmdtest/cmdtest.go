package cmdtest

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
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

// Run executes the gplay binary with the given arguments and returns the result.
func Run(t *testing.T, args ...string) Result {
	t.Helper()
	if BinaryPath == "" {
		t.Fatal("cmdtest.BinaryPath not set; call cmdtest.Build in TestMain")
	}

	cmd := exec.Command(BinaryPath, args...)
	cmd.Env = append(cmd.Environ(), "GPLAY_NO_UPDATE=1")

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
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
	dir := t.TempDir()
	binary := dir + "/gplay"
	if os.PathSeparator == '\\' {
		binary += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", binary, ".")
	cmd.Dir = findProjectRoot()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build gplay binary: %v\n%s", err, out)
	}
	BinaryPath = binary
}

// findProjectRoot walks up from cwd to find go.mod
func findProjectRoot() string {
	// The binary is at the module root
	cmd := exec.Command("go", "env", "GOMOD")
	out, err := cmd.Output()
	if err != nil {
		return "."
	}
	modPath := strings.TrimSpace(string(out))
	if modPath == "" {
		return "."
	}
	// go.mod path -> directory
	idx := strings.LastIndex(modPath, "/")
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
