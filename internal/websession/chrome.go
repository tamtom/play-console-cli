package websession

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tamtom/play-console-cli/internal/webdriver"
)

const (
	chromeEpochOffsetMicros = int64(11644473600000000)
	playGoogleOrigin        = "https://play.google.com/"
)

const chromeCookieQuery = `
SELECT
  host_key,
  name,
  value,
  hex(encrypted_value) AS encrypted_value,
  expires_utc,
  CAST((SELECT value FROM meta WHERE key = 'version') AS INTEGER) AS db_version
FROM cookies
WHERE top_frame_site_key = ''
  AND host_key IN ('.google.com', 'google.com', '.play.google.com', 'play.google.com')
ORDER BY host_key, name, path`

type chromeCookieRow struct {
	HostKey        string `json:"host_key"`
	Name           string `json:"name"`
	Value          string `json:"value"`
	EncryptedValue string `json:"encrypted_value"`
	ExpiresUTC     int64  `json:"expires_utc"`
	DBVersion      int    `json:"db_version"`
}

// ImportChromeCookies reads the current macOS user's Google Chrome profiles
// and returns each profile with Play Console authentication cookies.
func ImportChromeCookies(ctx context.Context) ([]map[string][]Cookie, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("finding home directory: %w", err)
	}
	return ImportChromeCookiesFrom(ctx, filepath.Join(home, "Library", "Application Support", "Google", "Chrome"))
}

// BrowserProfileDir is the Chrome user-data directory for the dedicated
// instance gplay drives, so a Play Console login survives across runs
// independently of the user's everyday Chrome profiles.
func BrowserProfileDir() string {
	return filepath.Join(Dir(), "browser")
}

// ImportChromeCookiesFrom reads Play Console cookies from an explicit Chrome
// user-data directory. The macOS Safe Storage key is app-wide, so a dedicated
// user-data directory decrypts with the same key as the default profiles.
func ImportChromeCookiesFrom(ctx context.Context, userDataDir string) ([]map[string][]Cookie, error) {
	if runtime.GOOS != "darwin" {
		return nil, fmt.Errorf("automatic Chrome cookie import is currently supported only on macOS; use --cookies or --cookies-file")
	}
	databases, err := discoverChromeCookieDBs(userDataDir)
	if err != nil {
		return nil, err
	}
	password, err := chromeSafeStoragePassword(ctx)
	if err != nil {
		return nil, err
	}
	key, err := deriveChromeKey(password)
	if err != nil {
		return nil, fmt.Errorf("deriving Chrome cookie key: %w", err)
	}

	var candidates []map[string][]Cookie
	var lastErr error
	for _, database := range databases {
		rows, err := readChromeRows(ctx, database)
		if err != nil {
			lastErr = err
			continue
		}
		cookies, err := chromeCookies(rows, key, time.Now())
		if err != nil {
			lastErr = err
			continue
		}
		if hasChromeSAPISID(cookies) {
			candidates = append(candidates, map[string][]Cookie{playGoogleOrigin: cookies})
		}
	}
	if len(candidates) > 0 {
		return candidates, nil
	}
	if lastErr != nil {
		return nil, fmt.Errorf("no usable Play Console cookies found in Chrome: %w", lastErr)
	}
	return nil, fmt.Errorf("no SAPISID cookie found in Chrome; sign in to https://play.google.com/console and retry")
}

func discoverChromeCookieDBs(userDataDir string) ([]string, error) {
	entries, err := os.ReadDir(userDataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no Chrome profile directory at %s", userDataDir)
		}
		return nil, fmt.Errorf("reading Chrome profiles: %w", err)
	}

	var databases []string
	for _, entry := range entries {
		if !entry.IsDir() || (entry.Name() != "Default" && !strings.HasPrefix(entry.Name(), "Profile ")) {
			continue
		}
		for _, relative := range []string{filepath.Join("Network", "Cookies"), "Cookies"} {
			path := filepath.Join(userDataDir, entry.Name(), relative)
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				databases = append(databases, path)
				break
			}
		}
	}
	sort.Strings(databases)
	if len(databases) == 0 {
		return nil, fmt.Errorf("no Chrome cookie database found under %s", userDataDir)
	}
	return databases, nil
}

