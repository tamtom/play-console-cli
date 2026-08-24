package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/tamtom/play-console-cli/internal/rootfs"
)

// readFile reads a file and returns its contents.
func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// SaveState persists execution results to disk for resume support.
func SaveState(path string, result *ExecutionResult) error {
	if result == nil {
		return fmt.Errorf("cannot save nil execution result")
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal workflow state: %w", err)
	}

	if err := rootfs.AtomicWriteFile(path, data, 0o600, 0o700); err != nil {
		return fmt.Errorf("persist workflow state: %w", err)
	}
	return nil
}

// LoadState reads a previously saved execution result from disk.
// Returns nil, nil if the file does not exist.
func LoadState(path string) (*ExecutionResult, error) {
	data, err := rootfs.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read workflow state: %w", err)
	}

	var result ExecutionResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse workflow state: %w", err)
	}
	return &result, nil
}
