package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cliruntime "github.com/tamtom/play-console-cli/internal/cli/runtime"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

func TestGlobalDryRunReachesCommandsWithLocalDryRun(t *testing.T) {
	t.Run("workflow run", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		marker := filepath.Join(dir, "marker")
		def := `{"workflows":{"ship":{"steps":[{"name":"touch","command":"touch ` + marker + `"}]}}}`
		if err := os.WriteFile("wf.json", []byte(def), 0o600); err != nil {
			t.Fatal(err)
		}
		code, _, stderr := runReleaseCommand(t, []string{"--dry-run", "workflow", "run", "wf.json"}, nil)
		if code != ExitSuccess {
			t.Fatalf("exit %d, stderr %q", code, stderr)
		}
		if _, err := os.Stat(marker); !os.IsNotExist(err) {
			t.Fatalf("global --dry-run executed a workflow step (marker stat err=%v)", err)
		}
		if _, err := os.Stat(filepath.Join(dir, ".gplay")); !os.IsNotExist(err) {
			t.Fatalf("global --dry-run wrote workflow state (stat err=%v)", err)
		}
	})

	t.Run("migrate fastlane", func(t *testing.T) {
		dir := t.TempDir()
		locale := filepath.Join(dir, "src", "en-US")
		if err := os.MkdirAll(locale, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(locale, "title.txt"), []byte("My App"), 0o600); err != nil {
			t.Fatal(err)
		}
		out := filepath.Join(dir, "out")
		code, stdout, stderr := runReleaseCommand(t, []string{"--dry-run", "migrate", "fastlane", "--source", filepath.Join(dir, "src"), "--output-dir", out}, nil)
		if code != ExitSuccess {
			t.Fatalf("exit %d, stderr %q", code, stderr)
		}
		if _, err := os.Stat(out); !os.IsNotExist(err) {
			t.Fatalf("global --dry-run wrote migrated files (stat err=%v)", err)
		}
		if !strings.Contains(stdout, `"dryRun":true`) {
			t.Fatalf("summary does not report dry run: %s", stdout)
		}
	})
}

func TestOutputValidationIgnoresDirectoryOutputFlags(t *testing.T) {
	for _, args := range [][]string{
		{"generated-apks", "download", "--package", "com.example.test", "--version-code", "1", "--download-id", "abc"},
		{"system-apks", "download", "--package", "com.example.test", "--version-code", "1", "--variant-id", "1"},
	} {
		t.Run(args[0], func(t *testing.T) {
			_, _, stderr := runReleaseCommand(t, append(args, "--output", t.TempDir()), nil)
			if strings.Contains(stderr, "output format") {
				t.Fatalf("directory --output was validated as a format: %q", stderr)
			}
		})
	}
}

func TestDefaultOutputEnvironmentDoesNotBreakCommands(t *testing.T) {
	t.Setenv("GPLAY_DEFAULT_OUTPUT", "table")
	for _, args := range [][]string{
		{"rtdn", "decode", "--data", `{"version":"1.0","packageName":"com.example.test","eventTimeMillis":"1"}`},
		{"auth", "doctor"},
	} {
		t.Run(args[0], func(t *testing.T) {
			_, _, stderr := runReleaseCommand(t, args, nil)
			for _, bad := range []string{"unsupported", "--pretty is only valid"} {
				if strings.Contains(stderr, bad) {
					t.Fatalf("GPLAY_DEFAULT_OUTPUT broke the command: %q", stderr)
				}
			}
		})
	}
}

func TestFlagParseErrorsExitWithUsageCodeAndPrintOnce(t *testing.T) {
	for _, tc := range []struct {
		args []string
		msg  string
	}{
		{[]string{"tracks", "list", "--bogus"}, "flag provided but not defined: -bogus"},
		{[]string{"--bogus"}, "flag provided but not defined: -bogus"},
		{[]string{"tracks", "list", "--package"}, "flag needs an argument: -package"},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			code, _, stderr := runReleaseCommand(t, tc.args, nil)
			if code != ExitUsage {
				t.Fatalf("exit %d, want %d (usage); stderr %q", code, ExitUsage, stderr)
			}
			if n := strings.Count(stderr, tc.msg); n != 1 {
				t.Fatalf("stderr has the parse error %d times, want 1: %q", n, stderr)
			}
		})
	}
}

func TestErrorOutputRedactsSecretsInURLs(t *testing.T) {
	runErr := errors.New(`Post "https://example.test/v1/apps/com.example/purchases/subscriptions/monthly/tokens/SECRET_MARKER:acknowledge?key=SECRET_MARKER&alt=json": dial tcp: connection refused`)

	fs := shared.FilesystemFrom(context.Background())
	report := filepath.Join(t.TempDir(), "junit.xml")
	if err := writeJUnitReport(fs, report, "gplay purchases", runErr, 0); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "SECRET_MARKER") || !strings.Contains(string(data), "alt=json") {
		t.Fatalf("JUnit report does not redact the URL: %s", data)
	}

	code, _, stderr := runReleaseCommand(t, []string{"purchases", "products", "get", "--package", "com.example.test", "--product-id", "p", "--token", "SECRET_MARKER"}, func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("response writer cannot hijack")
		}
		conn, _, _ := hj.Hijack()
		_ = conn.Close()
	})
	if code == ExitSuccess {
		t.Fatalf("expected a transport error; stderr %q", stderr)
	}
	if strings.Contains(stderr, "SECRET_MARKER") {
		t.Fatalf("stderr shows the token: %q", stderr)
	}
}

