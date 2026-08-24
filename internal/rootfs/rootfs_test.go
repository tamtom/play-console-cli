package rootfs

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootRejectsPathsOutsideTrustedTree(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = root.Close() }()

	for _, name := range []string{"../escape", filepath.Join("..", "escape"), filepath.Join(string(filepath.Separator), "absolute")} {
		if err := root.AtomicWrite(name, []byte("unsafe"), 0o600); !errors.Is(err, ErrEscapesRoot) {
			t.Errorf("AtomicWrite(%q) error = %v, want ErrEscapesRoot", name, err)
		}
	}
}

func TestRootRejectsSymlinkedParentsAndDestination(t *testing.T) {
	trusted := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(trusted, "linked")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	root, err := Open(trusted)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = root.Close() }()

	if err := root.AtomicWrite(filepath.Join("linked", "state.json"), []byte("unsafe"), 0o600); !errors.Is(err, ErrSymlink) {
		t.Fatalf("symlinked parent error = %v, want ErrSymlink", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "state.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("write escaped through a symlinked parent")
	}

	sentinel := filepath.Join(outside, "sentinel")
	if err := os.WriteFile(sentinel, []byte("original"), 0o600); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}
	if err := os.Symlink(sentinel, filepath.Join(trusted, "state.json")); err != nil {
		t.Fatalf("symlink destination: %v", err)
	}
	if err := root.AtomicWrite("state.json", []byte("unsafe"), 0o600); !errors.Is(err, ErrSymlink) {
		t.Fatalf("symlinked destination error = %v, want ErrSymlink", err)
	}
	got, err := os.ReadFile(sentinel)
	if err != nil || string(got) != "original" {
		t.Fatalf("sentinel changed: content=%q error=%v", got, err)
	}
}

func TestRootAtomicWriteCreatesDirectoriesAndReplacesContent(t *testing.T) {
	trusted := t.TempDir()
	root, err := Open(trusted)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = root.Close() }()

	name := filepath.Join("nested", "state.json")
	if err := root.AtomicWrite(name, []byte("first"), 0o600); err != nil {
		t.Fatalf("first AtomicWrite: %v", err)
	}
	if err := root.AtomicWrite(name, []byte("second"), 0o600); err != nil {
		t.Fatalf("second AtomicWrite: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(trusted, name))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "second" {
		t.Fatalf("content = %q, want second", got)
	}
	info, err := os.Lstat(filepath.Join(trusted, name))
	if err != nil {
		t.Fatalf("Lstat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}
	entries, err := os.ReadDir(filepath.Join(trusted, "nested"))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "state.json" {
		t.Fatalf("unexpected directory entries: %#v", entries)
	}
}

func TestRootReadFileRejectsSymlinks(t *testing.T) {
	trusted := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(trusted, "input")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	root, err := Open(trusted)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = root.Close() }()

	if _, err := root.ReadFile("input"); !errors.Is(err, ErrSymlink) {
		t.Fatalf("ReadFile error = %v, want ErrSymlink", err)
	}
}

func TestRootAtomicWriteFromStreamsContents(t *testing.T) {
	trusted := filepath.Join(t.TempDir(), "new-output")
	root, err := OpenOrCreate(trusted, 0o755)
	if err != nil {
		t.Fatalf("OpenOrCreate: %v", err)
	}
	defer func() { _ = root.Close() }()

	written, err := root.AtomicWriteFrom("reports/report.csv", strings.NewReader("header\nvalue\n"), 0o644)
	if err != nil {
		t.Fatalf("AtomicWriteFrom: %v", err)
	}
	if written != int64(len("header\nvalue\n")) {
		t.Fatalf("written = %d", written)
	}
	got, err := root.ReadFile("reports/report.csv")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "header\nvalue\n" {
		t.Fatalf("content = %q", got)
	}
	if _, err := root.AtomicWriteFrom("../escape", io.LimitReader(strings.NewReader("unsafe"), 6), 0o600); !errors.Is(err, ErrEscapesRoot) {
		t.Fatalf("escape error = %v, want ErrEscapesRoot", err)
	}
}
