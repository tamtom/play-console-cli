package shared

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenUploadFile_RejectsEmptyAndNonRegularFiles(t *testing.T) {
	tempDir := t.TempDir()
	empty := filepath.Join(tempDir, "empty.aab")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "empty", path: empty, want: "empty"},
		{name: "directory", path: tempDir, want: "regular file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := OpenUploadFile(tt.path, "test artifact")
			if file != nil {
				_ = file.Close()
				t.Fatal("expected no file")
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestOpenUploadFile_ReturnsValidatedDescriptor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.aab")
	if err := os.WriteFile(path, []byte("bundle"), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := OpenUploadFile(path, "bundle file")
	if err != nil {
		t.Fatalf("OpenUploadFile: %v", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 6 {
		t.Fatalf("size = %d, want 6", info.Size())
	}
}
