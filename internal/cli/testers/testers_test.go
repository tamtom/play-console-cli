package testers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

func TestTestersCommand_Name(t *testing.T) {
	cmd := TestersCommand()
	if cmd.Name != "testers" {
		t.Errorf("expected name %q, got %q", "testers", cmd.Name)
	}
}

func TestTestersCommand_ShortHelp(t *testing.T) {
	cmd := TestersCommand()
	if cmd.ShortHelp == "" {
		t.Error("expected non-empty ShortHelp")
	}
}

func TestTestersCommand_UsageFunc(t *testing.T) {
	cmd := TestersCommand()
	if cmd.UsageFunc == nil {
		t.Error("expected UsageFunc to be set")
	}
}

func TestTestersCommand_HasSubcommands(t *testing.T) {
	cmd := TestersCommand()
	if len(cmd.Subcommands) == 0 {
		t.Error("expected subcommands")
	}
}

func TestTestersCommand_SubcommandNames(t *testing.T) {
	cmd := TestersCommand()
	expected := map[string]bool{
		"get":    false,
		"update": false,
		"patch":  false,
	}
	for _, sub := range cmd.Subcommands {
		if _, ok := expected[sub.Name]; ok {
			expected[sub.Name] = true
		} else {
			t.Errorf("unexpected subcommand: %s", sub.Name)
		}
	}
	for name, found := range expected {
		if !found {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

func TestTestersCommand_SubcommandsHaveUsageFunc(t *testing.T) {
	cmd := TestersCommand()
	for _, sub := range cmd.Subcommands {
		if sub.UsageFunc == nil {
			t.Errorf("subcommand %q missing UsageFunc", sub.Name)
		}
	}
}

func TestTestersCommand_SubcommandsHaveShortHelp(t *testing.T) {
	cmd := TestersCommand()
	for _, sub := range cmd.Subcommands {
		if sub.ShortHelp == "" {
			t.Errorf("subcommand %q missing ShortHelp", sub.Name)
		}
	}
}

func TestTestersCommand_NoArgs_ReturnsHelp(t *testing.T) {
	cmd := TestersCommand()
	err := cmd.Exec(context.Background(), nil)
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

// --- testers get ---

func TestTestersGetCommand_Name(t *testing.T) {
	cmd := GetCommand()
	if cmd.Name != "get" {
		t.Errorf("expected name %q, got %q", "get", cmd.Name)
	}
}

func TestTestersGetCommand_MissingEdit(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{"--track", "internal"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --edit")
	}
	if !strings.Contains(err.Error(), "--edit") {
		t.Errorf("error should mention --edit, got: %s", err.Error())
	}
}

func TestTestersGetCommand_MissingTrack(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{"--edit", "abc123"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --track")
	}
	if !strings.Contains(err.Error(), "--track") {
		t.Errorf("error should mention --track, got: %s", err.Error())
	}
}

func TestTestersGetCommand_WhitespaceEdit(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{"--edit", "   ", "--track", "internal"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --edit")
	}
	if !strings.Contains(err.Error(), "--edit") {
		t.Errorf("error should mention --edit, got: %s", err.Error())
	}
}

func TestTestersGetCommand_WhitespaceTrack(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{"--edit", "abc123", "--track", "   "}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --track")
	}
	if !strings.Contains(err.Error(), "--track") {
		t.Errorf("error should mention --track, got: %s", err.Error())
	}
}

func TestTestersGetCommand_InvalidOutputFormat(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "xml"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

func TestTestersGetCommand_PrettyWithTable(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "table", "--pretty"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for --pretty with table output")
	}
	if !strings.Contains(err.Error(), "--pretty") {
		t.Errorf("error should mention --pretty, got: %s", err.Error())
	}
}

// --- testers update ---

func TestTestersUpdateCommand_Name(t *testing.T) {
	cmd := UpdateCommand()
	if cmd.Name != "update" {
		t.Errorf("expected name %q, got %q", "update", cmd.Name)
	}
}

func TestTestersUpdateCommand_MissingEdit(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--track", "internal"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --edit")
	}
	if !strings.Contains(err.Error(), "--edit") {
		t.Errorf("error should mention --edit, got: %s", err.Error())
	}
}

func TestTestersUpdateCommand_MissingTrack(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--edit", "abc123"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --track")
	}
	if !strings.Contains(err.Error(), "--track") {
		t.Errorf("error should mention --track, got: %s", err.Error())
	}
}

