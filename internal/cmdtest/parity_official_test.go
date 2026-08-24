package cmdtest_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/checksclient"
	cliruntime "github.com/tamtom/play-console-cli/internal/cli/runtime"
	"github.com/tamtom/play-console-cli/internal/developeridclient"
	"github.com/tamtom/play-console-cli/internal/gamesclient"
	"github.com/tamtom/play-console-cli/internal/integrityclient"
	"github.com/tamtom/play-console-cli/internal/playclient"
	"github.com/tamtom/play-console-cli/internal/reportingclient"
)

func TestAppSigningRootFlow_PlansOfflineAndDryRunsWithoutReceipt(t *testing.T) {
	tempDir := t.TempDir()
	planPath := filepath.Join(tempDir, "plan.json")
	receiptPath := filepath.Join(tempDir, "receipt.json")
	request := `{"enrollExistingApp":{"cloudKmsKey":{"cryptoKeyVersionResource":"projects/p/locations/l/keyRings/r/cryptoKeys/k/cryptoKeyVersions/1"}}}`
	result := runCommandWithRuntime(t, nil,
		"app-signing", "plan-enroll", "--package", "app.example", "--json", request,
		"--plan-file", planPath, "--enterprise-self-hosted-kms")
	if result.exitCode != 0 {
		t.Fatalf("plan exit=%d stderr=%s", result.exitCode, result.stderr)
	}
	var plan struct {
		PlanHash string `json:"planHash"`
	}
	if err := json.Unmarshal([]byte(result.stdout), &plan); err != nil || plan.PlanHash == "" {
		t.Fatalf("plan output=%s error=%v", result.stdout, err)
	}
	result = runCommandWithRuntime(t, nil,
		"--dry-run", "app-signing", "apply", "--plan-file", planPath,
		"--receipt-file", receiptPath, "--confirm-plan", plan.PlanHash, "--enterprise-self-hosted-kms")
	if result.exitCode != 0 || !strings.Contains(result.stdout, `"status":"planned"`) {
		t.Fatalf("apply exit=%d stdout=%s stderr=%s", result.exitCode, result.stdout, result.stderr)
	}
	if _, err := os.Stat(receiptPath); !os.IsNotExist(err) {
		t.Fatalf("dry run wrote receipt: %v", err)
	}
	result = runCommandWithRuntime(t, nil,
		"--dry-run", "app-signing", "apply", "--plan-file", planPath,
		"--receipt-file", receiptPath, "--confirm-plan", strings.Repeat("0", 64), "--enterprise-self-hosted-kms")
	if result.exitCode == 0 || !strings.Contains(result.stderr, "must exactly match") {
		t.Fatalf("wrong hash exit=%d stderr=%s", result.exitCode, result.stderr)
	}
}

