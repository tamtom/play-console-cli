package websession

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// stubSecurityScript is a test double for macOS security(1). It stores
// generic-password items as files under $GPLAY_STUB_KEYCHAIN_DIR and, when a
// "locked" sentinel file exists there, fails every operation the way a locked
// Keychain does.
const stubSecurityScript = `#!/bin/sh
set -eu
dir="$GPLAY_STUB_KEYCHAIN_DIR"
if [ -f "$dir/locked" ]; then
  echo "security: SecKeychainItemModifyContent: errSecInteractionNotAllowed" >&2
  exit 253
fi
cmd="$1"; shift
service=""; account=""; password=""
while [ $# -gt 0 ]; do
  case "$1" in
    -s) service="$2"; shift 2 ;;
    -a) account="$2"; shift 2 ;;
    -w)
      # -w takes the password value for add, but is a bare flag for find.
      if [ "$cmd" = "add-generic-password" ]; then
        password="$2"; shift 2
      else
        shift
      fi
      ;;
    -U) shift ;;
    *) echo "stub security: unexpected argument $1" >&2; exit 2 ;;
  esac
done
item="$dir/$service--$account"
case "$cmd" in
  add-generic-password)
    mkdir -p "$dir"
    printf '%s' "$password" > "$item"
    ;;
  find-generic-password)
    if [ ! -f "$item" ]; then
      echo "The specified item could not be found in the keychain." >&2
      exit 44
    fi
    cat "$item"
    ;;
  delete-generic-password)
    if [ ! -f "$item" ]; then
      echo "The specified item could not be found in the keychain." >&2
      exit 44
    fi
    rm -f "$item"
    ;;
  *) echo "stub security: unexpected command $cmd" >&2; exit 2 ;;
esac
`

// useStubKeychain activates the Keychain backend against the stub security
// binary: backend selection is forced to macOS with no dir override, and HOME
// is redirected so last.json/index.json land in a throwaway dir. Returns the
// stub's item directory.
func useStubKeychain(t *testing.T) string {
	t.Helper()
	itemDir := t.TempDir()
	t.Setenv("GPLAY_STUB_KEYCHAIN_DIR", itemDir)
	t.Setenv("HOME", t.TempDir())
	t.Setenv(dirEnvVar, "")

	bin := filepath.Join(t.TempDir(), "security")
	if err := os.WriteFile(bin, []byte(stubSecurityScript), 0o755); err != nil {
		t.Fatalf("writing stub security binary: %v", err)
	}
	origBinary, origGOOS := securityBinary, hostGOOS
	securityBinary, hostGOOS = bin, "darwin"
	t.Cleanup(func() { securityBinary, hostGOOS = origBinary, origGOOS })
	return itemDir
}

// stubItemPath is where the stub security binary stores the item for a key.
func stubItemPath(itemDir, key string) string {
	return filepath.Join(itemDir, keychainService+"--"+key)
}

// saveLegacySession writes a session with the file backend, for migration
// tests, then reactivates the Keychain backend.
func saveLegacySession(t *testing.T, email string) {
	t.Helper()
	hostGOOS = "linux"
	if err := Save(sampleSession(email)); err != nil {
		t.Fatalf("Save(legacy): %v", err)
	}
	hostGOOS = "darwin"
}

// assertDirLacks fails when any file under dir contains the needle.
func assertDirLacks(t *testing.T, dir, needle string) {
	t.Helper()
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), needle) {
			t.Errorf("%s contains cookie material %q", path, needle)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", dir, err)
	}
}

func TestUseKeychain_BackendSelection(t *testing.T) {
	orig := hostGOOS
	t.Cleanup(func() { hostGOOS = orig })

	t.Run("darwin without override", func(t *testing.T) {
		hostGOOS = "darwin"
		t.Setenv(dirEnvVar, "")
		if !useKeychain() {
			t.Error("want Keychain backend on macOS without GPLAY_WEB_SESSION_DIR")
		}
	})
	t.Run("darwin with override", func(t *testing.T) {
		hostGOOS = "darwin"
		t.Setenv(dirEnvVar, t.TempDir())
		if useKeychain() {
			t.Error("want file backend when GPLAY_WEB_SESSION_DIR is set")
		}
	})
	t.Run("non-darwin without override", func(t *testing.T) {
		hostGOOS = "linux"
		t.Setenv(dirEnvVar, "")
		if useKeychain() {
			t.Error("want file backend on non-macOS platforms")
		}
	})
}

