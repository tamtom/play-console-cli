package auditcmd

import (
	"context"
	"errors"
	"flag"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tamtom/play-console-cli/internal/audit"
)

func setupAuditEnv(t *testing.T) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "audit.log")
	t.Setenv(audit.PathEnvVar, path)
	audit.SetEnabled(true)
}

func TestAuditCommandStructure(t *testing.T) {
	cmd := AuditCommand()
	if cmd.Name != "audit" {
		t.Errorf("name = %q", cmd.Name)
	}
	want := map[string]bool{"list": false, "search": false, "clear": false, "path": false}
	for _, sub := range cmd.Subcommands {
		if _, ok := want[sub.Name]; ok {
			want[sub.Name] = true
		}
	}
	for k, v := range want {
		if !v {
			t.Errorf("missing subcommand: %s", k)
		}
	}
}

func TestAuditUnknownSubcommand(t *testing.T) {
	cmd := AuditCommand()
	if err := cmd.Exec(context.Background(), []string{"what"}); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected ErrHelp, got %v", err)
	}
}

func TestSearchRequiresFilter(t *testing.T) {
	setupAuditEnv(t)
	cmd := searchCommand()
	_ = cmd.FlagSet.Parse(nil)
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("expected required error, got %v", err)
	}
}

func TestClearRequiresConfirm(t *testing.T) {
	setupAuditEnv(t)
	cmd := clearCommand()
	_ = cmd.FlagSet.Parse(nil)
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--confirm") {
		t.Fatalf("expected --confirm error, got %v", err)
	}
}

func TestClearTruncates(t *testing.T) {
	setupAuditEnv(t)
	if err := audit.Write(audit.Entry{Command: "test"}); err != nil {
		t.Fatal(err)
	}
	cmd := clearCommand()
	_ = cmd.FlagSet.Parse([]string{"--confirm"})
	if err := cmd.Exec(context.Background(), nil); err != nil {
		t.Fatalf("clear: %v", err)
	}
	entries, _ := audit.Read(audit.Query{})
	if len(entries) != 0 {
		t.Fatalf("expected empty log, got %d entries", len(entries))
	}
}

func TestParseSince(t *testing.T) {
	cases := []struct {
		in      string
		wantErr bool
	}{
		{"24h", false},
		{"7d", false},
		{"2026-04-19T00:00:00Z", false},
		{"abc", true},
		{"", true},
	}
	for _, c := range cases {
		_, err := parseSince(c.in)
		if (err != nil) != c.wantErr {
			t.Errorf("parseSince(%q) err=%v wantErr=%v", c.in, err, c.wantErr)
		}
	}
}

func TestListFiltersSince(t *testing.T) {
	setupAuditEnv(t)
	now := time.Now().UTC()
	_ = audit.Write(audit.Entry{Command: "old", Timestamp: now.Add(-48 * time.Hour)})
	_ = audit.Write(audit.Entry{Command: "recent", Timestamp: now.Add(-1 * time.Hour)})

	entries, err := audit.Read(audit.Query{Since: now.Add(-24 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Command != "recent" {
		t.Fatalf("unexpected entries: %+v", entries)
	}
}