func TestIgnoredGlobalYAMLConfigShowsWarning(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("GPLAY_CONFIG_PATH", "")
	t.Setenv("GPLAY_AUDIT", "0")
	t.Setenv("GPLAY_NO_UPDATE", "1")
	if err := os.MkdirAll(filepath.Join(home, ".gplay"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".gplay", "config.yaml"), []byte("default_package: com.example.legacy\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())

	var stdout, stderr bytes.Buffer
	code := RunWithRuntime([]string{"rtdn", "decode", "--data", `{"version":"1.0","packageName":"com.example.test","eventTimeMillis":"1"}`}, "1.0.0", func(rt *cliruntime.Runtime) {
		rt.WithIO(&stdout, &stderr).WithAuditSink(nil)
	})
	if code != ExitSuccess {
		t.Fatalf("exit %d, stderr %q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "config.yaml is ignored") {
		t.Fatalf("no warning about the ignored config.yaml: %q", stderr.String())
	}
}

// rolloutServer returns a fake Play API that serves one release on GET and
// records the PUT payload and the commit.
func rolloutServer(t *testing.T, release string) (http.HandlerFunc, *[]map[string]any, *int) {
	t.Helper()
	var releases []map[string]any
	writes := 0
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			fmt.Fprintf(w, `{"track":"production","releases":[%s]}`, release)
		case http.MethodPut:
			writes++
			var payload struct {
				Releases []map[string]any `json:"releases"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			releases = payload.Releases
			fmt.Fprint(w, `{}`)
		default:
			if strings.HasSuffix(r.URL.Path, ":commit") {
				writes++
			}
			fmt.Fprint(w, `{"id":"edit"}`)
		}
	}, &releases, &writes
}

func TestRolloutRejectsAmbiguousChanges(t *testing.T) {
	for _, tc := range []struct {
		name    string
		release string
		args    []string
		want    string
	}{
		{"resume halted release without fraction", `{"status":"halted","versionCodes":["100"]}`, []string{"resume"}, "rollout complete"},
		{"resume active release", `{"status":"inProgress","versionCodes":["100"],"userFraction":0.1}`, []string{"resume"}, "rollout update"},
		{"update to the same fraction", `{"status":"inProgress","versionCodes":["100"],"userFraction":0.1}`, []string{"update", "--rollout", "0.1"}, "greater than the current fraction"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler, _, writes := rolloutServer(t, tc.release)
			args := append([]string{"rollout"}, tc.args...)
			args = append(args, "--package", "com.example.test")
			code, _, stderr := runReleaseCommand(t, args, handler)
			if code == ExitSuccess || *writes != 0 || !strings.Contains(stderr, tc.want) {
				t.Fatalf("code=%d writes=%d stderr=%q, want an error that contains %q", code, *writes, stderr, tc.want)
			}
		})
	}
}

func TestRolloutCountryTargetingIsKeptUntilCompletion(t *testing.T) {
	const targeted = `{"status":"inProgress","versionCodes":["100"],"userFraction":0.1,"countryTargeting":{"countries":["DE"]}}`
	for _, tc := range []struct {
		action string
		keep   bool
	}{
		{"halt", true},
		{"complete", false},
	} {
		t.Run(tc.action, func(t *testing.T) {
			handler, releases, _ := rolloutServer(t, targeted)
			code, _, stderr := runReleaseCommand(t, []string{"rollout", tc.action, "--package", "com.example.test"}, handler)
			if code != ExitSuccess || len(*releases) != 1 {
				t.Fatalf("code=%d releases=%v stderr=%q", code, *releases, stderr)
			}
			if _, ok := (*releases)[0]["countryTargeting"]; ok != tc.keep {
				t.Fatalf("countryTargeting present = %v, want %v: %v", ok, tc.keep, (*releases)[0])
			}
		})
	}
}

func TestReleaseCommandsRejectNaNRollout(t *testing.T) {
	aab := filepath.Join(t.TempDir(), "app.aab")
	if err := os.WriteFile(aab, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"release", "--package", "com.example.test", "--track", "beta", "--bundle", aab, "--rollout", "NaN"},
		{"promote", "--package", "com.example.test", "--from", "beta", "--to", "production", "--rollout", "NaN"},
		{"publish", "track", "--package", "com.example.test", "--track", "beta", "--bundle", aab, "--rollout", "NaN"},
	} {
		t.Run(args[0], func(t *testing.T) {
			requests := 0
			code, _, stderr := runReleaseCommand(t, args, func(w http.ResponseWriter, r *http.Request) { requests++; fmt.Fprint(w, `{}`) })
			if code == ExitSuccess || requests != 0 || !strings.Contains(stderr, "--rollout") {
				t.Fatalf("code=%d requests=%d stderr=%q", code, requests, stderr)
			}
		})
	}
}

func TestAuditKeepsArgumentAfterBooleanSkipFlag(t *testing.T) {
	got := strings.Join(scrubArgs([]string{"preflight", "--skip-secrets", "app.aab"}), " ")
	if got != "preflight --skip-secrets app.aab" {
		t.Fatalf("scrubArgs = %q", got)
	}
}
