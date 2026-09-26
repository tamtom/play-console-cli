//go:build !windows

package update

import (
	"io"
	"path/filepath"

	"github.com/tamtom/play-console-cli/internal/rootfs"
)

func replaceExecutable(path string, source io.Reader) error {
	root, err := rootfs.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	_, err = root.AtomicWriteFrom(filepath.Base(path), source, 0o755)
	return err
}
