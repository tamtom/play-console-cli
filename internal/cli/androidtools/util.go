package androidtools

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tamtom/play-console-cli/internal/rootfs"
)

var lookupExecutable = exec.LookPath

type localArtifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"sizeBytes"`
}

func inspectLocalArtifact(path string) (localArtifact, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return localArtifact{}, err
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return localArtifact{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return localArtifact{}, fmt.Errorf("artifact must be a regular, non-symlink file: %s", abs)
	}
	root, err := rootfs.Open(filepath.Dir(abs))
	if err != nil {
		return localArtifact{}, err
	}
	defer func() { _ = root.Close() }()
	file, err := root.OpenRead(filepath.Base(abs))
	if err != nil {
		return localArtifact{}, err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return localArtifact{}, err
	}
	return localArtifact{Path: abs, SHA256: hex.EncodeToString(hash.Sum(nil)), Size: info.Size()}, nil
}

func resolvedExecutable(name string) (string, error) {
	path, err := lookupExecutable(name)
	if err != nil {
		return "", fmt.Errorf("%s is required: %w", name, err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", name, err)
	}
	return abs, nil
}

func diagnostic(output string) string {
	trimmed := strings.TrimSpace(output)
	const max = 8192
	if len(trimmed) > max {
		return trimmed[:max] + "... (truncated)"
	}
	return trimmed
}
