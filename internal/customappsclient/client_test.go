package customappsclient

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/api/googleapi"
	playcustomapp "google.golang.org/api/playcustomapp/v1"
)

func TestNewServiceWithClientAppliesBasePath(t *testing.T) {
	svc, err := NewServiceWithClient(context.Background(), http.DefaultClient, "http://example.invalid/")
	if err != nil {
		t.Fatalf("NewServiceWithClient error: %v", err)
	}
	if svc.API == nil {
		t.Fatal("expected playcustomapp service to be built")
	}
	if svc.API.BasePath != "http://example.invalid/" {
		t.Fatalf("BasePath = %q, want override applied", svc.API.BasePath)
	}
}

// TestCreateCustomAppUploadsAndReturns exercises the real create+media-upload
// wire path against a mock server: the request must carry the APK bytes and the
// numeric account, and the decoded CustomApp response is returned to the caller.
func TestCreateCustomAppUploadsAndReturns(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"packageName":"com.example.custom123","title":"Field Tool"}`))
	}))
	defer srv.Close()

	svc, err := NewServiceWithClient(context.Background(), srv.Client(), srv.URL+"/")
	if err != nil {
		t.Fatalf("NewServiceWithClient error: %v", err)
	}

	body := &playcustomapp.CustomApp{Title: "Field Tool", LanguageCode: "en-US"}
	call := svc.API.Accounts.CustomApps.Create(1234567890, body)
	call.Media(strings.NewReader("APK-BYTES"), googleapi.ContentType("application/octet-stream"))
	resp, err := call.Context(context.Background()).Do()
	if err != nil {
		t.Fatalf("Create.Do error: %v", err)
	}
	if resp.PackageName != "com.example.custom123" {
		t.Fatalf("PackageName = %q, want com.example.custom123", resp.PackageName)
	}
	if !strings.Contains(gotBody, "APK-BYTES") {
		t.Fatalf("uploaded request body missing APK bytes; got %q", gotBody)
	}
}
