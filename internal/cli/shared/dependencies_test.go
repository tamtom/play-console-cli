package shared

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestContextClockOverridesProcessTime(t *testing.T) {
	want := time.Date(2042, time.March, 4, 5, 6, 7, 0, time.UTC)
	ctx := ContextWithClock(context.Background(), ClockFunc(func() time.Time { return want }))
	if got := Now(ctx); !got.Equal(want) {
		t.Fatalf("Now = %s, want %s", got, want)
	}
}

type recordingFilesystem struct {
	writes []string
}

func (f *recordingFilesystem) ReadFile(path string) ([]byte, error) { return []byte(path), nil }
func (f *recordingFilesystem) AtomicWriteFile(path string, _ []byte, _, _ os.FileMode) error {
	f.writes = append(f.writes, path)
	return nil
}

func TestContextFilesystemOverridesHostFiles(t *testing.T) {
	filesystem := &recordingFilesystem{}
	ctx := ContextWithFilesystem(context.Background(), filesystem)
	if err := FilesystemFrom(ctx).AtomicWriteFile("receipt.json", []byte("{}"), 0o600, 0o700); err != nil {
		t.Fatalf("AtomicWriteFile: %v", err)
	}
	if len(filesystem.writes) != 1 || filesystem.writes[0] != "receipt.json" {
		t.Fatalf("writes = %#v", filesystem.writes)
	}
}

func TestPrintOutputContextUsesInjectedWriter(t *testing.T) {
	var stdout, stderr bytes.Buffer
	ctx := ContextWithIO(context.Background(), &stdout, &stderr)
	if err := PrintOutputContext(ctx, map[string]string{"status": "ok"}, "json", false); err != nil {
		t.Fatalf("PrintOutputContext: %v", err)
	}
	if strings.TrimSpace(stdout.String()) != `{"status":"ok"}` {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
