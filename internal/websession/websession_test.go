package websession

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// useTempDir points the session dir at a t.TempDir().
func useTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(dirEnvVar, dir)
	return dir
}

func sampleSession(email string) *Session {
	return &Session{
		UserEmail: email,
		Cookies: map[string][]Cookie{
			"https://play.google.com/": {
				{Name: "SID", Value: "sid-value"},
				{Name: "SAPISID", Value: "sapisid-value", Expires: time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)},
			},
			"https://accounts.google.com/": {
				{Name: "SID", Value: "sid-value"},
			},
		},
	}
}

func TestDir_EnvOverride(t *testing.T) {
	dir := useTempDir(t)
	if Dir() != dir {
		t.Errorf("Dir() = %q, want %q", Dir(), dir)
	}
}

func TestDir_Default(t *testing.T) {
	t.Setenv(dirEnvVar, "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	want := filepath.Join(home, ".gplay", "web")
	if Dir() != want {
		t.Errorf("Dir() = %q, want %q", Dir(), want)
	}
}

func TestAccountKey_LowercaseSHA256(t *testing.T) {
	want := sha256.Sum256([]byte("user@example.com"))
	if got := AccountKey("User@Example.com"); got != hex.EncodeToString(want[:]) {
		t.Errorf("AccountKey = %q, want sha256 of lowercase email %q", got, hex.EncodeToString(want[:]))
	}
	if AccountKey("user@example.com") != AccountKey("USER@example.COM") {
		t.Error("AccountKey should be case-insensitive")
	}
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	dir := useTempDir(t)
	sess := sampleSession("user@example.com")

	if err := Save(sess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Directory must be 0700.
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf("dir perm = %o, want 700", perm)
	}

	// Session file must exist with 0600 and the derived name.
	path := sessionPath(AccountKey("user@example.com"))
	info, err = os.Stat(path)
	if err != nil {
		t.Fatalf("stat session file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("session file perm = %o, want 600", perm)
	}

	// Version and UpdatedAt must be set by Save.
	if sess.Version != sessionVersion {
		t.Errorf("Version = %d, want %d", sess.Version, sessionVersion)
	}
	if sess.UpdatedAt.IsZero() {
		t.Error("UpdatedAt not set by Save")
	}

	// Load by explicit account.
	loaded, err := Load("user@example.com")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.UserEmail != sess.UserEmail {
		t.Errorf("UserEmail = %q, want %q", loaded.UserEmail, sess.UserEmail)
	}
	cookies := loaded.Cookies["https://play.google.com/"]
	if len(cookies) != 2 {
		t.Fatalf("loaded %d play.google.com cookies, want 2", len(cookies))
	}
	if cookies[0].Name != "SID" || cookies[0].Value != "sid-value" {
		t.Errorf("cookie[0] = %+v, want SID/sid-value", cookies[0])
	}
	if cookies[1].Expires.IsZero() {
		t.Error("expiry not round-tripped")
	}

	// Load with empty account must follow last.json.
	last, err := Load("")
	if err != nil {
		t.Fatalf("Load(last): %v", err)
	}
	if last.UserEmail != sess.UserEmail {
		t.Errorf("Load(last) UserEmail = %q, want %q", last.UserEmail, sess.UserEmail)
	}

	// last.json must be 0600 and point at the right key.
	linfo, err := os.Stat(lastPath())
	if err != nil {
		t.Fatalf("stat last.json: %v", err)
	}
	if perm := linfo.Mode().Perm(); perm != 0o600 {
		t.Errorf("last.json perm = %o, want 600", perm)
	}
	data, err := os.ReadFile(lastPath())
	if err != nil {
		t.Fatalf("read last.json: %v", err)
	}
	var p lastPointer
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatalf("parse last.json: %v", err)
	}
	if p.Key != AccountKey("user@example.com") {
		t.Errorf("last.json key = %q, want %q", p.Key, AccountKey("user@example.com"))
	}
}

func TestSave_RewriteKeepsBackup(t *testing.T) {
	useTempDir(t)
	sess := sampleSession("user@example.com")
	if err := Save(sess); err != nil {
		t.Fatalf("Save: %v", err)
	}
	sess.Cookies["https://play.google.com/"][0].Value = "updated-value"
	if err := Save(sess); err != nil {
		t.Fatalf("Save(rewrite): %v", err)
	}

	bakPath := sessionPath(AccountKey("user@example.com")) + ".bak"
	data, err := os.ReadFile(bakPath)
	if err != nil {
		t.Fatalf("reading .bak: %v", err)
	}
	if !strings.Contains(string(data), "sid-value") {
		t.Error(".bak should contain the previous session content")
	}

	loaded, err := Load("user@example.com")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := loaded.Cookies["https://play.google.com/"][0].Value; got != "updated-value" {
		t.Errorf("reloaded value = %q, want updated-value", got)
	}
}

func TestLoad_NotFound(t *testing.T) {
	useTempDir(t)
	if _, err := Load("missing@example.com"); err == nil {
		t.Error("expected error for missing session")
	}
	if _, err := Load(""); err == nil {
		t.Error("expected error when no last-session pointer exists")
	}
}

func TestDelete(t *testing.T) {
	useTempDir(t)
	sess := sampleSession("user@example.com")
	if err := Save(sess); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := Save(sess); err != nil { // create .bak
		t.Fatalf("Save(rewrite): %v", err)
	}

	if err := Delete("user@example.com"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	key := AccountKey("user@example.com")
	if _, err := os.Stat(sessionPath(key)); !os.IsNotExist(err) {
		t.Error("session file should be removed")
	}
	if _, err := os.Stat(sessionPath(key) + ".bak"); !os.IsNotExist(err) {
		t.Error(".bak file should be removed")
	}
	// Pointer must be removed because it pointed at the deleted session.
	if _, err := os.Stat(lastPath()); !os.IsNotExist(err) {
		t.Error("last.json should be removed")
	}

	if err := Delete("user@example.com"); err == nil {
		t.Error("expected error deleting a missing session")
	}
}

func TestDelete_EmptyAccountFollowsLast(t *testing.T) {
	useTempDir(t)
	if err := Save(sampleSession("user@example.com")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := Delete(""); err != nil {
		t.Fatalf("Delete(last): %v", err)
	}
	if _, err := Load("user@example.com"); err == nil {
		t.Error("session should be gone")
	}
}

func TestDeleteAll(t *testing.T) {
	useTempDir(t)
	if err := Save(sampleSession("a@example.com")); err != nil {
		t.Fatalf("Save a: %v", err)
	}
	if err := Save(sampleSession("b@example.com")); err != nil {
		t.Fatalf("Save b: %v", err)
	}
	if err := Save(sampleSession("b@example.com")); err != nil { // create .bak
		t.Fatalf("Save b rewrite: %v", err)
	}

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
	if _, err := os.Stat(lastPath()); !os.IsNotExist(err) {
		t.Error("last.json should be removed")
	}
}

func TestList(t *testing.T) {
	useTempDir(t)
	emails, err := List()
	if err != nil {
		t.Fatalf("List empty: %v", err)
	}
	if len(emails) != 0 {
		t.Errorf("List on empty dir = %v, want empty", emails)
	}

	if err := Save(sampleSession("b@example.com")); err != nil {
		t.Fatalf("Save b: %v", err)
	}
	if err := Save(sampleSession("a@example.com")); err != nil {
		t.Fatalf("Save a: %v", err)
	}
	emails, err = List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(emails) != 2 || emails[0] != "a@example.com" || emails[1] != "b@example.com" {
		t.Errorf("List = %v, want [a@example.com b@example.com]", emails)
	}
}

func TestSave_RequiresEmail(t *testing.T) {
	useTempDir(t)
	if err := Save(&Session{}); err == nil {
		t.Error("expected error saving session without email")
	}
}
