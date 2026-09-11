package onetimeproducts

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

type requestExpectation struct {
	method string
	path   string
	query  url.Values
	body   any
}

func TestBatchUpdateReportsPerItemPartialFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"oneTimeProducts":[{"packageName":"com.example.app","productId":"one"}]}`)
	}))
	t.Cleanup(server.Close)

	ctx := playclient.ContextWithServiceFactory(context.Background(), func(ctx context.Context) (*playclient.Service, error) {
		return playclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
	})
	var stdout bytes.Buffer
	ctx = shared.ContextWithIO(ctx, &stdout, io.Discard)
	command := BatchUpdateCommand()
	if err := command.FlagSet.Parse([]string{
		"--package", "com.example.app",
		"--json", `{"requests":[{"oneTimeProduct":{"productId":"one","listings":[{"languageCode":"en-US","title":"One","description":"One"}]},"updateMask":"listings"},{"oneTimeProduct":{"productId":"two","listings":[{"languageCode":"en-US","title":"Two","description":"Two"}]},"updateMask":"listings"}]}`,
	}); err != nil {
		t.Fatal(err)
	}
	err := command.Exec(ctx, nil)
	if err == nil || !shared.IsReportedError(err) {
		t.Fatalf("partial response error = %v", err)
	}
	for _, fragment := range []string{`"productId":"one","status":"succeeded"`, `"productId":"two","status":"failed"`} {
		if !strings.Contains(stdout.String(), fragment) {
			t.Fatalf("per-item output missing %q: %s", fragment, stdout.String())
		}
	}
}

func TestOneTimeProductRequestShapes(t *testing.T) {
	t.Parallel()

	productBody := decodeJSON(t, `{"packageName":"com.example.app","productId":"coins.v1","listings":[{"languageCode":"en-US","title":"Coins","description":"Coins"}]}`)
	batchUpdateBody := decodeJSON(t, `{"requests":[{"oneTimeProduct":{"packageName":"com.example.app","productId":"coins.v1","listings":[{"languageCode":"en-US","title":"Coins","description":"Coins"}]},"updateMask":"listings","latencyTolerance":"PRODUCT_UPDATE_LATENCY_TOLERANCE_LATENCY_TOLERANT"}]}`)
	batchDeleteBody := decodeJSON(t, `{"requests":[{"packageName":"com.example.app","productId":"coins.v1","latencyTolerance":"PRODUCT_UPDATE_LATENCY_TOLERANCE_LATENCY_TOLERANT"}]}`)
	tests := []struct {
		name     string
		command  func() *ffcli.Command
		args     []string
		want     requestExpectation
		response string
	}{
		{
			name:     "list",
			command:  ListCommand,
			args:     []string{"--package", "com.example.app"},
			want:     requestExpectation{method: http.MethodGet, path: "/androidpublisher/v3/applications/com.example.app/oneTimeProducts", query: url.Values{"pageSize": {"100"}}},
			response: `{"oneTimeProducts":[{"packageName":"com.example.app","productId":"coins.v1"}]}`,
		},
		{
			name:     "get preserves legal period",
			command:  GetCommand,
			args:     []string{"--package", "com.example.app", "--product-id", "coins.v1"},
			want:     requestExpectation{method: http.MethodGet, path: "/androidpublisher/v3/applications/com.example.app/oneTimeProducts/coins.v1", query: url.Values{}},
			response: `{"packageName":"com.example.app","productId":"coins.v1"}`,
		},
		{
			name:    "create uses patch allow missing",
			command: CreateCommand,
			args:    []string{"--package", "com.example.app", "--product-id", "coins.v1", "--json", `{"listings":[{"languageCode":"en-US","title":"Coins","description":"Coins"}]}`, "--regions-version", "2025/02", "--latency-tolerance", latencyTolerant},
			want: requestExpectation{
				method: http.MethodPatch,
				path:   "/androidpublisher/v3/applications/com.example.app/onetimeproducts/coins.v1",
				query:  url.Values{"allowMissing": {"true"}, "latencyTolerance": {latencyTolerant}, "regionsVersion.version": {"2025/02"}, "updateMask": {"listings"}},
				body:   productBody,
			},
			response: `{"packageName":"com.example.app","productId":"coins.v1"}`,
		},
		{
			name:    "update forbids allow missing",
			command: UpdateCommand,
			args:    []string{"--package", "com.example.app", "--product-id", "coins.v1", "--json", `{"listings":[{"languageCode":"en-US","title":"Coins","description":"Coins"}]}`, "--update-mask", "listings", "--regions-version", "2025/02", "--latency-tolerance", latencyTolerant},
			want: requestExpectation{
				method: http.MethodPatch,
				path:   "/androidpublisher/v3/applications/com.example.app/onetimeproducts/coins.v1",
				query:  url.Values{"allowMissing": {"false"}, "latencyTolerance": {latencyTolerant}, "regionsVersion.version": {"2025/02"}, "updateMask": {"listings"}},
				body:   productBody,
			},
			response: `{"packageName":"com.example.app","productId":"coins.v1"}`,
		},
		{
			name:     "delete",
			command:  DeleteCommand,
			args:     []string{"--package", "com.example.app", "--product-id", "coins.v1", "--confirm", "--latency-tolerance", latencyTolerant},
			want:     requestExpectation{method: http.MethodDelete, path: "/androidpublisher/v3/applications/com.example.app/oneTimeProducts/coins.v1", query: url.Values{"latencyTolerance": {latencyTolerant}}},
			response: `{}`,
		},
		{
			name:     "batch get",
			command:  BatchGetCommand,
			args:     []string{"--package", "com.example.app", "--product-ids", "coins.v1,gems_2"},
			want:     requestExpectation{method: http.MethodGet, path: "/androidpublisher/v3/applications/com.example.app/oneTimeProducts:batchGet", query: url.Values{"productIds": {"coins.v1", "gems_2"}}},
			response: `{"oneTimeProducts":[]}`,
		},
		{
			name:     "batch update",
			command:  BatchUpdateCommand,
			args:     []string{"--package", "com.example.app", "--json", `{"requests":[{"oneTimeProduct":{"productId":"coins.v1","listings":[{"languageCode":"en-US","title":"Coins","description":"Coins"}]},"updateMask":"listings"}]}`, "--latency-tolerance", latencyTolerant},
			want:     requestExpectation{method: http.MethodPost, path: "/androidpublisher/v3/applications/com.example.app/oneTimeProducts:batchUpdate", query: url.Values{}, body: batchUpdateBody},
			response: `{"oneTimeProducts":[{"packageName":"com.example.app","productId":"coins.v1"}]}`,
		},
		{
			name:     "batch delete",
			command:  BatchDeleteCommand,
			args:     []string{"--package", "com.example.app", "--json", `{"requests":[{"productId":"coins.v1"}]}`, "--confirm", "--latency-tolerance", latencyTolerant},
			want:     requestExpectation{method: http.MethodPost, path: "/androidpublisher/v3/applications/com.example.app/oneTimeProducts:batchDelete", query: url.Values{}, body: batchDeleteBody},
			response: `{}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runRequestShape(t, test.command(), test.args, test.want, test.response)
		})
	}
}

