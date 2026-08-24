//go:build livesmoke

package livesmoke

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tamtom/play-console-cli/internal/cmdtest"
)

// The live smoke suite executes the compiled gplay binary against the real
// Google Play API. It is gated three times:
//
//  1. Build tag `livesmoke` keeps it out of normal test runs.
//  2. GPLAY_LIVE_SMOKE=1 must be set, with GPLAY_SERVICE_ACCOUNT_JSON
//     pointing at a scoped service-account key.
//  3. Mutations additionally require GPLAY_LIVE_SMOKE_MUTATIONS=1 and only
//     ever touch AllowedMutationPackage (compile-time constant).
//
// Quota budget: the whole suite creates at most ONE edit per run. The edit
// is shared by all track/listing checks and is deleted without commit.

var (
	homeDir    string
	keyPath    string
	runID      string
	ledgerPath string
)

func TestMain(m *testing.M) {
	if v := os.Getenv("GPLAY_LIVE_SMOKE"); v != "1" && v != "true" {
		// Report the skip and pass: normal CI must stay green.
		fmt.Println("livesmoke: skipped; set GPLAY_LIVE_SMOKE=1 to run")
		os.Exit(0)
	}

	keyPath = os.Getenv("GPLAY_SERVICE_ACCOUNT_JSON")
	if keyPath == "" {
		fmt.Fprintln(os.Stderr, "livesmoke: GPLAY_SERVICE_ACCOUNT_JSON is required")
		os.Exit(1)
	}

	runID = os.Getenv("GITHUB_RUN_ID")
	if runID == "" {
		runID = fmt.Sprintf("local%d", time.Now().Unix())
	}

	var err error
	homeDir, err = os.MkdirTemp("", "gplay-livesmoke-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "livesmoke: temp dir:", err)
		os.Exit(1)
	}

	ledgerPath = os.Getenv("GPLAY_LIVE_SMOKE_LEDGER")
	if ledgerPath == "" {
		ledgerPath = filepath.Join(homeDir, "ledger.jsonl")
	}

	code := m.Run()
	cmdtest.Cleanup()
	_ = os.RemoveAll(homeDir)
	os.Exit(code)
}

func mutationsEnabled() bool {
	v := os.Getenv("GPLAY_LIVE_SMOKE_MUTATIONS")
	return v == "1" || v == "true"
}

type result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// runCLI executes the compiled binary with a minimal, explicit environment.
// The child process never inherits the parent environment, so stray
// developer credentials cannot leak into test behavior or output.
func runCLI(t *testing.T, args ...string) result {
	t.Helper()
	return runCLIEnv(t, nil, args...)
}

// runCLIEnv is runCLI with per-test environment overrides. A key in overrides
// replaces the default value; the environment stays minimal and explicit.
func runCLIEnv(t *testing.T, overrides map[string]string, args ...string) result {
	t.Helper()
	cmdtest.Build(t)

	env := map[string]string{
		"PATH":                       os.Getenv("PATH"),
		"HOME":                       homeDir,
		"GPLAY_CONFIG_PATH":          filepath.Join(homeDir, "config.json"),
		"GPLAY_SERVICE_ACCOUNT_JSON": keyPath,
		"GPLAY_SERVICE_ACCOUNT":      keyPath,
		"GPLAY_NO_UPDATE":            "1",
		"GPLAY_TIMEOUT":              "60s",
	}
	for k, v := range overrides {
		env[k] = v
	}

	cmd := exec.Command(cmdtest.BinaryPath, args...)
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("running %v: %v", args, err)
	}
	return result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: exitCode}
}

func mustJSON(t *testing.T, r result, args ...string) map[string]any {
	t.Helper()
	if r.ExitCode != 0 {
		t.Fatalf("%v: exit %d\nstderr: %s", args, r.ExitCode, r.Stderr)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(r.Stdout), &parsed); err != nil {
		t.Fatalf("%v: stdout is not JSON: %v\nstdout: %s", args, err, r.Stdout)
	}
	return parsed
}

