package appsigningclient

import (
	"context"
	"fmt"
	"net/http"

	"github.com/tamtom/play-console-cli/internal/config"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

const androidPublisherBaseURL = "https://androidpublisher.googleapis.com/"

// Service owns the official App Signing REST adapter and resolved CLI config.
type Service struct {
	API *Client
	Cfg *config.Config
}

// ServiceFactory creates an App Signing service. Runtime wiring installs one
// in tests so the high-impact endpoint can never fall through to host auth.
type ServiceFactory func(context.Context) (*Service, error)

type serviceFactoryContextKey struct{}

// ContextWithServiceFactory installs a command-scoped service factory.
func ContextWithServiceFactory(ctx context.Context, factory ServiceFactory) context.Context {
	if factory == nil {
		return ctx
	}
	return context.WithValue(ctx, serviceFactoryContextKey{}, factory)
}

// NewService creates the official authenticated App Signing REST adapter.
func NewService(ctx context.Context) (*Service, error) {
	if factory, ok := ctx.Value(serviceFactoryContextKey{}).(ServiceFactory); ok && factory != nil {
		return factory(ctx)
	}
	client, cfg, err := playclient.NewAuthenticatedClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("authenticate App Signing service: %w", err)
	}
	return NewServiceWithClient(client, androidPublisherBaseURL, cfg), nil
}

// NewServiceWithClient builds a service on a caller-provided transport.
func NewServiceWithClient(client *http.Client, baseURL string, cfg *config.Config) *Service {
	if cfg == nil {
		cfg = &config.Config{}
	}
	return &Service{API: New(client, baseURL), Cfg: cfg}
}
