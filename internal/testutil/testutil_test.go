package testutil

import (
	"encoding/json"
	"os"
	"testing"
)

func TestMockServiceAccount(t *testing.T) {
	path := MockServiceAccount(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading mock service account: %v", err)
	}
	var sa map[string]string
	if err := json.Unmarshal(data, &sa); err != nil {
		t.Fatalf("parsing mock service account: %v", err)
	}
	requiredFields := []string{"type", "project_id", "private_key_id", "private_key", "client_email", "client_id", "auth_uri", "token_uri"}
	for _, field := range requiredFields {
		if sa[field] == "" {
			t.Errorf("missing required field %q", field)
		}
	}
	if sa["type"] != "service_account" {
		t.Errorf("type = %q, want %q", sa["type"], "service_account")
	}
}
