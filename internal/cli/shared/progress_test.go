package shared

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestProgressReaderImplementsIOReader(t *testing.T) {
	pr := &ProgressReader{reader: strings.NewReader("hello")}
	var _ io.Reader = pr
}

func TestProgressReaderCountsBytes(t *testing.T) {
	data := "hello, world!"
	pr := &ProgressReader{reader: strings.NewReader(data)}

	buf := make([]byte, 4)
	total := 0
	for {
		n, err := pr.Read(buf)
		total += n
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}

	if total != len(data) {
		t.Errorf("read %d bytes, want %d", total, len(data))
	}
	if pr.read != int64(len(data)) {
		t.Errorf("tracked %d bytes, want %d", pr.read, len(data))
	}
}

func TestProgressReaderNoOutputWhenWriterNil(t *testing.T) {
	data := strings.Repeat("x", 1024)
	pr := &ProgressReader{reader: strings.NewReader(data), writer: nil}

	_, err := io.ReadAll(pr)
	if err != nil {
		t.Fatal(err)
	}
	// No panic, no output — writer is nil so nothing is written.
}

func TestProgressReaderWritesToWriter(t *testing.T) {
	data := strings.Repeat("x", 1024)
	var buf bytes.Buffer
	pr := &ProgressReader{
		reader:   strings.NewReader(data),
		total:    int64(len(data)),
		filename: "test.apk",
		writer:   &buf,
	}

	_, err := io.ReadAll(pr)
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "test.apk") {
		t.Errorf("output should contain filename, got: %s", output)
	}
	if !strings.Contains(output, "1.0 KB") {
		t.Errorf("output should contain formatted size, got: %s", output)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
		{1610612736, "1.5 GB"},
	}
	for _, tt := range tests {
		got := formatBytes(tt.input)
		if got != tt.want {
			t.Errorf("formatBytes(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
