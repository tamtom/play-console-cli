package cmdtest_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/cmdtest"
	"github.com/tamtom/play-console-cli/internal/sandbox"
	"github.com/tamtom/play-console-cli/internal/testutil"
)

// These tests run the compiled gplay binary against the in-memory sandbox
// API server. They cover the full edit workflow — authentication, flags,
// JSON output, readback, commit, and error behavior — with no credentials,
// no network beyond localhost, and no quota.

func startSandbox(t *testing.T) {
	t.Helper()
	srv, _, err := sandbox.Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(srv.Close)
	t.Setenv("GPLAY_API_BASE_URL", srv.URL)
	t.Setenv("GPLAY_SERVICE_ACCOUNT_JSON", testutil.SandboxServiceAccount(t, srv.URL+"/token"))
}

func sandboxJSON(t *testing.T, args ...string) map[string]any {
	t.Helper()
	r := cmdtest.Run(t, args...)
	if r.ExitCode != 0 {
		t.Fatalf("%v: exit %d\nstderr: %s", args, r.ExitCode, r.Stderr)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(r.Stdout), &parsed); err != nil {
		t.Fatalf("%v: stdout is not JSON: %v\nstdout: %s", args, err, r.Stdout)
	}
	return parsed
}

func TestSandboxWorkflow_FullEditFlow(t *testing.T) {
	cmdtest.Build(t)
	startSandbox(t)
	pkg := sandbox.Package

	created := sandboxJSON(t, "edits", "create", "--package", pkg)
	editID, _ := created["id"].(string)
	if editID == "" {
		t.Fatalf("edits create returned no id: %v", created)
	}

	tracks := sandboxJSON(t, "tracks", "list", "--package", pkg, "--edit", editID)
	raw, _ := json.Marshal(tracks)
	if !strings.Contains(string(raw), "internal") || !strings.Contains(string(raw), "production") {
		t.Fatalf("tracks list must show the seeded tracks: %s", raw)
	}

	track := sandboxJSON(t, "tracks", "get", "--package", pkg, "--edit", editID, "--track", "internal")
	if track["track"] != "internal" {
		t.Fatalf("tracks get: %v", track)
	}

	updated := sandboxJSON(t, "listings", "update", "--package", pkg, "--edit", editID,
		"--locale", "en-US", "--title", "Black-box Title")
	if updated["title"] != "Black-box Title" {
		t.Fatalf("listings update: %v", updated)
	}

	readback := sandboxJSON(t, "listings", "get", "--package", pkg, "--edit", editID, "--locale", "en-US")
	if readback["title"] != "Black-box Title" {
		t.Fatalf("readback title = %v, want the updated title", readback["title"])
	}

	sandboxJSON(t, "edits", "validate", "--package", pkg, "--edit", editID)
	sandboxJSON(t, "edits", "commit", "--package", pkg, "--edit", editID)

	// The committed edit is gone: a follow-up call must fail.
	r := cmdtest.Run(t, "edits", "get", "--package", pkg, "--edit", editID)
	if r.ExitCode == 0 {
		t.Fatalf("edits get after commit must fail\nstdout: %s", r.Stdout)
	}
}

func TestSandboxWorkflow_ReviewsList(t *testing.T) {
	cmdtest.Build(t)
	startSandbox(t)

	reviews := sandboxJSON(t, "reviews", "list", "--package", sandbox.Package)
	raw, _ := json.Marshal(reviews)
	if !strings.Contains(string(raw), "sandbox-review-1") {
		t.Fatalf("reviews list must show the seeded review: %s", raw)
	}
}

func TestSandboxWorkflow_ErrorBehavior(t *testing.T) {
	cmdtest.Build(t)
	startSandbox(t)
	pkg := sandbox.Package

	t.Run("unknown package fails with 404 on stderr", func(t *testing.T) {
		r := cmdtest.Run(t, "edits", "create", "--package", "com.other.app")
		if r.ExitCode == 0 {
			t.Fatalf("must fail\nstdout: %s", r.Stdout)
		}
		if !strings.Contains(r.Stderr, "404") && !strings.Contains(r.Stderr, "not found") && !strings.Contains(r.Stderr, "Package not found") {
			t.Fatalf("stderr must carry the API error, got: %s", r.Stderr)
		}
	})

	t.Run("invalid edit ID fails", func(t *testing.T) {
		r := cmdtest.Run(t, "tracks", "list", "--package", pkg, "--edit", "bogus-edit")
		if r.ExitCode == 0 {
			t.Fatalf("must fail\nstdout: %s", r.Stdout)
		}
	})

	t.Run("unknown track fails", func(t *testing.T) {
		created := sandboxJSON(t, "edits", "create", "--package", pkg)
		editID := created["id"].(string)
		r := cmdtest.Run(t, "tracks", "get", "--package", pkg, "--edit", editID, "--track", "wildcard")
		if r.ExitCode == 0 {
			t.Fatalf("must fail\nstdout: %s", r.Stdout)
		}
	})
}
