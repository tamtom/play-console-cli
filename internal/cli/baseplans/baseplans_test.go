package baseplans

import (
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

func TestBasePlanMissingEndpointRequestShapes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command func() *ffcli.Command
		args    []string
		method  string
		path    string
		body    any
	}{
		{
			name:    "delete",
			command: DeleteCommand,
			args:    []string{"--package", "com.example.app", "--product-id", "premium.v1", "--base-plan-id", "monthly", "--confirm"},
			method:  http.MethodDelete,
			path:    "/androidpublisher/v3/applications/com.example.app/subscriptions/premium.v1/basePlans/monthly",
		},
		{
			name:    "batch update states",
			command: BatchUpdateStatesCommand,
			args: []string{
				"--package", "com.example.app",
				"--product-id", "premium.v1",
				"--json", `{"requests":[{"activateBasePlanRequest":{"basePlanId":"monthly"}},{"deactivateBasePlanRequest":{"basePlanId":"yearly"}}]}`,
				"--latency-tolerance", "PRODUCT_UPDATE_LATENCY_TOLERANCE_LATENCY_TOLERANT",
			},
			method: http.MethodPost,
			path:   "/androidpublisher/v3/applications/com.example.app/subscriptions/premium.v1/basePlans:batchUpdateStates",
			body:   decodeBasePlanJSON(t, `{"requests":[{"activateBasePlanRequest":{"basePlanId":"monthly","packageName":"com.example.app","productId":"premium.v1","latencyTolerance":"PRODUCT_UPDATE_LATENCY_TOLERANCE_LATENCY_TOLERANT"}},{"deactivateBasePlanRequest":{"basePlanId":"yearly","packageName":"com.example.app","productId":"premium.v1","latencyTolerance":"PRODUCT_UPDATE_LATENCY_TOLERANCE_LATENCY_TOLERANT"}}]}`),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var received int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				received++
				if r.Method != test.method {
					t.Errorf("method = %q, want %q", r.Method, test.method)
				}
				if r.URL.EscapedPath() != test.path {
					t.Errorf("path = %q, want %q", r.URL.EscapedPath(), test.path)
				}
				query := r.URL.Query()
				query.Del("alt")
				query.Del("prettyPrint")
				if !reflect.DeepEqual(query, url.Values{}) {
					t.Errorf("query = %#v, want empty", query)
				}
				if test.body != nil {
					var body any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Errorf("decode body: %v", err)
					} else if !reflect.DeepEqual(body, test.body) {
						t.Errorf("body = %#v, want %#v", body, test.body)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				if test.method == http.MethodPost {
					_, _ = io.WriteString(w, `{"subscriptions":[]}`)
				} else {
					_, _ = io.WriteString(w, `{}`)
				}
			}))
			t.Cleanup(server.Close)

			ctx := playclient.ContextWithServiceFactory(context.Background(), func(ctx context.Context) (*playclient.Service, error) {
				return playclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
			})
			ctx = shared.ContextWithIO(ctx, io.Discard, io.Discard)
			command := test.command()
			if err := command.FlagSet.Parse(test.args); err != nil {
				t.Fatal(err)
			}
			if err := command.Exec(ctx, nil); err != nil {
				t.Fatal(err)
			}
			if received != 1 {
				t.Fatalf("received %d requests, want 1", received)
			}
		})
	}
}

func TestBatchUpdateStatesEnforcesLimit(t *testing.T) {
	requests := make([]string, maxBasePlanBatchSize+1)
	for index := range requests {
		requests[index] = `{"activateBasePlanRequest":{"basePlanId":"plan` + strings.Repeat("x", index+1) + `"}}`
	}
	command := BatchUpdateStatesCommand()
	if err := command.FlagSet.Parse([]string{"--product-id", "premium", "--json", `{"requests":[` + strings.Join(requests, ",") + `]}`}); err != nil {
		t.Fatal(err)
	}
	err := command.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "at most 100") {
		t.Fatalf("oversized batch error = %v", err)
	}
}

func decodeBasePlanJSON(t *testing.T, value string) any {
	t.Helper()
	var decoded any
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		t.Fatal(err)
	}
	return decoded
}
