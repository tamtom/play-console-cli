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

func PrintMarkdown(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println("```json")
	fmt.Println(string(data))
	fmt.Println("```")
	return nil
}

func PrintTable(v interface{}) error {
	// Minimal table fallback: pretty JSON for now.
	return PrintPrettyJSON(v)
}
