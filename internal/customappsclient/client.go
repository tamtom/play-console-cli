// Package customappsclient provides authenticated access to the Google Play
// Custom App Publishing API (playcustomapp). This is the API behind Managed
// Google Play private app distribution: it publishes an APK as a custom app
// scoped to one or more organizations. It shares the androidpublisher OAuth
// scope, so this client reuses playclient's credential resolution instead of
// duplicating it.
package customappsclient

import (
	"context"
	"net/http"

	"google.golang.org/api/option"
	playcustomapp "google.golang.org/api/playcustomapp/v1"

	"github.com/tamtom/play-console-cli/internal/config"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

// scopeAndroidPublisher is the OAuth scope required by the Play Custom App
// Publishing API — the same scope the rest of gplay already uses.
const scopeAndroidPublisher = "https://www.googleapis.com/auth/androidpublisher"

// Service wraps the Play Custom App Publishing API surface and the loaded config.
type Service struct {
	API *playcustomapp.Service
	Cfg *config.Config
}

// NewService creates an authenticated Play Custom App Publishing service using
// the same credentials as the Android Publisher API.
func NewService(ctx context.Context) (*Service, error) {
	client, cfg, err := playclient.NewAuthenticatedClientWithScopes(ctx, scopeAndroidPublisher)
	if err != nil {
		return nil, err
	}
	return newServiceFromClient(ctx, client, cfg, "")
}

// NewServiceWithClient builds the service from a caller-provided HTTP client.
// Tests use this to point the generated client at a mock server.
func NewServiceWithClient(ctx context.Context, client *http.Client, basePath string) (*Service, error) {
	return newServiceFromClient(ctx, client, &config.Config{}, basePath)
}

func newServiceFromClient(ctx context.Context, client *http.Client, cfg *config.Config, basePath string) (*Service, error) {
	api, err := playcustomapp.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	if basePath != "" {
		api.BasePath = basePath
	}
	return &Service{API: api, Cfg: cfg}, nil
}
