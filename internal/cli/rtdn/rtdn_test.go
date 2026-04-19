package rtdn

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeRunner struct {
	lookPathErr error
	responses   []response
	calls       [][]string
	idx         int
}

type response struct {
	out []byte
	err error
}

func (f *fakeRunner) LookPath(string) (string, error) {
	if f.lookPathErr != nil {
		return "", f.lookPathErr
	}
	return "/usr/bin/gcloud", nil
}

func (f *fakeRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	if f.idx >= len(f.responses) {
		return nil, nil
	}
	r := f.responses[f.idx]
	f.idx++
	return r.out, r.err
}

func TestRtdnCommandStructure(t *testing.T) {
	cmd := RtdnCommand()
	if cmd.Name != "rtdn" {
		t.Errorf("name=%q", cmd.Name)
	}
	want := map[string]bool{"setup": false, "status": false, "decode": false}
	for _, sub := range cmd.Subcommands {
		if _, ok := want[sub.Name]; ok {
			want[sub.Name] = true
		}
	}
	for k, v := range want {
		if !v {
			t.Errorf("missing subcommand: %s", k)
		}
	}
}

func TestSetupRequiresProject(t *testing.T) {
	err := runSetup(context.Background(), SetupOptions{
		Runner: &fakeRunner{},
		Stdout: io.Discard,
	})
	if err == nil || !strings.Contains(err.Error(), "--project") {
		t.Fatalf("expected --project error, got %v", err)
	}
}

func TestSetupRequiresGcloud(t *testing.T) {
	err := runSetup(context.Background(), SetupOptions{
		Project: "p",
		Runner:  &fakeRunner{lookPathErr: errors.New("missing")},
		Stdout:  io.Discard,
	})
	if err == nil || !strings.Contains(err.Error(), "gcloud") {
		t.Fatalf("expected gcloud error, got %v", err)
	}
}

func TestSetupDryRunPrintsSteps(t *testing.T) {
	out := &bytes.Buffer{}
	r := &fakeRunner{}
	err := runSetup(context.Background(), SetupOptions{
		Project: "my-proj",
		Topic:   "custom",
		DryRun:  true,
		Runner:  r,
		Stdout:  out,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.calls) != 0 {
		t.Errorf("dry-run shouldn't execute; got %d calls", len(r.calls))
	}
	if !strings.Contains(out.String(), "projects/my-proj/topics/custom") {
		t.Errorf("expected topic resource in output: %s", out.String())
	}
}

func TestSetupExecutesSteps(t *testing.T) {
	r := &fakeRunner{}
	err := runSetup(context.Background(), SetupOptions{
		Project: "p",
		Topic:   "play-rtdn",
		Runner:  r,
		Stdout:  io.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.calls) != 2 {
		t.Fatalf("expected 2 gcloud calls, got %d", len(r.calls))
	}
}

func TestSetupIgnoresAlreadyExists(t *testing.T) {
	r := &fakeRunner{
		responses: []response{
			{err: errors.New("Topic projects/p/topics/play-rtdn already exists.")},
			{},
		},
	}
	err := runSetup(context.Background(), SetupOptions{
		Project: "p",
		Runner:  r,
		Stdout:  io.Discard,
	})
	if err != nil {
		t.Fatalf("expected tolerant behavior, got %v", err)
	}
}

func TestStatusDetectsBinding(t *testing.T) {
	policy := `{"bindings":[{"members":["serviceAccount:google-play-developer-notifications@system.gserviceaccount.com"]}]}`
	r := &fakeRunner{responses: []response{{out: []byte(policy)}}}
	buf := &bytes.Buffer{}
	err := runStatus(context.Background(), StatusOptions{
		Project: "p",
		Topic:   "play-rtdn",
		Runner:  r,
		Stdout:  buf,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "[OK]") {
		t.Errorf("expected OK marker, got %s", buf.String())
	}
}

func TestStatusMissingBinding(t *testing.T) {
	r := &fakeRunner{responses: []response{{out: []byte(`{"bindings":[]}`)}}}
	buf := &bytes.Buffer{}
	err := runStatus(context.Background(), StatusOptions{
		Project: "p",
		Topic:   "play-rtdn",
		Runner:  r,
		Stdout:  buf,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "[WARN]") {
		t.Errorf("expected WARN marker, got %s", buf.String())
	}
}

func TestDecodeFromFile(t *testing.T) {
	inner := map[string]any{
		"packageName": "com.ex",
		"subscriptionNotification": map[string]any{
			"notificationType": float64(2),
		},
	}
	innerBytes, _ := json.Marshal(inner)
	env := map[string]any{
		"message": map[string]any{
			"data": base64.StdEncoding.EncodeToString(innerBytes),
		},
	}
	payload, _ := json.Marshal(env)
	path := filepath.Join(t.TempDir(), "p.json")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := decodeCommand()
	_ = cmd.FlagSet.Parse([]string{"--file", path, "--output", "json"})
	if err := cmd.Exec(context.Background(), nil); err != nil {
		t.Fatalf("decode: %v", err)
	}
}

func TestDecodeRequiresInput(t *testing.T) {
	cmd := decodeCommand()
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("expected required error, got %v", err)
	}
}

func TestLoadPayloadStdin(t *testing.T) {
	stdin := strings.NewReader(`{"message":{"data":""}}`)
	got, err := loadPayload("-", "", stdin)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "message") {
		t.Errorf("unexpected: %s", got)
	}
}
