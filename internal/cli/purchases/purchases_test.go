package purchases

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

	"github.com/tamtom/play-console-cli/internal/playclient"
)

func TestPurchasesCommand_HasSubscriptionsV2(t *testing.T) {
	cmd := PurchasesCommand()
	found := false
	for _, sub := range cmd.Subcommands {
		if sub.Name == "subscriptionsv2" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected purchases command to expose subscriptionsv2")
	}
}

func TestSubscriptionsCommand_HasAcknowledge(t *testing.T) {
	cmd := SubscriptionsCommand()
	found := false
	for _, sub := range cmd.Subcommands {
		if sub.Name == "acknowledge" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected purchases subscriptions to expose acknowledge")
	}
}

func TestSubscriptionsAcknowledgeCommand_CallsAPI(t *testing.T) {
	var gotPath, gotBody string
	installMockPlayService(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)
	})

	cmd := SubscriptionsAcknowledgeCommand()
	_ = cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--subscription-id", "premium", "--token", "tok", "--developer-payload", "payload"})
	stdout, err := capturePurchasesStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotPath != "/androidpublisher/v3/applications/com.example.app/purchases/subscriptions/premium/tokens/tok:acknowledge" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if !strings.Contains(gotBody, "payload") {
		t.Fatalf("expected developer payload in body, got %s", gotBody)
	}
	if !strings.Contains(stdout, `"acknowledged":true`) {
		t.Fatalf("expected acknowledged output, got %s", stdout)
	}
}

func TestSubscriptionsV2CancelCommand_CallsAPI(t *testing.T) {
	var gotPath, gotBody string
	installMockPlayService(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	})

	cmd := SubscriptionsV2CancelCommand()
	_ = cmd.FlagSet.Parse([]string{
		"--package", "com.example.app",
		"--token", "tok",
		"--json", `{"cancellationContext":{"cancellationType":"USER_REQUESTED_STOP_RENEWALS"}}`,
		"--confirm",
	})
	stdout, err := capturePurchasesStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotPath != "/androidpublisher/v3/applications/com.example.app/purchases/subscriptionsv2/tokens/tok:cancel" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if !strings.Contains(gotBody, "USER_REQUESTED_STOP_RENEWALS") {
		t.Fatalf("expected cancellation request body, got %s", gotBody)
	}
	if !strings.Contains(stdout, `"canceled":true`) {
		t.Fatalf("expected canceled output, got %s", stdout)
	}
}

func installMockPlayService(t *testing.T, handler http.HandlerFunc) {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	original := newPlayService
	newPlayService = func(ctx context.Context) (*playclient.Service, error) {
		return playclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
	}
	t.Cleanup(func() {
		newPlayService = original
	})
}

func capturePurchasesStdout(fn func() error) (string, error) {
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
