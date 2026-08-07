package websession

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// keychainService labels the generic-password items holding web sessions in
// the macOS Keychain; each item's account is the AccountKey of the email.
const keychainService = "gplay web session"

// securityBinary is the macOS security(1) tool. A var so tests can point it
// at a stub script.
var securityBinary = "/usr/bin/security"

// hostGOOS gates the Keychain backend. A var so tests can exercise that
// backend with the stub security binary on any platform.
var hostGOOS = runtime.GOOS

// errKeychainItemNotFound reports a Keychain lookup miss for an account.
var errKeychainItemNotFound = errors.New("no web session found in macOS Keychain")

// useKeychain reports whether sessions are stored in the macOS Keychain: on
// macOS with no GPLAY_WEB_SESSION_DIR override. The override forces file
// storage and is the escape hatch for locked or headless Keychains.
func useKeychain() bool {
	return hostGOOS == "darwin" && strings.TrimSpace(os.Getenv(dirEnvVar)) == ""
}

// Store describes where the session for an account is (or would be) stored:
// "macOS Keychain" under the Keychain backend, the session file path under
// the file backend. An empty account follows the last-session pointer.
func Store(account string) string {
	if useKeychain() {
		return "macOS Keychain"
	}
	path, err := SessionFile(account)
	if err != nil {
		return ""
	}
	return path
}

// saveKeychain stores the session in the Keychain and records its metadata in
// index.json and last.json. Any legacy plaintext session file for the account
// is removed: under this backend cookies never touch disk.
func saveKeychain(s *Session) error {
	key := AccountKey(s.UserEmail)
	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("encoding session: %w", err)
	}
	if err := keychainSet(key, data); err != nil {
		return err
	}
	// The Keychain upsert is atomic, so no .bak backup is needed.
	_ = os.Remove(sessionPath(key))
	_ = os.Remove(sessionPath(key) + ".bak")
	if err := indexSet(key, indexEntry{UserEmail: s.UserEmail, UpdatedAt: s.UpdatedAt}); err != nil {
		return err
	}
	return writeLastPointer(key)
}

// loadKeychain reads the session from the Keychain. A lookup miss falls back
// to a legacy plaintext session file, which is migrated into the Keychain
// (read-through migration).
func loadKeychain(key string) (*Session, error) {
	data, err := keychainGet(key)
	if err == nil {
		var s Session
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, fmt.Errorf("parsing web session from the macOS Keychain: %w", err)
		}
		return &s, nil
	}
	if !errors.Is(err, errKeychainItemNotFound) {
		return nil, err
	}
	legacy, err := loadByKey(key)
	if err != nil {
		return nil, err
	}
	// Save stores the session in the Keychain and drops the plaintext file.
	if err := Save(legacy); err != nil {
		return nil, fmt.Errorf("migrating web session into the macOS Keychain: %w", err)
	}
	return legacy, nil
}

// deleteKeychain removes the Keychain item, index entry, and any legacy
// plaintext session files for a key.
func deleteKeychain(key string) error {
	kerr := keychainDelete(key)
	if kerr != nil && !errors.Is(kerr, errKeychainItemNotFound) {
		return kerr
	}
	legacy := sessionPath(key)
	_, statErr := os.Stat(legacy)
	if errors.Is(kerr, errKeychainItemNotFound) && statErr != nil {
		return fmt.Errorf("no web session found for this account")
	}
	_ = os.Remove(legacy)
	_ = os.Remove(legacy + ".bak")
	if err := indexRemove(key); err != nil {
		return err
	}
	clearLastPointer(key)
	return nil
}

// deleteAllKeychain removes every Keychain item recorded in the index or in a
// legacy plaintext session file, then clears those files, index.json, and
// last.json.
func deleteAllKeychain() error {
	keys := map[string]struct{}{}
	idx, err := readIndex()
	if err != nil {
		return err
	}
	for key := range idx.Accounts {
		keys[key] = struct{}{}
	}
	dir := Dir()
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading session dir: %w", err)
	}
	for _, e := range entries {
		if key, ok := legacySessionKey(e.Name()); ok {
			keys[key] = struct{}{}
		}
	}
	for key := range keys {
		if err := keychainDelete(key); err != nil && !errors.Is(err, errKeychainItemNotFound) {
			return err
		}
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "session-") || name == "last.json" || name == "index.json" {
			if err := os.Remove(filepath.Join(dir, name)); err != nil {
				return fmt.Errorf("removing %s: %w", name, err)
			}
		}
	}
	return nil
}

// listKeychain merges the index with any legacy plaintext session files,
// without migrating them.
func listKeychain() ([]string, error) {
	seen := map[string]bool{}
	var emails []string
	idx, err := readIndex()
	if err != nil {
		return nil, err
	}
	for _, entry := range idx.Accounts {
		if email := strings.TrimSpace(entry.UserEmail); email != "" && !seen[email] {
			seen[email] = true
			emails = append(emails, email)
		}
	}
	legacy, err := listFileEmails()
	if err != nil {
		return nil, err
	}
	for _, email := range legacy {
		if !seen[email] {
			seen[email] = true
			emails = append(emails, email)
		}
	}
	sort.Strings(emails)
	return emails, nil
}

