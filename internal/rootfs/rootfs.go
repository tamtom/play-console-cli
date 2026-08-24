// Package rootfs provides filesystem operations anchored to an operator-chosen
// directory. Paths from manifests, APIs, and workflow state are kept beneath
// that root and symlinked components are rejected.
package rootfs

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var (
	// ErrEscapesRoot reports an absolute path, parent traversal, or volume change.
	ErrEscapesRoot = errors.New("path escapes trusted root")
	// ErrSymlink reports a symlink below the trusted root.
	ErrSymlink = errors.New("refusing to follow symlink")
)

// Root is an open descriptor anchored to a trusted directory.
type Root struct {
	path string
	root *os.Root
}

// AtomicWriteFile atomically writes one operator-selected path. The path's
// parent directory is the trusted root; a symlinked destination is refused.
func AtomicWriteFile(path string, data []byte, fileMode, dirMode os.FileMode) error {
	_, err := AtomicWriteFileFrom(path, bytes.NewReader(data), fileMode, dirMode)
	return err
}

// AtomicWriteFileFrom streams one operator-selected path atomically. The
// path's parent directory is the trusted root.
func AtomicWriteFileFrom(path string, source io.Reader, fileMode, dirMode os.FileMode) (int64, error) {
	if strings.TrimSpace(path) == "" {
		return 0, fmt.Errorf("%w: invalid empty path", ErrEscapesRoot)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return 0, fmt.Errorf("create trusted parent directory %q: %w", dir, err)
	}
	root, err := Open(dir)
	if err != nil {
		return 0, err
	}
	defer func() { _ = root.Close() }()
	return root.AtomicWriteFrom(filepath.Base(path), source, fileMode)
}

// ReadFile reads one operator-selected path while refusing a symlinked final
// component.
func ReadFile(path string) ([]byte, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("%w: invalid empty path", ErrEscapesRoot)
	}
	root, err := Open(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	return root.ReadFile(filepath.Base(path))
}

// Open opens an existing trusted directory.
func Open(path string) (*Root, error) {
	if strings.TrimSpace(path) == "" || strings.ContainsRune(path, 0) {
		return nil, fmt.Errorf("%w: invalid trusted root", ErrEscapesRoot)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve trusted root: %w", err)
	}
	opened, err := os.OpenRoot(abs)
	if err != nil {
		return nil, fmt.Errorf("open trusted root %q: %w", path, err)
	}
	return &Root{path: abs, root: opened}, nil
}

// OpenOrCreate creates and opens an operator-selected trusted directory.
func OpenOrCreate(path string, mode os.FileMode) (*Root, error) {
	if strings.TrimSpace(path) == "" || strings.ContainsRune(path, 0) {
		return nil, fmt.Errorf("%w: invalid trusted root", ErrEscapesRoot)
	}
	if mode&^os.FileMode(0o777) != 0 {
		return nil, fmt.Errorf("invalid directory mode %o", mode)
	}
	if err := os.MkdirAll(path, mode); err != nil {
		return nil, fmt.Errorf("create trusted root %q: %w", path, err)
	}
	return Open(path)
}

// Close releases the root descriptor.
func (r *Root) Close() error {
	if r == nil || r.root == nil {
		return nil
	}
	return r.root.Close()
}

// ReadFile reads a regular file without accepting symlinked path components.
func (r *Root) ReadFile(name string) ([]byte, error) {
	clean, err := normalize(name)
	if err != nil {
		return nil, err
	}
	if err := r.rejectSymlinks(clean, false); err != nil {
		return nil, err
	}
	return r.root.ReadFile(clean)
}