func TestOfficialParityFamilies_RunThroughProductionRoot(t *testing.T) {
	t.Run("app stores", func(t *testing.T) {
		server := officialTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/androidpublisher/v3/appstorecatalog/store.example/recentAppViews/app.example" {
				t.Errorf("path=%s", r.URL.Path)
			}
			jsonResponse(w, `{}`)
		})
		result := runCommandWithRuntime(t, func(rt *cliruntime.Runtime) {
			rt.WithPlayServiceFactory(func(ctx context.Context) (*playclient.Service, error) {
				return playclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
			})
		}, "app-stores", "recent-app-view", "--app-store-package", "store.example", "--package", "app.example", "--registered-third-party-store")
		assertRuntimeSuccess(t, result)
	})

	t.Run("checks manifest dry run", func(t *testing.T) {
		request := `{"cliVersion":"1.0.0","localScanPath":".","cliAnalysis":{"codeScans":[{"sourceCode":{"path":"src/main.kt","startLine":1,"endLine":1,"code":"safe snippet"}}]},"scmMetadata":{"branch":"main","remoteUri":"https://github.com/acme/app","revisionId":"abc"}}`
		requestPath := filepath.Join(t.TempDir(), "repo-scan.json")
		if err := os.WriteFile(requestPath, []byte(request), 0o600); err != nil {
			t.Fatal(err)
		}
		result := runCommandWithRuntime(t, func(rt *cliruntime.Runtime) {
			rt.WithChecksServiceFactory(func(context.Context) (*checksclient.Service, error) {
				t.Fatal("dry run authenticated")
				return nil, nil
			})
		}, "--dry-run", "checks", "repo-scans", "generate", "--account", "123", "--repo", "repo1", "--json", "@"+requestPath, "--confirm")
		assertRuntimeSuccess(t, result)
		if !strings.Contains(result.stdout, `"manifestHash"`) || strings.Contains(result.stdout, "safe snippet") {
			t.Fatalf("unsafe manifest output=%s", result.stdout)
		}
	})

	t.Run("integrity", func(t *testing.T) {
		tokenPath := filepath.Join(t.TempDir(), "token.txt")
		if err := os.WriteFile(tokenPath, []byte("token"), 0o600); err != nil {
			t.Fatal(err)
		}
		server := officialTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1/app.example:decodeIntegrityToken" {
				t.Errorf("path=%s", r.URL.Path)
			}
			jsonResponse(w, `{}`)
		})
		result := runCommandWithRuntime(t, func(rt *cliruntime.Runtime) {
			rt.WithIntegrityServiceFactory(func(ctx context.Context) (*integrityclient.Service, error) {
				return integrityclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
			})
		}, "integrity", "decode", "--package", "app.example", "--token-file", tokenPath)
		assertRuntimeSuccess(t, result)
	})

	t.Run("developer verification", func(t *testing.T) {
		server := officialTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1/packages/app-example/packageRegistrationStatus:check" {
				t.Errorf("path=%s", r.URL.Path)
			}
			jsonResponse(w, `{}`)
		})
		result := runCommandWithRuntime(t, func(rt *cliruntime.Runtime) {
			rt.WithDeveloperIDServiceFactory(func(ctx context.Context, key string) (*developeridclient.Service, error) {
				if key != "test-key" {
					t.Fatalf("key=%q", key)
				}
				return developeridclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
			})
		}, "verification", "status", "--package", "app.example", "--api-key", "test-key")
		assertRuntimeSuccess(t, result)
	})

	t.Run("games global reset", func(t *testing.T) {
		server := officialTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				jsonResponse(w, `{"items":[]}`)
				return
			}
			if r.URL.Path != "/games/v1management/achievements/resetAllForAllPlayers" {
				t.Errorf("path=%s", r.URL.Path)
			}
			w.WriteHeader(http.StatusNoContent)
		})
		result := runCommandWithRuntime(t, func(rt *cliruntime.Runtime) {
			rt.WithGamesServiceFactory(func(ctx context.Context) (*gamesclient.Service, error) {
				return gamesclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
			})
		}, "games", "achievements", "reset-all-for-all-players", "--application-id", "123", "--confirm-application-id", "123", "--confirm")
		assertRuntimeSuccess(t, result)
	})

	t.Run("vitals metric set", func(t *testing.T) {
		server := officialTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1beta1/apps/app.example/bitmapMemoryUsageMetricSet" {
				t.Errorf("path=%s", r.URL.Path)
			}
			jsonResponse(w, `{}`)
		})
		result := runCommandWithRuntime(t, func(rt *cliruntime.Runtime) {
			rt.WithReportingServiceFactory(func(ctx context.Context) (*reportingclient.Service, error) {
				return reportingclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
			})
		}, "vitals", "metric-sets", "get", "--package", "app.example", "--metric-set", "bitmap-memory")
		assertRuntimeSuccess(t, result)
	})
}

