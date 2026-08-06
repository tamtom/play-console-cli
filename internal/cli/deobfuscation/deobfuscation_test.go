package deobfuscation

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/tamtom/play-console-cli/internal/playclient"
)

func TestDeobfuscationCommand_NoArgsReturnsHelp(t *testing.T) {
	cmd := DeobfuscationCommand()
	if err := cmd.Exec(context.Background(), nil); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp, got %v", err)
	}
}

// TestUploadCommand_SendsExactAPICasing pins the casing of the
// deobfuscationFileType path segment. The Play API rejects a lowercased
// "nativecode" with HTTP 400, so --type must be matched case-insensitively but
// sent in the exact casing the API defines.
func TestUploadCommand_SendsExactAPICasing(t *testing.T) {
	tests := []struct {
		name     string
		flagType string
		wantPath string
	}{
		{name: "nativeCode as documented", flagType: "nativeCode", wantPath: "nativeCode"},
		{name: "nativecode lowercased by user", flagType: "nativecode", wantPath: "nativeCode"},
		{name: "NATIVECODE uppercased by user", flagType: "NATIVECODE", wantPath: "nativeCode"},
		{name: "proguard", flagType: "proguard", wantPath: "proguard"},
		{name: "ProGuard mixed case", flagType: "ProGuard", wantPath: "proguard"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotPath string
			installMockDeobfuscationPlayService(t, func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"deobfuscationFile":{"symbolType":"nativeCode"}}`)
			})

			cmd := UploadCommand()
			if err := cmd.FlagSet.Parse([]string{
				"--package", "com.example.app",
				"--edit", "edit-1",
				"--apk-version", "42",
				"--file", writeTempMappingFile(t),
				"--type", tc.flagType,
			}); err != nil {
				t.Fatalf("parse flags: %v", err)
			}

			if _, err := captureDeobfuscationStdout(func() error {
				return cmd.Exec(context.Background(), nil)
			}); err != nil {
				t.Fatalf("exec: %v", err)
			}

			wantSuffix := "/deobfuscationFiles/" + tc.wantPath
			if !strings.HasSuffix(gotPath, wantSuffix) {
				t.Fatalf("request path %q does not end with %q", gotPath, wantSuffix)
			}
		})
	}
}

func TestUploadCommand_RejectsUnknownType(t *testing.T) {
	cmd := UploadCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--package", "com.example.app",
		"--edit", "edit-1",
		"--apk-version", "42",
		"--file", writeTempMappingFile(t),
		"--type", "bogus",
	}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--type must be") {
		t.Fatalf("expected --type validation error, got %v", err)
	}
}

func TestUploadCommand_RequiredFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "missing edit",
			args:    []string{"--package", "com.example.app", "--apk-version", "42", "--file", "x"},
			wantErr: "--edit is required",
		},
		{
			name:    "missing apk-version",
			args:    []string{"--package", "com.example.app", "--edit", "edit-1", "--file", "x"},
			wantErr: "--apk-version is required",
		},
		{
			name:    "missing file",
			args:    []string{"--package", "com.example.app", "--edit", "edit-1", "--apk-version", "42"},
			wantErr: "--file is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := UploadCommand()
			if err := cmd.FlagSet.Parse(tc.args); err != nil {
				t.Fatalf("parse flags: %v", err)
			}
			err := cmd.Exec(context.Background(), nil)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func writeTempMappingFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mapping.txt")
	if err := os.WriteFile(path, []byte("com.example.Foo -> a:\n"), 0o600); err != nil {
		t.Fatalf("writing temp mapping file: %v", err)
	}
	return path
}

func installMockDeobfuscationPlayService(t *testing.T, handler http.HandlerFunc) {
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

func captureDeobfuscationStdout(fn func() error) (string, error) {
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
