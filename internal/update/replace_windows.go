package update

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/tamtom/play-console-cli/internal/rootfs"
)

// Windows permits renaming a running executable, but not replacing it in place.
// Keep the old image available for rollback until the new name is installed.
func replaceExecutable(path string, source io.Reader) error {
	dir, name := filepath.Dir(path), filepath.Base(path)
	staged := name + ".new-" + rand.Text()
	backup := name + ".old-" + rand.Text()
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	removeStaleBackups(root, name)
	info, err := root.Lstat(name)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("update destination must be a regular executable")
	}
	if _, err := rootfs.AtomicWriteFileFrom(filepath.Join(dir, staged), source, 0o755, 0o755); err != nil {
		return err
	}
	defer func() { _ = root.Remove(staged) }()
	if err := root.Rename(name, backup); err != nil {
		return fmt.Errorf("move current executable: %w", err)
	}
	if err := root.Rename(staged, name); err != nil {
		if rollback := root.Rename(backup, name); rollback != nil {
			return fmt.Errorf("install failed: %w; rollback failed: %w (previous binary: %s)", err, rollback, filepath.Join(dir, backup))
		}
		return fmt.Errorf("install failed; previous executable restored: %w", err)
	}
	// The running image may remain locked until this process exits.
	_ = root.Remove(backup)
	return nil
}
