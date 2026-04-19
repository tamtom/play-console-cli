// Package audit provides a file-based journal of gplay command invocations.
//
// Entries are written as newline-delimited JSON to ~/.gplay/audit.log (or a
// user-selected path via GPLAY_AUDIT_LOG). Disabled when GPLAY_AUDIT=0.
// The log feeds `gplay audit` and `gplay quota` commands.
package audit

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	// EnableEnvVar toggles logging. Any value except "0"/"false"/"off" enables it.
	EnableEnvVar = "GPLAY_AUDIT"
	// PathEnvVar overrides the audit log file location.
	PathEnvVar = "GPLAY_AUDIT_LOG"

	defaultDirName  = ".gplay"
	defaultFileName = "audit.log"
)

// Entry is a single audit record.
type Entry struct {
	Timestamp time.Time `json:"timestamp"`
	Command   string    `json:"command"`
	Args      []string  `json:"args,omitempty"`
	Package   string    `json:"package,omitempty"`
	Profile   string    `json:"profile,omitempty"`
	Status    string    `json:"status"` // "ok", "error", "started"
	DurationM int64     `json:"duration_ms,omitempty"`
	Error     string    `json:"error,omitempty"`
	// APICall is set when the entry represents a single API request rather than
	// a command invocation. Used by quota tracking.
	APICall string `json:"api_call,omitempty"`
}

var (
	mu       sync.Mutex
	disabled bool
)

func init() {
	disabled = parseDisabled(os.Getenv(EnableEnvVar))
}

func parseDisabled(value string) bool {
	v := strings.ToLower(strings.TrimSpace(value))
	switch v {
	case "0", "false", "off", "no":
		return true
	default:
		return false
	}
}

// Enabled reports whether audit logging is currently active.
func Enabled() bool {
	mu.Lock()
	defer mu.Unlock()
	return !disabled
}

// SetEnabled overrides the enabled state (mainly for tests).
func SetEnabled(on bool) {
	mu.Lock()
	defer mu.Unlock()
	disabled = !on
}

// Path returns the active audit log path.
func Path() (string, error) {
	if p := strings.TrimSpace(os.Getenv(PathEnvVar)); p != "" {
		return filepath.Clean(p), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home: %w", err)
	}
	return filepath.Join(home, defaultDirName, defaultFileName), nil
}

// Write appends an entry to the audit log. No-op when disabled.
// Returns nil on all I/O errors to avoid breaking user commands.
func Write(entry Entry) error {
	if !Enabled() {
		return nil
	}
	return writeTo("", entry)
}

// WriteTo appends to a specific path (used in tests).
func WriteTo(path string, entry Entry) error {
	return writeTo(path, entry)
}

func writeTo(path string, entry Entry) error {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	if strings.TrimSpace(entry.Status) == "" {
		entry.Status = "ok"
	}

	var err error
	if path == "" {
		path, err = Path()
		if err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	mu.Lock()
	defer mu.Unlock()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

// Query filters audit entries.
type Query struct {
	Since   time.Time
	Until   time.Time
	Command string // substring match
	Status  string // exact match, e.g. "error"
	Limit   int
}

// Read returns entries matching q, newest first.
func Read(q Query) ([]Entry, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	return ReadFrom(path, q)
}

// ReadFrom reads and filters from a specific path.
func ReadFrom(path string, q Query) ([]Entry, error) {
	f, err := os.Open(path) // #nosec G304 -- path comes from audit config
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue // skip malformed rows rather than failing entire query
		}
		if !q.matches(e) {
			continue
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return entries, err
	}

	// Newest first.
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	if q.Limit > 0 && len(entries) > q.Limit {
		entries = entries[:q.Limit]
	}
	return entries, nil
}

func (q Query) matches(e Entry) bool {
	if !q.Since.IsZero() && e.Timestamp.Before(q.Since) {
		return false
	}
	if !q.Until.IsZero() && e.Timestamp.After(q.Until) {
		return false
	}
	if q.Command != "" && !strings.Contains(e.Command, q.Command) {
		return false
	}
	if q.Status != "" && e.Status != q.Status {
		return false
	}
	return true
}

// Clear truncates the audit log.
func Clear() error {
	path, err := Path()
	if err != nil {
		return err
	}
	return ClearAt(path)
}

// ClearAt truncates a specific audit log path. Returns nil if the file does
// not exist.
func ClearAt(path string) error {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	mu.Lock()
	defer mu.Unlock()
	return os.Truncate(path, 0)
}