func TestLocalAndOneShotParityCommands_RunThroughProductionRoot(t *testing.T) {
	t.Run("android build dry run", func(t *testing.T) {
		project := t.TempDir()
		wrapper := filepath.Join(project, "gradlew")
		if err := os.WriteFile(wrapper, []byte("#!/bin/sh\nexit 99\n"), 0o700); err != nil {
			t.Fatal(err)
		}
		result := runCommandWithRuntime(t, nil, "--dry-run", "android", "build", "--project", project)
		assertRuntimeSuccess(t, result)
		if !strings.Contains(result.stdout, `"mode":"planned"`) || !strings.Contains(result.stdout, `:app:testReleaseUnitTest`) {
			t.Fatalf("build plan=%s", result.stdout)
		}
	})

	t.Run("android signing dry run", func(t *testing.T) {
		toolDir := t.TempDir()
		writeExecutable(t, filepath.Join(toolDir, "jarsigner"))
		t.Setenv("PATH", toolDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		artifact := filepath.Join(t.TempDir(), "app.aab")
		if err := os.WriteFile(artifact, []byte("bundle"), 0o600); err != nil {
			t.Fatal(err)
		}
		result := runCommandWithRuntime(t, nil, "--dry-run", "android", "signing", "verify", "--file", artifact)
		assertRuntimeSuccess(t, result)
		if !strings.Contains(result.stdout, `"mode":"planned"`) || !strings.Contains(result.stdout, "jarsigner") {
			t.Fatalf("signing plan=%s", result.stdout)
		}
	})

	t.Run("android screenshot dry run", func(t *testing.T) {
		toolDir := t.TempDir()
		writeExecutable(t, filepath.Join(toolDir, "adb"))
		t.Setenv("PATH", toolDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		outputDir := filepath.Join(t.TempDir(), "metadata")
		result := runCommandWithRuntime(t, nil, "--dry-run", "android", "screenshots", "capture", "--serial", "emulator-5554", "--output-dir", outputDir)
		assertRuntimeSuccess(t, result)
		if !strings.Contains(result.stdout, `"mode":"planned"`) || !strings.Contains(result.stdout, "phoneScreenshots") {
			t.Fatalf("screenshot plan=%s", result.stdout)
		}
		if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
			t.Fatalf("screenshot dry run wrote output: %v", err)
		}
	})

	t.Run("verified skills preview", func(t *testing.T) {
		destination := filepath.Join(t.TempDir(), "skills")
		result := runCommandWithRuntime(t, nil, "install-skills", "--preview", "--dest", destination)
		assertRuntimeSuccess(t, result)
		if !strings.Contains(result.stdout, `"mode":"preview"`) || !strings.Contains(result.stdout, `"sourceCommit"`) {
			t.Fatalf("preview=%s", result.stdout)
		}
		if _, err := os.Stat(destination); !os.IsNotExist(err) {
			t.Fatalf("preview wrote destination: %v", err)
		}
	})

	t.Run("experiment support and exact winner", func(t *testing.T) {
		result := runCommandWithRuntime(t, nil, "experiments", "support")
		assertRuntimeSuccess(t, result)
		if !strings.Contains(result.stdout, `"privateInterfacesUsed":false`) || !strings.Contains(result.stdout, `"manualConsoleRequired":true`) {
			t.Fatalf("support=%s", result.stdout)
		}
		result = runCommandWithRuntime(t, nil, "experiments", "apply-winner", "--package", "app.example", "--edit", "edit-1", "--winner", "A", "--confirm-winner", "B", "--dir", t.TempDir())
		if result.exitCode == 0 || !strings.Contains(result.stderr, "must exactly match") {
			t.Fatalf("winner mismatch exit=%d stderr=%s", result.exitCode, result.stderr)
		}
	})

	t.Run("one-shot listing sync", func(t *testing.T) {
		metadataDir := t.TempDir()
		localeDir := filepath.Join(metadataDir, "en-US")
		if err := os.MkdirAll(localeDir, 0o755); err != nil {
			t.Fatal(err)
		}
		for name, contents := range map[string]string{
			"title.txt": "New title\n", "short_description.txt": "Short\n", "full_description.txt": "Full\n",
		} {
			if err := os.WriteFile(filepath.Join(localeDir, name), []byte(contents), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		currentTitle := "Old title"
		mutations := 0
		server := officialTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/edits/edit-1") {
				_, _ = io.WriteString(w, `{"id":"edit-1","expiryTimeSeconds":"4102444800"}`)
				return
			}
			if r.Method == http.MethodPut {
				mutations++
				currentTitle = "New title"
				_, _ = io.WriteString(w, `{"language":"en-US","title":"New title","shortDescription":"Short","fullDescription":"Full"}`)
				return
			}
			if strings.HasSuffix(r.URL.Path, "/listings/en-US") {
				_, _ = io.WriteString(w, `{"language":"en-US","title":"`+currentTitle+`","shortDescription":"Short","fullDescription":"Full"}`)
				return
			}
			_, _ = io.WriteString(w, `{"listings":[{"language":"en-US","title":"`+currentTitle+`","shortDescription":"Short","fullDescription":"Full"}]}`)
		})
		stateDir := t.TempDir()
		result := runCommandWithRuntime(t, func(rt *cliruntime.Runtime) {
			rt.WithPlayServiceFactory(func(ctx context.Context) (*playclient.Service, error) {
				return playclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
			})
		}, "sync", "run", "--package", "app.example", "--edit", "edit-1", "--dir", metadataDir, "--state-dir", stateDir)
		assertRuntimeSuccess(t, result)
		if mutations != 1 || !strings.Contains(result.stdout, `"status":"complete"`) {
			t.Fatalf("mutations=%d stdout=%s", mutations, result.stdout)
		}
		for _, directory := range []string{"plans", "receipts"} {
			entries, err := os.ReadDir(filepath.Join(stateDir, directory))
			if err != nil || len(entries) != 1 {
				t.Fatalf("%s artifacts=%v error=%v", directory, entries, err)
			}
		}
	})

	t.Run("experiment winner apply", func(t *testing.T) {
		metadataDir := writeRootListing(t, "Winner title")
		server := listingSyncServer(t, "Old title")
		result := runCommandWithRuntime(t, func(rt *cliruntime.Runtime) {
			rt.WithPlayServiceFactory(func(ctx context.Context) (*playclient.Service, error) {
				return playclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
			})
		}, "experiments", "apply-winner", "--package", "app.example", "--edit", "edit-1", "--winner", "variant-b", "--confirm-winner", "variant-b", "--dir", metadataDir, "--state-dir", t.TempDir())
		assertRuntimeSuccess(t, result)
		if !strings.Contains(result.stdout, `"selectionSource":"manual"`) || !strings.Contains(result.stdout, `"provider":"official-api"`) {
			t.Fatalf("winner output=%s", result.stdout)
		}
	})
}

