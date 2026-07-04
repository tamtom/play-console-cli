package playclient

import (
	"context"
	"errors"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/option"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/config"
)

var scopes = []string{"https://www.googleapis.com/auth/androidpublisher"}

// Service wraps the Android Publisher service and config.
type Service struct {
	API *androidpublisher.Service
	Cfg *config.Config
}

// NewService creates an authenticated Android Publisher service.
func NewService(ctx context.Context) (*Service, error) {
	client, cfg, err := NewAuthenticatedClient(ctx)
	if err != nil {
		return nil, err
	}
	api, err := androidpublisher.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	return &Service{API: api, Cfg: cfg}, nil
}

// NewAuthenticatedClient returns an HTTP client authenticated with the
// androidpublisher OAuth scope, together with the loaded config. Sibling
// clients that share this scope (e.g. the Play Games publishing API) build
// their generated services on top of this client instead of duplicating the
// credential-resolution logic. The transport is wrapped for dry-run when
// dry-run mode is active, mirroring NewService.
func NewAuthenticatedClient(ctx context.Context) (*http.Client, *config.Config, error) {
	return NewAuthenticatedClientWithScopes(ctx)
}

// NewAuthenticatedClientWithScopes is like NewAuthenticatedClient but requests
// an explicit set of OAuth scopes. Pass no scopes to default to the
// androidpublisher scope. Sibling APIs that need additional scopes (e.g. the
// Play Games management API additionally requires the games scope) pass the
// full list so a single token covers every service they build.
func NewAuthenticatedClientWithScopes(ctx context.Context, scopeList ...string) (*http.Client, *config.Config, error) {
	cfg, err := config.Load()
	if err != nil && !errors.Is(err, config.ErrNotFound) {
		return nil, nil, shared.NewActionableError(
			"failed to load config",
			err,
			"Check that your config file is valid JSON and readable. Use `gplay auth init` to recreate it.",
		)
	}
	client, err := newHTTPClient(ctx, cfg, scopeList...)
	if err != nil {
		return nil, nil, err
	}

	// Wrap transport with DryRunTransport when dry-run is active.
	if shared.IsDryRun(ctx) {
		client.Transport = &shared.DryRunTransport{
			Base:   client.Transport,
			Writer: os.Stderr,
		}
	}
	return client, cfg, nil
}

// NewServiceWithClient creates an Android Publisher service using a provided
// HTTP client. Tests use this to point the generated client at a mock server.
func NewServiceWithClient(ctx context.Context, client *http.Client, basePath string) (*Service, error) {
	api, err := androidpublisher.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	if basePath != "" {
		api.BasePath = basePath
	}
	return &Service{API: api, Cfg: &config.Config{}}, nil
}

func newHTTPClient(ctx context.Context, cfg *config.Config, scopeList ...string) (*http.Client, error) {
	creds, err := resolveCredentials(ctx, cfg, scopeList...)
	if err != nil {
		return nil, err
	}
	return oauth2.NewClient(ctx, creds.TokenSource), nil
}
