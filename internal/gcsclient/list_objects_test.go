package gcsclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestListObjects_StopsWhenGeneratedPagerRepeatsToken(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"nextPageToken": "repeated"})
	}))
	t.Cleanup(server.Close)
	service, err := NewServiceWithClient(context.Background(), server.Client(), server.URL+"/storage/v1/")
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.ListObjects(context.Background(), "bucket", "prefix")
	if err == nil || !strings.Contains(err.Error(), "repeated page token") {
		t.Fatalf("error = %v, want repeated token guard", err)
	}
	if got := requests.Load(); got != 2 {
		t.Fatalf("requests = %d, want 2", got)
	}
}
