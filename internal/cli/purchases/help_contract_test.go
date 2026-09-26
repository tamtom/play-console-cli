package purchases

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/tamtom/play-console-cli/internal/playclient"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestDeferralHelpExamplesOnTheWire(t *testing.T) {
	for _, makeCommand := range []func() *ffcli.Command{SubscriptionsDeferCommand, SubscriptionsV2DeferCommand} {
		cmd := makeCommand()
		t.Run(cmd.FlagSet.Name(), func(t *testing.T) {
			start := strings.Index(cmd.LongHelp, "{")
			end := strings.LastIndex(cmd.LongHelp, "}")
			if start < 0 || end < start {
				t.Fatal("help missing JSON example")
			}
			example := cmd.LongHelp[start : end+1]
			var got []byte
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got, _ = io.ReadAll(r.Body)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{}`)
			}))
			defer server.Close()
			ctx := playclient.ContextWithServiceFactory(context.Background(), func(ctx context.Context) (*playclient.Service, error) {
				return playclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
			})
			args := []string{"--package", "com.example.test", "--token", "test-token", "--json", example}
			if cmd.FlagSet.Lookup("subscription-id") != nil {
				args = append(args, "--subscription-id", "monthly")
			}
			if err := cmd.ParseAndRun(ctx, args); err != nil {
				t.Fatal(err)
			}
			if len(got) == 0 {
				t.Fatal("request not sent")
			}
			if strings.Contains(cmd.FlagSet.Name(), "subscriptionsv2") {
				var payload struct {
					Context struct {
						Duration json.RawMessage `json:"deferDuration"`
					} `json:"deferralContext"`
				}
				if err := json.Unmarshal(got, &payload); err != nil {
					t.Fatal(err)
				}
				var duration durationpb.Duration
				if err := protojson.Unmarshal(payload.Context.Duration, &duration); err != nil {
					t.Fatal(err)
				}
				if duration.Seconds != 604800 {
					t.Fatalf("duration=%v", &duration)
				}
			}
		})
	}
}

func TestV2DeferralRejectsInvalidContextBeforeRequests(t *testing.T) {
	for _, input := range []string{`{}`, `{"deferralContext":{"deferDuration":"604800s"}}`, `{"deferralContext":{"etag":"test","deferDuration":"P7D"}}`, `{"deferralContext":{"etag":"test","deferDuration":"0s"}}`, `{"deferralContext":{"etag":"test","deferDuration":"-1s"}}`} {
		t.Run(input, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests++; fmt.Fprint(w, `{}`) }))
			defer server.Close()
			ctx := playclient.ContextWithServiceFactory(context.Background(), func(ctx context.Context) (*playclient.Service, error) {
				return playclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
			})
			err := SubscriptionsV2DeferCommand().ParseAndRun(ctx, []string{"--package", "com.example.test", "--token", "test", "--json", input})
			if err == nil || requests != 0 || !strings.Contains(err.Error(), "deferralContext") {
				t.Fatalf("err=%v requests=%d", err, requests)
			}
		})
	}
}