func TestTestersUpdateCommand_PrettyWithMarkdown(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "markdown", "--pretty"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for --pretty with markdown output")
	}
	if !strings.Contains(err.Error(), "--pretty") {
		t.Errorf("error should mention --pretty, got: %s", err.Error())
	}
}

func TestTestersUpdateCommand_RejectsUnsupportedIndividualEmailsBeforeServiceCreation(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--package", "com.example.app",
		"--edit", "edit-1",
		"--track", "internal",
		"--emails", "person@example.com",
	}); err != nil {
		t.Fatal(err)
	}

	serviceCreated := false
	ctx := playclient.ContextWithServiceFactory(context.Background(), func(context.Context) (*playclient.Service, error) {
		serviceCreated = true
		return nil, errors.New("service should not be created")
	})

	err := cmd.Exec(ctx, nil)
	if err == nil {
		t.Fatal("expected --emails to be rejected")
	}
	if !strings.Contains(err.Error(), "--google-groups") {
		t.Fatalf("error = %q, want guidance to use --google-groups", err)
	}
	if serviceCreated {
		t.Fatal("service was created before rejecting unsupported --emails")
	}
}

func TestTestersUpdateCommand_SendsGoogleGroups(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody struct {
		GoogleGroups []string `json:"googleGroups"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"googleGroups":["beta-testers@example.com"]}`))
	}))
	t.Cleanup(server.Close)

	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--package", "com.example.app",
		"--edit", "edit-1",
		"--track", "internal",
		"--google-groups", " beta-testers@example.com ",
		"--confirm",
	}); err != nil {
		t.Fatal(err)
	}
	ctx := playclient.ContextWithServiceFactory(context.Background(), func(ctx context.Context) (*playclient.Service, error) {
		return playclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
	})
	ctx = shared.ContextWithIO(ctx, &bytes.Buffer{}, &bytes.Buffer{})
	if err := cmd.Exec(ctx, nil); err != nil {
		t.Fatalf("update testers: %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Fatalf("method = %q, want PUT", gotMethod)
	}
	wantPath := "/androidpublisher/v3/applications/com.example.app/edits/edit-1/testers/internal"
	if gotPath != wantPath {
		t.Fatalf("path = %q, want %q", gotPath, wantPath)
	}
	if len(gotBody.GoogleGroups) != 1 || gotBody.GoogleGroups[0] != "beta-testers@example.com" {
		t.Fatalf("googleGroups = %#v, want trimmed group", gotBody.GoogleGroups)
	}
}

func TestTestersUpdateCommand_RequiresConfirmationBeforeServiceCreation(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--package", "com.example.app",
		"--edit", "edit-1",
		"--track", "internal",
		"--google-groups", "beta-testers@example.com",
	}); err != nil {
		t.Fatal(err)
	}
	serviceCreated := false
	ctx := playclient.ContextWithServiceFactory(context.Background(), func(context.Context) (*playclient.Service, error) {
		serviceCreated = true
		return nil, errors.New("service should not be created")
	})
	if err := cmd.Exec(ctx, nil); err == nil || !strings.Contains(err.Error(), "--confirm") {
		t.Fatalf("error = %v, want --confirm requirement", err)
	}
	if serviceCreated {
		t.Fatal("service was created before enforcing confirmation")
	}
}