// mustJSONArray parses list output. Paginated list commands print a JSON
// array (or null when empty), not an object.
func mustJSONArray(t *testing.T, r result, args ...string) []map[string]any {
	t.Helper()
	if r.ExitCode != 0 {
		t.Fatalf("%v: exit %d\nstderr: %s", args, r.ExitCode, r.Stderr)
	}
	var parsed []map[string]any
	if err := json.Unmarshal([]byte(r.Stdout), &parsed); err != nil {
		t.Fatalf("%v: stdout is not a JSON array: %v\nstdout: %s", args, err, r.Stdout)
	}
	return parsed
}

// --- Read-only smoke: no edit, no quota ---

func TestLive_AppsList_ShowsFixtureApp(t *testing.T) {
	r := runCLI(t, "apps", "list", "--output", "json")
	parsed := mustJSON(t, r, "apps list")

	raw, err := json.Marshal(parsed)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), AllowedMutationPackage) {
		t.Fatalf("apps list must include the fixture app %s", AllowedMutationPackage)
	}
}

func TestLive_ReviewsList_ReturnsJSON(t *testing.T) {
	r := runCLI(t, "reviews", "list", "--package", AllowedMutationPackage, "--output", "json")
	mustJSON(t, r, "reviews list")
}

func TestLive_InvalidPackage_FailsWithError(t *testing.T) {
	r := runCLI(t, "reviews", "list", "--package", "com.invalid.package.gplay.livesmoke", "--output", "json")
	if r.ExitCode == 0 {
		t.Fatalf("invalid package must fail\nstdout: %s", r.Stdout)
	}
	if strings.TrimSpace(r.Stderr) == "" {
		t.Fatal("failure must print an error on stderr")
	}
	if strings.Contains(r.Stderr, "private_key") {
		t.Fatal("stderr must never contain key material")
	}
}

func TestLive_InvalidEditID_FailsWithError(t *testing.T) {
	r := runCLI(t, "tracks", "list", "--package", AllowedMutationPackage, "--edit", "invalid-edit-id-999", "--output", "json")
	if r.ExitCode == 0 {
		t.Fatalf("invalid edit ID must fail\nstdout: %s", r.Stdout)
	}
	if strings.TrimSpace(r.Stderr) == "" {
		t.Fatal("failure must print an error on stderr")
	}
}

func TestLive_TracksReleasesList_Shape(t *testing.T) {
	parsed := mustJSON(t, runCLI(t, "tracks", "releases", "list", "--package", AllowedMutationPackage, "--track", "internal", "--output", "json"), "tracks releases list")

	// The fixture-release script depends on this shape. A drift here broke
	// the first fixture release (versionCodes vs activeArtifacts).
	releases, _ := parsed["releases"].([]any)
	if len(releases) == 0 {
		t.Fatal("internal track must have at least one release")
	}
	release, _ := releases[0].(map[string]any)
	artifacts, _ := release["activeArtifacts"].([]any)
	if len(artifacts) == 0 {
		t.Fatalf("release must carry activeArtifacts: %v", release)
	}
	artifact, _ := artifacts[0].(map[string]any)
	if code, ok := artifact["versionCode"].(float64); !ok || code < 1 {
		t.Fatalf("activeArtifacts[0].versionCode must be a positive number: %v", artifact)
	}
}

func TestLive_OneTimeProductsList_ReturnsJSON(t *testing.T) {
	mustJSON(t, runCLI(t, "onetimeproducts", "list", "--package", AllowedMutationPackage, "--output", "json"), "onetimeproducts list")
}

// --- Behavior smoke: safety promises against the real transport ---

func TestLive_DryRun_InterceptsWrite(t *testing.T) {
	r := runCLI(t, "--dry-run", "edits", "create", "--package", AllowedMutationPackage, "--output", "json")
	if r.ExitCode != 0 {
		t.Fatalf("dry-run must succeed: %s", r.Stderr)
	}
	if !strings.Contains(r.Stderr, "[DRY RUN]") || !strings.Contains(r.Stderr, "POST") {
		t.Fatalf("stderr must log the intercepted POST: %s", r.Stderr)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(r.Stdout), &parsed); err != nil {
		t.Fatalf("stdout must stay JSON: %v", err)
	}
	if id, _ := parsed["id"].(string); id != "" {
		t.Fatalf("dry-run must not create an edit, got id %q", id)
	}
}

