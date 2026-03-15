package output

import (
	"encoding/json"
	"fmt"
)

func PrintJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func PrintPrettyJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// PrintMarkdown renders data as markdown. If the type is registered,
// it uses a markdown table. Otherwise wraps JSON in a code fence.
func PrintMarkdown(v interface{}) error {
	if rendered, err := RenderRegisteredToStdout(v, "markdown"); rendered {
		return err
	}
	// Fallback: JSON in code fence
	fmt.Println("```json")
	if err := PrintPrettyJSON(v); err != nil {
		return err
	}
	fmt.Println("```")
	return nil
}

// PrintTable renders data as a table. If the type is registered in the
// output registry, it uses the registered renderer. Otherwise falls back to JSON.
func PrintTable(v interface{}) error {
	if rendered, err := RenderRegisteredToStdout(v, "table"); rendered {
		return err
	}
	// Fallback: pretty JSON (unregistered types)
	return PrintPrettyJSON(v)
}
