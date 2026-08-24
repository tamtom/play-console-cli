package shared

import (
	"os"

	"github.com/tamtom/play-console-cli/internal/rootfs"
)

// AtomicWrite writes data to path atomically using a temp-file-then-rename pattern.
// This prevents corruption if the process crashes during the write.
func AtomicWrite(path string, data []byte, mode os.FileMode) error {
	return rootfs.AtomicWriteFile(path, data, mode, 0o755)
}
