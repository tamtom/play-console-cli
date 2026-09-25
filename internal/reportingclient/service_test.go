package reportingclient

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewHTTPClient_AppliesRetryPolicy(t *testing.T) {
	tokenPath := filepath.Join(t.TempDir(), "token.json")
	if err := os.WriteFile(tokenPath, []byte(`{"access_token":"token","token_type":"Bearer","expiry":"2099-01-01T00:00:00Z"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(serviceAccountEnvVar, "")
	t.Setenv(oauthTokenEnvVar, tokenPath)
	t.Setenv(oauthClientIDEnvVar, "client")
	t.Setenv(oauthClientSecretEnvVar, "secret")
	t.Setenv("GPLAY_MAX_RETRIES", "invalid")
	_, err := newHTTPClient(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "apply HTTP retry policy") {
		t.Fatalf("error = %v, want retry-policy wiring error", err)
	}
}
