package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDefinitionRejectsNullRetryAndTimeoutPolicies(t *testing.T) {
	for _, field := range []string{"retry", "timeout"} {
		t.Run(field, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "workflow.json")
			body := `{"name":"release","steps":[{"name":"publish","run":"true","` + field + `":null}]}`
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatalf("write workflow: %v", err)
			}
			_, err := LoadDefinition(path)
			if err == nil || !strings.Contains(err.Error(), field+" must not be null") {
				t.Fatalf("LoadDefinition error = %v, want explicit null rejection", err)
			}
		})
	}
}
