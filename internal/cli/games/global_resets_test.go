package games

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/tamtom/play-console-cli/internal/gamesclient"
)

func TestGlobalResetCommandsAreExposed(t *testing.T) {
	assertSubcommands(t, AchievementsCommand(), "reset-all-for-all-players", "reset-multiple-for-all-players")
	assertSubcommands(t, EventsCommand(), "reset-all-for-all-players", "reset-multiple-for-all-players")
	assertSubcommands(t, ScoresCommand(), "reset-all-for-all-players", "reset-multiple-for-all-players")
}

func TestGlobalResetVerifiesApplicationBeforeMutation(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_, _ = io.WriteString(w, `{"items":[]}`)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	original := newGlobalResetService
	newGlobalResetService = func(ctx context.Context) (*gamesclient.Service, error) {
		return gamesclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
	}
	t.Cleanup(func() { newGlobalResetService = original })

	cmd := achievementsResetAllForAllCommand()
	if err := cmd.FlagSet.Parse([]string{"--application-id", "123", "--confirm-application-id", "123", "--confirm"}); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Exec(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	want := []string{"/games/v1configuration/applications/123/achievements", "/games/v1management/achievements/resetAllForAllPlayers"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths = %#v, want %#v", paths, want)
	}
}

func TestGlobalResetRequiresExactApplicationConfirmation(t *testing.T) {
	cmd := achievementsResetAllForAllCommand()
	if err := cmd.FlagSet.Parse([]string{"--application-id", "123", "--confirm-application-id", "456", "--confirm"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "must exactly match") {
		t.Fatalf("expected target mismatch, got %v", err)
	}
}

func assertSubcommands(t *testing.T, command *ffcli.Command, names ...string) {
	t.Helper()
	want := make(map[string]bool, len(names))
	for _, name := range names {
		want[name] = false
	}
	for _, sub := range command.Subcommands {
		if _, ok := want[sub.Name]; ok {
			want[sub.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("%s missing %s", command.Name, name)
		}
	}
}
