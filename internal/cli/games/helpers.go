package games

import (
	"errors"
	"fmt"

	"github.com/tamtom/play-console-cli/internal/config"
	"github.com/tamtom/play-console-cli/internal/gamesclient"
)

// resolveApplicationID resolves the Play Games application ID from the flag,
// GPLAY_GAMES_APP_ID, or config, loading config independently so validation can
// run before the authenticated service is built.
func resolveApplicationID(flagValue string) (string, error) {
	cfg, err := config.Load()
	if err != nil && !errors.Is(err, config.ErrNotFound) {
		return "", fmt.Errorf("load config: %w", err)
	}
	return gamesclient.ResolveApplicationID(flagValue, cfg), nil
}

// missingApplicationIDError is the shared usage message for commands that
// require an application ID.
const missingApplicationIDMsg = "--application-id is required (or set GPLAY_GAMES_APP_ID/games_application_id)"
