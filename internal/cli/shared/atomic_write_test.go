package shared

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWrite_CreatesFileWithCorrectContentAndPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	data := []byte("hello, atomic world")
	mode := os.FileMode(0o644)

	if err := AtomicWrite(path, data, mode); err != nil {
		t.Fatalf("AtomicWrite failed: %v", err)
	}

	// Verify content
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("content = %q; want %q", got, data)
	}

	// Verify permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != mode {
		t.Errorf("permissions = %v; want %v", info.Mode().Perm(), mode)
	}
}

func TestAtomicWrite_CreatesParentDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "test.txt")
	data := []byte("nested content")
	mode := os.FileMode(0o600)

	if err := AtomicWrite(path, data, mode); err != nil {
		t.Fatalf("AtomicWrite failed: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("content = %q; want %q", got, data)
	}
}

func TestAtomicWrite_OverwritesExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	// Write initial content
	if err := os.WriteFile(path, []byte("old content"), 0o644); err != nil {
		t.Fatalf("writing initial file: %v", err)
	}

	// Overwrite atomically
	newData := []byte("new content")
	if err := AtomicWrite(path, newData, 0o644); err != nil {
		t.Fatalf("AtomicWrite failed: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}
	if string(got) != string(newData) {
		t.Errorf("content = %q; want %q", got, newData)
	}
}

func TestAtomicWrite_NoTempFileLeftOnSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	if err := AtomicWrite(path, []byte("data"), 0o644); err != nil {
		t.Fatalf("AtomicWrite failed: %v", err)
	}

	// Check that no temp files remain
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading directory: %v", err)
	}
	for _, entry := range entries {
		if entry.Name() != "test.txt" {
			t.Errorf("unexpected file in directory: %s", entry.Name())
		}
	}
}

func TestAtomicWrite_EmptyData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")

	if err := AtomicWrite(path, []byte{}, 0o644); err != nil {
		t.Fatalf("AtomicWrite failed: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("content length = %d; want 0", len(got))
	}
}
