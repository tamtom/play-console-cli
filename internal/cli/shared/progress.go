package shared

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// ProgressReader wraps an io.Reader and reports progress to stderr.
type ProgressReader struct {
	reader    io.Reader
	total     int64
	read      int64
	filename  string
	writer    io.Writer // nil to disable output
	mu        sync.Mutex
	startTime time.Time
	lastPrint time.Time
}

// NewProgressReader creates a progress-reporting reader.
// If stderr is not a TTY, output is disabled.
// total can be 0 if unknown.
func NewProgressReader(r io.Reader, total int64, filename string) *ProgressReader {
	var w io.Writer
	if info, err := os.Stderr.Stat(); err == nil && info.Mode()&os.ModeCharDevice != 0 {
		w = os.Stderr
	}
	return &ProgressReader{
		reader:    r,
		total:     total,
		filename:  filename,
		writer:    w,
		startTime: time.Now(),
	}
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.mu.Lock()
	pr.read += int64(n)
	now := time.Now()
	if pr.writer != nil && (now.Sub(pr.lastPrint) > 100*time.Millisecond || err == io.EOF) {
		pr.printProgress()
		pr.lastPrint = now
	}
	pr.mu.Unlock()
	if err == io.EOF && pr.writer != nil {
		pr.printFinal()
	}
	return n, err
}

func (pr *ProgressReader) printProgress() {
	elapsed := time.Since(pr.startTime).Seconds()
	speed := float64(pr.read) / elapsed / 1024 / 1024 // MB/s
	if pr.total > 0 {
		pct := float64(pr.read) / float64(pr.total) * 100
		fmt.Fprintf(pr.writer, "\rUploading %s  %s / %s (%.0f%%)  %.1f MB/s",
			pr.filename, formatBytes(pr.read), formatBytes(pr.total), pct, speed)
	} else {
		fmt.Fprintf(pr.writer, "\rUploading %s  %s  %.1f MB/s",
			pr.filename, formatBytes(pr.read), speed)
	}
}

func (pr *ProgressReader) printFinal() {
	elapsed := time.Since(pr.startTime)
	fmt.Fprintf(pr.writer, "\rUploaded %s  %s in %s\n",
		pr.filename, formatBytes(pr.read), elapsed.Round(time.Millisecond))
}

func formatBytes(b int64) string {
	switch {
	case b >= 1024*1024*1024:
		return fmt.Sprintf("%.1f GB", float64(b)/1024/1024/1024)
	case b >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(b)/1024/1024)
	case b >= 1024:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	default:
		return fmt.Sprintf("%d B", b)
	}
}