func TestLive_CorruptKey_FailsCleanly(t *testing.T) {
	corrupt := filepath.Join(homeDir, "corrupt-key.json")
	if err := os.WriteFile(corrupt, []byte(`{"type":"service_account","private_key":"not-a-key"`), 0o600); err != nil {
		t.Fatal(err)
	}
	r := runCLIEnv(t, map[string]string{
		"GPLAY_SERVICE_ACCOUNT_JSON": corrupt,
		"GPLAY_SERVICE_ACCOUNT":      corrupt,
	}, "apps", "list", "--output", "json")
	if r.ExitCode == 0 {
		t.Fatalf("a corrupt key must fail\nstdout: %s", r.Stdout)
	}
	if strings.TrimSpace(r.Stderr) == "" {
		t.Fatal("failure must print an error on stderr")
	}
	if strings.Contains(r.Stderr, "not-a-key") || strings.Contains(r.Stdout, "not-a-key") {
		t.Fatal("output must never echo key material")
	}
}

func TestLive_TinyTimeout_FailsCleanly(t *testing.T) {
	r := runCLIEnv(t, map[string]string{"GPLAY_TIMEOUT": "1ms"},
		"reviews", "list", "--package", AllowedMutationPackage, "--output", "json")
	if r.ExitCode == 0 {
		t.Fatalf("a 1ms timeout must fail\nstdout: %s", r.Stdout)
	}
	if strings.TrimSpace(r.Stderr) == "" {
		t.Fatal("timeout must print an error on stderr, not panic silently")
	}
	if strings.Contains(r.Stderr, "panic:") {
		t.Fatalf("timeout must not panic: %s", r.Stderr)
	}
}

// --- Mutating smoke: exactly one disposable edit, deleted without commit ---