func chromeSafeStoragePassword(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "/usr/bin/security",
		"find-generic-password", "-w",
		"-s", "Chrome Safe Storage",
		"-a", "Chrome",
	)
	output, err := cmd.Output()
	if err != nil {
		return "", commandError(err, "reading Chrome Safe Storage from Keychain (approve the macOS access prompt)")
	}
	password := strings.TrimSuffix(string(output), "\n")
	if password == "" {
		return "", fmt.Errorf("empty Chrome Safe Storage password")
	}
	return password, nil
}

func readChromeRows(ctx context.Context, database string) ([]chromeCookieRow, error) {
	cmd := exec.CommandContext(ctx, "/usr/bin/sqlite3", "-readonly", "-json", database, chromeCookieQuery)
	output, err := cmd.Output()
	if err != nil {
		return nil, commandError(err, "reading Chrome cookie database")
	}
	return parseChromeRows(output)
}

func commandError(err error, action string) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if stderr := strings.TrimSpace(string(exitErr.Stderr)); stderr != "" {
			return fmt.Errorf("%s: %s", action, stderr)
		}
	}
	return fmt.Errorf("%s: %w", action, err)
}

func parseChromeRows(data []byte) ([]chromeCookieRow, error) {
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, nil
	}
	var rows []chromeCookieRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, fmt.Errorf("parsing Chrome cookie database output: %w", err)
	}
	return rows, nil
}

func chromeCookies(rows []chromeCookieRow, key []byte, now time.Time) ([]Cookie, error) {
	byName := make(map[string]Cookie)
	for _, row := range rows {
		if strings.TrimSpace(row.Name) == "" {
			continue
		}
		expires := chromeTime(row.ExpiresUTC)
		if !expires.IsZero() && !expires.After(now) {
			continue
		}
		if row.Value != "" && row.EncryptedValue != "" {
			if isSAPISIDName(row.Name) {
				return nil, fmt.Errorf("cookie %s from Chrome has both plaintext and encrypted values", row.Name)
			}
			continue
		}

		value := row.Value
		if row.EncryptedValue != "" {
			encrypted, err := hex.DecodeString(row.EncryptedValue)
			if err != nil {
				if isSAPISIDName(row.Name) {
					return nil, fmt.Errorf("decoding Chrome %s cookie: %w", row.Name, err)
				}
				continue
			}
			value, err = decryptChromeValue(key, row.HostKey, encrypted, row.DBVersion)
			if err != nil {
				if isSAPISIDName(row.Name) {
					return nil, fmt.Errorf("decrypting Chrome %s cookie: %w", row.Name, err)
				}
				continue
			}
		}
		byName[row.Name] = Cookie{Name: row.Name, Value: value, Expires: expires}
	}

	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)
	cookies := make([]Cookie, 0, len(names))
	for _, name := range names {
		cookies = append(cookies, byName[name])
	}
	return cookies, nil
}

func deriveChromeKey(password string) ([]byte, error) {
	return pbkdf2.Key(sha1.New, password, []byte("saltysalt"), 1003, 16)
}

func decryptChromeValue(key []byte, host string, encrypted []byte, dbVersion int) (string, error) {
	// Schema 24 added the domain-hash prefix checked below; newer schemas keep
	// it, and a wrong guess still fails on the "v10" prefix or the hash.
	if dbVersion < 24 {
		return "", fmt.Errorf("unsupported Chrome cookie schema %d (need 24 or newer)", dbVersion)
	}
	if len(encrypted) < 3 || string(encrypted[:3]) != "v10" {
		return "", fmt.Errorf("unsupported Chrome encryption prefix")
	}
	ciphertext := encrypted[3:]
	if len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return "", fmt.Errorf("invalid Chrome ciphertext length")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	plaintext := make([]byte, len(ciphertext))
	iv := []byte("                ")
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plaintext, ciphertext)

	padding := int(plaintext[len(plaintext)-1])
	if padding == 0 || padding > aes.BlockSize || padding > len(plaintext) {
		return "", fmt.Errorf("invalid Chrome cookie padding")
	}
	for _, b := range plaintext[len(plaintext)-padding:] {
		if int(b) != padding {
			return "", fmt.Errorf("invalid Chrome cookie padding")
		}
	}
	plaintext = plaintext[:len(plaintext)-padding]

	hash := sha256.Sum256([]byte(host))
	if len(plaintext) < len(hash) || subtle.ConstantTimeCompare(plaintext[:len(hash)], hash[:]) != 1 {
		return "", fmt.Errorf("domain hash mismatch for Chrome cookie")
	}
	plaintext = plaintext[len(hash):]
	return string(plaintext), nil
}

