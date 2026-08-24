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
	cmdtest.Build(t)

	cmd := exec.Command(cmdtest.BinaryPath, args...)
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + homeDir,
		"GPLAY_CONFIG_PATH=" + filepath.Join(homeDir, "config.json"),
		"GPLAY_SERVICE_ACCOUNT_JSON=" + keyPath,
		"GPLAY_SERVICE_ACCOUNT=" + keyPath,
		"GPLAY_NO_UPDATE=1",
		"GPLAY_TIMEOUT=60s",
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
	parsed := mustJSON(t, runCLI(t, "iap", "list", "--package", pkg, "--paginate", "--output", "json"), "iap list")

	products, _ := parsed["inappproduct"].([]any)
	removed := 0
	for _, p := range products {
		product, _ := p.(map[string]any)
		sku, _ := product["sku"].(string)
		if !IsManagedResourceName(sku) {
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
	t.Logf("janitor: removed %d leftover products", removed)
}
