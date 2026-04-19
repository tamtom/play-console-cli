package quota

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

func setupAudit(t *testing.T) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "audit.log")
	t.Setenv(audit.PathEnvVar, path)
	audit.SetEnabled(true)
}

func TestQuotaCommandStructure(t *testing.T) {
	cmd := QuotaCommand()
	if cmd.Name != "quota" {
		t.Errorf("name = %q", cmd.Name)
	}
	found := false
	for _, sub := range cmd.Subcommands {
		if sub.Name == "status" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected status subcommand")
	}
}

func TestQuotaUnknownSubcommand(t *testing.T) {
	cmd := QuotaCommand()
	if err := cmd.Exec(context.Background(), []string{"boom"}); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected ErrHelp, got %v", err)
	}
}

func TestStatusRejectsBadFlags(t *testing.T) {
	setupAudit(t)
	cmd := statusCommand()
	_ = cmd.FlagSet.Parse([]string{"--days", "-1"})
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--days") {
		t.Fatalf("expected --days error, got %v", err)
	}

	cmd = statusCommand()
	_ = cmd.FlagSet.Parse([]string{"--top", "-1"})
	err = cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--top") {
		t.Fatalf("expected --top error, got %v", err)
	}
}

func TestComputeBuckets(t *testing.T) {
	now := time.Now().UTC()
	entries := []audit.Entry{
		{Command: "apps list", Timestamp: now.Add(-30 * time.Second), Status: "ok"},
		{Command: "apps list", Timestamp: now.Add(-45 * time.Second), Status: "ok"},
		{Command: "tracks list", Timestamp: now.Add(-10 * time.Minute), Status: "error"},
		{Command: "tracks list", Timestamp: now.Add(-5 * time.Hour), Status: "ok"},
		{Command: "apps get", Timestamp: now.Add(-48 * time.Hour), Status: "ok"}, // outside daily window
	}
	s := Compute(entries, now, 5)

	if s.MinuteCount != 2 {
		t.Errorf("minute count = %d, want 2", s.MinuteCount)
	}
	// 48h-old entry is outside the daily window but Compute received it anyway;
	// the command filters at Read(). Here we still expect only entries <24h counted.
	if s.DailyCount != 4 {
		t.Errorf("daily count = %d, want 4", s.DailyCount)
	}
	if s.DailyLimit != dailyCap {
		t.Errorf("daily limit should be %d", dailyCap)
	}
	if s.MinuteLimit != perMinCap {
		t.Errorf("minute limit should be %d", perMinCap)
	}
	if s.ErrorRateRatio == 0 {
		t.Errorf("expected non-zero error rate")
	}
	if len(s.TopCommands) == 0 {
		t.Fatal("expected top commands populated")
	}
	// apps list and tracks list both have count 2 -> first by alphabetic tie-break.
	if s.TopCommands[0].Count != 2 {
		t.Errorf("top count = %d, want 2", s.TopCommands[0].Count)
	}
}

func TestComputeWarningThreshold(t *testing.T) {
	now := time.Now().UTC()
	count := int(float64(dailyCap)*warnRatio) + 1
	entries := make([]audit.Entry, 0, count)
	// Spread entries across the last 12 hours to keep them all inside the daily window.
	step := (12 * time.Hour) / time.Duration(count)
	for i := 0; i < count; i++ {
		entries = append(entries, audit.Entry{
			Command:   "apps list",
			Timestamp: now.Add(-time.Duration(i) * step),
		})
	}
	s := Compute(entries, now, 3)
	if !s.DailyWarning {
		t.Errorf("expected daily warning at %d entries (ratio %.3f)", s.DailyCount, s.DailyRatio)
	}
}

func TestComputeEmpty(t *testing.T) {
	s := Compute(nil, time.Now().UTC(), 5)
	if s.DailyCount != 0 || s.MinuteCount != 0 {
		t.Errorf("expected zero counts, got %+v", s)
	}
	if s.ErrorRateRatio != 0 {
		t.Errorf("expected zero error rate, got %v", s.ErrorRateRatio)
	}
	if s.DailyWarning || s.MinuteWarning {
		t.Error("no warnings on empty")
	}
}

func TestComputeTopNLimit(t *testing.T) {
	now := time.Now().UTC()
	entries := []audit.Entry{
		{Command: "a", Timestamp: now, Status: "ok"},
		{Command: "b", Timestamp: now, Status: "ok"},
		{Command: "b", Timestamp: now, Status: "ok"},
		{Command: "c", Timestamp: now, Status: "ok"},
		{Command: "c", Timestamp: now, Status: "ok"},
		{Command: "c", Timestamp: now, Status: "ok"},
	}
	s := Compute(entries, now, 2)
	if len(s.TopCommands) != 2 {
		t.Fatalf("expected top 2, got %d", len(s.TopCommands))
	}
	if s.TopCommands[0].Command != "c" || s.TopCommands[1].Command != "b" {
		t.Errorf("unexpected ordering: %+v", s.TopCommands)
	}
}