// legacySessionKey extracts the storage key from a session-<key>.json name.
func legacySessionKey(name string) (string, bool) {
	if !strings.HasPrefix(name, "session-") || !strings.HasSuffix(name, ".json") {
		return "", false
	}
	return strings.TrimSuffix(strings.TrimPrefix(name, "session-"), ".json"), true
}

// keychainSet stores session JSON as the generic-password item for a key.
// -U makes the write an atomic upsert.
func keychainSet(key string, data []byte) error {
	cmd := exec.CommandContext(context.Background(), securityBinary, // #nosec G204 -- fixed path or test override
		"add-generic-password", "-U",
		"-s", keychainService,
		"-a", key,
		"-w", string(data),
	)
	if _, err := cmd.Output(); err != nil {
		return keychainError("storing web session in the macOS Keychain", err)
	}
	return nil
}

// keychainGet reads the session JSON stored for a key.
func keychainGet(key string) ([]byte, error) {
	cmd := exec.CommandContext(context.Background(), securityBinary, // #nosec G204 -- fixed path or test override
		"find-generic-password", "-w",
		"-s", keychainService,
		"-a", key,
	)
	output, err := cmd.Output()
	if err != nil {
		if isKeychainNotFound(err) {
			return nil, errKeychainItemNotFound
		}
		return nil, keychainError("reading web session from the macOS Keychain", err)
	}
	data := bytes.TrimSuffix(output, []byte("\n"))
	if len(data) == 0 {
		return nil, errKeychainItemNotFound
	}
	return data, nil
}

// keychainDelete removes the item for a key.
func keychainDelete(key string) error {
	cmd := exec.CommandContext(context.Background(), securityBinary, // #nosec G204 -- fixed path or test override
		"delete-generic-password",
		"-s", keychainService,
		"-a", key,
	)
	if _, err := cmd.Output(); err != nil {
		if isKeychainNotFound(err) {
			return errKeychainItemNotFound
		}
		return keychainError("deleting web session from the macOS Keychain", err)
	}
	return nil
}

// isKeychainNotFound reports whether a security(1) failure means the item is
// absent (exit code 44, "could not be found in the keychain").
func isKeychainNotFound(err error) bool {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return false
	}
	return exitErr.ExitCode() == 44 || strings.Contains(string(exitErr.Stderr), "could not be found in the keychain")
}

// keychainError formats a security(1) failure, naming the macOS Keychain and
// the GPLAY_WEB_SESSION_DIR escape hatch for locked or headless environments
// (e.g. errSecInteractionNotAllowed).
func keychainError(action string, err error) error {
	return fmt.Errorf("%w; the macOS Keychain may be locked or unavailable — set %s to store web sessions as files instead", commandError(err, action), dirEnvVar)
}

// sessionIndex is the content of index.json: per-account metadata kept on
// disk under the Keychain backend so List and web auth status work without
// reading Keychain items. It holds no cookie material.
type sessionIndex struct {
	Version  int                   `json:"version"`
	Accounts map[string]indexEntry `json:"accounts"`
}

// indexEntry is the index.json record for one account, keyed by AccountKey.
type indexEntry struct {
	UserEmail string    `json:"user_email"`
	UpdatedAt time.Time `json:"updated_at"`
}

// indexPath returns the path of the session index file.
func indexPath() string {
	return filepath.Join(Dir(), "index.json")
}

// readIndex reads index.json, returning an empty index when it is missing.
func readIndex() (*sessionIndex, error) {
	idx := &sessionIndex{Version: sessionVersion, Accounts: map[string]indexEntry{}}
	data, err := os.ReadFile(indexPath())
	if err != nil {
		if os.IsNotExist(err) {
			return idx, nil
		}
		return nil, fmt.Errorf("reading session index: %w", err)
	}
	if err := json.Unmarshal(data, idx); err != nil {
		return nil, fmt.Errorf("parsing session index: %w", err)
	}
	if idx.Accounts == nil {
		idx.Accounts = map[string]indexEntry{}
	}
	return idx, nil
}

// writeIndex persists index.json (0600).
func writeIndex(idx *sessionIndex) error {
	idx.Version = sessionVersion
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding session index: %w", err)
	}
	return shared.AtomicWrite(indexPath(), append(data, '\n'), 0o600)
}

// indexSet records an account in the index.
func indexSet(key string, entry indexEntry) error {
	idx, err := readIndex()
	if err != nil {
		return err
	}
	idx.Accounts[key] = entry
	return writeIndex(idx)
}

// indexRemove drops an account from the index.
func indexRemove(key string) error {
	idx, err := readIndex()
	if err != nil {
		return err
	}
	if _, ok := idx.Accounts[key]; !ok {
		return nil
	}
	delete(idx.Accounts, key)
	return writeIndex(idx)
}
