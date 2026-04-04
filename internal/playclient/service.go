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

	// Wrap transport with DryRunTransport when dry-run is active.
	if shared.IsDryRun(ctx) {
		client.Transport = &shared.DryRunTransport{
			Base:   client.Transport,
			Writer: os.Stderr,
		}
	}

	api, err := androidpublisher.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	return &Service{API: api, Cfg: cfg}, nil
}

func newHTTPClient(ctx context.Context, cfg *config.Config) (*http.Client, error) {
	creds, err := resolveCredentials(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return oauth2.NewClient(ctx, creds.TokenSource), nil
}
