package externaltx

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/playclient"
)

// mockService points the generated client at a local server and records the
// request path, so the test asserts the exact official URL.
func mockService(t *testing.T, record func(*http.Request)) context.Context {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		record(r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	}))
	t.Cleanup(server.Close)
	return playclient.ContextWithServiceFactory(context.Background(),
		func(ctx context.Context) (*playclient.Service, error) {
			return playclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
		})
}

// The official createexternaltransaction method requires the parent in the
// form applications/{packageName}. A bare package name produces a URL the
// API rejects.
func TestCreate_UsesOfficialApplicationsParent(t *testing.T) {
	var gotPath string
	ctx := mockService(t, func(r *http.Request) { gotPath = r.URL.Path })

	cmd := CreateCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--package", "com.example.app",
		"--external-transaction-id", "tx-1",
		"--json", `{"originalPreTaxAmount":{"priceMicros":"9990000","currency":"EUR"}}`,
	}); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Exec(ctx, nil); err != nil {
		t.Fatal(err)
	}

	want := "/androidpublisher/v3/applications/com.example.app/externalTransactions"
	if gotPath != want {
		t.Fatalf("request path = %q, want %q", gotPath, want)
	}
}

func TestGet_UsesOfficialResourceName(t *testing.T) {
	var gotPath string
	ctx := mockService(t, func(r *http.Request) { gotPath = r.URL.Path })

	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--external-transaction-id", "tx-1"}); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Exec(ctx, nil); err != nil {
		t.Fatal(err)
	}

	want := "/androidpublisher/v3/applications/com.example.app/externalTransactions/tx-1"
	if gotPath != want {
		t.Fatalf("request path = %q, want %q", gotPath, want)
	}
}

func TestRefund_UsesOfficialResourceName(t *testing.T) {
	var gotPath string
	ctx := mockService(t, func(r *http.Request) { gotPath = r.URL.Path })

	cmd := RefundCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--package", "com.example.app",
		"--external-transaction-id", "tx-1",
		"--json", `{"refundTime":"2026-08-24T00:00:00Z"}`,
		"--confirm",
	}); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Exec(ctx, nil); err != nil {
		t.Fatal(err)
	}

	want := "/androidpublisher/v3/applications/com.example.app/externalTransactions/tx-1:refund"
	if !strings.HasPrefix(gotPath, want) {
		t.Fatalf("request path = %q, want prefix %q", gotPath, want)
	}
}
