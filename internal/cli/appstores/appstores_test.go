package appstores

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/playclient"
)

func TestCommand_ExposesAllOfficialThirdPartyStoreMethods(t *testing.T) {
	cmd := Command()
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
