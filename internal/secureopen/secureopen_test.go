package secureopen_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/secureopen"
)

func TestSecureOpen_RegularFileInAllowedDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	f, err := secureopen.SecureOpen(path, dir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer f.Close()

	buf := make([]byte, 5)
	n, err := f.Read(buf)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if string(buf[:n]) != "hello" {
		t.Fatalf("expected %q, got %q", "hello", string(buf[:n]))
	}
}

func TestSecureOpen_SymlinkWithinAllowedDir(t *testing.T) {
	dir := t.TempDir()
	realFile := filepath.Join(dir, "real.txt")
	if err := os.WriteFile(realFile, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	linkPath := filepath.Join(dir, "link.txt")
	if err := os.Symlink(realFile, linkPath); err != nil {
		t.Fatal(err)
	}

	f, err := secureopen.SecureOpen(linkPath, dir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	f.Close()
}

func TestSecureOpen_SymlinkOutsideAllowedDir(t *testing.T) {
	allowedDir := t.TempDir()
	outsideDir := t.TempDir()

	outsideFile := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}

	linkPath := filepath.Join(allowedDir, "escape.txt")
	if err := os.Symlink(outsideFile, linkPath); err != nil {
		t.Fatal(err)
	}

	_, err := secureopen.SecureOpen(linkPath, allowedDir)
	if err == nil {
		t.Fatal("expected error for symlink escaping allowed dir, got nil")
	}
	if !strings.Contains(err.Error(), "outside allowed directories") {
		t.Fatalf("expected 'outside allowed directories' error, got %v", err)
	}
}

func TestSecureOpen_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	traversal := filepath.Join(dir, "..", "..", "..", "etc", "passwd")

	_, err := secureopen.SecureOpen(traversal, dir)
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
}

func TestSecureOpen_NonexistentFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.txt")

	_, err := secureopen.SecureOpen(path, dir)
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
	if !strings.Contains(err.Error(), "resolving path") {
		t.Fatalf("expected 'resolving path' error, got %v", err)
	}
}

func TestSecureOpen_DirectoryPath(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "subdir")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatal(err)
	}

	_, err := secureopen.SecureOpen(subdir, dir)
	if err == nil {
		t.Fatal("expected error for directory path, got nil")
	}
	if !strings.Contains(err.Error(), "is a directory") {
		t.Fatalf("expected 'is a directory' error, got %v", err)
	}
}

func TestSecureOpen_NoAllowedDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := secureopen.SecureOpen(path)
	if err == nil {
		t.Fatal("expected error for no allowed dirs, got nil")
	}
	if !strings.Contains(err.Error(), "at least one allowed directory is required") {
		t.Fatalf("expected 'at least one allowed directory' error, got %v", err)
	}
}

func TestSecureOpen_MultipleAllowedDirs(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	dir3 := t.TempDir()

	path := filepath.Join(dir2, "test.txt")
	if err := os.WriteFile(path, []byte("found"), 0644); err != nil {
		t.Fatal(err)
	}

	f, err := secureopen.SecureOpen(path, dir1, dir2, dir3)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	f.Close()
}

func TestSecureOpenFile_CustomFlags(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "writable.txt")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	f, err := secureopen.SecureOpenFile(path, os.O_WRONLY|os.O_TRUNC, 0644, dir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer f.Close()

	if _, err := f.WriteString("written"); err != nil {
		t.Fatalf("write error: %v", err)
	}
	f.Close()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "written" {
		t.Fatalf("expected %q, got %q", "written", string(data))
	}
}

func TestSecureOpen_FileOutsideAllowedDir(t *testing.T) {
	allowedDir := t.TempDir()
	outsideDir := t.TempDir()

	path := filepath.Join(outsideDir, "outside.txt")
	if err := os.WriteFile(path, []byte("nope"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := secureopen.SecureOpen(path, allowedDir)
	if err == nil {
		t.Fatal("expected error for file outside allowed dir, got nil")
	}
	if !strings.Contains(err.Error(), "outside allowed directories") {
		t.Fatalf("expected 'outside allowed directories' error, got %v", err)
	}
}

func TestSecureOpen_NestedSubdirectory(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(nested, "deep.txt")
	if err := os.WriteFile(path, []byte("deep"), 0644); err != nil {
		t.Fatal(err)
	}

	f, err := secureopen.SecureOpen(path, dir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	f.Close()
}
