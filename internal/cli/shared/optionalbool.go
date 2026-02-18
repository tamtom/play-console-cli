package shared

import (
	"fmt"
	"strings"
)

// OptionalBool is a tri-state boolean: unset, true, or false.
// It implements flag.Value and reports IsBoolFlag() = true so
// --flag (without value) sets it to true.
type OptionalBool struct {
	val bool
	set bool
}

// IsSet reports whether the flag was explicitly set.
func (o *OptionalBool) IsSet() bool { return o.set }

// Value returns the boolean value (only meaningful when IsSet is true).
func (o *OptionalBool) Value() bool { return o.val }

// String implements flag.Value.
func (o *OptionalBool) String() string {
	if !o.set {
		return ""
	}
	if o.val {
		return "true"
	}
	return "false"
}

// Set implements flag.Value. Accepts: true, false, 1, 0, yes, no (case-insensitive).
func (o *OptionalBool) Set(s string) error {
	o.set = true
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1", "yes":
		o.val = true
	case "false", "0", "no":
		o.val = false
	default:
		o.set = false
		return fmt.Errorf("invalid boolean value: %q (use true/false, 1/0, yes/no)", s)
	}
	return nil
}

// IsBoolFlag tells the flag package this can be used without "=value".
func (o *OptionalBool) IsBoolFlag() bool { return true }
