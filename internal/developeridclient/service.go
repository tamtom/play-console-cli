// Package developeridclient provides access to the official Android
// Developer ID Status API through a context-injectable service boundary.
package developeridclient

import (
	"context"
	"fmt"
	"net/http"

	"google.golang.org/api/androiddeveloperidstatus/v1"
	"google.golang.org/api/option"
)

type Service struct {
	API *androiddeveloperidstatus.Service
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
	api, err := androiddeveloperidstatus.NewService(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("create Android Developer ID Status service: %w", err)
	}
	return &Service{API: api}, nil
}

// NewServiceWithClient creates a testable service that targets basePath.
func NewServiceWithClient(ctx context.Context, client *http.Client, basePath string) (*Service, error) {
	api, err := androiddeveloperidstatus.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("create Android Developer ID Status test service: %w", err)
	}
	if basePath != "" {
		api.BasePath = basePath
	}
	return &Service{API: api}, nil
}
