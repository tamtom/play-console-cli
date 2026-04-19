package auth

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/config"
)

type fakeRunner struct {
	lookPathErr error
	calls       [][]string
	// responses keyed by first two args joined by space (e.g. "gcloud config").
	responses map[string]runResponse
}

type runResponse struct {
	stdout []byte
	err    error
}

func (f *fakeRunner) LookPath(name string) (string, error) {
	if f.lookPathErr != nil {
		return "", f.lookPathErr
	}
	return "/usr/local/bin/" + name, nil
}

func (f *fakeRunner) Run(ctx context.Context, stdin []byte, name string, args ...string) ([]byte, error) {
	argv := append([]string{name}, args...)
	f.calls = append(f.calls, argv)
	for key, resp := range f.responses {
		parts := strings.Fields(key)
		if matchesPrefix(argv, parts) {
			return resp.stdout, resp.err
		}
	}
	return nil, nil
}

func matchesPrefix(argv, prefix []string) bool {
	if len(prefix) > len(argv) {
		return false
	}
	for i, p := range prefix {
		if argv[i] != p {
			return false
		}
	}
	return true
}

func writeFakeKey(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(map[string]string{
		"type":         "service_account",
		"client_email": "play-console-cli@my-project.iam.gserviceaccount.com",
	})
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestRunSetupRequiresAuto(t *testing.T) {
	err := RunSetup(context.Background(), SetupOptions{Auto: false}, os.Stdout)
	if err == nil || !strings.Contains(err.Error(), "--auto") {
		t.Fatalf("expected guidance about --auto, got %v", err)
	}
}

func TestRunSetupRequiresGcloud(t *testing.T) {
	runner := &fakeRunner{lookPathErr: errors.New("not found")}
	err := RunSetup(context.Background(), SetupOptions{Auto: true, Runner: runner}, os.Stdout)
	if err == nil || !strings.Contains(err.Error(), "gcloud") {
		t.Fatalf("expected gcloud error, got %v", err)
	}
}

func TestRunSetupRequiresProject(t *testing.T) {
	runner := &fakeRunner{
		responses: map[string]runResponse{
			"gcloud config get-value project": {stdout: []byte("(unset)\n")},
		},
	}
	err := RunSetup(context.Background(), SetupOptions{Auto: true, Runner: runner}, os.Stdout)
	if err == nil || !strings.Contains(err.Error(), "project") {
		t.Fatalf("expected project error, got %v", err)
	}
}

func TestRunSetupDryRunPrintsSteps(t *testing.T) {
	runner := &fakeRunner{
		responses: map[string]runResponse{
			"gcloud config get-value project": {stdout: []byte("my-proj\n")},
		},
	}
	tmp := t.TempDir()
	keyOut := filepath.Join(tmp, "sa.json")
	opts := SetupOptions{
		Auto:    true,
		Runner:  runner,
		DryRun:  true,
		KeyOut:  keyOut,
		HomeDir: func() (string, error) { return tmp, nil },
		Output:  "json",
	}
	if err := RunSetup(context.Background(), opts, os.Stdout); err != nil {
		t.Fatalf("RunSetup: %v", err)
	}
	if _, err := os.Stat(keyOut); err == nil {
		t.Error("dry-run should not write key file")
	}
}

func TestRunSetupHappyPathCreatesKeyAndConfig(t *testing.T) {
	tmp := t.TempDir()
	keyOut := filepath.Join(tmp, "sa.json")
	runner := &fakeRunner{
		responses: map[string]runResponse{
			"gcloud config get-value project": {stdout: []byte("my-proj\n")},
			// describe returns error -> triggers create
			"gcloud iam service-accounts describe": {err: errors.New("not found")},
			// keys create "creates" the key (the test writes it)
			"gcloud iam service-accounts keys create": {},
		},
	}
	saved := false
	opts := SetupOptions{
		Auto:    true,
		Project: "my-proj",
		Runner:  runner,
		KeyOut:  keyOut,
		HomeDir: func() (string, error) { return tmp, nil },
		SaveConfig: func(profile config.Profile, setDefault bool) (string, error) {
			saved = true
			return filepath.Join(tmp, "config.json"), nil
		},
		Output: "json",
	}

	// Pre-write the key so that post-key validation passes. In real flow
	// gcloud writes the file. We simulate that by writing it just before the
	// validate step via runner hook — simpler: write ahead of time.
	writeFakeKey(t, keyOut)

	if err := RunSetup(context.Background(), opts, os.Stdout); err != nil {
		t.Fatalf("RunSetup: %v", err)
	}
	if !saved {
		t.Error("expected SaveConfig to be called")
	}
	// gcloud should have been invoked at least 4 times.
	if len(runner.calls) < 4 {
		t.Errorf("expected >=4 gcloud calls, got %d", len(runner.calls))
	}
}

func TestValidateServiceAccountKeyRejectsBadJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateServiceAccountKey(path); err == nil {
		t.Error("expected error for bad json")
	}
}

func TestValidateServiceAccountKeyRejectsWrongType(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oauth.json")
	if err := os.WriteFile(path, []byte(`{"type":"oauth2","client_email":"x@y"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateServiceAccountKey(path); err == nil {
		t.Error("expected error for wrong type")
	}
}

func TestValidateServiceAccountKeyOK(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sa.json")
	writeFakeKey(t, path)
	if err := validateServiceAccountKey(path); err != nil {
		t.Errorf("expected ok, got %v", err)
	}
}

func TestAuthSetupCommandRegistered(t *testing.T) {
	cmd := AuthCommand()
	found := false
	for _, sub := range cmd.Subcommands {
		if sub.Name == "setup" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected setup subcommand registered on auth")
	}
}
