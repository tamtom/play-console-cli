package playclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/tamtom/play-console-cli/internal/testutil"
)

// GPLAY_API_BASE_URL points the generated client at a local sandbox server.
// The override exists for hermetic black-box tests only.

func TestNewService_HonorsBaseURLOverride(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"sandbox","token_type":"Bearer","expires_in":3600}`))
			return
		}
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"kind":"androidpublisher#appsListResponse"}`))
	}))
	defer srv.Close()

	t.Setenv("GPLAY_CONFIG_PATH", filepath.Join(t.TempDir(), "config.json"))
	t.Setenv("GPLAY_SERVICE_ACCOUNT_JSON", testutil.SandboxServiceAccount(t, srv.URL+"/token"))
	t.Setenv("GPLAY_API_BASE_URL", srv.URL)

	service, err := NewService(context.Background())
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	if _, err := service.API.Users.List("developers/1").Do(); err != nil {
		t.Fatalf("call through override: %v", err)
	}
	if gotPath == "" {
		t.Fatal("the request did not reach the sandbox server")
	}
}

func TestNewService_NoOverrideKeepsGoogleBasePath(t *testing.T) {
	t.Setenv("GPLAY_CONFIG_PATH", filepath.Join(t.TempDir(), "config.json"))
	t.Setenv("GPLAY_SERVICE_ACCOUNT_JSON", testutil.SandboxServiceAccount(t, "https://oauth2.googleapis.com/token"))
	t.Setenv("GPLAY_API_BASE_URL", "")

	service, err := NewService(context.Background())
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if service.API.BasePath != "https://androidpublisher.googleapis.com/" {
		t.Fatalf("BasePath = %q, want the Google default", service.API.BasePath)
	}
}
