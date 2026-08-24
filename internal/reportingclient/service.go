package reportingclient

import (
	"context"
	"errors"
	"net/http"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/playdeveloperreporting/v1beta1"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/config"
)

var scopes = []string{
	"https://www.googleapis.com/auth/playdeveloperreporting",
}

// Service wraps the Play Developer Reporting service and config.
type Service struct {
	API        *playdeveloperreporting.Service
	HTTPClient *http.Client
	BasePath   string
	Cfg        *config.Config
}

type (
	ServiceFactory           func(context.Context) (*Service, error)
	serviceFactoryContextKey struct{}
)

func ContextWithServiceFactory(ctx context.Context, factory ServiceFactory) context.Context {
	if factory == nil {
		return ctx
	}
	return context.WithValue(ctx, serviceFactoryContextKey{}, factory)
}

// NewService creates an authenticated Play Developer Reporting service.
func NewService(ctx context.Context) (*Service, error) {
	if factory, ok := ctx.Value(serviceFactoryContextKey{}).(ServiceFactory); ok && factory != nil {
		return factory(ctx)
	}
	cfg, err := config.Load()
	if err != nil && !errors.Is(err, config.ErrNotFound) {
		return nil, shared.NewActionableError(
			"failed to load config",
			err,
			"Check that your config file is valid JSON and readable. Use `gplay auth init` to recreate it.",
		)
	}
	client, err := newHTTPClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	api, err := playdeveloperreporting.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	return &Service{API: api, HTTPClient: client, BasePath: api.BasePath, Cfg: cfg}, nil
}

// NewServiceWithClient creates a reporting service using a provided HTTP client.
// This is useful for tests that need to point the generated API client at a mock server.
func NewServiceWithClient(ctx context.Context, client *http.Client, basePath string) (*Service, error) {
	api, err := playdeveloperreporting.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	if basePath != "" {
		api.BasePath = basePath
	}
	return &Service{API: api, HTTPClient: client, BasePath: api.BasePath, Cfg: &config.Config{}}, nil
}

func newHTTPClient(ctx context.Context, cfg *config.Config) (*http.Client, error) {
	creds, err := resolveCredentials(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return oauth2.NewClient(ctx, creds.TokenSource), nil
}
