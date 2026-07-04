package cmdtest_test

import (
	"errors"
	"flag"
	"strings"
	"testing"
)

func TestGames_Help(t *testing.T) {
	stdout, stderr, err := runCommand(t, "games")
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected help error, got %v", err)
	}
	combined := stdout + stderr
	for _, want := range []string{"games", "achievements", "leaderboards", "scores", "events", "players"} {
		if !strings.Contains(combined, want) {
			t.Fatalf("help should mention %q, got: %q", want, combined)
		}
	}
}

func TestGamesAchievements_Help(t *testing.T) {
	stdout, stderr, err := runCommand(t, "games", "achievements")
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected help error, got %v", err)
	}
	combined := stdout + stderr
	for _, want := range []string{"list", "get", "create", "update", "delete", "reset", "reset-for-all-players"} {
		if !strings.Contains(combined, want) {
			t.Fatalf("achievements help should mention %q, got: %q", want, combined)
		}
	}
}

func TestGamesAchievementsList_MissingApplicationID(t *testing.T) {
	t.Setenv("GPLAY_GAMES_APP_ID", "")
	_, stderr, err := runCommand(t, "games", "achievements", "list")
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected usage error, got %v", err)
	}
	if !strings.Contains(stderr, "--application-id is required") {
		t.Fatalf("stderr should mention missing application id, got: %q", stderr)
	}
}

func TestGamesAchievementsGet_MissingID(t *testing.T) {
	_, stderr, err := runCommand(t, "games", "achievements", "get")
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected usage error, got %v", err)
	}
	if !strings.Contains(stderr, "--achievement-id is required") {
		t.Fatalf("stderr should mention missing achievement id, got: %q", stderr)
	}
}

func TestGamesAchievementsCreate_MissingData(t *testing.T) {
	_, stderr, err := runCommand(t, "games", "achievements", "create", "--application-id", "123")
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected usage error, got %v", err)
	}
	if !strings.Contains(stderr, "--data is required") {
		t.Fatalf("stderr should mention missing data, got: %q", stderr)
	}
}

func TestGamesAchievementsDelete_RequiresConfirm(t *testing.T) {
	_, stderr, err := runCommand(t, "games", "achievements", "delete", "--achievement-id", "abc")
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected usage error, got %v", err)
	}
	if !strings.Contains(stderr, "--confirm") {
		t.Fatalf("stderr should mention --confirm, got: %q", stderr)
	}
}

func TestGamesLeaderboardsList_MissingApplicationID(t *testing.T) {
	t.Setenv("GPLAY_GAMES_APP_ID", "")
	_, stderr, err := runCommand(t, "games", "leaderboards", "list")
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected usage error, got %v", err)
	}
	if !strings.Contains(stderr, "--application-id is required") {
		t.Fatalf("stderr should mention missing application id, got: %q", stderr)
	}
}

func TestGamesScoresReset_MissingLeaderboardID(t *testing.T) {
	_, stderr, err := runCommand(t, "games", "scores", "reset")
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected usage error, got %v", err)
	}
	if !strings.Contains(stderr, "--leaderboard-id is required") {
		t.Fatalf("stderr should mention missing leaderboard id, got: %q", stderr)
	}
}

func TestGamesEventsResetForAll_RequiresConfirm(t *testing.T) {
	_, stderr, err := runCommand(t, "games", "events", "reset-for-all-players", "--event-id", "e1")
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected usage error, got %v", err)
	}
	if !strings.Contains(stderr, "--confirm") {
		t.Fatalf("stderr should mention --confirm, got: %q", stderr)
	}
}

func TestGamesPlayersHide_MissingPlayerID(t *testing.T) {
	_, stderr, err := runCommand(t, "games", "players", "hide", "--application-id", "123")
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected usage error, got %v", err)
	}
	if !strings.Contains(stderr, "--player-id is required") {
		t.Fatalf("stderr should mention missing player id, got: %q", stderr)
	}
}
