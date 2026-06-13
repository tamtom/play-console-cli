package expansion

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/tamtom/play-console-cli/internal/playclient"
)

func TestExpansionCommand_HasUpdate(t *testing.T) {
	cmd := ExpansionCommand()
	expected := map[string]bool{
		"get":    false,
		"upload": false,
		"patch":  false,
		"update": false,
	}
	for _, sub := range cmd.Subcommands {
		if _, ok := expected[sub.Name]; ok {
			expected[sub.Name] = true
		}
	}
	for name, found := range expected {
		if !found {
			t.Fatalf("missing subcommand %q", name)
		}
	}
}

func TestExpansionCommand_NoArgsReturnsHelp(t *testing.T) {
	cmd := ExpansionCommand()
	if err := cmd.Exec(context.Background(), nil); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp, got %v", err)
	}
}

func TestExpansionUpdateCommand_CallsAPI(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	installMockExpansionPlayService(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"referencesVersion":123}`)
	})

	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--package", "com.example.app",
		"--edit", "edit-1",
		"--apk-version", "456",
		"--type", "main",
		"--references-version", "123",
	}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	stdout, err := captureExpansionStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Fatalf("method = %s, want PUT", gotMethod)
	}
	if gotPath != "/androidpublisher/v3/applications/com.example.app/edits/edit-1/apks/456/expansionFiles/main" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if !strings.Contains(gotBody, `"referencesVersion":123`) {
		t.Fatalf("expected referencesVersion in body, got %s", gotBody)
	}
	if !strings.Contains(stdout, "123") {
		t.Fatalf("expected expansion file output, got %s", stdout)
	}
}

func installMockExpansionPlayService(t *testing.T, handler http.HandlerFunc) {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	original := newPlayService
	newPlayService = func(ctx context.Context) (*playclient.Service, error) {
		return playclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
	}
	t.Cleanup(func() {
		newPlayService = original
	})
}

func captureExpansionStdout(fn func() error) (string, error) {
	origStdout := os.Stdout
	rOut, wOut, err := os.Pipe()
	if err != nil {
		return "", err
	}

	os.Stdout = wOut

	var buf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(&buf, rOut)
	}()

	runErr := fn()

	_ = wOut.Close()
	os.Stdout = origStdout
	wg.Wait()
	_ = rOut.Close()

	return buf.String(), runErr
}