func decodeJSON(t *testing.T, value string) any {
	t.Helper()
	var decoded any
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		t.Fatal(err)
	}
	return decoded
}

func runRequestShape(t *testing.T, command *ffcli.Command, args []string, want requestExpectation, response string) {
	t.Helper()

	var received int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received++
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		if r.Method != want.method {
			t.Errorf("method = %q, want %q", r.Method, want.method)
		}
		if r.URL.EscapedPath() != want.path {
			t.Errorf("path = %q, want %q", r.URL.EscapedPath(), want.path)
		}
		query := r.URL.Query()
		query.Del("alt")
		query.Del("prettyPrint")
		if !reflect.DeepEqual(query, want.query) {
			t.Errorf("query = %#v, want %#v", query, want.query)
		}
		if want.body != nil {
			var gotBody any
			if err := json.Unmarshal(body, &gotBody); err != nil {
				t.Errorf("decode body %q: %v", body, err)
			} else if !reflect.DeepEqual(gotBody, want.body) {
				t.Errorf("body = %#v, want %#v", gotBody, want.body)
			}
		} else if len(bytes.TrimSpace(body)) != 0 {
			t.Errorf("body = %q, want empty", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, response)
	}))
	t.Cleanup(server.Close)

	ctx := playclient.ContextWithServiceFactory(context.Background(), func(ctx context.Context) (*playclient.Service, error) {
		return playclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
	})
	ctx = shared.ContextWithIO(ctx, io.Discard, io.Discard)
	if err := command.FlagSet.Parse(args); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := command.Exec(ctx, nil); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if received != 1 {
		t.Fatalf("received %d requests, want 1", received)
	}
}
