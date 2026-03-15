package output

import (
	"io"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// RenderTable writes a formatted table to stdout.
func RenderTable(headers []string, rows [][]string) {
	RenderTableTo(os.Stdout, headers, rows)
}

// RenderTableTo writes a formatted table to the given writer.
func RenderTableTo(w io.Writer, headers []string, rows [][]string) {
	table := tablewriter.NewTable(w,
		tablewriter.WithConfig(tablewriter.Config{
			Header: tw.CellConfig{
				Formatting: tw.CellFormatting{
					AutoFormat: tw.On,
				},
				Alignment: tw.CellAlignment{Global: tw.AlignLeft},
			},
			Row: tw.CellConfig{
				Alignment: tw.CellAlignment{Global: tw.AlignLeft},
			},
		}),
	)
	table.Header(headers)

	for _, row := range rows {
		sanitized := make([]string, len(row))
		for i, cell := range row {
			sanitized[i] = SanitizeTerminal(cell)
		}
		_ = table.Append(sanitized)
	}

	_ = table.Render()
}

// Truncate shortens a string to maxLen, appending "..." if truncated.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
