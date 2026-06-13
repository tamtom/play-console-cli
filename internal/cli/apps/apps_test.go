package apps

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/tamtom/play-console-cli/internal/reportingclient"
)

func TestAppsCommand_Help(t *testing.T) {
	cmd := AppsCommand(nil)
	if cmd.Name != "apps" {
		t.Errorf("Name = %q, want %q", cmd.Name, "apps")
	}
	if len(cmd.Subcommands) == 0 {
		t.Error("expected at least one subcommand")
	}
}

func TestListCommand_Flags(t *testing.T) {
	cmd := ListCommand(nil)
	if cmd.Name != "list" {
		t.Errorf("Name = %q, want %q", cmd.Name, "list")
	}
	var buf bytes.Buffer
	_ = buf
}

func TestAppsCommand_UnknownSubcommand(t *testing.T) {
	cmd := AppsCommand(nil)
	err := cmd.ParseAndRun(context.Background(), []string{"nonexistent"})
	if err == nil {
		t.Error("expected error for unknown subcommand")
	}
}

func TestListCommand_CallsReportingAppsSearch(t *testing.T) {
	var gotPath, gotPageSize string
	installMockReportingService(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotPageSize = r.URL.Query().Get("pageSize")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"apps":[{"name":"apps/com.example.app","packageName":"com.example.app","displayName":"Example"}]}`)
	})

	cmd := ListCommand(nil)
	if err := cmd.FlagSet.Parse([]string{"--page-size", "25"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	stdout, err := captureAppsStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotPath != "/v1beta1/apps:search" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if gotPageSize != "25" {
		t.Fatalf("pageSize = %q, want 25", gotPageSize)
	}
	if !strings.Contains(stdout, "com.example.app") {
		t.Fatalf("expected app in output, got %s", stdout)
	}
}

func TestListCommand_PaginatesReportingAppsSearch(t *testing.T) {
	var pageTokens []string
	installMockReportingService(t, func(w http.ResponseWriter, r *http.Request) {
		pageTokens = append(pageTokens, r.URL.Query().Get("pageToken"))
		w.Header().Set("Content-Type", "application/json")
		if len(pageTokens) == 1 {
			_, _ = io.WriteString(w, `{"apps":[{"packageName":"com.example.one"}],"nextPageToken":"next"}`)
			return
		}
		_, _ = io.WriteString(w, `{"apps":[{"packageName":"com.example.two"}]}`)
	})

	cmd := ListCommand(nil)
	if err := cmd.FlagSet.Parse([]string{"--paginate", "--page-size", "1"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	stdout, err := captureAppsStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(pageTokens) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(pageTokens))
	}
	if pageTokens[0] != "" || pageTokens[1] != "next" {
		t.Fatalf("unexpected page tokens: %#v", pageTokens)
	}
	if !strings.Contains(stdout, "com.example.one") || !strings.Contains(stdout, "com.example.two") {
		t.Fatalf("expected both apps in paginated output, got %s", stdout)
	}
}

func installMockReportingService(t *testing.T, handler http.HandlerFunc) {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	original := newReportingService
	newReportingService = func(ctx context.Context) (*reportingclient.Service, error) {
		return reportingclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
	}
	t.Cleanup(func() {
		newReportingService = original
	})
}

func captureAppsStdout(fn func() error) (string, error) {
	origStdout := os.Stdout
	rOut, wOut, err := os.Pipe()
	if err != nil {
		return "", err
	}

	os.Stdout = wOut

	var buf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(&buf, rOut)
	}()

	runErr := fn()

	_ = wOut.Close()
	os.Stdout = origStdout
	wg.Wait()
	_ = rOut.Close()

	return buf.String(), runErr
}
