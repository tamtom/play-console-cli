package orders

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/tamtom/play-console-cli/internal/playclient"
)

func TestBatchGetCommand_UsesBatchEndpoint(t *testing.T) {
	var gotPath string
	var gotOrderIDs []string
	installMockOrdersService(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotOrderIDs = r.URL.Query()["orderIds"]
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"orders":[{"orderId":"GPA.1"},{"orderId":"GPA.2"}]}`)
	})

	cmd := BatchGetCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--package", "dev.example.app",
		"--order-ids", "GPA.1,GPA.2",
	}); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureOrdersStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("batch get failed: %v", err)
	}
	if gotPath != "/androidpublisher/v3/applications/dev.example.app/orders:batchGet" {
		t.Fatalf("path = %q", gotPath)
	}
	if !reflect.DeepEqual(gotOrderIDs, []string{"GPA.1", "GPA.2"}) {
		t.Fatalf("orderIds = %#v", gotOrderIDs)
	}
	if !strings.Contains(stdout, `"orderId":"GPA.2"`) {
		t.Fatalf("unexpected output: %s", stdout)
	}
}

func TestReviewRefundCommand_SubmitsExplicitPreference(t *testing.T) {
	var gotPath, gotBody string
	installMockOrdersService(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusNoContent)
	})

	cmd := ReviewRefundCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--package", "dev.example.app",
		"--order-id", "GPA.123",
		"--json", `{"pendingRefundToken":"token","refundPreference":"DECLINE","sampleContentProvided":true}`,
		"--confirm",
	}); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureOrdersStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("review refund failed: %v", err)
	}
	if gotPath != "/androidpublisher/v3/applications/dev.example.app/orders/GPA.123:reviewrefund" {
		t.Fatalf("path = %q", gotPath)
	}
	if !strings.Contains(gotBody, `"refundPreference":"DECLINE"`) || !strings.Contains(gotBody, `"pendingRefundToken":"token"`) {
		t.Fatalf("unexpected request body: %s", gotBody)
	}
	if !strings.Contains(stdout, `"reviewed":true`) {
		t.Fatalf("unexpected output: %s", stdout)
	}
}

func TestParseOrderIDs_DeduplicatesAndEnforcesOfficialLimit(t *testing.T) {
	ids, err := parseOrderIDs("GPA.1, GPA.2, GPA.1")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ids, []string{"GPA.1", "GPA.2"}) {
		t.Fatalf("ids = %#v", ids)
	}

	tooMany := make([]string, 1001)
	for i := range tooMany {
		tooMany[i] = "GPA." + strconv.Itoa(i)
	}
	if _, err := parseOrderIDs(strings.Join(tooMany, ",")); err == nil || !strings.Contains(err.Error(), "1,000") {
		t.Fatalf("expected official limit error, got %v", err)
	}
}

func TestReviewRefund_RequiresExplicitSampleContentBoolean(t *testing.T) {
	cmd := ReviewRefundCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "dev.example", "--order-id", "GPA.1", "--json", `{"pendingRefundToken":"token","refundPreference":"NEUTRAL"}`, "--confirm"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "sampleContentProvided") {
		t.Fatalf("expected required boolean error, got %v", err)
	}
}

func installMockOrdersService(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	original := newPlayService
	newPlayService = func(ctx context.Context) (*playclient.Service, error) {
		return playclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
	}
	t.Cleanup(func() { newPlayService = original })
}

func captureOrdersStdout(fn func() error) (string, error) {
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = writer
	var output bytes.Buffer
	var wait sync.WaitGroup
	wait.Add(1)
	go func() {
		defer wait.Done()
		_, _ = io.Copy(&output, reader)
	}()
	runErr := fn()
	_ = writer.Close()
	os.Stdout = original
	wait.Wait()
	_ = reader.Close()
	return output.String(), runErr
}