func TestLive_DisposableEditWorkflow(t *testing.T) {
	if !mutationsEnabled() {
		t.Skip("skipping mutations; set GPLAY_LIVE_SMOKE_MUTATIONS=1")
	}
	pkg := AllowedMutationPackage
	if err := EnsureMutationAllowed(pkg); err != nil {
		t.Fatal(err)
	}

	// Create the single shared edit.
	created := mustJSON(t, runCLI(t, "edits", "create", "--package", pkg, "--output", "json"), "edits create")
	editID, _ := created["id"].(string)
	if editID == "" {
		t.Fatalf("edits create returned no id: %+v", created)
	}

	ledger := NewLedger(ledgerPath)
	if err := ledger.Append(Resource{Kind: "edit", ID: editID, Package: pkg, RunID: runID}); err != nil {
		t.Fatalf("recording edit in ledger: %v", err)
	}

	deleted := false
	deleteEdit := func() {
		if deleted {
			return
		}
		deleted = true
		r := runCLI(t, "edits", "delete", "--package", pkg, "--edit", editID, "--confirm")
		if r.ExitCode != 0 {
			t.Logf("edit delete (cleanup step will retry): %s", r.Stderr)
		}
	}
	defer deleteEdit()

	t.Run("tracks list", func(t *testing.T) {
		parsed := mustJSON(t, runCLI(t, "tracks", "list", "--package", pkg, "--edit", editID, "--output", "json"), "tracks list")
		raw, _ := json.Marshal(parsed)
		if !strings.Contains(string(raw), "internal") {
			t.Fatalf("tracks list must include the internal track: %s", raw)
		}
	})

	t.Run("tracks get internal", func(t *testing.T) {
		mustJSON(t, runCLI(t, "tracks", "get", "--package", pkg, "--edit", editID, "--track", "internal", "--output", "json"), "tracks get")
	})

	t.Run("listings list", func(t *testing.T) {
		mustJSON(t, runCLI(t, "listings", "list", "--package", pkg, "--edit", editID, "--output", "json"), "listings list")
	})

	t.Run("bundles list has used version codes", func(t *testing.T) {
		parsed := mustJSON(t, runCLI(t, "bundles", "list", "--package", pkg, "--edit", editID, "--output", "json"), "bundles list")
		// The fixture-release script computes the next version code from
		// this list. The bundles must carry integer versionCode fields.
		bundles, _ := parsed["bundles"].([]any)
		if len(bundles) == 0 {
			t.Fatal("the fixture app must have at least one uploaded bundle")
		}
		first, _ := bundles[0].(map[string]any)
		if _, ok := first["versionCode"].(float64); !ok {
			t.Fatalf("bundles[0].versionCode must be a number: %v", first)
		}
	})

	t.Run("apks list", func(t *testing.T) {
		mustJSON(t, runCLI(t, "apks", "list", "--package", pkg, "--edit", editID, "--output", "json"), "apks list")
	})

	t.Run("details get", func(t *testing.T) {
		parsed := mustJSON(t, runCLI(t, "details", "get", "--package", pkg, "--edit", editID, "--output", "json"), "details get")
		if lang, _ := parsed["defaultLanguage"].(string); lang == "" {
			t.Fatalf("details must carry a default language: %v", parsed)
		}
	})

	t.Run("testers get internal", func(t *testing.T) {
		mustJSON(t, runCLI(t, "testers", "get", "--package", pkg, "--edit", editID, "--track", "internal", "--output", "json"), "testers get")
	})

	t.Run("availability get production", func(t *testing.T) {
		// The API rejects the internal track for country availability, so
		// this reads the production track. It is a read-only call; the
		// suite never mutates the production track.
		mustJSON(t, runCLI(t, "availability", "get", "--package", pkg, "--edit", editID, "--track", "production", "--output", "json"), "availability get")
	})

	t.Run("images list", func(t *testing.T) {
		mustJSON(t, runCLI(t, "images", "list", "--package", pkg, "--edit", editID, "--locale", "en-US", "--type", "phoneScreenshots", "--output", "json"), "images list")
	})

	t.Run("listing patch and readback inside edit", func(t *testing.T) {
		marker := ResourceName(runID, "t")
		if len(marker) > 30 {
			marker = marker[:30] // Play listing titles are capped at 30 chars
		}
		// listings update is a full PUT and would wipe the descriptions
		// inside the edit, which then fails edits validate. Patch changes
		// only the title.
		update := runCLI(t, "listings", "patch", "--package", pkg, "--edit", editID,
			"--locale", "en-US", "--title", marker)
		if update.ExitCode != 0 {
			t.Fatalf("listings patch: %s", update.Stderr)
		}

		got := mustJSON(t, runCLI(t, "listings", "get", "--package", pkg, "--edit", editID, "--locale", "en-US", "--output", "json"), "listings get")
		if title, _ := got["title"].(string); title != marker {
			t.Fatalf("readback title = %q, want %q", got["title"], marker)
		}
	})

	t.Run("scratch locale create and delete inside edit", func(t *testing.T) {
		// de-DE does not exist on the fixture app. Create it inside the
		// disposable edit, read it back, then delete it again so the edit
		// still validates (a title-only listing would fail validation).
		marker := ResourceName(runID, "loc")
		if len(marker) > 30 {
			marker = marker[:30]
		}
		created := runCLI(t, "listings", "update", "--package", pkg, "--edit", editID,
			"--locale", "de-DE", "--title", marker)
		if created.ExitCode != 0 {
			t.Fatalf("listings update de-DE: %s", created.Stderr)
		}

		got := mustJSON(t, runCLI(t, "listings", "get", "--package", pkg, "--edit", editID, "--locale", "de-DE", "--output", "json"), "listings get de-DE")
		if title, _ := got["title"].(string); title != marker {
			t.Fatalf("readback title = %q, want %q", got["title"], marker)
		}

		deleted := runCLI(t, "listings", "delete", "--package", pkg, "--edit", editID, "--locale", "de-DE", "--confirm")
		if deleted.ExitCode != 0 {
			t.Fatalf("listings delete de-DE: %s", deleted.Stderr)
		}

		after := runCLI(t, "listings", "get", "--package", pkg, "--edit", editID, "--locale", "de-DE", "--output", "json")
		if after.ExitCode == 0 {
			t.Fatalf("de-DE must be gone after delete\nstdout: %s", after.Stdout)
		}
	})

	t.Run("edit validates", func(t *testing.T) {
		r := runCLI(t, "edits", "validate", "--package", pkg, "--edit", editID, "--output", "json")
		if r.ExitCode != 0 {
			t.Fatalf("edits validate: %s", r.Stderr)
		}
	})

	// Delete without commit: the marker title must never reach the store.
	deleteEdit()
}

