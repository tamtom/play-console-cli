package appstores

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

func TestCommand_ExposesAllOfficialThirdPartyStoreMethods(t *testing.T) {
	cmd := AppStoresCommand()
	want := map[string]bool{
		"create-app": false, "update-app": false, "publish-status": false,
		"upload-apk": false, "upload-policy-file": false, "upload-image": false,
		"recent-app-view": false, "recent-update-events": false,
	}
	for _, sub := range cmd.Subcommands {
		if _, ok := want[sub.Name]; ok {
			want[sub.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestCreateApp_UsesOfficialAppStoreReviewEndpoint(t *testing.T) {
	var gotPath, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()
	original := newPlayService
	newPlayService = func(ctx context.Context) (*playclient.Service, error) {
		return playclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
	}
	t.Cleanup(func() { newPlayService = original })

	cmd := CreateAppCommand()
	if err := cmd.FlagSet.Parse([]string{"--app-store-package", "store.example", "--package", "app.example", "--registered-third-party-store", "--confirm"}); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Exec(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/androidpublisher/v3/appstore/store.example/apps:create" {
		t.Fatalf("path = %q", gotPath)
	}
	if !strings.Contains(gotBody, `"packageName":"app.example"`) {
		t.Fatalf("body = %s", gotBody)
	}
}

func TestCreateApp_RequiresProgramAcknowledgement(t *testing.T) {
	cmd := CreateAppCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--app-store-package", "store.example",
		"--package", "app.example",
		"--confirm",
	}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--registered-third-party-store") {
		t.Fatalf("expected scope acknowledgement error, got %v", err)
	}
}

func TestPublishStatus_RejectsInvalidStateBeforeAuthentication(t *testing.T) {
	cmd := PublishStatusCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--app-store-package", "store.example",
		"--package", "app.example",
		"--state", "PUBLIC",
		"--registered-third-party-store",
		"--confirm",
	}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "PUBLISHED or UNPUBLISHED") {
		t.Fatalf("expected state validation error, got %v", err)
	}
}

func TestOfficialThirdPartyStoreEndpoints_SuccessAndAPIError(t *testing.T) {
	tempDir := t.TempDir()
	apk := filepath.Join(tempDir, "app.apk")
	pdf := filepath.Join(tempDir, "policy.pdf")
	png := filepath.Join(tempDir, "listing.png")
	for path, contents := range map[string]string{apk: "apk", pdf: "%PDF-test", png: "png"} {
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name       string
		command    func() *ffcli.Command
		args       []string
		method     string
		path       string
		errorLabel string
	}{
		{"create", CreateAppCommand, []string{"--app-store-package", "store.example", "--package", "app.example", "--registered-third-party-store", "--confirm"}, http.MethodPost, "/androidpublisher/v3/appstore/store.example/apps:create", "create third-party"},
		{"update", UpdateAppCommand, []string{"--app-store-package", "store.example", "--json", `{"packageName":"app.example"}`, "--registered-third-party-store", "--confirm"}, http.MethodPost, "/androidpublisher/v3/appstore/store.example/apps:update", "update third-party"},
		{"publish", PublishStatusCommand, []string{"--app-store-package", "store.example", "--package", "app.example", "--state", "PUBLISHED", "--registered-third-party-store", "--confirm"}, http.MethodPost, "/androidpublisher/v3/appstore/store.example/apps/app.example:updateAppStoreHostedAppPublishStatus", "publish status"},
		{"upload-apk", UploadAPKCommand, []string{"--app-store-package", "store.example", "--package", "app.example", "--file", apk, "--registered-third-party-store", "--confirm"}, http.MethodPost, "/upload/androidpublisher/v3/appstore/store.example/apps/app.example/apks:upload", "upload third-party app-store APK"},
		{"upload-policy", UploadPolicyFileCommand, []string{"--app-store-package", "store.example", "--package", "app.example", "--file", pdf, "--registered-third-party-store", "--confirm"}, http.MethodPost, "/upload/androidpublisher/v3/appstore/store.example/apps/app.example/policyDeclarationFiles:upload", "upload third-party app-store policy"},
		{"upload-image", UploadImageCommand, []string{"--app-store-package", "store.example", "--package", "app.example", "--file", png, "--registered-third-party-store", "--confirm"}, http.MethodPost, "/upload/androidpublisher/v3/appstore/store.example/apps/app.example/images:upload", "upload third-party app-store image"},
		{"recent-app-view", RecentAppViewCommand, []string{"--app-store-package", "store.example", "--package", "app.example", "--registered-third-party-store"}, http.MethodGet, "/androidpublisher/v3/appstorecatalog/store.example/recentAppViews/app.example", "recent Play Catalog app view"},
		{"recent-update-events", RecentUpdateEventsCommand, []string{"--app-store-package", "store.example", "--registered-third-party-store", "--paginate"}, http.MethodGet, "/androidpublisher/v3/appstorecatalog/store.example/recentUpdateEvents", "recent Play Catalog update events"},
	}

	for _, tc := range tests {
		t.Run(tc.name+" success", func(t *testing.T) {
			var gotMethod, gotPath string
			server := newAppStoreTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath = r.Method, r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{}`)
			})
			installAppStoreService(t, server)
			cmd := tc.command()
			if err := cmd.FlagSet.Parse(tc.args); err != nil {
				t.Fatal(err)
			}
			if err := cmd.Exec(context.Background(), nil); err != nil {
				t.Fatal(err)
			}
			if gotMethod != tc.method || gotPath != tc.path {
				t.Fatalf("request = %s %s, want %s %s", gotMethod, gotPath, tc.method, tc.path)
			}
		})

		t.Run(tc.name+" API error", func(t *testing.T) {
			server := newAppStoreTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, `{"error":{"message":"denied"}}`, http.StatusForbidden)
			})
			installAppStoreService(t, server)
			cmd := tc.command()
			if err := cmd.FlagSet.Parse(tc.args); err != nil {
				t.Fatal(err)
			}
			err := cmd.Exec(context.Background(), nil)
			if err == nil || !strings.Contains(err.Error(), tc.errorLabel) {
				t.Fatalf("expected contextual API error containing %q, got %v", tc.errorLabel, err)
			}
		})
	}
}

func newAppStoreTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func installAppStoreService(t *testing.T, server *httptest.Server) {
	t.Helper()
	original := newPlayService
	newPlayService = func(ctx context.Context) (*playclient.Service, error) {
		return playclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
	}
	t.Cleanup(func() { newPlayService = original })
}
