// Package developeridclient provides access to the official Android
// Developer ID Status API through a context-injectable service boundary.
package developeridclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"google.golang.org/api/androiddeveloperidstatus/v1"
	"google.golang.org/api/option"
	transport "google.golang.org/api/transport/http"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/config"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

type Service struct {
	API *androiddeveloperidstatus.Service
	Cfg *config.Config
}

type (
	ServiceFactory           func(context.Context, string) (*Service, error)
	serviceFactoryContextKey struct{}
)

func ContextWithServiceFactory(ctx context.Context, factory ServiceFactory) context.Context {
	if factory == nil {
		return ctx
	}
	return context.WithValue(ctx, serviceFactoryContextKey{}, factory)
}

func NewService(ctx context.Context, apiKey string) (*Service, error) {
	if factory, ok := ctx.Value(serviceFactoryContextKey{}).(ServiceFactory); ok && factory != nil {
		return factory(ctx, apiKey)
	}
	client, _, err := transport.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("create Android Developer ID Status service: %w", err)
	}
	return NewServiceWithClient(ctx, client, "")
}

// NewServiceWithClient creates a testable service that targets basePath.
func NewServiceWithClient(ctx context.Context, client *http.Client, basePath string) (*Service, error) {
	cfg, err := config.Load()
	if err != nil && !errors.Is(err, config.ErrNotFound) {
		return nil, fmt.Errorf("load config: %w", err)
	}
	copyClient := *client
	timeout, _ := shared.ParseTimeouts(cfg)
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	copyClient.Timeout = timeout
	if err := playclient.ApplyRetryPolicy(&copyClient, cfg); err != nil {
		return nil, err
	}
	client = &copyClient
	api, err := androiddeveloperidstatus.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("create Android Developer ID Status test service: %w", err)
	}
	if basePath != "" {
		api.BasePath = basePath
	}
	return &Service{API: api, Cfg: cfg}, nil
}
