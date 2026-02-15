package gcsclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	storage "google.golang.org/api/storage/v1"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/config"
)

const (
	serviceAccountEnvVar    = "GPLAY_SERVICE_ACCOUNT_JSON"
	oauthTokenEnvVar        = "GPLAY_OAUTH_TOKEN_PATH"
	oauthClientIDEnvVar     = "GPLAY_OAUTH_CLIENT_ID"
	oauthClientSecretEnvVar = "GPLAY_OAUTH_CLIENT_SECRET"
	oauthRedirectEnvVar     = "GPLAY_OAUTH_REDIRECT_URI"
)

var scopes = []string{
	"https://www.googleapis.com/auth/devstorage.read_only",
}

// Service wraps the Google Cloud Storage service and config.
type Service struct {
	API *storage.Service
	Cfg *config.Config
}

// NewService creates an authenticated GCS service.
func NewService(ctx context.Context) (*Service, error) {
	cfg, err := config.Load()
	if err != nil && !errors.Is(err, config.ErrNotFound) {
		return nil, shared.NewActionableError(
			"failed to load config",
			err,
			"Check that your config file is valid and readable. Use `gplay auth init` to recreate it.",
		)
	}
	client, err := newHTTPClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	api, err := storage.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	return &Service{API: api, Cfg: cfg}, nil
}

// NewServiceWithClient creates a GCS service using a provided HTTP client.
// This is useful for testing with mock servers.
func NewServiceWithClient(ctx context.Context, client *http.Client, basePath string) (*Service, error) {
	api, err := storage.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	if basePath != "" {
		api.BasePath = basePath
	}
	return &Service{API: api}, nil
}

// ObjectInfo contains metadata about a GCS object.
type ObjectInfo struct {
	Name    string `json:"name"`
	Size    uint64 `json:"size"`
	Updated string `json:"updated"`
}

// ListObjects lists objects in a GCS bucket matching the given prefix.
func (s *Service) ListObjects(ctx context.Context, bucket, prefix string) ([]ObjectInfo, error) {
	var objects []ObjectInfo
	call := s.API.Objects.List(bucket).Prefix(prefix).Context(ctx)

	err := call.Pages(ctx, func(page *storage.Objects) error {
		for _, obj := range page.Items {
			objects = append(objects, ObjectInfo{
				Name:    obj.Name,
				Size:    obj.Size,
				Updated: obj.Updated,
			})
		}
		return nil
	})
	if err != nil {
		return nil, shared.WrapGoogleAPIError("list reports", err)
	}
	return objects, nil
}

// DownloadObject downloads a single object from GCS and returns a ReadCloser.
func (s *Service) DownloadObject(ctx context.Context, bucket, object string) (io.ReadCloser, error) {
	resp, err := s.API.Objects.Get(bucket, object).Context(ctx).Download()
	if err != nil {
		return nil, shared.WrapGoogleAPIError("download report", err)
	}
	return resp.Body, nil
}

func newHTTPClient(ctx context.Context, cfg *config.Config) (*http.Client, error) {
	creds, err := resolveCredentials(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return oauth2.NewClient(ctx, creds.TokenSource), nil
}

type resolvedCredentials struct {
	TokenSource oauth2.TokenSource
	Source      string
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
				return nil, shared.NewAuthError(
					"authentication failed",
					fmt.Errorf("strict auth: profile selected but environment credentials also present"),
					"Unset environment credentials or set GPLAY_STRICT_AUTH=false.",
				)
			}
			creds.ProfileName = profileName
			creds.Source = "profile"
			return creds, nil
		}
		return nil, shared.NewAuthError(
			"authentication failed",
			fmt.Errorf("profile not found: %s", profileName),
			"Run `gplay auth login --profile <name>` or set GPLAY_PROFILE to an existing profile.",
		)
	}

	if envAuthPresent() {
		creds, err := credentialsFromEnv(ctx)
		if err != nil {
			return nil, err
		}
		creds.Source = "env"
		return creds, nil
	}

	return nil, shared.NewAuthError(
		"authentication failed",
		errors.New("no credentials found"),
		"Run `gplay auth login` or set GPLAY_SERVICE_ACCOUNT_JSON / GPLAY_OAUTH_TOKEN_PATH.",
	)
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
			return nil, shared.NewAuthError(
				"invalid auth profile",
				errors.New("service account profile missing key_path"),
				"Set key_path in config.json or re-run `gplay auth login` with --service-account.",
			)
		}
		ts, err := credentialsFromServiceAccount(ctx, profile.KeyPath)
		if err != nil {
			return nil, err
		}
		return &resolvedCredentials{TokenSource: ts}, nil
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
		ts, err := credentialsFromOAuth(ctx, profile.TokenPath, clientID, clientSecret, redirectURIFromEnv())
		if err != nil {
			return nil, err
		}
		return &resolvedCredentials{TokenSource: ts}, nil
	default:
		return nil, shared.NewAuthError(
			"invalid auth profile",
			fmt.Errorf("unknown profile type: %s", profile.Type),
			"Use type service_account or oauth.",
		)
	}
}

func credentialsFromEnv(ctx context.Context) (*resolvedCredentials, error) {
	if keyPath := strings.TrimSpace(os.Getenv(serviceAccountEnvVar)); keyPath != "" {
		ts, err := credentialsFromServiceAccount(ctx, keyPath)
		if err != nil {
			return nil, err
		}
		return &resolvedCredentials{TokenSource: ts}, nil
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
		ts, err := credentialsFromOAuth(ctx, tokenPath, clientID, clientSecret, redirectURIFromEnv())
		if err != nil {
			return nil, err
		}
		return &resolvedCredentials{TokenSource: ts}, nil
	}

	return nil, errors.New("no credentials found")
}

func credentialsFromServiceAccount(ctx context.Context, keyPath string) (oauth2.TokenSource, error) {
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, shared.NewAuthError(
			"failed to read service account file",
			err,
			fmt.Sprintf("Check that %s exists and is readable.", keyPath),
		)
	}
	creds, err := google.CredentialsFromJSON(ctx, data, scopes...) //nolint:staticcheck // no replacement available yet
	if err != nil {
		return nil, shared.NewAuthError(
			"failed to parse service account JSON",
			err,
			"Ensure the file is a valid service account JSON with GCS read access.",
		)
	}
	return creds.TokenSource, nil
}

func credentialsFromOAuth(ctx context.Context, tokenPath, clientID, clientSecret, redirectURI string) (oauth2.TokenSource, error) {
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
	oauthCfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       scopes,
		RedirectURL:  redirectURI,
	}
	return oauthCfg.TokenSource(ctx, &token), nil
}

func redirectURIFromEnv() string {
	if v := strings.TrimSpace(os.Getenv(oauthRedirectEnvVar)); v != "" {
		return v
	}
	return "urn:ietf:wg:oauth:2.0:oob"
}
