package checks

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"
	checksapi "google.golang.org/api/checks/v1alpha"

	"github.com/tamtom/play-console-cli/internal/checksclient"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

func TestRepoScansCommand_ExposesCompleteOfficialSurface(t *testing.T) {
	cmd := RepoScansCommand()
	want := map[string]bool{"generate": false, "get": false, "list": false, "operation": false}
	for _, sub := range cmd.Subcommands {
		if _, ok := want[sub.Name]; ok {
			want[sub.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("missing %s", name)
		}
	}
}

func TestRepoScanGenerate_RequiresConfirmationBeforeAuthentication(t *testing.T) {
	cmd := RepoScanGenerateCommand()
	if err := cmd.FlagSet.Parse([]string{"--account", "123", "--repo", "repo1", "--json", `{}`}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--confirm") {
		t.Fatalf("expected --confirm error, got %v", err)
	}
}

func TestRepoScanGenerateRejectsLiteralJSONBeforeAuthentication(t *testing.T) {
	cmd := RepoScanGenerateCommand()
	if err := cmd.Parse([]string{"--account", "123", "--repo", "repo1", "--json", `{}`, "--confirm"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "must use @file") {
		t.Fatalf("expected @file privacy error, got %v", err)
	}
}

func TestRepoResource(t *testing.T) {
	if got := repoResource("123", "repo1"); got != "accounts/123/repos/repo1" {
		t.Fatalf("got %q", got)
	}
	if got := repoScanResource("123", "repo1", "scan1"); got != "accounts/123/repos/repo1/scans/scan1" {
		t.Fatalf("got %q", got)
	}
}

func TestRepoScanGenerateRejectsCredentialShapedSourceBeforeAuthentication(t *testing.T) {
	authCalls := 0
	ctx := checksclient.ContextWithServiceFactory(context.Background(), func(context.Context) (*checksclient.Service, error) {
		authCalls++
		return nil, nil
	})
	command := RepoScanGenerateCommand()
	request := validRepoScanRequest()
	request.CliAnalysis.CodeScans[0].SourceCode.Code = `const token = "eyJfakeheader0.fakepayload0.fakesignature0"`
	if err := command.Parse([]string{"--account", "123", "--repo", "repo1", "--json", writeRepoScanRequest(t, request), "--confirm"}); err != nil {
		t.Fatal(err)
	}
	err := command.Run(ctx)
	if err == nil || !strings.Contains(err.Error(), "credential-shaped") {
		t.Fatalf("error = %v", err)
	}
	if authCalls != 0 {
		t.Fatalf("authentication calls = %d", authCalls)
	}
}

func TestRepoScanGenerateDryRunPrintsRedactedManifestWithoutAuthentication(t *testing.T) {
	authCalls := 0
	ctx := checksclient.ContextWithServiceFactory(context.Background(), func(context.Context) (*checksclient.Service, error) {
		authCalls++
		return nil, nil
	})
	ctx = shared.ContextWithDryRun(ctx, true)
	var stdout bytes.Buffer
	ctx = shared.ContextWithIO(ctx, &stdout, &bytes.Buffer{})
	request := validRepoScanRequest()
	command := RepoScanGenerateCommand()
	if err := command.Parse([]string{"--account", "123", "--repo", "repo1", "--json", writeRepoScanRequest(t, request), "--confirm"}); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(ctx); err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if authCalls != 0 {
		t.Fatalf("authentication calls = %d", authCalls)
	}
	output := stdout.String()
	if !strings.Contains(output, `"sourceCodeBytes"`) || !strings.Contains(output, `"manifestHash"`) || strings.Contains(output, "safe source snippet") {
		t.Fatalf("manifest is missing or disclosed source: %s", output)
	}
}

func TestRepoScanGenerateRequiresExactManifestHashBeforeAuthentication(t *testing.T) {
	request := validRepoScanRequest()
	command := RepoScanGenerateCommand()
	if err := command.Parse([]string{
		"--account", "123", "--repo", "repo1", "--json", writeRepoScanRequest(t, request),
		"--confirm", "--confirm-manifest", strings.Repeat("0", 64),
	}); err != nil {
		t.Fatal(err)
	}
	err := command.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "must exactly match") {
		t.Fatalf("error = %v", err)
	}
}

func TestRepoScanGenerateSendsValidatedBodyAndReturnsManifest(t *testing.T) {
	request := validRepoScanRequest()
	manifest, err := buildRepoUploadManifest("123", "repo1", request)
	if err != nil {
		t.Fatal(err)
	}
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"accounts/123/repos/repo1/operations/op1"}`))
	}))
	t.Cleanup(server.Close)
	ctx := checksclient.ContextWithServiceFactory(context.Background(), func(ctx context.Context) (*checksclient.Service, error) {
		return checksclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
	})
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	command := RepoScanGenerateCommand()
	if err := command.Parse([]string{
		"--account", "123", "--repo", "repo1", "--json", writeRepoScanRequest(t, request), "--confirm",
		"--confirm-manifest", manifest.ManifestHash, "--manifest-file", manifestPath,
	}); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if err := command.Run(shared.ContextWithIO(ctx, &stdout, &bytes.Buffer{})); err != nil {
		t.Fatalf("generate: %v", err)
	}
	if requests != 1 || !strings.Contains(stdout.String(), manifest.ManifestHash) || strings.Contains(stdout.String(), "safe source snippet") {
		t.Fatalf("requests=%d output=%s", requests, stdout.String())
	}
	if info, err := os.Stat(manifestPath); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("manifest mode/error = %v, %v", info, err)
	}
}

func TestEveryRepoScanEndpoint_SuccessAndAPIError(t *testing.T) {
	request := validRepoScanRequest()
	manifest, err := buildRepoUploadManifest("123", "repo1", request)
	if err != nil {
		t.Fatal(err)
	}
	requestArg := writeRepoScanRequest(t, request)
	tests := []struct {
		name       string
		command    func() *ffcli.Command
		args       []string
		method     string
		path       string
		errorLabel string
	}{
		{"generate", RepoScanGenerateCommand, []string{"--account", "123", "--repo", "repo1", "--json", requestArg, "--confirm", "--confirm-manifest", manifest.ManifestHash}, http.MethodPost, "/v1alpha/accounts/123/repos/repo1/scans:generate", "generate Checks repository scan"},
		{"get", RepoScanGetCommand, []string{"--account", "123", "--repo", "repo1", "--scan", "scan1"}, http.MethodGet, "/v1alpha/accounts/123/repos/repo1/scans/scan1", "get Checks repository scan"},
		{"list", RepoScanListCommand, []string{"--account", "123", "--repo", "repo1", "--paginate"}, http.MethodGet, "/v1alpha/accounts/123/repos/repo1/scans", "list Checks repository scans"},
		{"operation", RepoOperationGetCommand, []string{"--account", "123", "--repo", "repo1", "--operation", "op1"}, http.MethodGet, "/v1alpha/accounts/123/repos/repo1/operations/op1", "get Checks repository operation"},
	}

	for _, tc := range tests {
		for _, status := range []int{http.StatusOK, http.StatusForbidden} {
			caseName := "success"
			if status != http.StatusOK {
				caseName = "API error"
			}
			t.Run(tc.name+" "+caseName, func(t *testing.T) {
				var gotMethod, gotPath string
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					gotMethod, gotPath = r.Method, r.URL.Path
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(status)
					if status == http.StatusOK {
						_, _ = w.Write([]byte(`{}`))
						return
					}
					_, _ = w.Write([]byte(`{"error":{"message":"denied"}}`))
				}))
				t.Cleanup(server.Close)
				ctx := checksclient.ContextWithServiceFactory(context.Background(), func(ctx context.Context) (*checksclient.Service, error) {
					return checksclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
				})
				ctx = shared.ContextWithIO(ctx, &bytes.Buffer{}, &bytes.Buffer{})
				cmd := tc.command()
				if err := cmd.Parse(tc.args); err != nil {
					t.Fatal(err)
				}
				err := cmd.Run(ctx)
				if status != http.StatusOK {
					if err == nil || !strings.Contains(err.Error(), tc.errorLabel) {
						t.Fatalf("expected contextual API error containing %q, got %v", tc.errorLabel, err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if gotMethod != tc.method || gotPath != tc.path {
					t.Fatalf("request = %s %s, want %s %s", gotMethod, gotPath, tc.method, tc.path)
				}
			})
		}
	}
}

func validRepoScanRequest() *checksapi.GoogleChecksRepoScanV1alphaGenerateScanRequest {
	return &checksapi.GoogleChecksRepoScanV1alphaGenerateScanRequest{
		CliVersion: "1.0.0", LocalScanPath: ".",
		CliAnalysis: &checksapi.GoogleChecksRepoScanV1alphaCliAnalysis{
			CodeScans: []*checksapi.GoogleChecksRepoScanV1alphaCodeScan{{
				SourceCode: &checksapi.GoogleChecksRepoScanV1alphaSourceCode{Path: "src/main.kt", StartLine: 1, EndLine: 1, Code: "safe source snippet"},
			}},
		},
		ScmMetadata: &checksapi.GoogleChecksRepoScanV1alphaScmMetadata{Branch: "main", RemoteUri: "https://github.com/acme/app", RevisionId: "abc123"},
	}
}

func writeRepoScanRequest(t *testing.T, request *checksapi.GoogleChecksRepoScanV1alphaGenerateScanRequest) string {
	t.Helper()
	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "repo-scan.json")
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	return "@" + path
}
