// Package websession manages browser-session storage for the "gplay web"
// command family. Sessions hold cookies captured from a signed-in browser.
// On macOS they live in the Keychain (service "gplay web session") and only
// metadata (last.json, index.json) is kept under ~/.gplay/web/; setting
// GPLAY_WEB_SESSION_DIR, or running on another platform, stores per-account
// session files there instead. They are deliberately kept out of the regular
// gplay config profiles.
package websession

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// dirEnvVar overrides the session storage directory (used by tests).
const dirEnvVar = "GPLAY_WEB_SESSION_DIR"

// sessionVersion is the current on-disk session schema version.
const sessionVersion = 1

// Cookie is a single browser cookie.
type Cookie struct {
	Name    string    `json:"name"`
	Value   string    `json:"value"`
	Expires time.Time `json:"expires,omitempty"`
}

// Session is a stored browser session for one Google account.
type Session struct {
	Version   int                 `json:"version"`
	UpdatedAt time.Time           `json:"updated_at"`
	UserEmail string              `json:"user_email"`
	Cookies   map[string][]Cookie `json:"cookies"` // keyed by origin URL, e.g. "https://play.google.com/"
}

// lastPointer is the content of last.json, pointing at the most recently
// used session file.
type lastPointer struct {
	Version int    `json:"version"`
	Key     string `json:"key"`
}

// Dir returns the directory holding web session files.
func Dir() string {
	if env := strings.TrimSpace(os.Getenv(dirEnvVar)); env != "" {
		return env
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".gplay", "web")
	}
	return filepath.Join(home, ".gplay", "web")
}

// AccountKey derives the storage key for an account email.
func AccountKey(email string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return hex.EncodeToString(sum[:])
}

// sessionPath returns the session file path for a storage key.
func sessionPath(key string) string {
	return filepath.Join(Dir(), "session-"+key+".json")
}

// lastPath returns the path of the last-session pointer file.
func lastPath() string {
	return filepath.Join(Dir(), "last.json")
}

// Save stores the session: in the macOS Keychain by default on macOS, or as
// a 0600 file under Dir() when GPLAY_WEB_SESSION_DIR is set or on other
// platforms. Either way it updates the last-session pointer.
func Save(s *Session) error {
	if s == nil {
		return fmt.Errorf("session is nil")
	}
	if strings.TrimSpace(s.UserEmail) == "" {
		return fmt.Errorf("session has no user email")
	}

	dir := Dir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating session dir: %w", err)
	}
	// Enforce 0700 even if the directory already existed.
	_ = os.Chmod(dir, 0o700)

	s.Version = sessionVersion
	s.UpdatedAt = time.Now().UTC()

	if useKeychain() {
		return saveKeychain(s)
	}
	return saveFile(s)
}

// saveFile writes the session to disk (0600), keeping any previous file as a
// .bak sibling, and updates the last-session pointer.
func saveFile(s *Session) error {
	key := AccountKey(s.UserEmail)
	path := sessionPath(key)

	// Keep the previous session file as .bak before rewriting.
	if _, err := os.Stat(path); err == nil {
		if err := os.Rename(path, path+".bak"); err != nil {
			return fmt.Errorf("backing up previous session: %w", err)
		}
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding session: %w", err)
	}
	if err := shared.AtomicWrite(path, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return writeLastPointer(key)
}

// writeLastPointer points last.json at a storage key.
func writeLastPointer(key string) error {
	pointer, err := json.Marshal(lastPointer{Version: sessionVersion, Key: key})
	if err != nil {
		return fmt.Errorf("encoding last pointer: %w", err)
	}
	return shared.AtomicWrite(lastPath(), append(pointer, '\n'), 0o600)
}

// Load reads a session. An empty account follows the last-session pointer.
func Load(account string) (*Session, error) {
	key, err := resolveKey(account)
	if err != nil {
		return nil, err
	}
	if useKeychain() {
		return loadKeychain(key)
	}
	return loadByKey(key)
}

// SessionFile returns the session file path for an account (empty account
// follows the last-session pointer).
func SessionFile(account string) (string, error) {
	key, err := resolveKey(account)
	if err != nil {
		return "", err
	}
	return sessionPath(key), nil
}

// resolveKey maps an account email to a storage key, following last.json
// when account is empty.
func resolveKey(account string) (string, error) {
	if strings.TrimSpace(account) != "" {
		return AccountKey(account), nil
	}
	data, err := os.ReadFile(lastPath())
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no web session found (no last-session pointer)")
		}
		return "", fmt.Errorf("reading last-session pointer: %w", err)
	}
	var p lastPointer
	if err := json.Unmarshal(data, &p); err != nil {
		return "", fmt.Errorf("parsing last-session pointer: %w", err)
	}
	if strings.TrimSpace(p.Key) == "" {
		return "", fmt.Errorf("last-session pointer is empty")
	}
	return p.Key, nil
}

// loadByKey reads and decodes the session file for a storage key.
func loadByKey(key string) (*Session, error) {
	data, err := os.ReadFile(sessionPath(key))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no web session found for this account")
		}
		return nil, fmt.Errorf("reading session file: %w", err)
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing session file: %w", err)
	}
	return &s, nil
}

// Delete removes the stored session for an account. An empty account follows
// the last-session pointer. If the deleted session was the last one, the
// pointer is removed too.
func Delete(account string) error {
	key, err := resolveKey(account)
	if err != nil {
		return err
	}
	if useKeychain() {
		return deleteKeychain(key)
	}
	return deleteFile(key)
}

// deleteFile removes the session file (and its .bak sibling) for a key.
func deleteFile(key string) error {
	path := sessionPath(key)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("no web session found for this account")
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("removing session file: %w", err)
	}
	_ = os.Remove(path + ".bak")
	clearLastPointer(key)
	return nil
}

// clearLastPointer removes last.json when it points at key.
func clearLastPointer(key string) {
	if data, err := os.ReadFile(lastPath()); err == nil {
		var p lastPointer
		if json.Unmarshal(data, &p) == nil && p.Key == key {
			_ = os.Remove(lastPath())
		}
	}
}

// DeleteAll removes all stored sessions and the last-session pointer.
func DeleteAll() error {
	if useKeychain() {
		return deleteAllKeychain()
	}
	dir := Dir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading session dir: %w", err)
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "session-") || name == "last.json" {
			if err := os.Remove(filepath.Join(dir, name)); err != nil {
				return fmt.Errorf("removing %s: %w", name, err)
			}
		}
	}
	return nil
}

// List returns the account emails of all stored sessions, sorted.
func List() ([]string, error) {
	if useKeychain() {
		return listKeychain()
	}
	return listFileEmails()
}

// listFileEmails returns the account emails of all plaintext session files,
// sorted.
func listFileEmails() ([]string, error) {
	dir := Dir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading session dir: %w", err)
	}
	var emails []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "session-") || !strings.HasSuffix(name, ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		var s Session
		if json.Unmarshal(data, &s) == nil && strings.TrimSpace(s.UserEmail) != "" {
			emails = append(emails, s.UserEmail)
		}
	}
	sort.Strings(emails)
	return emails, nil
}
