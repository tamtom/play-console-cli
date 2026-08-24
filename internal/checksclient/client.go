package checksclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/checks/v1alpha"
	"google.golang.org/api/option"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/config"
)

const (
	Scope = "https://www.googleapis.com/auth/checks"

	serviceAccountEnvVar       = "GPLAY_SERVICE_ACCOUNT_JSON"
	serviceAccountPathEnvVar   = "GPLAY_SERVICE_ACCOUNT"
	oauthTokenEnvVar           = "GPLAY_OAUTH_TOKEN_PATH"
	oauthClientIDEnvVar        = "GPLAY_OAUTH_CLIENT_ID"
	oauthClientSecretEnvVar    = "GPLAY_OAUTH_CLIENT_SECRET"
	oauthRedirectEnvVar        = "GPLAY_OAUTH_REDIRECT_URI"
	checksAccountEnvVar        = "GPLAY_CHECKS_ACCOUNT"
	defaultOAuthRedirectURI    = "urn:ietf:wg:oauth:2.0:oob"
	serviceAccountProfileTypes = "service_account, service-account, serviceaccount"
)

// Service wraps the Checks API service and config.
type Service struct {
	API *checks.Service
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

// NewService creates an authenticated Checks API service.
func NewService(ctx context.Context) (*Service, error) {
	if factory, ok := ctx.Value(serviceFactoryContextKey{}).(ServiceFactory); ok && factory != nil {
		return factory(ctx)
	}
	cfg, err := config.Load()
	if err != nil && !errors.Is(err, config.ErrNotFound) {
		return nil, shared.NewActionableError(
			"failed to load config",
			err,
			"Check that your config file is valid JSON and readable. Use `gplay auth login` to recreate it.",
		)
	}
	client, err := newHTTPClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if shared.IsDryRun(ctx) {
		client.Transport = &shared.DryRunTransport{
			Base:   client.Transport,
			Writer: os.Stderr,
		}
	}
	api, err := checks.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	return &Service{API: api, Cfg: cfg}, nil
}

// ResolveAccount returns the Checks account ID from flag, env, or config.
func ResolveAccount(flagValue string, cfg *config.Config) string {
	if v := strings.TrimSpace(flagValue); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv(checksAccountEnvVar)); v != "" {
		return v
	}
	if cfg != nil && strings.TrimSpace(cfg.ChecksAccount) != "" {
		return strings.TrimSpace(cfg.ChecksAccount)
	}
	return ""
}

