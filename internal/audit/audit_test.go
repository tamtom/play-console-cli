package audit

import (
	"path/filepath"
	"testing"
	"time"
)

func TestWriteAndRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	t.Setenv(PathEnvVar, path)
	SetEnabled(true)

	entries := []Entry{
		{Command: "apps list", Status: "ok", DurationM: 120},
		{Command: "vitals crashes query", Status: "error", Error: "timeout"},
		{Command: "tracks list", Status: "ok"},
	}
	for _, e := range entries {
		if err := Write(e); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}

	got, err := Read(Query{})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(got))
	}
	// Newest first -> last written is first.
	if got[0].Command != "tracks list" {
		t.Errorf("expected newest first, got %q", got[0].Command)
	}
}

func TestReadFilters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	t.Setenv(PathEnvVar, path)
	SetEnabled(true)

	now := time.Now().UTC()
	entries := []Entry{
		{Command: "apps list", Status: "ok", Timestamp: now.Add(-2 * time.Hour)},
		{Command: "apps get", Status: "error", Timestamp: now.Add(-1 * time.Hour)},
		{Command: "tracks list", Status: "ok", Timestamp: now},
	}
	for _, e := range entries {
		if err := Write(e); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}

	t.Run("command substring", func(t *testing.T) {
		got, err := Read(Query{Command: "apps"})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2, got %d", len(got))
		}
	})

	t.Run("status exact", func(t *testing.T) {
		got, err := Read(Query{Status: "error"})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].Status != "error" {
			t.Fatalf("expected single error entry, got %+v", got)
		}
	})

	t.Run("since cutoff", func(t *testing.T) {
		got, err := Read(Query{Since: now.Add(-90 * time.Minute)})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 entries after cutoff, got %d", len(got))
		}
	})

	t.Run("limit", func(t *testing.T) {
		got, err := Read(Query{Limit: 1})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1, got %d", len(got))
		}
	})
}

func TestDisabledSkipsWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	t.Setenv(PathEnvVar, path)
	SetEnabled(false)
	t.Cleanup(func() { SetEnabled(true) })

	if err := Write(Entry{Command: "apps list"}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := Read(Query{})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected zero entries when disabled, got %d", len(got))
	}
}

func TestParseDisabled(t *testing.T) {
	cases := map[string]bool{
		"":      false,
		"1":     false,
		"true":  false,
		"on":    false,
		"0":     true,
		"false": true,
		"OFF":   true,
		"no":    true,
	}
	for input, want := range cases {
		if got := parseDisabled(input); got != want {
			t.Errorf("parseDisabled(%q)=%v want %v", input, got, want)
		}
	}
}

func TestClear(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	t.Setenv(PathEnvVar, path)
	SetEnabled(true)

	if err := Write(Entry{Command: "apps list"}); err != nil {
		t.Fatal(err)
	}
	if err := Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	got, err := Read(Query{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty after clear, got %d", len(got))
	}
}

func TestReadMalformedLinesSkipped(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	t.Setenv(PathEnvVar, path)
	SetEnabled(true)

	if err := Write(Entry{Command: "apps list"}); err != nil {
		t.Fatal(err)
	}
	// Append garbage then a valid entry.
	if err := appendRaw(path, "not-json\n"); err != nil {
		t.Fatal(err)
	}
	if err := Write(Entry{Command: "tracks list"}); err != nil {
		t.Fatal(err)
	}

	got, err := Read(Query{})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 valid entries, got %d", len(got))
	}
}

func TestReadMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent.log")
	got, err := ReadFrom(path, Query{})
	if err != nil {
		t.Fatalf("ReadFrom missing file: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(got))
	}
}
