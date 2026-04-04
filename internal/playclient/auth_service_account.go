package playclient

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

func credentialsFromServiceAccount(ctx context.Context, keyPath string) (oauth2.TokenSource, error) {
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, shared.NewAuthError(
			"failed to read service account file",
			err,
			fmt.Sprintf("Check that %s exists and is readable (configured via profile key_path or %s).", keyPath, serviceAccountEnvVar),
		)
	}
	creds, err := google.CredentialsFromJSON(ctx, data, scopes...) //nolint:staticcheck // no replacement available yet
	if err != nil {
		return nil, shared.NewAuthError(
			"failed to parse service account JSON",
			err,
			"Ensure the file is a valid service account JSON with Android Publisher access.",
		)
	}
	return creds.TokenSource, nil
}
