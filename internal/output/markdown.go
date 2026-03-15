package output

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// RenderMarkdownTable writes a pipe-separated markdown table.
func RenderMarkdownTable(w io.Writer, headers []string, rows [][]string) error {
	if len(headers) == 0 {
		return nil
	}

	// Header row
	fmt.Fprintf(w, "| %s |\n", strings.Join(escapeMarkdownCells(headers), " | "))

	// Separator row
	seps := make([]string, len(headers))
	for i := range seps {
		seps[i] = "---"
	}
	fmt.Fprintf(w, "| %s |\n", strings.Join(seps, " | "))

	// Data rows
	for _, row := range rows {
		// Pad row to match header count
		padded := make([]string, len(headers))
		for i := range padded {
			if i < len(row) {
				padded[i] = row[i]
			}
		}
		fmt.Fprintf(w, "| %s |\n", strings.Join(escapeMarkdownCells(padded), " | "))
	}

	return nil
}

// RenderMarkdownTableToStdout is a convenience wrapper.
func RenderMarkdownTableToStdout(headers []string, rows [][]string) error {
	return RenderMarkdownTable(os.Stdout, headers, rows)
}

func escapeMarkdownCells(cells []string) []string {
	escaped := make([]string, len(cells))
	for i, cell := range cells {
		escaped[i] = strings.ReplaceAll(cell, "|", "\\|")
	}
	return escaped
}
