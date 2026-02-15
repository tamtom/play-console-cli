package output

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
)

// RenderTable writes a formatted table to stdout with the given headers and rows.
func RenderTable(headers []string, rows [][]string) {
	RenderTableTo(os.Stdout, headers, rows)
}

// RenderTableTo writes a formatted table to the given writer.
func RenderTableTo(w io.Writer, headers []string, rows [][]string) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

	// Print headers
	fmt.Fprintln(tw, strings.Join(headers, "\t"))

	// Print separator
	seps := make([]string, len(headers))
	for i, h := range headers {
		seps[i] = strings.Repeat("-", len(h))
	}
	fmt.Fprintln(tw, strings.Join(seps, "\t"))

	// Print rows
	if len(rows) == 0 {
		fmt.Fprintln(tw, "No results found.")
	} else {
		for _, row := range rows {
			fmt.Fprintln(tw, strings.Join(row, "\t"))
		}
	}

	tw.Flush()
}

// Truncate truncates a string to maxLen characters, adding "..." if truncated.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
