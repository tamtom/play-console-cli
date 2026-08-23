package checks

import (
	"context"
	"strings"
	"testing"
)

func TestRepoScansCommand_ExposesCompleteOfficialSurface(t *testing.T) {
	cmd := RepoScansCommand()
	want := map[string]bool{"generate": false, "get": false, "list": false, "operation": false}
	for _, sub := range cmd.Subcommands {
		if _, ok := want[sub.Name]; ok {
			want[sub.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("missing %s", name)
		}
	}
}

func TestRepoScanGenerate_RequiresConfirmationBeforeAuthentication(t *testing.T) {
	cmd := RepoScanGenerateCommand()
	if err := cmd.FlagSet.Parse([]string{"--account", "123", "--repo", "repo1", "--json", `{}`}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--confirm") {
		t.Fatalf("expected --confirm error, got %v", err)
	}
}

func TestRepoResource(t *testing.T) {
	if got := repoResource("123", "repo1"); got != "accounts/123/repos/repo1" {
		t.Fatalf("got %q", got)
	}
	if got := repoScanResource("123", "repo1", "scan1"); got != "accounts/123/repos/repo1/scans/scan1" {
		t.Fatalf("got %q", got)
	}
}
