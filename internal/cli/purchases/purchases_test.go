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

func TestSubscriptionsRevokeCommand_CallsV2API(t *testing.T) {
	var gotPath, gotBody string
	installMockPlayService(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	})

	cmd := SubscriptionsRevokeCommand()
	_ = cmd.FlagSet.Parse([]string{
		"--package", "com.example.app",
		"--subscription-id", "premium",
		"--token", "tok",
		"--confirm",
	})
	stdout, err := capturePurchasesStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// The v1 endpoint was removed from androidpublisher/v3; this command now
	// rides on subscriptionsv2, which is keyed by token alone.
	if gotPath != "/androidpublisher/v3/applications/com.example.app/purchases/subscriptionsv2/tokens/tok:revoke" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	// v2 requires an explicit revocation context; a full refund preserves the
	// behavior the v1 endpoint applied implicitly.
	if !strings.Contains(gotBody, "fullRefund") {
		t.Fatalf("expected fullRefund revocation context, got %s", gotBody)
	}
	if !strings.Contains(stdout, `"revoked":true`) {
		t.Fatalf("expected revoked output, got %s", stdout)
	}
	if !strings.Contains(stdout, `"subscriptionId":"premium"`) {
		t.Fatalf("expected subscriptionId echoed in output, got %s", stdout)
	}
}

// --subscription-id is meaningless to the v2 endpoint, so it must not be
// required any more — but passing it stays valid for existing scripts.
func TestSubscriptionsRevokeCommand_SubscriptionIDOptional(t *testing.T) {
	var gotPath string
	installMockPlayService(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	})

	cmd := SubscriptionsRevokeCommand()
	_ = cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--token", "tok", "--confirm"})
	if _, err := capturePurchasesStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	}); err != nil {
		t.Fatalf("expected no error without --subscription-id, got %v", err)
	}
	if gotPath != "/androidpublisher/v3/applications/com.example.app/purchases/subscriptionsv2/tokens/tok:revoke" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
}

func TestSubscriptionsRevokeCommand_Validation(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"missing token", []string{"--package", "com.example.app", "--confirm"}, "--token is required"},
		{"missing confirm", []string{"--package", "com.example.app", "--token", "tok"}, "--confirm is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := SubscriptionsRevokeCommand()
			_ = cmd.FlagSet.Parse(tt.args)
			err := cmd.Exec(context.Background(), nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got %v", tt.want, err)
			}
		})
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
