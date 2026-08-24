package verification

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/developeridclient"
)

func TestStatusRequiresAPIKeyBeforeNetwork(t *testing.T) {
	t.Setenv("GPLAY_ANDROID_DEVELOPER_ID_API_KEY", "")
	cmd := StatusCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "dev.example"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--api-key") {
		t.Fatalf("expected API key error, got %v", err)
	}
}

func TestCertificateFingerprintValidation(t *testing.T) {
	cmd := StatusCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "dev.example", "--api-key", "test", "--certificate-fingerprint", "bad"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("expected fingerprint error, got %v", err)
	}
}

func TestStatus_UsesOfficialEndpointAndPropagatesAPIError(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		wantError string
	}{
		{"success", http.StatusOK, ""},
		{"API error", http.StatusForbidden, "check Android developer package registration"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotPath, gotFingerprint, gotKey string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotFingerprint = r.URL.Query().Get("certificateFingerprint")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				if tc.status == http.StatusOK {
					_, _ = io.WriteString(w, `{"packageName":"dev.example"}`)
					return
				}
				_, _ = io.WriteString(w, `{"error":{"message":"denied"}}`)
			}))
			t.Cleanup(server.Close)
			original := newDeveloperIDService
			newDeveloperIDService = func(ctx context.Context, apiKey string) (*developeridclient.Service, error) {
				gotKey = apiKey
				return developeridclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
			}
			t.Cleanup(func() { newDeveloperIDService = original })

			fingerprint := strings.Repeat("a", 64)
			cmd := StatusCommand()
			if err := cmd.FlagSet.Parse([]string{"--package", "dev.example", "--api-key", "test-key", "--certificate-fingerprint", fingerprint}); err != nil {
				t.Fatal(err)
			}
			err := cmd.Exec(context.Background(), nil)
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("expected %q error, got %v", tc.wantError, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if gotPath != "/v1/packages/dev-example/packageRegistrationStatus:check" {
				t.Fatalf("path = %q", gotPath)
			}
			if gotFingerprint != fingerprint || gotKey != "test-key" {
				t.Fatalf("fingerprint/key = %q/%q", gotFingerprint, gotKey)
			}
		})
	}
}
