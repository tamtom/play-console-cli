package playclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tamtom/play-console-cli/internal/config"
)

func TestNewAuthenticatedClient_RetriesTransientReadFailures(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempt := requests.Add(1)
		if attempt < 3 {
			http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)

	tokenPath := filepath.Join(t.TempDir(), "oauth-token.json")
	token := fmt.Sprintf(`{"access_token":"test-token","token_type":"Bearer","expiry":%q}`, time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano))
	if err := os.WriteFile(tokenPath, []byte(token), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GPLAY_CONFIG_PATH", filepath.Join(t.TempDir(), "missing-config.json"))
	t.Setenv(oauthTokenEnvVar, tokenPath)
	t.Setenv(oauthClientIDEnvVar, "test-client")
	t.Setenv(oauthClientSecretEnvVar, "test-secret")
	t.Setenv("GPLAY_MAX_RETRIES", "2")
	t.Setenv("GPLAY_RETRY_DELAY", "1ms")

	client, _, err := NewAuthenticatedClient(context.Background())
	if err != nil {
		t.Fatalf("NewAuthenticatedClient: %v", err)
	}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if got := requests.Load(); got != 3 {
		t.Fatalf("requests = %d, want 3", got)
	}
}

func TestRetryTransport_DoesNotReplayMutation(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	client := &http.Client{Transport: &retryTransport{
		base:       server.Client().Transport,
		maxRetries: 3,
		baseDelay:  time.Millisecond,
	}}
	resp, err := client.Post(server.URL, "application/json", nil)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("mutation requests = %d, want exactly 1", got)
	}
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, time.August, 30, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		value string
		want  time.Duration
		ok    bool
	}{
		{name: "seconds", value: "3", want: 3 * time.Second, ok: true},
		{name: "http date", value: now.Add(5 * time.Second).Format(http.TimeFormat), want: 5 * time.Second, ok: true},
		{name: "invalid", value: "later", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseRetryAfter(tt.value, now)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("parseRetryAfter(%q) = (%s, %t), want (%s, %t)", tt.value, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestRetrySettings_ConfiguredZeroDisablesRetries(t *testing.T) {
	var cfg config.Config
	if err := json.Unmarshal([]byte(`{"max_retries":0}`), &cfg); err != nil {
		t.Fatal(err)
	}
	maxRetries, _, err := retrySettings(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	if maxRetries != 0 {
		t.Fatalf("max retries = %d, want 0", maxRetries)
	}
}

func TestApplyRetryPolicy_WrapsInvalidSettings(t *testing.T) {
	t.Setenv("GPLAY_MAX_RETRIES", "invalid")
	err := ApplyRetryPolicy(&http.Client{}, nil)
	if err == nil || !strings.Contains(err.Error(), "resolve retry settings") {
		t.Fatalf("error = %v, want retry settings context", err)
	}
}

func TestRetryTransport_StopsAfterConfiguredAttempts(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)
	client := &http.Client{Transport: &retryTransport{base: server.Client().Transport, maxRetries: 2, baseDelay: time.Millisecond}}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if got := requests.Load(); got != 3 {
		t.Fatalf("requests = %d, want initial request plus 2 retries", got)
	}
}

func TestRetryTransport_HonorsCancellationDuringBackoff(t *testing.T) {
	var requests atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
		go cancel()
	}))
	t.Cleanup(server.Close)
	client := &http.Client{Transport: &retryTransport{base: server.Client().Transport, maxRetries: 3, baseDelay: time.Hour}}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Do(request)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context canceled", err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("requests = %d, want one request before cancellation stopped retries", got)
	}
}

func TestRetryTransport_RetriesRetryAfterResponse(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			http.Error(w, "slow down", http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	client := &http.Client{Transport: &retryTransport{base: server.Client().Transport, maxRetries: 1, baseDelay: time.Hour}}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || requests.Load() != 2 {
		t.Fatalf("status = %d, requests = %d; want 200 and 2", resp.StatusCode, requests.Load())
	}
}
