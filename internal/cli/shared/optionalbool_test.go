package shared

import (
	"flag"
	"testing"
)

func TestOptionalBool_UnsetByDefault(t *testing.T) {
	var ob OptionalBool
	if ob.IsSet() {
		t.Error("IsSet() = true, want false for zero value")
	}
	if ob.Value() {
		t.Error("Value() = true, want false for zero value")
	}
}

func TestOptionalBool_SetTrue(t *testing.T) {
	for _, input := range []string{"true", "1", "yes", "TRUE", "Yes", "YES", "True"} {
		t.Run(input, func(t *testing.T) {
			var ob OptionalBool
			if err := ob.Set(input); err != nil {
				t.Fatalf("Set(%q) returned error: %v", input, err)
			}
			if !ob.IsSet() {
				t.Error("IsSet() = false after Set()")
			}
			if !ob.Value() {
				t.Errorf("Value() = false after Set(%q), want true", input)
			}
		})
	}
}

func TestOptionalBool_SetFalse(t *testing.T) {
	for _, input := range []string{"false", "0", "no", "FALSE", "No", "NO", "False"} {
		t.Run(input, func(t *testing.T) {
			var ob OptionalBool
			if err := ob.Set(input); err != nil {
				t.Fatalf("Set(%q) returned error: %v", input, err)
			}
			if !ob.IsSet() {
				t.Error("IsSet() = false after Set()")
			}
			if ob.Value() {
				t.Errorf("Value() = true after Set(%q), want false", input)
			}
		})
	}
}

func TestOptionalBool_InvalidInput(t *testing.T) {
	for _, input := range []string{"maybe", "2", "", "yep", "nope", "on", "off"} {
		t.Run(input, func(t *testing.T) {
			var ob OptionalBool
			err := ob.Set(input)
			if err == nil {
				t.Fatalf("Set(%q) returned nil error, want error", input)
			}
			if ob.IsSet() {
				t.Errorf("IsSet() = true after invalid Set(%q)", input)
			}
		})
	}
}

func TestOptionalBool_StringUnset(t *testing.T) {
	var ob OptionalBool
	if got := ob.String(); got != "" {
		t.Errorf("String() = %q, want empty string for unset", got)
	}
}

func TestOptionalBool_StringTrue(t *testing.T) {
	var ob OptionalBool
	ob.Set("true")
	if got := ob.String(); got != "true" {
		t.Errorf("String() = %q, want %q", got, "true")
	}
}

func TestOptionalBool_StringFalse(t *testing.T) {
	var ob OptionalBool
	ob.Set("false")
	if got := ob.String(); got != "false" {
		t.Errorf("String() = %q, want %q", got, "false")
	}
}

func TestOptionalBool_IsBoolFlag(t *testing.T) {
	var ob OptionalBool
	if !ob.IsBoolFlag() {
		t.Error("IsBoolFlag() = false, want true")
	}
}

func TestOptionalBool_FlagSetNoValue(t *testing.T) {
	var ob OptionalBool
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.Var(&ob, "myflag", "test flag")

	if err := fs.Parse([]string{"--myflag"}); err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if !ob.IsSet() {
		t.Error("IsSet() = false after --myflag (no value)")
	}
	if !ob.Value() {
		t.Error("Value() = false after --myflag (no value), want true")
	}
}

func TestOptionalBool_FlagSetExplicitFalse(t *testing.T) {
	var ob OptionalBool
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.Var(&ob, "myflag", "test flag")

	if err := fs.Parse([]string{"--myflag=false"}); err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if !ob.IsSet() {
		t.Error("IsSet() = false after --myflag=false")
	}
	if ob.Value() {
		t.Error("Value() = true after --myflag=false, want false")
	}
}

func TestOptionalBool_FlagSetUnset(t *testing.T) {
	var ob OptionalBool
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.Var(&ob, "myflag", "test flag")

	if err := fs.Parse([]string{}); err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if ob.IsSet() {
		t.Error("IsSet() = true when flag not provided")
	}
	if ob.Value() {
		t.Error("Value() = true when flag not provided")
	}
}

func TestOptionalBool_WhitespaceHandling(t *testing.T) {
	var ob OptionalBool
	if err := ob.Set("  true  "); err != nil {
		t.Fatalf("Set with whitespace returned error: %v", err)
	}
	if !ob.Value() {
		t.Error("Value() = false after Set(\"  true  \"), want true")
	}
}
