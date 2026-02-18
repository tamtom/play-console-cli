package tracks

import (
	"context"
	"errors"
	"flag"
	"strings"
	"testing"
)

func TestTracksCommand_Name(t *testing.T) {
	cmd := TracksCommand()
	if cmd.Name != "tracks" {
		t.Errorf("expected name %q, got %q", "tracks", cmd.Name)
	}
}

func TestTracksCommand_ShortHelp(t *testing.T) {
	cmd := TracksCommand()
	if cmd.ShortHelp == "" {
		t.Error("expected non-empty ShortHelp")
	}
}

func TestTracksCommand_UsageFunc(t *testing.T) {
	cmd := TracksCommand()
	if cmd.UsageFunc == nil {
		t.Error("expected UsageFunc to be set")
	}
}

func TestTracksCommand_HasSubcommands(t *testing.T) {
	cmd := TracksCommand()
	if len(cmd.Subcommands) == 0 {
		t.Error("expected subcommands")
	}
}

func TestTracksCommand_SubcommandNames(t *testing.T) {
	cmd := TracksCommand()
	expected := map[string]bool{
		"list":   false,
		"get":    false,
		"create": false,
		"update": false,
		"patch":  false,
	}
	for _, sub := range cmd.Subcommands {
		if _, ok := expected[sub.Name]; ok {
			expected[sub.Name] = true
		} else {
			t.Errorf("unexpected subcommand: %s", sub.Name)
		}
	}
	for name, found := range expected {
		if !found {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

func TestTracksCommand_SubcommandsHaveUsageFunc(t *testing.T) {
	cmd := TracksCommand()
	for _, sub := range cmd.Subcommands {
		if sub.UsageFunc == nil {
			t.Errorf("subcommand %q missing UsageFunc", sub.Name)
		}
	}
}

func TestTracksCommand_SubcommandsHaveShortHelp(t *testing.T) {
	cmd := TracksCommand()
	for _, sub := range cmd.Subcommands {
		if sub.ShortHelp == "" {
			t.Errorf("subcommand %q missing ShortHelp", sub.Name)
		}
	}
}

func TestTracksCommand_NoArgs_ReturnsHelp(t *testing.T) {
	cmd := TracksCommand()
	err := cmd.Exec(context.Background(), nil)
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

// --- tracks list ---

func TestTracksListCommand_Name(t *testing.T) {
	cmd := ListCommand()
	if cmd.Name != "list" {
		t.Errorf("expected name %q, got %q", "list", cmd.Name)
	}
}

func TestTracksListCommand_InvalidOutputFormat(t *testing.T) {
	cmd := ListCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "xml"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

func TestTracksListCommand_PrettyWithTable(t *testing.T) {
	cmd := ListCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "table", "--pretty"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for --pretty with table output")
	}
	if !strings.Contains(err.Error(), "--pretty") {
		t.Errorf("error should mention --pretty, got: %s", err.Error())
	}
}

// --- tracks get ---

func TestTracksGetCommand_Name(t *testing.T) {
	cmd := GetCommand()
	if cmd.Name != "get" {
		t.Errorf("expected name %q, got %q", "get", cmd.Name)
	}
}

func TestTracksGetCommand_MissingTrack(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --track")
	}
	if !strings.Contains(err.Error(), "--track") {
		t.Errorf("error should mention --track, got: %s", err.Error())
	}
}

func TestTracksGetCommand_WhitespaceTrack(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{"--track", "   "}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --track")
	}
	if !strings.Contains(err.Error(), "--track") {
		t.Errorf("error should mention --track, got: %s", err.Error())
	}
}

func TestTracksGetCommand_InvalidOutputFormat(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "yaml"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

// --- tracks create ---

func TestTracksCreateCommand_Name(t *testing.T) {
	cmd := CreateCommand()
	if cmd.Name != "create" {
		t.Errorf("expected name %q, got %q", "create", cmd.Name)
	}
}

func TestTracksCreateCommand_MissingTrack(t *testing.T) {
	cmd := CreateCommand()
	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --track")
	}
	if !strings.Contains(err.Error(), "--track") {
		t.Errorf("error should mention --track, got: %s", err.Error())
	}
}

func TestTracksCreateCommand_WhitespaceTrack(t *testing.T) {
	cmd := CreateCommand()
	if err := cmd.FlagSet.Parse([]string{"--track", "  "}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --track")
	}
	if !strings.Contains(err.Error(), "--track") {
		t.Errorf("error should mention --track, got: %s", err.Error())
	}
}

func TestTracksCreateCommand_PrettyWithMarkdown(t *testing.T) {
	cmd := CreateCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "markdown", "--pretty"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for --pretty with markdown output")
	}
	if !strings.Contains(err.Error(), "--pretty") {
		t.Errorf("error should mention --pretty, got: %s", err.Error())
	}
}

// --- tracks update ---

func TestTracksUpdateCommand_Name(t *testing.T) {
	cmd := UpdateCommand()
	if cmd.Name != "update" {
		t.Errorf("expected name %q, got %q", "update", cmd.Name)
	}
}

func TestTracksUpdateCommand_MissingTrack(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--releases", `[{"status":"completed"}]`}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --track")
	}
	if !strings.Contains(err.Error(), "--track") {
		t.Errorf("error should mention --track, got: %s", err.Error())
	}
}

func TestTracksUpdateCommand_MissingReleases(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--track", "production"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --releases")
	}
	if !strings.Contains(err.Error(), "--releases") {
		t.Errorf("error should mention --releases, got: %s", err.Error())
	}
}

func TestTracksUpdateCommand_PrettyWithTable(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "table", "--pretty"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for --pretty with table output")
	}
	if !strings.Contains(err.Error(), "--pretty") {
		t.Errorf("error should mention --pretty, got: %s", err.Error())
	}
}

// --- tracks patch ---

func TestTracksPatchCommand_Name(t *testing.T) {
	cmd := PatchCommand()
	if cmd.Name != "patch" {
		t.Errorf("expected name %q, got %q", "patch", cmd.Name)
	}
}

func TestTracksPatchCommand_MissingTrack(t *testing.T) {
	cmd := PatchCommand()
	if err := cmd.FlagSet.Parse([]string{"--releases", `[{"status":"completed"}]`}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --track")
	}
	if !strings.Contains(err.Error(), "--track") {
		t.Errorf("error should mention --track, got: %s", err.Error())
	}
}

func TestTracksPatchCommand_MissingReleases(t *testing.T) {
	cmd := PatchCommand()
	if err := cmd.FlagSet.Parse([]string{"--track", "beta"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --releases")
	}
	if !strings.Contains(err.Error(), "--releases") {
		t.Errorf("error should mention --releases, got: %s", err.Error())
	}
}