func TestTestersUpdateCommand_RejectsUnknownJSONBeforeServiceCreation(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--package", "com.example.app",
		"--edit", "edit-1",
		"--track", "internal",
		"--json", `{"emails":["person@example.com"]}`,
		"--confirm",
	}); err != nil {
		t.Fatal(err)
	}
	serviceCreated := false
	ctx := playclient.ContextWithServiceFactory(context.Background(), func(context.Context) (*playclient.Service, error) {
		serviceCreated = true
		return nil, errors.New("service should not be created")
	})
	err := cmd.Exec(ctx, nil)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("error = %v, want unknown JSON field rejection", err)
	}
	if serviceCreated {
		t.Fatal("service was created before rejecting unknown JSON")
	}
}

// --- testers patch ---

func TestTestersPatchCommand_Name(t *testing.T) {
	cmd := PatchCommand()
	if cmd.Name != "patch" {
		t.Errorf("expected name %q, got %q", "patch", cmd.Name)
	}
}

func TestTestersPatchCommand_MissingEdit(t *testing.T) {
	cmd := PatchCommand()
	if err := cmd.FlagSet.Parse([]string{"--track", "internal"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --edit")
	}
	if !strings.Contains(err.Error(), "--edit") {
		t.Errorf("error should mention --edit, got: %s", err.Error())
	}
}

func TestTestersPatchCommand_MissingTrack(t *testing.T) {
	cmd := PatchCommand()
	if err := cmd.FlagSet.Parse([]string{"--edit", "abc123"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --track")
	}
	if !strings.Contains(err.Error(), "--track") {
		t.Errorf("error should mention --track, got: %s", err.Error())
	}
}

func TestTestersPatchCommand_RejectsUnsupportedIndividualEmailsBeforeServiceCreation(t *testing.T) {
	cmd := PatchCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--package", "com.example.app",
		"--edit", "edit-1",
		"--track", "internal",
		"--emails", "person@example.com",
	}); err != nil {
		t.Fatal(err)
	}

	serviceCreated := false
	ctx := playclient.ContextWithServiceFactory(context.Background(), func(context.Context) (*playclient.Service, error) {
		serviceCreated = true
		return nil, errors.New("service should not be created")
	})

	err := cmd.Exec(ctx, nil)
	if err == nil {
		t.Fatal("expected --emails to be rejected")
	}
	if !strings.Contains(err.Error(), "--google-groups") {
		t.Fatalf("error = %q, want guidance to use --google-groups", err)
	}
	if serviceCreated {
		t.Fatal("service was created before rejecting unsupported --emails")
	}
}

func TestTestersUpdateCommand_GoogleGroupsDropsEmptyEntries(t *testing.T) {
	tests := []struct {
		name     string
		groups   string
		wantBody string
	}{
		{name: "empty value clears groups", groups: "", wantBody: `{"googleGroups":[]}`},
		{name: "blank entries are dropped", groups: "a@example.com, ,b@example.com,", wantBody: `{"googleGroups":["a@example.com","b@example.com"]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotBody string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var buf bytes.Buffer
				_, _ = buf.ReadFrom(r.Body)
				gotBody = strings.TrimSpace(buf.String())
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`))
			}))
			t.Cleanup(server.Close)

			cmd := UpdateCommand()
			if err := cmd.FlagSet.Parse([]string{
				"--package", "com.example.app",
				"--edit", "edit-1",
				"--track", "internal",
				"--google-groups", tt.groups,
				"--confirm",
			}); err != nil {
				t.Fatal(err)
			}
			ctx := playclient.ContextWithServiceFactory(context.Background(), func(ctx context.Context) (*playclient.Service, error) {
				return playclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
			})
			ctx = shared.ContextWithIO(ctx, &bytes.Buffer{}, &bytes.Buffer{})
			if err := cmd.Exec(ctx, nil); err != nil {
				t.Fatalf("update testers: %v", err)
			}
			if gotBody != tt.wantBody {
				t.Fatalf("body = %s, want %s", gotBody, tt.wantBody)
			}
		})
	}
}
