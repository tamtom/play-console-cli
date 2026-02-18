package secureopen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SecureOpen opens a file after verifying it resolves within one of the allowedDirs.
// It cleans the path, resolves symlinks, and checks the resolved path is within bounds.
func SecureOpen(path string, allowedDirs ...string) (*os.File, error) {
	return SecureOpenFile(path, os.O_RDONLY, 0, allowedDirs...)
}

// SecureOpenFile opens a file with custom flags and permissions after path validation.
func SecureOpenFile(path string, flag int, perm os.FileMode, allowedDirs ...string) (*os.File, error) {
	if len(allowedDirs) == 0 {
		return nil, fmt.Errorf("secureopen: at least one allowed directory is required")
	}

	// Clean the path
	cleaned := filepath.Clean(path)

	// Resolve symlinks
	resolved, err := filepath.EvalSymlinks(cleaned)
	if err != nil {
		return nil, fmt.Errorf("secureopen: resolving path %q: %w", path, err)
	}

	// Check it's not a directory
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, fmt.Errorf("secureopen: stat %q: %w", path, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("secureopen: %q is a directory, not a file", path)
	}

	// Verify resolved path is within at least one allowed directory
	allowed := false
	for _, dir := range allowedDirs {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			continue
		}
		// Resolve symlinks in the allowed dir too
		resolvedDir, err := filepath.EvalSymlinks(absDir)
		if err != nil {
			continue
		}
		if strings.HasPrefix(resolved, resolvedDir+string(filepath.Separator)) || resolved == resolvedDir {
			allowed = true
			break
		}
	}

	if !allowed {
		return nil, fmt.Errorf("secureopen: path %q resolves to %q which is outside allowed directories", path, resolved)
	}

	return os.OpenFile(resolved, flag, perm)
}