func chromeTime(microseconds int64) time.Time {
	if microseconds <= chromeEpochOffsetMicros {
		return time.Time{}
	}
	return time.UnixMicro(microseconds - chromeEpochOffsetMicros).UTC()
}

func hasChromeSAPISID(cookies []Cookie) bool {
	for _, cookie := range cookies {
		if isSAPISIDName(cookie.Name) && cookie.Value != "" {
			return true
		}
	}
	return false
}

// chromeBinaryEnv overrides Chrome executable discovery without recompiling.
const chromeBinaryEnv = "GPLAY_CHROME_BINARY"

// chromeBinary locates the Google Chrome executable. Auto-discovery only
// knows the macOS install locations; elsewhere the env override is the way in.
func chromeBinary() (string, error) {
	if custom := strings.TrimSpace(os.Getenv(chromeBinaryEnv)); custom != "" {
		return custom, nil
	}
	if runtime.GOOS != "darwin" {
		return "", fmt.Errorf("browser-driven web commands find Chrome automatically only on macOS today; set %s to the Chrome executable path to run them here", chromeBinaryEnv)
	}
	home, _ := os.UserHomeDir()
	for _, candidate := range []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		filepath.Join(home, "Applications", "Google Chrome.app", "Contents", "MacOS", "Google Chrome"),
	} {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no Google Chrome install found; install it or set %s to the executable path", chromeBinaryEnv)
}

// LaunchChrome opens a visible Chrome window that later commands can drive
// over the DevTools protocol (see internal/webdriver). The process is
// deliberately detached from this command: the window outlives the command so
// the user can review what the driver staged.
func LaunchChrome(ctx context.Context, userDataDir, startURL string) error {
	cmd, err := startChrome(userDataDir, startURL, webdriver.LaunchArgs(userDataDir))
	if err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }() // reap the child without blocking
	return nil
}

// LaunchInteractiveChrome opens a visible Chrome window for the user to drive
// themselves (sign-in). It deliberately opens no DevTools port: the window
// holds a live Google session, DevTools has no authentication, and any local
// process that read DevToolsActivePort could drive that session for as long
// as the window stayed open. The returned function closes the window
// gracefully; it tolerates the process having already exited (a second launch
// against the same user-data-dir forwards to the running Chrome and exits).
func LaunchInteractiveChrome(ctx context.Context, userDataDir, startURL string) (terminate func(), err error) {
	cmd, err := startChrome(userDataDir, startURL, webdriver.InteractiveArgs(userDataDir))
	if err != nil {
		return nil, err
	}
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			_ = terminateChrome(cmd.Process) // best effort: may already be gone
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				_ = cmd.Process.Kill()
				<-done
			}
		})
	}, nil
}

// startChrome launches Chrome with the given flags bound to its own user-data
// directory. Not CommandContext: the window outlives the command on purpose.
func startChrome(userDataDir, startURL string, args []string) (*exec.Cmd, error) {
	binary, err := chromeBinary()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(userDataDir, 0o700); err != nil {
		return nil, fmt.Errorf("creating Chrome profile directory: %w", err)
	}
	cmd := exec.Command(binary, append(args, startURL)...) // #nosec G204 -- path is a fixed location or operator-set env override
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("launching Chrome: %w", err)
	}
	return cmd, nil
}

func isSAPISIDName(name string) bool {
	switch name {
	case "SAPISID", "__Secure-1PAPISID", "__Secure-3PAPISID":
		return true
	default:
		return false
	}
}
