package playclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
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

// The override attaches the OAuth bearer token to every request it sends.
// A non-loopback host would therefore receive a live Google access token,
// so NewService must refuse it. It must fail closed: silently falling back
// to the real API would let a "hermetic" test hit production instead.
func TestNewService_RefusesNonLoopbackBaseURL(t *testing.T) {
	for _, base := range []string{
		"https://evil.example.com",
		"http://10.0.0.5:8080",
		"https://androidpublisher.googleapis.com.evil.example",
		"not-a-url",
		"file:///etc/passwd",
	} {
		t.Run(base, func(t *testing.T) {
			t.Setenv("GPLAY_CONFIG_PATH", filepath.Join(t.TempDir(), "config.json"))
			t.Setenv("GPLAY_SERVICE_ACCOUNT_JSON", testutil.SandboxServiceAccount(t, "https://oauth2.googleapis.com/token"))
			t.Setenv("GPLAY_API_BASE_URL", base)

			_, err := NewService(context.Background())
			if err == nil {
				t.Fatalf("base URL %q must be refused", base)
			}
			if !strings.Contains(err.Error(), "GPLAY_API_BASE_URL") {
				t.Fatalf("error must name the variable, got: %v", err)
			}
		})
	}
}

func TestNewService_AcceptsLoopbackBaseURL(t *testing.T) {
	for _, base := range []string{
		"http://127.0.0.1:8080",
		"http://localhost:9999",
		"http://[::1]:8080",
	} {
		t.Run(base, func(t *testing.T) {
			t.Setenv("GPLAY_CONFIG_PATH", filepath.Join(t.TempDir(), "config.json"))
			t.Setenv("GPLAY_SERVICE_ACCOUNT_JSON", testutil.SandboxServiceAccount(t, "https://oauth2.googleapis.com/token"))
			t.Setenv("GPLAY_API_BASE_URL", base)

			service, err := NewService(context.Background())
			if err != nil {
				t.Fatalf("loopback base URL %q must be accepted: %v", base, err)
			}
			if service.API.BasePath != base+"/" {
				t.Fatalf("BasePath = %q, want %q", service.API.BasePath, base+"/")
			}
		})
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
