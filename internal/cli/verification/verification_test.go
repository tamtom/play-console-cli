package verification

import (
	"context"
	"strings"
	"testing"
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