func TestKeychain_SaveLoadRoundTrip(t *testing.T) {
	itemDir := useStubKeychain(t)
	sess := sampleSession("user@example.com")
	if err := Save(sess); err != nil {
		t.Fatalf("Save: %v", err)
	}
	key := AccountKey("user@example.com")

	// Cookies must land in the (stub) Keychain, never in a plaintext file.
	if _, err := os.Stat(stubItemPath(itemDir, key)); err != nil {
		t.Fatalf("expected Keychain item: %v", err)
	}
	if _, err := os.Stat(sessionPath(key)); !os.IsNotExist(err) {
		t.Error("no plaintext session file should be written under the Keychain backend")
	}
	assertDirLacks(t, Dir(), "sapisid-value")

	// index.json carries the account metadata with 0600 permissions.
	idx, err := readIndex()
	if err != nil {
		t.Fatalf("readIndex: %v", err)
	}
	entry, ok := idx.Accounts[key]
	if !ok || entry.UserEmail != "user@example.com" {
		t.Fatalf("index entry = %+v, ok=%v, want user@example.com", entry, ok)
	}
	if entry.UpdatedAt.IsZero() {
		t.Error("index entry UpdatedAt not set")
	}
	info, err := os.Stat(indexPath())
	if err != nil {
		t.Fatalf("stat index.json: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("index.json perm = %o, want 600", perm)
	}

	// last.json still points at the account.
	data, err := os.ReadFile(lastPath())
	if err != nil {
		t.Fatalf("read last.json: %v", err)
	}
	var p lastPointer
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatalf("parse last.json: %v", err)
	}
	if p.Key != key {
		t.Errorf("last.json key = %q, want %q", p.Key, key)
	}

	loaded, err := Load("user@example.com")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.UserEmail != sess.UserEmail {
		t.Errorf("UserEmail = %q, want %q", loaded.UserEmail, sess.UserEmail)
	}
	cookies := loaded.Cookies["https://play.google.com/"]
	if len(cookies) != 2 || cookies[0].Value != "sid-value" || cookies[1].Value != "sapisid-value" {
		t.Errorf("loaded cookies = %+v, want the saved ones", cookies)
	}
}

func TestKeychain_SaveRewriteHasNoBackup(t *testing.T) {
	useStubKeychain(t)
	sess := sampleSession("user@example.com")
	if err := Save(sess); err != nil {
		t.Fatalf("Save: %v", err)
	}
	sess.Cookies["https://play.google.com/"][0].Value = "updated-value"
	if err := Save(sess); err != nil {
		t.Fatalf("Save(rewrite): %v", err)
	}

	key := AccountKey("user@example.com")
	if _, err := os.Stat(sessionPath(key) + ".bak"); !os.IsNotExist(err) {
		t.Error("Keychain backend must not write .bak files")
	}
	loaded, err := Load("user@example.com")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := loaded.Cookies["https://play.google.com/"][0].Value; got != "updated-value" {
		t.Errorf("reloaded value = %q, want updated-value", got)
	}
}

func TestKeychain_LoadNotFound(t *testing.T) {
	useStubKeychain(t)
	if _, err := Load("missing@example.com"); err == nil ||
		!strings.Contains(err.Error(), "no web session found") {
		t.Errorf("err = %v, want no-session error", err)
	}
}

func TestKeychain_LoadMigratesLegacyFile(t *testing.T) {
	itemDir := useStubKeychain(t)
	// A pre-Keychain plaintext session, rewritten once so a .bak exists.
	hostGOOS = "linux"
	if err := Save(sampleSession("legacy@example.com")); err != nil {
		t.Fatalf("Save(legacy): %v", err)
	}
	if err := Save(sampleSession("legacy@example.com")); err != nil {
		t.Fatalf("Save(legacy rewrite): %v", err)
	}
	hostGOOS = "darwin"
	key := AccountKey("legacy@example.com")
	if _, err := os.Stat(sessionPath(key) + ".bak"); err != nil {
		t.Fatalf("expected legacy .bak: %v", err)
	}

	loaded, err := Load("legacy@example.com")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.UserEmail != "legacy@example.com" {
		t.Errorf("UserEmail = %q", loaded.UserEmail)
	}

	// The session moved into the Keychain; both plaintext files are gone.
	if _, err := os.Stat(stubItemPath(itemDir, key)); err != nil {
		t.Errorf("expected migrated Keychain item: %v", err)
	}
	if _, err := os.Stat(sessionPath(key)); !os.IsNotExist(err) {
		t.Error("legacy session file should be removed after migration")
	}
	if _, err := os.Stat(sessionPath(key) + ".bak"); !os.IsNotExist(err) {
		t.Error("legacy .bak file should be removed after migration")
	}
	assertDirLacks(t, Dir(), "sapisid-value")

	// The index knows the account, and the last-session pointer still works.
	emails, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(emails) != 1 || emails[0] != "legacy@example.com" {
		t.Errorf("List = %v, want [legacy@example.com]", emails)
	}
	if _, err := Load(""); err != nil {
		t.Errorf("Load(last): %v", err)
	}
}

