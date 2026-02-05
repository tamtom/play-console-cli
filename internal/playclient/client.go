package playclient

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
	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/option"

	"github.com/tamtom/play-console-cli/internal/config"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

const (
	serviceAccountEnvVar = "GPLAY_SERVICE_ACCOUNT_JSON"
	oauthTokenEnvVar     = "GPLAY_OAUTH_TOKEN_PATH"
	oauthClientIDEnvVar  = "GPLAY_OAUTH_CLIENT_ID"
	oauthClientSecretEnvVar = "GPLAY_OAUTH_CLIENT_SECRET"
	oauthRedirectEnvVar  = "GPLAY_OAUTH_REDIRECT_URI"
)

var scopes = []string{"https://www.googleapis.com/auth/androidpublisher"}

// Service wraps the Android Publisher service and config.
type Service struct {
	API *androidpublisher.Service
	Cfg *config.Config
}

// NewService creates an authenticated Android Publisher service.
func NewService(ctx context.Context) (*Service, error) {
	cfg, _ := config.Load()
	client, err := newHTTPClient(ctx, cfg)
	if err != nil {
		return nil, err
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

type credentialSource string

const (
	sourceProfile credentialSource = "profile"
	sourceEnv     credentialSource = "env"
)

type resolvedCredentials struct {
	TokenSource oauth2.TokenSource
	Source      credentialSource
	ProfileName string
}

func resolveCredentials(ctx context.Context, cfg *config.Config) (*resolvedCredentials, error) {
	profileName := shared.ResolveProfileName(cfg)
	if profileName != "" && cfg != nil {
		if profile, ok := findProfile(cfg, profileName); ok {
			creds, err := credentialsFromProfile(ctx, profile)
			if err != nil {
				return nil, err
			}
			if shared.StrictAuthEnabled() && envAuthPresent() {
				return nil, fmt.Errorf("strict auth: profile selected but environment credentials also present")
			}
			creds.ProfileName = profileName
			creds.Source = sourceProfile
			return creds, nil
		}
		return nil, fmt.Errorf("profile not found: %s", profileName)
	}

	if envAuthPresent() {
		creds, err := credentialsFromEnv(ctx)
		if err != nil {
			return nil, err
		}
		creds.Source = sourceEnv
		return creds, nil
	}

	return nil, errors.New("no credentials found (use gplay auth login or set env vars)")
}

func envAuthPresent() bool {
	if strings.TrimSpace(os.Getenv(serviceAccountEnvVar)) != "" {
		return true
	}
	if strings.TrimSpace(os.Getenv(oauthTokenEnvVar)) != "" {
		return true
	}
	return false
}

func findProfile(cfg *config.Config, name string) (config.Profile, bool) {
	for _, p := range cfg.Profiles {
		if p.Name == name {
			return p, true
		}
	}
	return config.Profile{}, false
}

func credentialsFromProfile(ctx context.Context, profile config.Profile) (*resolvedCredentials, error) {
	switch strings.ToLower(strings.TrimSpace(profile.Type)) {
	case "service_account", "service-account", "serviceaccount":
		if strings.TrimSpace(profile.KeyPath) == "" {
			return nil, fmt.Errorf("service account profile missing key_path")
		}
		creds, err := credentialsFromServiceAccount(ctx, profile.KeyPath)
		if err != nil {
			return nil, err
		}
		return &resolvedCredentials{TokenSource: creds}, nil
	case "oauth":
		if strings.TrimSpace(profile.TokenPath) == "" {
			return nil, fmt.Errorf("oauth profile missing token_path")
		}
		clientID := strings.TrimSpace(profile.ClientID)
		clientSecret := strings.TrimSpace(profile.ClientSecret)
		if clientID == "" || clientSecret == "" {
			return nil, fmt.Errorf("oauth profile missing client_id or client_secret")
		}
		creds, err := credentialsFromOAuth(ctx, profile.TokenPath, clientID, clientSecret, redirectURIFromEnv())
		if err != nil {
			return nil, err
		}
		return &resolvedCredentials{TokenSource: creds}, nil
	default:
		return nil, fmt.Errorf("unknown profile type: %s", profile.Type)
	}
}

func credentialsFromEnv(ctx context.Context) (*resolvedCredentials, error) {
	if keyPath := strings.TrimSpace(os.Getenv(serviceAccountEnvVar)); keyPath != "" {
		tokenSource, err := credentialsFromServiceAccount(ctx, keyPath)
		if err != nil {
			return nil, err
		}
		return &resolvedCredentials{TokenSource: tokenSource}, nil
	}

	tokenPath := strings.TrimSpace(os.Getenv(oauthTokenEnvVar))
	clientID := strings.TrimSpace(os.Getenv(oauthClientIDEnvVar))
	clientSecret := strings.TrimSpace(os.Getenv(oauthClientSecretEnvVar))
	if tokenPath != "" {
		if clientID == "" || clientSecret == "" {
			return nil, fmt.Errorf("oauth env vars require %s and %s", oauthClientIDEnvVar, oauthClientSecretEnvVar)
		}
		tokenSource, err := credentialsFromOAuth(ctx, tokenPath, clientID, clientSecret, redirectURIFromEnv())
		if err != nil {
			return nil, err
		}
		return &resolvedCredentials{TokenSource: tokenSource}, nil
	}

	return nil, errors.New("no credentials found")
}

func credentialsFromServiceAccount(ctx context.Context, keyPath string) (oauth2.TokenSource, error) {
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	creds, err := google.CredentialsFromJSON(ctx, data, scopes...)
	if err != nil {
		return nil, err
	}
	return creds.TokenSource, nil
}

func credentialsFromOAuth(ctx context.Context, tokenPath, clientID, clientSecret, redirectURI string) (oauth2.TokenSource, error) {
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, err
	}
	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       scopes,
		RedirectURL:  redirectURI,
	}
	return cfg.TokenSource(ctx, &token), nil
}

func redirectURIFromEnv() string {
	if v := strings.TrimSpace(os.Getenv(oauthRedirectEnvVar)); v != "" {
		return v
	}
	return "urn:ietf:wg:oauth:2.0:oob"
}
