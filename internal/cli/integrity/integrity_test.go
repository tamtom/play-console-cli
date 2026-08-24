package integrity

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/tamtom/play-console-cli/internal/integrityclient"
)

func TestDecodeRequiresTokenBeforeAuthentication(t *testing.T) {
	cmd := DecodeAndroidCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "dev.example"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--token") {
		t.Fatalf("expected token error, got %v", err)
	}
}

func TestDeviceRecallRequiresSecurityAcknowledgement(t *testing.T) {
	cmd := DeviceRecallWriteCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "dev.example", "--json", `{"integrityToken":"x","newValues":{"bitFirst":true}}`, "--confirm"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--security-fraud-abuse-use") {
		t.Fatalf("expected use restriction error, got %v", err)
	}
}

func TestIntegrityEndpoints_SuccessAndAPIError(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "token.txt")
	if err := os.WriteFile(tokenFile, []byte("encoded-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	recallFile := filepath.Join(t.TempDir(), "recall.json")
	if err := os.WriteFile(recallFile, []byte(`{"integrityToken":"encoded-token","newValues":{"bitFirst":true}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		command    func() *ffcli.Command
		args       []string
		path       string
		errorLabel string
	}{
		{"android", DecodeAndroidCommand, []string{"--package", "dev.example", "--token-file", tokenFile}, "/v1/dev.example:decodeIntegrityToken", "decode integrity token"},
		{"pc", DecodePCCommand, []string{"--package", "dev.example", "--token-file", tokenFile}, "/v1/dev.example:decodePcIntegrityToken", "decode PC integrity token"},
		{"device-recall", DeviceRecallWriteCommand, []string{"--package", "dev.example", "--json", "@" + recallFile, "--security-fraud-abuse-use", "--confirm"}, "/v1/dev.example/deviceRecall:write", "write Device Recall state"},
	}

	for _, tc := range tests {
		t.Run(tc.name+" success", func(t *testing.T) {
			var gotPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{}`)
			}))
			t.Cleanup(server.Close)
			installIntegrityService(t, server)
			cmd := tc.command()
			if err := cmd.FlagSet.Parse(tc.args); err != nil {
				t.Fatal(err)
			}
			if err := cmd.Exec(context.Background(), nil); err != nil {
				t.Fatal(err)
			}
			if gotPath != tc.path {
				t.Fatalf("path = %q, want %q", gotPath, tc.path)
			}
		})

		t.Run(tc.name+" API error", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, `{"error":{"message":"denied"}}`, http.StatusForbidden)
			}))
			t.Cleanup(server.Close)
			installIntegrityService(t, server)
			cmd := tc.command()
			if err := cmd.FlagSet.Parse(tc.args); err != nil {
				t.Fatal(err)
			}
			err := cmd.Exec(context.Background(), nil)
			if err == nil || !strings.Contains(err.Error(), tc.errorLabel) {
				t.Fatalf("expected contextual API error containing %q, got %v", tc.errorLabel, err)
			}
		})
	}
}

func installIntegrityService(t *testing.T, server *httptest.Server) {
	t.Helper()
	original := newIntegrityService
	newIntegrityService = func(ctx context.Context) (*integrityclient.Service, error) {
		return integrityclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
	}
	t.Cleanup(func() { newIntegrityService = original })
}