func TestKeychain_Delete(t *testing.T) {
	itemDir := useStubKeychain(t)
	if err := Save(sampleSession("a@example.com")); err != nil {
		t.Fatalf("Save a: %v", err)
	}
	if err := Save(sampleSession("b@example.com")); err != nil {
		t.Fatalf("Save b: %v", err)
	}
	saveLegacySession(t, "legacy@example.com")

	if err := Delete("a@example.com"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	key := AccountKey("a@example.com")
	if _, err := os.Stat(stubItemPath(itemDir, key)); !os.IsNotExist(err) {
		t.Error("Keychain item should be removed")
	}
	idx, err := readIndex()
	if err != nil {
		t.Fatalf("readIndex: %v", err)
	}
	if _, ok := idx.Accounts[key]; ok {
		t.Error("index entry should be removed")
	}

	// Deleting a legacy-only session removes its plaintext files too.
	legacyKey := AccountKey("legacy@example.com")
	if err := Delete("legacy@example.com"); err != nil {
		t.Fatalf("Delete(legacy): %v", err)
	}
	if _, err := os.Stat(sessionPath(legacyKey)); !os.IsNotExist(err) {
		t.Error("legacy session file should be removed")
	}

	emails, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(emails) != 1 || emails[0] != "b@example.com" {
		t.Errorf("List = %v, want [b@example.com]", emails)
	}
	if err := Delete("a@example.com"); err == nil {
		t.Error("expected error deleting a missing session")
	}
}

func TestKeychain_DeleteAll(t *testing.T) {
	itemDir := useStubKeychain(t)
	if err := Save(sampleSession("a@example.com")); err != nil {
		t.Fatalf("Save a: %v", err)
	}
	if err := Save(sampleSession("b@example.com")); err != nil {
		t.Fatalf("Save b: %v", err)
	}
	saveLegacySession(t, "legacy@example.com")

	if err := DeleteAll(); err != nil {
		t.Fatalf("DeleteAll: %v", err)
	}
	emails, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(emails) != 0 {
		t.Errorf("List after DeleteAll = %v, want empty", emails)
	}
	for _, email := range []string{"a@example.com", "b@example.com"} {
		if _, err := os.Stat(stubItemPath(itemDir, AccountKey(email))); !os.IsNotExist(err) {
			t.Errorf("Keychain item for %s should be removed", email)
		}
	}
	if _, err := os.Stat(sessionPath(AccountKey("legacy@example.com"))); !os.IsNotExist(err) {
		t.Error("legacy session file should be removed")
	}
	if _, err := os.Stat(indexPath()); !os.IsNotExist(err) {
		t.Error("index.json should be removed")
	}
	if _, err := os.Stat(lastPath()); !os.IsNotExist(err) {
		t.Error("last.json should be removed")
	}
}

func TestKeychain_ListMergesLegacyWithoutMigrating(t *testing.T) {
	itemDir := useStubKeychain(t)
	if err := Save(sampleSession("kc@example.com")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	saveLegacySession(t, "legacy@example.com")

	emails, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(emails) != 2 || emails[0] != "kc@example.com" || emails[1] != "legacy@example.com" {
		t.Errorf("List = %v, want [kc@example.com legacy@example.com]", emails)
	}

	// List has no side effects: the legacy file stays, unmigrated.
	legacyKey := AccountKey("legacy@example.com")
	if _, err := os.Stat(sessionPath(legacyKey)); err != nil {
		t.Error("List must not migrate the legacy session file")
	}
	if _, err := os.Stat(stubItemPath(itemDir, legacyKey)); !os.IsNotExist(err) {
		t.Error("List must not write Keychain items")
	}
}

func TestKeychain_LockedKeychainError(t *testing.T) {
	itemDir := useStubKeychain(t)
	if err := os.WriteFile(filepath.Join(itemDir, "locked"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	assertLockedError := func(op string, err error) {
		t.Helper()
		if err == nil {
			t.Fatalf("%s: expected locked-Keychain error", op)
		}
		if !strings.Contains(err.Error(), "Keychain") {
			t.Errorf("%s: err = %v, want it to name the Keychain", op, err)
		}
		if !strings.Contains(err.Error(), dirEnvVar) {
			t.Errorf("%s: err = %v, want the %s escape hatch", op, err, dirEnvVar)
		}
	}

	assertLockedError("Save", Save(sampleSession("user@example.com")))
	_, loadErr := Load("user@example.com")
	assertLockedError("Load", loadErr)
	assertLockedError("Delete", Delete("user@example.com"))
}

func TestStore(t *testing.T) {
	t.Run("file backend returns the session path", func(t *testing.T) {
		useTempDir(t)
		want := sessionPath(AccountKey("user@example.com"))
		if got := Store("user@example.com"); got != want {
			t.Errorf("Store = %q, want %q", got, want)
		}
	})
	t.Run("Keychain backend names the Keychain", func(t *testing.T) {
		useStubKeychain(t)
		if got := Store("user@example.com"); got != "macOS Keychain" {
			t.Errorf("Store = %q, want macOS Keychain", got)
		}
	})
}
