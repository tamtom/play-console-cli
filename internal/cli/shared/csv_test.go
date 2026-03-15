package shared

import (
	"reflect"
	"testing"
)

func TestSplitCSV(t *testing.T) {
	result := SplitCSV("a, b, c")
	expected := []string{"a", "b", "c"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("SplitCSV(\"a, b, c\") = %v, want %v", result, expected)
	}
}

func TestSplitCSV_Empty(t *testing.T) {
	result := SplitCSV("")
	if result != nil {
		t.Errorf("SplitCSV(\"\") = %v, want nil", result)
	}
}

func TestSplitCSV_Whitespace(t *testing.T) {
	result := SplitCSV("  a , , b  ")
	expected := []string{"a", "b"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("SplitCSV(\"  a , , b  \") = %v, want %v", result, expected)
	}
}

func TestSplitUniqueCSV(t *testing.T) {
	result := SplitUniqueCSV("a,b,a,c")
	expected := []string{"a", "b", "c"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("SplitUniqueCSV(\"a,b,a,c\") = %v, want %v", result, expected)
	}
}
