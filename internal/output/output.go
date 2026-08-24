package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func PrintJSON(v interface{}) error {
	return PrintJSONTo(os.Stdout, v)
}

func PrintJSONTo(w io.Writer, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}

func PrintPrettyJSON(v interface{}) error {
	return PrintPrettyJSONTo(os.Stdout, v)
}

func PrintPrettyJSONTo(w io.Writer, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}

// PrintMarkdown renders data as markdown. If the type is registered,
// it uses a markdown table. Otherwise wraps JSON in a code fence.
func PrintMarkdown(v interface{}) error {
	return PrintMarkdownTo(os.Stdout, v)
}

func PrintMarkdownTo(w io.Writer, v interface{}) error {
	if rendered, err := RenderRegistered(w, v, "markdown"); rendered {
		return err
	}
	// Fallback: JSON in code fence
	if _, err := fmt.Fprintln(w, "```json"); err != nil {
		return err
	}
	if err := PrintPrettyJSONTo(w, v); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w, "```")
	return err
}

// PrintTable renders data as a table. If the type is registered in the
// output registry, it uses the registered renderer. Otherwise falls back to JSON.
func PrintTable(v interface{}) error {
	return PrintTableTo(os.Stdout, v)
}

func PrintTableTo(w io.Writer, v interface{}) error {
	if rendered, err := RenderRegistered(w, v, "table"); rendered {
		return err
	}
	// Fallback: pretty JSON (unregistered types)
	return PrintPrettyJSONTo(w, v)
}