// OpenRead opens a regular file for streaming reads without accepting
// symlinked path components. The caller owns the returned file.
func (r *Root) OpenRead(name string) (*os.File, error) {
	clean, err := normalize(name)
	if err != nil {
		return nil, err
	}
	if clean == "." {
		return nil, fmt.Errorf("%w: source must name a file", ErrEscapesRoot)
	}
	if err := r.rejectSymlinks(clean, false); err != nil {
		return nil, err
	}
	file, err := r.root.Open(clean)
	if err != nil {
		return nil, fmt.Errorf("open rooted file %q: %w", name, err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("inspect rooted file %q: %w", name, err)
	}
	if !info.Mode().IsRegular() {
		_ = file.Close()
		return nil, fmt.Errorf("rooted source %q is not a regular file", name)
	}
	return file, nil
}

// ReadDir lists a directory beneath the root in lexical order without
// accepting symlinked path components.
func (r *Root) ReadDir(name string) ([]fs.DirEntry, error) {
	clean, err := normalize(name)
	if err != nil {
		return nil, err
	}
	if err := r.rejectSymlinks(clean, false); err != nil {
		return nil, err
	}
	directory, err := r.root.Open(clean)
	if err != nil {
		return nil, fmt.Errorf("open rooted directory %q: %w", name, err)
	}
	defer func() { _ = directory.Close() }()
	info, err := directory.Stat()
	if err != nil {
		return nil, fmt.Errorf("inspect rooted directory %q: %w", name, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("rooted path %q is not a directory", name)
	}
	entries, err := directory.ReadDir(-1)
	if err != nil {
		return nil, fmt.Errorf("read rooted directory %q: %w", name, err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}

// MkdirAll creates a directory tree without accepting existing symlinked
// components.
func (r *Root) MkdirAll(name string, mode os.FileMode) error {
	clean, err := normalize(name)
	if err != nil {
		return err
	}
	if mode&^os.FileMode(0o777) != 0 {
		return fmt.Errorf("invalid directory mode %o", mode)
	}
	if err := r.rejectSymlinks(clean, true); err != nil {
		return err
	}
	if err := r.root.MkdirAll(clean, mode); err != nil {
		return fmt.Errorf("create rooted directory %q: %w", name, err)
	}
	return r.rejectSymlinks(clean, false)
}

// AtomicWrite writes a file through an exclusive, unpredictable temporary file
// in the destination directory, syncs it, and atomically replaces the target.
func (r *Root) AtomicWrite(name string, data []byte, mode os.FileMode) error {
	_, err := r.AtomicWriteFrom(name, bytes.NewReader(data), mode)
	return err
}

// AtomicWriteFrom streams a file through an exclusive, unpredictable
// temporary file and returns the number of bytes written.
func (r *Root) AtomicWriteFrom(name string, source io.Reader, mode os.FileMode) (int64, error) {
	if source == nil {
		return 0, fmt.Errorf("source reader is required")
	}
	clean, err := normalize(name)
	if err != nil {
		return 0, err
	}
	if clean == "." {
		return 0, fmt.Errorf("%w: destination must name a file", ErrEscapesRoot)
	}
	if mode&^os.FileMode(0o777) != 0 {
		return 0, fmt.Errorf("invalid file mode %o", mode)
	}
	parent := filepath.Dir(clean)
	if err := r.MkdirAll(parent, 0o755); err != nil {
		return 0, err
	}
	if err := r.rejectSymlinks(clean, true); err != nil {
		return 0, err
	}
	if info, statErr := r.root.Lstat(clean); statErr == nil && info.IsDir() {
		return 0, fmt.Errorf("destination %q is a directory", name)
	} else if statErr != nil && !errors.Is(statErr, fs.ErrNotExist) {
		return 0, fmt.Errorf("inspect destination %q: %w", name, statErr)
	}

	tempName, err := r.createTempName(parent)
	if err != nil {
		return 0, err
	}
	temp, err := r.root.OpenFile(tempName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return 0, fmt.Errorf("create rooted temporary file: %w", err)
	}
	committed := false
	defer func() {
		_ = temp.Close()
		if !committed {
			_ = r.root.Remove(tempName)
		}
	}()

	written, err := io.Copy(temp, source)
	if err != nil {
		return written, fmt.Errorf("write rooted temporary file: %w", err)
	}
	if err := temp.Chmod(mode); err != nil {
		return written, fmt.Errorf("set rooted temporary file mode: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return written, fmt.Errorf("sync rooted temporary file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return written, fmt.Errorf("close rooted temporary file: %w", err)
	}
	if err := r.root.Rename(tempName, clean); err != nil {
		return written, fmt.Errorf("replace rooted destination %q: %w", name, err)
	}
	committed = true
	if err := syncDirectory(r.root, parent); err != nil {
		return written, fmt.Errorf("sync rooted destination directory: %w", err)
	}
	return written, nil
}

// Append adds one record to a rooted regular file, creating it when needed.
// Existing symlinks and symlinked parent components are rejected.
func (r *Root) Append(name string, data []byte, mode os.FileMode) error {
	clean, err := normalize(name)
	if err != nil {
		return err
	}
	if clean == "." {
		return fmt.Errorf("%w: destination must name a file", ErrEscapesRoot)
	}
	if mode&^os.FileMode(0o777) != 0 {
		return fmt.Errorf("invalid file mode %o", mode)
	}
	parent := filepath.Dir(clean)
	if err := r.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	if err := r.rejectSymlinks(clean, true); err != nil {
		return err
	}
	file, err := r.root.OpenFile(clean, os.O_WRONLY|os.O_APPEND|os.O_CREATE, mode)
	if err != nil {
		return fmt.Errorf("open rooted append destination %q: %w", name, err)
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("inspect rooted append destination %q: %w", name, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("rooted append destination %q is not a regular file", name)
	}
	if err := file.Chmod(mode); err != nil {
		return fmt.Errorf("set rooted append destination mode: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("append rooted destination %q: %w", name, err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync rooted append destination %q: %w", name, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close rooted append destination %q: %w", name, err)
	}
	if err := syncDirectory(r.root, parent); err != nil {
		return fmt.Errorf("sync rooted append directory: %w", err)
	}
	return nil
}

// CheckWritable verifies that the root can durably create and remove a file.
func (r *Root) CheckWritable() error {
	name, err := r.createTempName(".")
	if err != nil {
		return err
	}
	file, err := r.root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create rooted write probe: %w", err)
	}
	remove := true
	defer func() {
		_ = file.Close()
		if remove {
			_ = r.root.Remove(name)
		}
	}()
	if _, err := file.Write([]byte{0}); err != nil {
		return fmt.Errorf("write rooted probe: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync rooted probe: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close rooted probe: %w", err)
	}
	if err := r.root.Remove(name); err != nil {
		return fmt.Errorf("remove rooted probe: %w", err)
	}
	remove = false
	return syncDirectory(r.root, ".")
}

func (r *Root) createTempName(parent string) (string, error) {
	for attempt := 0; attempt < 10; attempt++ {
		var random [16]byte
		if _, err := rand.Read(random[:]); err != nil {
			return "", fmt.Errorf("generate temporary name: %w", err)
		}
		name := filepath.Join(parent, ".gplay-tmp-"+hex.EncodeToString(random[:]))
		if _, err := r.root.Lstat(name); errors.Is(err, fs.ErrNotExist) {
			return name, nil
		} else if err != nil {
			return "", fmt.Errorf("inspect temporary name: %w", err)
		}
	}
	return "", fmt.Errorf("allocate unique rooted temporary name")
}

func (r *Root) rejectSymlinks(name string, allowMissing bool) error {
	if r == nil || r.root == nil {
		return fmt.Errorf("trusted root is closed")
	}
	if name == "." {
		return nil
	}
	parts := strings.Split(name, string(filepath.Separator))
	current := ""
	for _, part := range parts {
		if current == "" {
			current = part
		} else {
			current = filepath.Join(current, part)
		}
		info, err := r.root.Lstat(current)
		if errors.Is(err, fs.ErrNotExist) && allowMissing {
			return nil
		}
		if err != nil {
			return fmt.Errorf("inspect rooted path %q: %w", name, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: %s", ErrSymlink, current)
		}
	}
	return nil
}

func normalize(name string) (string, error) {
	if strings.TrimSpace(name) == "" || strings.ContainsRune(name, 0) {
		return "", fmt.Errorf("%w: invalid empty path", ErrEscapesRoot)
	}
	if filepath.IsAbs(name) || filepath.VolumeName(name) != "" || strings.HasPrefix(name, `\\`) || strings.HasPrefix(name, "//") {
		return "", fmt.Errorf("%w: absolute path %q", ErrEscapesRoot, name)
	}
	for _, part := range strings.FieldsFunc(name, func(r rune) bool { return r == '/' || r == '\\' }) {
		if part == ".." {
			return "", fmt.Errorf("%w: parent traversal in %q", ErrEscapesRoot, name)
		}
	}
	clean := filepath.Clean(name)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: parent traversal in %q", ErrEscapesRoot, name)
	}
	return clean, nil
}
