package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
	cliruntime "github.com/tamtom/play-console-cli/internal/cli/runtime"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/config"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

func runReleaseCommand(t *testing.T, args []string, handler http.HandlerFunc) (int, string, string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("GPLAY_AUDIT", "0")
	t.Setenv("GPLAY_NO_UPDATE", "1")
	var stdout, stderr bytes.Buffer
	var server *httptest.Server
	if handler != nil {
		server = httptest.NewServer(handler)
		t.Cleanup(server.Close)
	}
	code := RunWithRuntime(args, "1.0.0", func(rt *cliruntime.Runtime) {
		rt.WithIO(&stdout, &stderr).WithAuditSink(nil)
		rt.WithPlayServiceFactory(func(ctx context.Context) (*playclient.Service, error) {
			if server == nil {
				return nil, fmt.Errorf("unexpected API service")
			}
			return playclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
		})
	})
	return code, stdout.String(), stderr.String()
}

func TestAuditRedactsSensitiveArgumentsAndErrors(t *testing.T) {
	for _, args := range [][]string{
		{"--api-key", "SECRET_MARKER"},
		{"--webhook-url=https://example.invalid/SECRET_MARKER"},
		{"-token", "SECRET_MARKER"},
		{"-token=SECRET_MARKER"},
		{"--data", `{"purchaseToken":"SECRET_MARKER"}`},
		{"--json", `{"nested":{"secret":"SECRET_MARKER"}}`},
		{"--param", "API_TOKEN=SECRET_MARKER"},
		{"--param=API_TOKEN=SECRET_MARKER"},
	} {
		if got := strings.Join(scrubArgs(args), " "); strings.Contains(got, "SECRET_MARKER") {
			t.Errorf("secret in args: %s", got)
		}
	}
	sink := &recordingAuditSink{}
	var stdout, stderr bytes.Buffer
	RunWithRuntime([]string{"tracks", "list", "--package", "com.example.test"}, "1.0.0", func(rt *cliruntime.Runtime) {
		rt.WithIO(&stdout, &stderr).WithAuditSink(sink).WithPlayServiceFactory(func(context.Context) (*playclient.Service, error) {
			return nil, fmt.Errorf("API error reflected credential SECRET_MARKER")
		})
	})
	encoded, err := json.Marshal(sink.entries)
	if err != nil {
		t.Fatal(err)
	}
	if len(sink.entries) == 0 || strings.Contains(string(encoded), "SECRET_MARKER") {
		t.Fatalf("unsafe audit entries: %s", encoded)
	}
}

func TestWebhookOutputDoesNotExposeCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{}`) }))
	defer server.Close()
	for _, dryRun := range []bool{false, true} {
		args := []string{"notify", "send", "--webhook-url", server.URL + "/SECRET_MARKER?key=SECRET_MARKER", "--message", "test"}
		if dryRun {
			args = append([]string{"--dry-run"}, args...)
		}
		code, stdout, stderr := runReleaseCommand(t, args, nil)
		if code != 0 || strings.Contains(stdout+stderr, "SECRET_MARKER") {
			t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout, stderr)
		}
	}
}

func TestInvalidOutputPreventsWrites(t *testing.T) {
	bundle := filepath.Join(t.TempDir(), "app.aab")
	if err := os.WriteFile(bundle, []byte("test bundle"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"rollout", "halt", "--package", "com.example.test"},
		{"bundles", "upload", "--package", "com.example.test", "--edit", "edit", "--file", bundle},
	} {
		t.Run(strings.Join(args[:2], "_"), func(t *testing.T) {
			requests := 0
			code, _, stderr := runReleaseCommand(t, append(args, "--output", "xml"), func(w http.ResponseWriter, r *http.Request) {
				requests++
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodGet {
					fmt.Fprint(w, `{"track":"production","releases":[{"status":"inProgress","versionCodes":["100"],"userFraction":0.1}]}`)
				} else {
					fmt.Fprint(w, `{"id":"edit"}`)
				}
			})
			if code == 0 || !strings.Contains(stderr, "unsupported") || requests != 0 {
				t.Fatalf("code=%d requests=%d stderr=%s", code, requests, stderr)
			}
		})
	}
}

func TestDryRunPreventsWebhook(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; fmt.Fprint(w, `{}`) }))
	defer server.Close()
	code, _, stderr := runReleaseCommand(t, []string{"--dry-run", "notify", "send", "--webhook-url", server.URL, "--message", "test"}, nil)
	if code != 0 || calls != 0 || !strings.Contains(stderr, "DRY RUN") {
		t.Fatalf("code=%d calls=%d stderr=%s", code, calls, stderr)
	}
}

func TestDryRunPreventsSetupWrites(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake gcloud shell process is POSIX; HTTP and native updater tests run on Windows")
	}
	for _, command := range [][]string{{"rtdn", "setup"}, {"setup", "--auto"}, {"auth", "setup", "--auto"}} {
		for _, global := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/global=%v", strings.Join(command, "_"), global), func(t *testing.T) {
				dir := t.TempDir()
				t.Chdir(dir)
				log := filepath.Join(dir, "calls")
				script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$GPLAY_TEST_GCLOUD_LOG\"\nprintf '{}\\n'\n"
				if err := os.WriteFile(filepath.Join(dir, "gcloud"), []byte(script), 0o700); err != nil {
					t.Fatal(err)
				}
				t.Setenv("PATH", dir)
				t.Setenv("GPLAY_TEST_GCLOUD_LOG", log)
				args := append(append([]string{}, command...), "--project", "test-project", "--output", "json")
				if command[0] != "rtdn" {
					args = append(args, "--key-out", filepath.Join(dir, "keys", "key.json"))
				}
				if global {
					args = append([]string{"--dry-run"}, args...)
				} else {
					args = append(args, "--dry-run")
				}
				code, _, stderr := runReleaseCommand(t, args, nil)
				calls, _ := os.ReadFile(log)
				_, keyDirErr := os.Stat(filepath.Join(dir, "keys"))
				if code != 0 || len(calls) != 0 || !os.IsNotExist(keyDirErr) {
					t.Fatalf("code=%d calls=%s keyDirErr=%v stderr=%s", code, calls, keyDirErr, stderr)
				}
			})
		}
	}
}

func TestInitCreatesLoadableConfiguration(t *testing.T) {
	t.Chdir(t.TempDir())
	code, _, stderr := runReleaseCommand(t, []string{"init", "--package", "com.example.release", "--service-account", "/test/key.json", "--timeout", "90s"}, nil)
	if code != 0 {
		t.Fatalf("init: %s", stderr)
	}
	path, err := config.LocalPath()
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadAt(path)
	if err != nil {
		t.Fatal(err)
	}
	duration, _ := cfg.Timeout.Value()
	if cfg.PackageName != "com.example.release" || duration != 90*time.Second || len(cfg.Profiles) != 1 || cfg.Profiles[0].KeyPath != "/test/key.json" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestInitDryRunAndInvalidTimeoutDoNotWrite(t *testing.T) {
	for _, args := range [][]string{{"--dry-run", "init"}, {"init", "--timeout", "bad"}} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			t.Chdir(t.TempDir())
			code, _, stderr := runReleaseCommand(t, args, nil)
			if args[0] == "init" && (code == 0 || !strings.Contains(stderr, "timeout")) {
				t.Fatalf("code=%d stderr=%s", code, stderr)
			}
			if _, err := os.Stat(".gplay"); !os.IsNotExist(err) {
				t.Fatalf("configuration directory created: %v", err)
			}
		})
	}
}

func TestRolloutWirePayload(t *testing.T) {
	for _, tc := range []struct{ action, fraction, initial, want string }{
		{"complete", "", "inProgress", "completed"},
		{"update", "1", "inProgress", "completed"},
		{"resume", "1", "halted", "completed"},
		{"halt", "", "completed", "halted"},
	} {
		t.Run(tc.action+tc.fraction, func(t *testing.T) {
			var payload struct {
				Releases []map[string]any `json:"releases"`
			}
			committed := false
			handler := func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.Method {
				case http.MethodGet:
					fraction := `,"userFraction":0.1`
					if tc.initial == "completed" {
						fraction = ""
					}
					fmt.Fprintf(w, `{"track":"production","releases":[{"status":%q,"versionCodes":["100"]%s}]}`, tc.initial, fraction)
				case http.MethodPut:
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Error(err)
					}
					fmt.Fprint(w, `{}`)
				default:
					committed = committed || strings.HasSuffix(r.URL.Path, ":commit")
					fmt.Fprint(w, `{"id":"edit"}`)
				}
			}
			args := []string{"rollout", tc.action, "--package", "com.example.test"}
			if tc.fraction != "" {
				args = append(args, "--rollout", tc.fraction)
			}
			code, _, stderr := runReleaseCommand(t, args, handler)
			if code != 0 || !committed || len(payload.Releases) != 1 {
				t.Fatalf("code=%d committed=%v payload=%v stderr=%s", code, committed, payload, stderr)
			}
			release := payload.Releases[0]
			if release["status"] != tc.want {
				t.Errorf("status = %v", release["status"])
			}
			if _, ok := release["userFraction"]; ok {
				t.Errorf("unexpected userFraction: %v", release)
			}
		})
	}
}

func TestRolloutRejectsInvalidFractionsBeforeRequests(t *testing.T) {
	for _, action := range []string{"update", "resume"} {
		for _, fraction := range []string{"NaN", "+Inf", "-Inf", "-0.1", "1.1"} {
			t.Run(action+fraction, func(t *testing.T) {
				requests := 0
				code, _, stderr := runReleaseCommand(t, []string{"rollout", action, "--package", "com.example.test", "--rollout", fraction}, func(w http.ResponseWriter, r *http.Request) { requests++; fmt.Fprint(w, `{}`) })
				if code == 0 || !strings.Contains(stderr, "--rollout") || requests != 0 {
					t.Fatalf("code=%d requests=%d stderr=%s", code, requests, stderr)
				}
			})
		}
	}
}

func TestRolloutUpdateDoesNotResumeHaltedRelease(t *testing.T) {
	for _, fraction := range []string{"0.5", "1"} {
		t.Run(fraction, func(t *testing.T) {
			trackWrites, commits := 0, 0
			code, _, stderr := runReleaseCommand(t, []string{"rollout", "update", "--package", "com.example.test", "--rollout", fraction}, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodGet {
					fmt.Fprint(w, `{"track":"production","releases":[{"status":"halted","versionCodes":["100"],"userFraction":0.1}]}`)
					return
				}
				if r.Method == http.MethodPut {
					trackWrites++
				}
				if strings.HasSuffix(r.URL.Path, ":commit") {
					commits++
				}
				fmt.Fprint(w, `{"id":"edit"}`)
			})
			if code == 0 || trackWrites != 0 || commits != 0 || !strings.Contains(stderr, "rollout resume") {
				t.Fatalf("code=%d trackWrites=%d commits=%d stderr=%s", code, trackWrites, commits, stderr)
			}
		})
	}
}

func TestRootHelpIncludesGlobalFlags(t *testing.T) {
	root := RootCommand("1.0.0")
	help := root.UsageFunc(root)
	for _, name := range []string{"--dry-run", "--profile", "--debug", "--report", "--report-file", "--version"} {
		if !strings.Contains(help, name) {
			t.Errorf("help omits %s", name)
		}
	}
}

func TestSetupAndReportFailuresExplainTheProblem(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"--report", "junit", "version"}, "--report-file"},
		{[]string{"setup"}, "--auto"},
		{[]string{"auth", "setup"}, "--auto"},
		{[]string{"rtdn", "status", "--project", "test"}, "gcloud"},
	} {
		code, _, stderr := runReleaseCommand(t, tc.args, nil)
		if code == 0 || !strings.Contains(stderr, tc.want) {
			t.Errorf("%v: code=%d stderr=%s", tc.args, code, stderr)
		}
	}
}

func TestDocumentedPackageEnvironmentPrecedence(t *testing.T) {
	t.Setenv("GPLAY_PACKAGE", "com.example.canonical")
	t.Setenv("GPLAY_PACKAGE_NAME", "com.example.legacy")
	cfg := &config.Config{PackageName: "com.example.config"}
	if got := shared.ResolvePackageName("", cfg); got != "com.example.canonical" {
		t.Errorf("package=%s", got)
	}
	if got := shared.ResolvePackageName("com.example.flag", cfg); got != "com.example.flag" {
		t.Errorf("flag package=%s", got)
	}
	t.Setenv("GPLAY_PACKAGE", "")
	if got := shared.ResolvePackageName("", cfg); got != "com.example.legacy" {
		t.Errorf("legacy package=%s", got)
	}
}

func TestRTDNExistingTopicStillGrantsPublisher(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake gcloud shell process is POSIX")
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$GPLAY_TEST_GCLOUD_LOG\"\ncase \"$*\" in *topics\\ create*) printf 'ERROR: already exists\\n' >&2; exit 1;; esac\nprintf '{}\\n'\n"
	if err := os.WriteFile(filepath.Join(dir, "gcloud"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("GPLAY_TEST_GCLOUD_LOG", log)
	code, _, stderr := runReleaseCommand(t, []string{"rtdn", "setup", "--project", "test", "--output", "json"}, nil)
	calls, _ := os.ReadFile(log)
	if code != 0 || !strings.Contains(string(calls), "add-iam-policy-binding") {
		t.Fatalf("code=%d calls=%s stderr=%s", code, calls, stderr)
	}
}

func TestSnitchExplicitHelpSucceeds(t *testing.T) {
	for _, args := range [][]string{{"snitch", "--help"}, {"snitch", "flush", "--help"}} {
		code, _, stderr := runReleaseCommand(t, args, nil)
		if code != 0 || !strings.Contains(stderr, "USAGE") {
			t.Errorf("%v: code=%d stderr=%s", args, code, stderr)
		}
	}
}

func TestSubscriptionArchiveFailsLocally(t *testing.T) {
	calls := 0
	code, _, stderr := runReleaseCommand(t, []string{"subscriptions", "archive", "--package", "com.example.test", "--product-id", "monthly"}, func(w http.ResponseWriter, r *http.Request) { calls++; fmt.Fprint(w, `{}`) })
	if code == 0 || calls != 0 || !strings.Contains(stderr, "not supported") || !strings.Contains(stderr, "gplay baseplans deactivate --help") {
		t.Fatalf("code=%d calls=%d stderr=%s", code, calls, stderr)
	}
	// The suggested command must exist.
	if code, _, stderr := runReleaseCommand(t, []string{"baseplans", "deactivate", "--help"}, nil); code != 0 {
		t.Fatalf("suggested command failed: code=%d stderr=%s", code, stderr)
	}
}

func TestDryRunPreventsHTTPMutation(t *testing.T) {
	requests := 0
	code, _, stderr := runReleaseCommand(t, []string{"--dry-run", "purchases", "subscriptions", "cancel", "--package", "com.example.test", "--subscription-id", "monthly", "--token", "test-token", "--confirm"}, func(w http.ResponseWriter, r *http.Request) { requests++; fmt.Fprint(w, `{}`) })
	if code != 0 || requests != 0 {
		t.Fatalf("code=%d requests=%d stderr=%s", code, requests, stderr)
	}
}

func TestDryRunPreservesLocalState(t *testing.T) {
	for _, args := range [][]string{
		{"auth", "init", "--force", "--local"},
		{"auth", "login", "--service-account", "key.json"},
		{"auth", "switch", "--profile", "other"},
		{"auth", "logout", "--profile", "default", "--confirm"},
		{"auth", "doctor", "--fix", "--confirm", "--output", "json"},
		{"snitch", "--local", "--repro", "example", "--expected", "works", "--actual", "fails"},
	} {
		t.Run(strings.Join(args[:2], "_"), func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			path := filepath.Join(dir, ".gplay", "config.json")
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			original := []byte(`{"default_profile":"default","profiles":[{"name":"default","type":"service_account","key_path":"missing.json"},{"name":"other","type":"service_account","key_path":"missing.json"}]}`)
			if err := os.WriteFile(path, original, 0o600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("GPLAY_CONFIG_PATH", path)
			t.Setenv("GITHUB_TOKEN", "")
			t.Setenv("GH_TOKEN", "")
			code, stdout, stderr := runReleaseCommand(t, append([]string{"--dry-run"}, args...), nil)
			actual, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(original, actual) {
				t.Fatalf("config changed: %s err=%v", actual, err)
			}
			if _, err := os.Stat(filepath.Join(dir, ".gplay", "snitch.log")); !os.IsNotExist(err) {
				t.Fatalf("local bug report was written: %v", err)
			}
			if code != 0 {
				t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout, stderr)
			}
		})
	}
}

func TestDryRunUpdateDoesNotCheckOrInstall(t *testing.T) {
	calls := 0
	original := http.DefaultTransport
	http.DefaultTransport = updateHTTPTransport(func(*http.Request) (*http.Response, error) {
		calls++
		return nil, fmt.Errorf("unexpected update network call")
	})
	defer func() { http.DefaultTransport = original }()
	code, _, stderr := runReleaseCommand(t, []string{"--dry-run", "update", "--force"}, nil)
	if code != 0 || calls != 0 || !strings.Contains(stderr, "DRY RUN") {
		t.Fatalf("code=%d calls=%d stderr=%s", code, calls, stderr)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(homeDir, ".cache")); !os.IsNotExist(err) {
		t.Fatalf("update wrote cache: %v", err)
	}
}

func TestOutputDefaultsAndExplicitPrecedence(t *testing.T) {
	for _, command := range [][]string{{"experiments", "support"}, {"schema", "--list"}} {
		t.Run(command[0], func(t *testing.T) {
			t.Setenv("GPLAY_DEFAULT_OUTPUT", "table")
			code, out, err := runReleaseCommand(t, command, nil)
			if code != 0 || json.Valid([]byte(out)) {
				t.Fatalf("table default not honored: code=%d out=%s stderr=%s", code, out, err)
			}
			code, out, err = runReleaseCommand(t, append(command, "--output", "json"), nil)
			if code != 0 || !json.Valid([]byte(out)) {
				t.Fatalf("explicit JSON did not win: code=%d out=%s stderr=%s", code, out, err)
			}
		})
	}
	t.Setenv("GPLAY_DEFAULT_OUTPUT", "xml")
	calls := 0
	code, _, err := runReleaseCommand(t, []string{"rollout", "halt", "--package", "com.example.app"}, func(w http.ResponseWriter, r *http.Request) { calls++; fmt.Fprint(w, `{}`) })
	if code == 0 || calls != 0 || !strings.Contains(err, "unsupported") {
		t.Fatalf("code=%d requests=%d stderr=%s", code, calls, err)
	}
}

func TestVersionWithGlobalFlags(t *testing.T) {
	code, out, err := runReleaseCommand(t, []string{"--dry-run", "--version"}, nil)
	if code != 0 || strings.TrimSpace(out) != "1.0.0" {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, out, err)
	}
}

func TestEveryCommandHelpExitsSuccessfully(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("GPLAY_NO_UPDATE", "1")
	var walk func(*ffcli.Command, []string)
	walk = func(command *ffcli.Command, path []string) {
		args := append(append([]string{}, path...), "--help")
		var stdout, stderr bytes.Buffer
		code := RunWithRuntime(args, "1.0.0", func(rt *cliruntime.Runtime) { rt.WithIO(&stdout, &stderr).WithAuditSink(nil) })
		if code != 0 || !strings.Contains(stdout.String()+stderr.String(), "USAGE") {
			t.Errorf("%v: code=%d stdout=%s stderr=%s", args, code, stdout.String(), stderr.String())
		}
		for _, child := range command.Subcommands {
			walk(child, append(append([]string{}, path...), child.Name))
		}
	}
	walk(RootCommand("1.0.0"), nil)
}
