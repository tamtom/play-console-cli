// Package gamesclient provides authenticated access to the Play Game Services
// Publishing API (gamesConfiguration) and the Play Games Services Management API
// (gamesManagement). Both share the androidpublisher OAuth scope, so this client
// reuses playclient's credential resolution instead of duplicating it.
package gamesclient

import (
	"context"
	"net/http"
	"os"
	"strings"

	gamesconfiguration "google.golang.org/api/gamesconfiguration/v1configuration"
	gamesmanagement "google.golang.org/api/gamesmanagement/v1management"
	"google.golang.org/api/option"

	"github.com/tamtom/play-console-cli/internal/config"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

// GamesApplicationIDEnvVar is the environment variable that supplies the numeric
// Play Games application (project) ID when the --application-id flag is omitted.
const GamesApplicationIDEnvVar = "GPLAY_GAMES_APP_ID"

// OAuth scopes required by the Play Games APIs. gamesConfiguration (publishing)
// uses the androidpublisher scope; gamesManagement (player progress/hide) uses
// the games scope. A single token requesting both covers every service this
// client builds.
const (
	scopeAndroidPublisher = "https://www.googleapis.com/auth/androidpublisher"
	scopeGames            = "https://www.googleapis.com/auth/games"
)

// Service wraps the two Play Games API surfaces and the loaded config.
type Service struct {
	// Configuration manages achievement and leaderboard definitions
	// (the "content" of the Play Games section in Play Console).
	Configuration *gamesconfiguration.Service
	// Management performs player/progress operations (hide players, reset
	// achievement/score/event progress).
	Management *gamesmanagement.Service
	Cfg        *config.Config
}

// NewService creates an authenticated Play Games service using the same
// credentials as the Android Publisher API.
func NewService(ctx context.Context) (*Service, error) {
	client, cfg, err := playclient.NewAuthenticatedClientWithScopes(ctx, scopeAndroidPublisher, scopeGames)
	if err != nil {
		return nil, err
	}
	return newServiceFromClient(ctx, client, cfg, "")
}

// NewServiceWithClient builds the service from a caller-provided HTTP client.
// Tests use this to point the generated clients at a mock server.
func NewServiceWithClient(ctx context.Context, client *http.Client, basePath string) (*Service, error) {
	return newServiceFromClient(ctx, client, &config.Config{}, basePath)
}

func newServiceFromClient(ctx context.Context, client *http.Client, cfg *config.Config, basePath string) (*Service, error) {
	confSvc, err := gamesconfiguration.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	mgmtSvc, err := gamesmanagement.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	if basePath != "" {
		confSvc.BasePath = basePath
		mgmtSvc.BasePath = basePath
	}
	return &Service{Configuration: confSvc, Management: mgmtSvc, Cfg: cfg}, nil
}

// ResolveApplicationID returns the Play Games application ID from the flag, the
// GPLAY_GAMES_APP_ID environment variable, or config, in that order.
func ResolveApplicationID(flagValue string, cfg *config.Config) string {
	if v := strings.TrimSpace(flagValue); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv(GamesApplicationIDEnvVar)); v != "" {
		return v
	}
	if cfg != nil && strings.TrimSpace(cfg.GamesApplicationID) != "" {
		return strings.TrimSpace(cfg.GamesApplicationID)
	}
	return ""
}