// --- Cleanup and janitor ---

// TestLive_Cleanup deletes every resource in the ledger. The workflow runs
// it in an always() step so a crashed run still gets cleaned.
func TestLive_Cleanup(t *testing.T) {
	if os.Getenv("GPLAY_LIVE_SMOKE_CLEANUP") != "1" {
		t.Skip("skipping cleanup; set GPLAY_LIVE_SMOKE_CLEANUP=1")
	}
	resources, err := NewLedger(ledgerPath).Load()
	if err != nil {
		t.Fatalf("loading ledger: %v", err)
	}
	for _, res := range resources {
		if err := EnsureMutationAllowed(res.Package); err != nil {
			t.Errorf("ledger entry refused: %v", err)
			continue
		}
		switch res.Kind {
		case "edit":
			// Deleting an already-deleted or expired edit fails; that is fine.
			r := runCLI(t, "edits", "delete", "--package", res.Package, "--edit", res.ID, "--confirm")
			t.Logf("cleanup edit %s: exit %d", res.ID, r.ExitCode)
		case "iap":
			r := runCLI(t, "iap", "delete", "--package", res.Package, "--sku", res.ID, "--confirm")
			t.Logf("cleanup iap %s: exit %d", res.ID, r.ExitCode)
		case "onetimeproduct":
			r := runCLI(t, "onetimeproducts", "delete", "--package", res.Package, "--product-id", res.ID, "--confirm")
			t.Logf("cleanup onetimeproduct %s: exit %d", res.ID, r.ExitCode)
		default:
			t.Errorf("ledger entry with unknown kind %q: manual cleanup needed for %+v", res.Kind, res)
		}
	}
}

// TestLive_Janitor removes leftovers from interrupted past runs. It only
// deletes products whose SKU carries the lsmoke- prefix. It runs in the
// same concurrency group as the smoke suite, so no live run is in flight
// while it deletes.
func TestLive_Janitor(t *testing.T) {
	if os.Getenv("GPLAY_LIVE_SMOKE_JANITOR") != "1" {
		t.Skip("skipping janitor; set GPLAY_LIVE_SMOKE_JANITOR=1")
	}
	pkg := AllowedMutationPackage
	removed := 0

	// Legacy in-app products created by the live suite (lsmoke-*). The
	// fixture app uses the new monetization model, so the legacy
	// inappproducts API refuses all calls with 403. That leaves nothing
	// to sweep; skip instead of failing.
	iapList := runCLI(t, "iap", "list", "--package", pkg, "--paginate", "--output", "json")
	if iapList.ExitCode != 0 && strings.Contains(iapList.Stderr, "migrate to the new publishing API") {
		t.Log("janitor: legacy iap API is unavailable for this app; skipping the legacy sweep")
	} else {
		for _, product := range mustJSONArray(t, iapList, "iap list") {
			sku, _ := product["sku"].(string)
			if !IsJanitorTarget(sku) {
				continue
			}
			if err := EnsureMutationAllowed(pkg); err != nil {
				t.Fatal(err)
			}
			r := runCLI(t, "iap", "delete", "--package", pkg, "--sku", sku, "--confirm")
			if r.ExitCode != 0 {
				t.Errorf("janitor: deleting %s failed: %s", sku, r.Stderr)
				continue
			}
			removed++
		}
	}

	// One-time products: the Go contract test creates ci_otp_* fixtures
	// (see internal/cli/monetizationpricing). A crashed run leaks them.
	otps := mustJSONArray(t, runCLI(t, "onetimeproducts", "list", "--package", pkg, "--paginate", "--output", "json"), "onetimeproducts list")
	for _, product := range otps {
		productID, _ := product["productId"].(string)
		if !IsJanitorTarget(productID) {
			continue
		}
		if err := EnsureMutationAllowed(pkg); err != nil {
			t.Fatal(err)
		}
		r := runCLI(t, "onetimeproducts", "delete", "--package", pkg, "--product-id", productID, "--confirm")
		if r.ExitCode != 0 {
			t.Errorf("janitor: deleting one-time product %s failed: %s", productID, r.Stderr)
			continue
		}
		removed++
	}
	t.Logf("janitor: removed %d leftover products", removed)
}
