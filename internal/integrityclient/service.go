// Package integrityclient provides authenticated access to the official Play
// Integrity API using the repository's existing credential resolution.
package integrityclient

import (
	"context"
	"net/http"

	"google.golang.org/api/option"
	"google.golang.org/api/playintegrity/v1"

	"github.com/tamtom/play-console-cli/internal/config"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

const Scope = "https://www.googleapis.com/auth/playintegrity"

type Service struct {
	API *playintegrity.Service
	Cfg *config.Config
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

func NewService(ctx context.Context) (*Service, error) {
	if factory, ok := ctx.Value(serviceFactoryContextKey{}).(ServiceFactory); ok && factory != nil {
		return factory(ctx)
	}
	client, cfg, err := playclient.NewAuthenticatedClientWithScopes(ctx, Scope)
	if err != nil {
		return nil, err
	}
	return newService(ctx, client, cfg, "")
}

func NewServiceWithClient(ctx context.Context, client *http.Client, basePath string) (*Service, error) {
	return newService(ctx, client, &config.Config{}, basePath)
}

func newService(ctx context.Context, client *http.Client, cfg *config.Config, basePath string) (*Service, error) {
	api, err := playintegrity.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	if basePath != "" {
		api.BasePath = basePath
	}
	return &Service{API: api, Cfg: cfg}, nil
}
