package playclient

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/config"
)

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
		creds, err := credentialsFromServiceAccount(ctx, profile.KeyPath)
		if err != nil {
			return nil, err
		}
		return &resolvedCredentials{TokenSource: creds}, nil
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
		creds, err := credentialsFromOAuth(ctx, profile.TokenPath, clientID, clientSecret, redirectURIFromEnv())
		if err != nil {
			return nil, err
		}
		return &resolvedCredentials{TokenSource: creds}, nil
	default:
		return nil, shared.NewAuthError(
			"invalid auth profile",
			fmt.Errorf("unknown profile type: %s", profile.Type),
			"Use type service_account or oauth.",
		)
	}
}
