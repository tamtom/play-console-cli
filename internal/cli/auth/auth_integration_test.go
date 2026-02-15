//go:build integration

package auth

import (
	"bytes"
	"context"
	"os"
	"testing"
)

func TestAuthDoctor_Integration(t *testing.T) {
	// Skip unless integration env var is also set (safety net)
	if v := os.Getenv("GPLAY_INTEGRATION_TEST"); v != "1" && v != "true" {
		t.Skip("skipping integration test; set GPLAY_INTEGRATION_TEST=1 to run")
	}

	saPath := os.Getenv("GPLAY_SERVICE_ACCOUNT")
	if saPath == "" {
		t.Skip("skipping: GPLAY_SERVICE_ACCOUNT not set")
	}

	cmd := DoctorCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOutput(&stdout)

	err := cmd.ParseAndRun(context.Background(), []string{"--service-account", saPath})
	if err != nil {
		t.Fatalf("auth doctor failed: %v", err)
	}

	_ = stderr // available for future assertions
}
