// Package quota implements the `gplay quota` command family.
//
// Quota tracking is derived from the audit log: every gplay invocation that
// made an API call is one tick against Google Play's documented quota windows
// (daily and per-minute). Limits match the publicly documented values.
package quota

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/audit"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// Documented Play Developer API quota caps.
// Source: https://developers.google.com/android-publisher/quotas
const (
	dailyCap    = 200_000
	perMinCap   = 6_000 // 100 req/s sustained bucket
	warnRatio   = 0.80
	defaultTop  = 5
	defaultDays = 1
)

// QuotaCommand builds the root `gplay quota` command.
func QuotaCommand() *ffcli.Command {
	fs := flag.NewFlagSet("quota", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "quota",
		ShortUsage: "gplay quota <subcommand> [flags]",
		ShortHelp:  "Inspect local API quota usage derived from the audit log.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			statusCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", args[0])
			return flag.ErrHelp
		},
	}
}

func statusCommand() *ffcli.Command {
	fs := flag.NewFlagSet("quota status", flag.ExitOnError)
	days := fs.Int("days", defaultDays, "Window in days to include in daily totals")
	top := fs.Int("top", defaultTop, "Show this many top commands by call count")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "status",
		ShortUsage: "gplay quota status [flags]",
		ShortHelp:  "Show current API quota usage (daily and per-minute).",
		LongHelp: `Show current API quota usage derived from the local audit log.

Google Play Developer API enforces ~200,000 calls/day and ~6,000/minute
(roughly 100/s sustained) per developer account. This command counts audit
entries to estimate usage; it can only see calls made by this machine.

Disable audit with GPLAY_AUDIT=0.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if *days <= 0 {
				return fmt.Errorf("--days must be positive")
			}
			if *top < 0 {
				return fmt.Errorf("--top must be non-negative")
			}
			now := time.Now().UTC()
			since := now.Add(-time.Duration(*days) * 24 * time.Hour)
			entries, err := audit.Read(audit.Query{Since: since})
			if err != nil {
				return fmt.Errorf("read audit log: %w", err)
			}
			status := Compute(entries, now, *top)
			return shared.PrintOutput(status, *outputFlag, *pretty)
		},
	}
}

// Status is the computed quota view returned by `quota status`.
type Status struct {
	DailyCount     int            `json:"daily_count"`
	DailyLimit     int            `json:"daily_limit"`
	DailyRatio     float64        `json:"daily_ratio"`
	DailyWarning   bool           `json:"daily_warning"`
	MinuteCount    int            `json:"minute_count"`
	MinuteLimit    int            `json:"minute_limit"`
	MinuteRatio    float64        `json:"minute_ratio"`
	MinuteWarning  bool           `json:"minute_warning"`
	WindowStart    time.Time      `json:"window_start"`
	WindowEnd      time.Time      `json:"window_end"`
	TopCommands    []CommandCount `json:"top_commands,omitempty"`
	AuditDisabled  bool           `json:"audit_disabled,omitempty"`
	AuditLogPath   string         `json:"audit_log_path,omitempty"`
	TotalEntries   int            `json:"total_entries"`
	ErrorRateRatio float64        `json:"error_rate_ratio"`
}

// CommandCount is a single bucket in the top-commands breakdown.
type CommandCount struct {
	Command string `json:"command"`
	Count   int    `json:"count"`
}

// Compute aggregates entries into a quota Status relative to `now`.
// Exported so other packages (tests) can reuse the math.
func Compute(entries []audit.Entry, now time.Time, topN int) Status {
	s := Status{
		DailyLimit:  dailyCap,
		MinuteLimit: perMinCap,
		WindowEnd:   now,
	}

	minuteStart := now.Add(-time.Minute)
	dayStart := now.Add(-24 * time.Hour)
	s.WindowStart = dayStart

	counts := map[string]int{}
	errorCount := 0
	for _, e := range entries {
		if e.Timestamp.After(dayStart) {
			s.DailyCount++
		}
		if e.Timestamp.After(minuteStart) {
			s.MinuteCount++
		}
		counts[e.Command]++
		if e.Status == "error" {
			errorCount++
		}
	}
	s.TotalEntries = len(entries)
	if s.DailyCount > 0 {
		s.ErrorRateRatio = float64(errorCount) / float64(s.DailyCount)
	}
	if dailyCap > 0 {
		s.DailyRatio = float64(s.DailyCount) / float64(dailyCap)
	}
	if perMinCap > 0 {
		s.MinuteRatio = float64(s.MinuteCount) / float64(perMinCap)
	}
	s.DailyWarning = s.DailyRatio >= warnRatio
	s.MinuteWarning = s.MinuteRatio >= warnRatio

	if topN > 0 && len(counts) > 0 {
		s.TopCommands = sortedTop(counts, topN)
	}

	s.AuditDisabled = !audit.Enabled()
	if path, err := audit.Path(); err == nil {
		s.AuditLogPath = path
	}
	return s
}

func sortedTop(counts map[string]int, topN int) []CommandCount {
	out := make([]CommandCount, 0, len(counts))
	for cmd, n := range counts {
		if strings.TrimSpace(cmd) == "" {
			continue
		}
		out = append(out, CommandCount{Command: cmd, Count: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Command < out[j].Command
		}
		return out[i].Count > out[j].Count
	})
	if len(out) > topN {
		out = out[:topN]
	}
	return out
}
