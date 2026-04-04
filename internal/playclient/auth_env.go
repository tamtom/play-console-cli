package playclient

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

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
			return nil, shared.NewAuthError(
				"oauth env vars incomplete",
				fmt.Errorf("missing %s or %s", oauthClientIDEnvVar, oauthClientSecretEnvVar),
				"Set both env vars or use `gplay auth login` to create a profile.",
			)
		}
		tokenSource, err := credentialsFromOAuth(ctx, tokenPath, clientID, clientSecret, redirectURIFromEnv())
		if err != nil {
			return nil, err
		}
		return &resolvedCredentials{TokenSource: tokenSource}, nil
	}

	return nil, errors.New("no credentials found")
}
