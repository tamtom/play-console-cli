package playclient

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/oauth2"

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
				return nil, shared.NewAuthError(
					"authentication failed",
					fmt.Errorf("strict auth: profile selected but environment credentials also present"),
					"Unset environment credentials or set GPLAY_STRICT_AUTH=false.",
				)
			}
			creds.ProfileName = profileName
			creds.Source = sourceProfile
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
		creds.Source = sourceEnv
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

func redirectURIFromEnv() string {
	if v := strings.TrimSpace(os.Getenv(oauthRedirectEnvVar)); v != "" {
		return v
	}
	return "urn:ietf:wg:oauth:2.0:oob"
}
