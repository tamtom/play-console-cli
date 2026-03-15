package output

import (
	"bytes"
	"strings"
	"testing"
)

type testApp struct {
	Name    string
	Package string
}

func init() {
	RegisterType([]testApp{}, []string{"Name", "Package"}, func(data any) [][]string {
		apps := data.([]testApp)
		rows := make([][]string, len(apps))
		for i, app := range apps {
			rows[i] = []string{app.Name, app.Package}
		}
		return rows
	})
}

func TestRegisterType_And_RenderRegistered_Table(t *testing.T) {
	apps := []testApp{
		{Name: "My App", Package: "com.example.app"},
		{Name: "Other", Package: "com.example.other"},
	}

	var buf bytes.Buffer
	rendered, err := RenderRegistered(&buf, apps, "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rendered {
		t.Fatal("expected type to be found in registry")
	}

	out := buf.String()
	if !strings.Contains(out, "My App") {
		t.Errorf("table output should contain 'My App', got: %q", out)
	}
	if !strings.Contains(out, "com.example.app") {
		t.Errorf("table output should contain 'com.example.app', got: %q", out)
	}
}

func TestRegisterType_And_RenderRegistered_Markdown(t *testing.T) {
	apps := []testApp{
		{Name: "My App", Package: "com.example.app"},
	}

	var buf bytes.Buffer
	rendered, err := RenderRegistered(&buf, apps, "markdown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rendered {
		t.Fatal("expected type to be found in registry")
	}

	out := buf.String()
	// Should be a markdown table with pipes
	if !strings.Contains(out, "|") {
		t.Errorf("markdown output should contain pipe separators, got: %q", out)
	}
	if !strings.Contains(out, "My App") {
		t.Errorf("markdown output should contain 'My App', got: %q", out)
	}
	if !strings.Contains(out, "---") {
		t.Errorf("markdown output should contain separator row, got: %q", out)
	}
}

func TestRenderRegistered_UnregisteredType(t *testing.T) {
	type unknownType struct{ X int }
	data := unknownType{X: 42}

	var buf bytes.Buffer
	rendered, err := RenderRegistered(&buf, data, "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rendered {
		t.Error("expected unregistered type to return rendered=false")
	}

	out := buf.String()
	if out != "" {
		t.Errorf("expected no output for unregistered type, got: %q", out)
	}
}

func TestRenderRegistered_MdAlias(t *testing.T) {
	apps := []testApp{
		{Name: "Test", Package: "com.test"},
	}

	var buf bytes.Buffer
	rendered, err := RenderRegistered(&buf, apps, "md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rendered {
		t.Fatal("expected 'md' format alias to work")
	}

	out := buf.String()
	if !strings.Contains(out, "Test") {
		t.Errorf("md output should contain 'Test', got: %q", out)
	}
}
