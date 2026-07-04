package gamesclient

import (
	"context"
	"net/http"
	"testing"

	"github.com/tamtom/play-console-cli/internal/config"
)

func TestResolveApplicationIDPrecedence(t *testing.T) {
	t.Setenv(GamesApplicationIDEnvVar, "env-app")
	cfg := &config.Config{GamesApplicationID: "config-app"}

	if got := ResolveApplicationID("flag-app", cfg); got != "flag-app" {
		t.Fatalf("ResolveApplicationID flag precedence = %q, want flag-app", got)
	}
	if got := ResolveApplicationID("", cfg); got != "env-app" {
		t.Fatalf("ResolveApplicationID env precedence = %q, want env-app", got)
	}
	t.Setenv(GamesApplicationIDEnvVar, "")
	if got := ResolveApplicationID("", cfg); got != "config-app" {
		t.Fatalf("ResolveApplicationID config fallback = %q, want config-app", got)
	}
	if got := ResolveApplicationID("", &config.Config{}); got != "" {
		t.Fatalf("ResolveApplicationID empty = %q, want empty", got)
	}
}

func TestNewServiceWithClientBuildsBothSurfaces(t *testing.T) {
	svc, err := NewServiceWithClient(context.Background(), http.DefaultClient, "http://example.invalid/")
	if err != nil {
		t.Fatalf("NewServiceWithClient error: %v", err)
	}
	if svc.Configuration == nil {
		t.Fatal("expected gamesConfiguration service to be built")
	}
	if svc.Management == nil {
		t.Fatal("expected gamesManagement service to be built")
	}
	if svc.Configuration.BasePath != "http://example.invalid/" {
		t.Fatalf("configuration BasePath = %q, want override applied", svc.Configuration.BasePath)
	}
	if svc.Management.BasePath != "http://example.invalid/" {
		t.Fatalf("management BasePath = %q, want override applied", svc.Management.BasePath)
	}
}
