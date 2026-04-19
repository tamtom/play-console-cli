//go:build !windows

package doctor

import "syscall"

// diskFree returns free bytes on the filesystem containing path.
func diskFree(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	// Bavail * Bsize; cast via uint64 for safety on 32-bit systems.
	return uint64(stat.Bavail) * uint64(stat.Bsize), nil //nolint:unconvert
}
