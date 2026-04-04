package reportingclient

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

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
	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       scopes,
		RedirectURL:  redirectURI,
	}
	return cfg.TokenSource(ctx, &token), nil
}
