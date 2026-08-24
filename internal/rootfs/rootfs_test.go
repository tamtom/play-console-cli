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

func TestRootOpenReadStreamsRegularFileAndRejectsSymlink(t *testing.T) {
	rootDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(rootDir, "asset.png"), []byte("image"), 0o644); err != nil {
		t.Fatalf("write asset: %v", err)
	}
	root, err := Open(rootDir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = root.Close() }()

	file, err := root.OpenRead("asset.png")
	if err != nil {
		t.Fatalf("OpenRead regular file: %v", err)
	}
	data, err := io.ReadAll(file)
	_ = file.Close()
	if err != nil || string(data) != "image" {
		t.Fatalf("read regular file = %q, %v", data, err)
	}

	outside := filepath.Join(t.TempDir(), "outside.png")
	if err := os.WriteFile(outside, []byte("outside"), 0o644); err != nil {
		t.Fatalf("write outside: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(rootDir, "linked.png")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := root.OpenRead("linked.png"); !errors.Is(err, ErrSymlink) {
		t.Fatalf("OpenRead symlink error = %v, want ErrSymlink", err)
	}
}

func TestRootReadDirListsRootedDirectoryAndRejectsSymlink(t *testing.T) {
	rootDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(rootDir, "images"), 0o755); err != nil {
		t.Fatalf("mkdir images: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "images", "01.png"), []byte("image"), 0o644); err != nil {
		t.Fatalf("write image: %v", err)
	}
	root, err := Open(rootDir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = root.Close() }()
	entries, err := root.ReadDir("images")
	if err != nil || len(entries) != 1 || entries[0].Name() != "01.png" {
		t.Fatalf("ReadDir = %#v, %v", entries, err)
	}

	if err := os.Symlink(filepath.Join(rootDir, "images"), filepath.Join(rootDir, "linked")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := root.ReadDir("linked"); !errors.Is(err, ErrSymlink) {
		t.Fatalf("ReadDir symlink error = %v, want ErrSymlink", err)
	}
}

func TestRootAppendWritesRecordsAndRejectsSymlink(t *testing.T) {
	rootDir := t.TempDir()
	root, err := Open(rootDir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = root.Close() }()
	if err := root.Append("logs/audit.log", []byte("one\n"), 0o600); err != nil {
		t.Fatalf("first Append: %v", err)
	}
	if err := root.Append("logs/audit.log", []byte("two\n"), 0o600); err != nil {
		t.Fatalf("second Append: %v", err)
	}
	data, err := root.ReadFile("logs/audit.log")
	if err != nil || string(data) != "one\ntwo\n" {
		t.Fatalf("appended data = %q, %v", data, err)
	}

	outside := filepath.Join(t.TempDir(), "outside.log")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatalf("write outside: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(rootDir, "linked.log")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := root.Append("linked.log", []byte("unsafe"), 0o600); !errors.Is(err, ErrSymlink) {
		t.Fatalf("Append symlink error = %v, want ErrSymlink", err)
	}
}

func TestRootCheckWritableLeavesNoProbeFile(t *testing.T) {
	rootDir := t.TempDir()
	root, err := Open(rootDir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = root.Close() }()
	if err := root.CheckWritable(); err != nil {
		t.Fatalf("CheckWritable: %v", err)
	}
	entries, err := os.ReadDir(rootDir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("probe leftovers = %#v, %v", entries, err)
	}
}

func TestCreateExclusiveFileNeverReplacesReservation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "receipt.json")
	if err := CreateExclusiveFile(path, []byte("first"), 0o600, 0o700); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if err := CreateExclusiveFile(path, []byte("second"), 0o600, 0o700); !errors.Is(err, os.ErrExist) {
		t.Fatalf("second create error = %v, want ErrExist", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "first" {
		t.Fatalf("reservation = %q, %v", data, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("reservation mode/error = %v, %v", info, err)
	}
}
