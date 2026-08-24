package livesmoke

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// Resource is one object the live smoke suite created in the fixture app.
type Resource struct {
	Kind      string `json:"kind"`
	ID        string `json:"id"`
	Package   string `json:"package"`
	RunID     string `json:"run_id"`
	CreatedAt string `json:"created_at"`
}

// Ledger is an append-only JSONL record of created resources. The workflow
// cleanup step and the janitor read it to delete leftovers after a crash.
type Ledger struct {
	path string
}

// NewLedger returns a ledger backed by the file at path.
func NewLedger(path string) *Ledger {
	return &Ledger{path: path}
}

// Append records one resource. It stamps CreatedAt and refuses resources
// that belong to any package other than the fixture app.
func (l *Ledger) Append(r Resource) error {
	if err := EnsureMutationAllowed(r.Package); err != nil {
		return err
	}
	if r.Kind == "" || r.ID == "" || r.RunID == "" {
		return fmt.Errorf("ledger: kind, id, and run_id are required, got %+v", r)
	}
	if r.CreatedAt == "" {
		r.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	line, err := json.Marshal(r)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

// Load returns all readable resources. A missing file yields an empty list.
// Corrupt lines are skipped so a partial write never blocks cleanup.
func (l *Ledger) Load() ([]Resource, error) {
	f, err := os.Open(l.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var resources []Resource
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var r Resource
		if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
			continue
		}
		if r.ID == "" {
			continue
		}
		resources = append(resources, r)
	}
	return resources, scanner.Err()
}
