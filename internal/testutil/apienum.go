package testutil

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/apischema"
)

type enumSchema struct {
	Enum           []string `json:"enum"`
	EnumDeprecated []bool   `json:"enumDeprecated"`
}

type enumProperty struct {
	enumSchema
	Items *enumSchema `json:"items"`
}

// OfficialEnum returns the values of an enum property in the embedded official
// API schema. The first list has the values that are in use. The second list
// has the deprecated values. The "_UNSPECIFIED" value is not in either list.
// The property can be an enum or an array of enums.
func OfficialEnum(t *testing.T, api, typeName, property string) (current, deprecated []string) {
	t.Helper()
	index, err := apischema.Load()
	if err != nil {
		t.Fatalf("load API schema index: %v", err)
	}
	types, err := index.FindTypes(apischema.Filter{API: api, Query: typeName})
	if err != nil {
		t.Fatalf("find type %s.%s: %v", api, typeName, err)
	}
	var definition *struct {
		Properties map[string]enumProperty `json:"properties"`
	}
	for _, item := range types {
		if item.Name != typeName {
			continue
		}
		if err := json.Unmarshal(item.Definition, &definition); err != nil {
			t.Fatalf("decode type %s.%s: %v", api, typeName, err)
		}
	}
	if definition == nil {
		t.Fatalf("type %s.%s is not in the API schema index", api, typeName)
	}
	prop, ok := definition.Properties[property]
	if !ok {
		t.Fatalf("property %s.%s.%s is not in the API schema index", api, typeName, property)
	}
	values := prop.enumSchema
	if prop.Items != nil {
		values = *prop.Items
	}
	if len(values.Enum) == 0 {
		t.Fatalf("property %s.%s.%s has no enum values", api, typeName, property)
	}
	for i, value := range values.Enum {
		if strings.HasSuffix(value, "_UNSPECIFIED") {
			continue
		}
		if i < len(values.EnumDeprecated) && values.EnumDeprecated[i] {
			deprecated = append(deprecated, value)
			continue
		}
		current = append(current, value)
	}
	return current, deprecated
}

// AssertHelpListsEnum fails the test when help does not name each current
// value, or when help names a deprecated value.
func AssertHelpListsEnum(t *testing.T, help string, current, deprecated []string) {
	t.Helper()
	for _, value := range current {
		if !containsWord(help, value) {
			t.Errorf("help does not list the official value %s", value)
		}
	}
	for _, value := range deprecated {
		if containsWord(help, value) {
			t.Errorf("help lists the deprecated value %s", value)
		}
	}
}

func containsWord(text, word string) bool {
	return regexp.MustCompile(`\b` + regexp.QuoteMeta(word) + `\b`).MatchString(text)
}