func newHTTPClient(ctx context.Context, cfg *config.Config) (*http.Client, error) {
	tokenSource, err := resolveTokenSource(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return oauth2.NewClient(ctx, tokenSource), nil
}

func resolveTokenSource(ctx context.Context, cfg *config.Config) (oauth2.TokenSource, error) {
	profileName := shared.ResolveProfileName(cfg)
	if profileName != "" && cfg != nil {
		if profile, ok := findProfile(cfg, profileName); ok {
			tokenSource, err := tokenSourceFromProfile(ctx, profile)
			if err != nil {
				return nil, err
			}
			if shared.StrictAuthEnabled() && envAuthPresent() {
				return nil, shared.NewAuthError(
					"authentication failed",
					fmt.Errorf("strict auth: profile selected but environment credentials also present"),
					"Unset environment credentials or set GPLAY_STRICT_AUTH=false.",
				)
			}
			return tokenSource, nil
		}
		return nil, shared.NewAuthError(
			"authentication failed",
			fmt.Errorf("profile not found: %s", profileName),
			"Run `gplay auth login --profile <name>` or set GPLAY_PROFILE to an existing profile.",
		)
	}

	if envAuthPresent() {
		return tokenSourceFromEnv(ctx)
	}

	return nil, shared.NewAuthError(
		"authentication failed",
		errors.New("no credentials found"),
		"Run `gplay auth login` or set GPLAY_SERVICE_ACCOUNT_JSON / GPLAY_OAUTH_TOKEN_PATH.",
	)
}

func tokenSourceFromProfile(ctx context.Context, profile config.Profile) (oauth2.TokenSource, error) {
	switch strings.ToLower(strings.TrimSpace(profile.Type)) {
	case "service_account", "service-account", "serviceaccount":
		if strings.TrimSpace(profile.KeyPath) == "" {
			return nil, shared.NewAuthError(
				"invalid auth profile",
				errors.New("service account profile missing key_path"),
				"Set key_path in config.json or re-run `gplay auth login` with --service-account.",
			)
		}
		return tokenSourceFromServiceAccount(ctx, profile.KeyPath)
	case "oauth":
		if strings.TrimSpace(profile.TokenPath) == "" {
			return nil, shared.NewAuthError(
				"invalid auth profile",
				errors.New("oauth profile missing token_path"),
				"Set token_path in config.json or re-run `gplay auth login` with --oauth-token.",
			)
		}
		clientID := strings.TrimSpace(profile.ClientID)
		clientSecret := strings.TrimSpace(profile.ClientSecret)
		if clientID == "" || clientSecret == "" {
			return nil, shared.NewAuthError(
				"invalid auth profile",
				errors.New("oauth profile missing client_id or client_secret"),
				"Set client_id/client_secret in config.json or re-run `gplay auth login` with --client-id/--client-secret.",
			)
		}
		return tokenSourceFromOAuth(ctx, profile.TokenPath, clientID, clientSecret, redirectURIFromEnv())
	default:
		return nil, shared.NewAuthError(
			"invalid auth profile",
			fmt.Errorf("unknown profile type: %s", profile.Type),
			fmt.Sprintf("Use one of: %s.", serviceAccountProfileTypes),
		)
	}
}

func tokenSourceFromEnv(ctx context.Context) (oauth2.TokenSource, error) {
	if keyPath := serviceAccountPathFromEnv(); keyPath != "" {
		return tokenSourceFromServiceAccount(ctx, keyPath)
	}

	tokenPath := strings.TrimSpace(os.Getenv(oauthTokenEnvVar))
	clientID := strings.TrimSpace(os.Getenv(oauthClientIDEnvVar))
	clientSecret := strings.TrimSpace(os.Getenv(oauthClientSecretEnvVar))
	if tokenPath != "" {
		if clientID == "" || clientSecret == "" {
			return nil, shared.NewAuthError(
				"oauth env vars incomplete",
				fmt.Errorf("missing %s or %s", oauthClientIDEnvVar, oauthClientSecretEnvVar),
				"Set both env vars or use `gplay auth login` to create a profile.",
			)
		}
		return tokenSourceFromOAuth(ctx, tokenPath, clientID, clientSecret, redirectURIFromEnv())
	}

	return nil, errors.New("no credentials found")
}

func tokenSourceFromServiceAccount(ctx context.Context, keyPath string) (oauth2.TokenSource, error) {
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, shared.NewAuthError(
			"failed to read service account file",
			err,
			fmt.Sprintf("Check that %s exists and is readable.", keyPath),
		)
	}
	creds, err := google.CredentialsFromJSON(ctx, data, Scope) //nolint:staticcheck // no replacement available yet
	if err != nil {
		return nil, shared.NewAuthError(
			"failed to parse service account JSON",
			err,
			"Ensure the file is a valid service account JSON with Checks API access.",
		)
	}
	return creds.TokenSource, nil
}

func tokenSourceFromOAuth(ctx context.Context, tokenPath, clientID, clientSecret, redirectURI string) (oauth2.TokenSource, error) {
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, shared.NewAuthError(
			"failed to read OAuth token file",
			err,
			fmt.Sprintf("Check that %s exists and is readable.", tokenPath),
		)
	}
	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, shared.NewAuthError(
			"failed to parse OAuth token JSON",
			err,
			"Ensure the OAuth token file contains valid JSON.",
		)
	}
	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{Scope},
		RedirectURL:  redirectURI,
	}
	return cfg.TokenSource(ctx, &token), nil
}

func envAuthPresent() bool {
	return serviceAccountPathFromEnv() != "" || strings.TrimSpace(os.Getenv(oauthTokenEnvVar)) != ""
}

func serviceAccountPathFromEnv() string {
	if v := strings.TrimSpace(os.Getenv(serviceAccountEnvVar)); v != "" {
		return v
	}
	return strings.TrimSpace(os.Getenv(serviceAccountPathEnvVar))
}

func findProfile(cfg *config.Config, name string) (config.Profile, bool) {
	for _, p := range cfg.Profiles {
		if p.Name == name {
			return p, true
		}
	}
	return config.Profile{}, false
}

func redirectURIFromEnv() string {
	if v := strings.TrimSpace(os.Getenv(oauthRedirectEnvVar)); v != "" {
		return v
	}
	return defaultOAuthRedirectURI
}