func TestParityCommands_ProductionRootFailurePaths(t *testing.T) {
	t.Setenv("GPLAY_ANDROID_DEVELOPER_ID_API_KEY", "")
	project := t.TempDir()
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"app stores", []string{"app-stores", "recent-app-view", "--app-store-package", "store.example", "--package", "app.example"}, "--registered-third-party-store"},
		{"checks source argv", []string{"checks", "repo-scans", "generate", "--account", "123", "--repo", "repo1", "--json", `{}`, "--confirm"}, "must use @file"},
		{"integrity", []string{"integrity", "decode", "--package", "app.example"}, "--token-file"},
		{"verification", []string{"verification", "status", "--package", "app.example"}, "--api-key"},
		{"games reset", []string{"games", "achievements", "reset-all-for-all-players", "--application-id", "123", "--confirm-application-id", "456", "--confirm"}, "must exactly match"},
		{"vitals metric", []string{"vitals", "metric-sets", "get", "--package", "app.example", "--metric-set", "invented"}, "documented values"},
		{"android", []string{"android", "build", "--project", project, "--artifact-type", "zip"}, "aab or apk"},
		{"android signing", []string{"android", "signing", "verify"}, "--file is required"},
		{"android screenshots", []string{"android", "screenshots", "capture"}, "--serial is required"},
		{"skills", []string{"install-skills", "--preview", "--force"}, "cannot be used together"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := runCommandWithRuntime(t, nil, tc.args...)
			if result.exitCode == 0 || !strings.Contains(result.stderr, tc.want) {
				t.Fatalf("exit=%d stderr=%s, want %q", result.exitCode, result.stderr, tc.want)
			}
		})
	}
}

func writeExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
}

func writeRootListing(t *testing.T, title string) string {
	t.Helper()
	metadataDir := t.TempDir()
	localeDir := filepath.Join(metadataDir, "en-US")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, contents := range map[string]string{
		"title.txt": title + "\n", "short_description.txt": "Short\n", "full_description.txt": "Full\n",
	} {
		if err := os.WriteFile(filepath.Join(localeDir, name), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return metadataDir
}

func listingSyncServer(t *testing.T, initialTitle string) *httptest.Server {
	t.Helper()
	currentTitle := initialTitle
	return officialTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/edits/edit-1") {
			_, _ = io.WriteString(w, `{"id":"edit-1","expiryTimeSeconds":"4102444800"}`)
			return
		}
		if r.Method == http.MethodPut {
			currentTitle = "Winner title"
			_, _ = io.WriteString(w, `{"language":"en-US","title":"Winner title","shortDescription":"Short","fullDescription":"Full"}`)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/listings/en-US") {
			_, _ = io.WriteString(w, `{"language":"en-US","title":"`+currentTitle+`","shortDescription":"Short","fullDescription":"Full"}`)
			return
		}
		_, _ = io.WriteString(w, `{"listings":[{"language":"en-US","title":"`+currentTitle+`","shortDescription":"Short","fullDescription":"Full"}]}`)
	})
}

func officialTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func jsonResponse(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, body)
}

func assertRuntimeSuccess(t *testing.T, result runtimeResult) {
	t.Helper()
	if result.exitCode != 0 {
		t.Fatalf("exit=%d stdout=%s stderr=%s", result.exitCode, result.stdout, result.stderr)
	}
}
