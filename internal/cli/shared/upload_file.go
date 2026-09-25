package shared

import (
	"fmt"
	"os"
	"strings"
)

// OpenUploadFile opens and validates a file before any network mutation. The
// returned descriptor is the same file callers should pass to the API client,
// avoiding a validate-then-reopen race.
func OpenUploadFile(path, description string) (*os.File, error) {
	description = strings.TrimSpace(description)
	if description == "" {
		description = "upload file"
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, WrapActionable(err, "failed to open "+description, "Check that the file exists and is readable.")
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("inspect %s: %w", description, err)
	}
	if !info.Mode().IsRegular() {
		_ = file.Close()
		return nil, fmt.Errorf("%s must be a regular file: %s", description, path)
	}
	if info.Size() == 0 {
		_ = file.Close()
		return nil, fmt.Errorf("%s is empty: %s", description, path)
	}
	return file, nil
}
