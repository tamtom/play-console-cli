package errfmt

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"google.golang.org/api/googleapi"
)

// fakeTimeoutErr implements net.Error with Timeout() returning true.
type fakeTimeoutErr struct{ msg string }

func (e *fakeTimeoutErr) Error() string   { return e.msg }
func (e *fakeTimeoutErr) Timeout() bool   { return true }
func (e *fakeTimeoutErr) Temporary() bool { return false }

func TestClassify_NilError(t *testing.T) {
	if got := Classify(nil); got != nil {
		t.Fatalf("Classify(nil) = %v; want nil", got)
	}
}

func TestClassify_GoogleAPI401(t *testing.T) {
	err := &googleapi.Error{Code: 401, Message: "unauthorized"}
	c := Classify(err)
	if c == nil {
		t.Fatal("Classify returned nil for 401")
	}
	if c.Category != CategoryAuth {
		t.Errorf("Category = %q; want %q", c.Category, CategoryAuth)
	}
	if !strings.Contains(c.Hint, "gplay auth login") {
		t.Errorf("Hint = %q; want it to contain 'gplay auth login'", c.Hint)
	}
}

func TestClassify_GoogleAPI403(t *testing.T) {
	err := &googleapi.Error{Code: 403, Message: "forbidden"}
	c := Classify(err)
	if c == nil {
		t.Fatal("Classify returned nil for 403")
	}
	if c.Category != CategoryPermission {
		t.Errorf("Category = %q; want %q", c.Category, CategoryPermission)
	}
	if !strings.Contains(c.Hint, "permissions") {
		t.Errorf("Hint = %q; want it to contain 'permissions'", c.Hint)
	}
}

func TestClassify_GoogleAPI404(t *testing.T) {
	err := &googleapi.Error{Code: 404, Message: "not found"}
	c := Classify(err)
	if c == nil {
		t.Fatal("Classify returned nil for 404")
	}
	if c.Category != CategoryNotFound {
		t.Errorf("Category = %q; want %q", c.Category, CategoryNotFound)
	}
	if !strings.Contains(c.Hint, "package name") {
		t.Errorf("Hint = %q; want it to contain 'package name'", c.Hint)
	}
}

func TestClassify_TimeoutError(t *testing.T) {
	err := &fakeTimeoutErr{msg: "i/o timeout"}
	c := Classify(err)
	if c == nil {
		t.Fatal("Classify returned nil for timeout error")
	}
	if c.Category != CategoryTimeout {
		t.Errorf("Category = %q; want %q", c.Category, CategoryTimeout)
	}
	if !strings.Contains(c.Hint, "GPLAY_TIMEOUT") {
		t.Errorf("Hint = %q; want it to contain 'GPLAY_TIMEOUT'", c.Hint)
	}
}

func TestClassify_OsNotExist(t *testing.T) {
	err := &os.PathError{Op: "open", Path: "/missing/key.json", Err: os.ErrNotExist}
	c := Classify(err)
	if c == nil {
		t.Fatal("Classify returned nil for os.ErrNotExist")
	}
	if c.Category != CategoryMissingAuth {
		t.Errorf("Category = %q; want %q", c.Category, CategoryMissingAuth)
	}
	if !strings.Contains(c.Hint, "gplay auth doctor") {
		t.Errorf("Hint = %q; want it to contain 'gplay auth doctor'", c.Hint)
	}
}

func TestClassify_ContextDeadlineExceeded(t *testing.T) {
	err := fmt.Errorf("request failed: context deadline exceeded")
	c := Classify(err)
	if c == nil {
		t.Fatal("Classify returned nil for context deadline exceeded")
	}
	if c.Category != CategoryTimeout {
		t.Errorf("Category = %q; want %q", c.Category, CategoryTimeout)
	}
	if !strings.Contains(c.Hint, "GPLAY_TIMEOUT") {
		t.Errorf("Hint = %q; want it to contain 'GPLAY_TIMEOUT'", c.Hint)
	}
}

func TestClassify_GenericError(t *testing.T) {
	err := fmt.Errorf("something went wrong")
	c := Classify(err)
	if c == nil {
		t.Fatal("Classify returned nil for generic error")
	}
	if c.Category != CategoryGeneric {
		t.Errorf("Category = %q; want %q", c.Category, CategoryGeneric)
	}
	if c.Hint != "" {
		t.Errorf("Hint = %q; want empty", c.Hint)
	}
}

func TestClassifiedError_Unwrap(t *testing.T) {
	orig := fmt.Errorf("original")
	c := Classify(orig)
	if !errors.Is(c, orig) {
		t.Error("Unwrap chain does not reach original error")
	}
}

func TestClassifiedError_ErrorReturnsOriginalMessage(t *testing.T) {
	orig := fmt.Errorf("original message")
	c := Classify(orig)
	if c.Error() != orig.Error() {
		t.Errorf("Error() = %q; want %q", c.Error(), orig.Error())
	}
}

func TestFormatStderr_WithHint(t *testing.T) {
	err := &googleapi.Error{Code: 401, Message: "unauthorized"}
	out := FormatStderr(err)
	if !strings.HasPrefix(out, "Error: ") {
		t.Errorf("output should start with 'Error: '; got %q", out)
	}
	if !strings.Contains(out, "\n\nHint: ") {
		t.Errorf("output should contain '\\n\\nHint: '; got %q", out)
	}
}

func TestFormatStderr_GenericNoHint(t *testing.T) {
	err := fmt.Errorf("generic failure")
	out := FormatStderr(err)
	if !strings.HasPrefix(out, "Error: ") {
		t.Errorf("output should start with 'Error: '; got %q", out)
	}
	if strings.Contains(out, "Hint:") {
		t.Errorf("output should not contain 'Hint:'; got %q", out)
	}
}

func TestFormatStderr_NilReturnsEmpty(t *testing.T) {
	out := FormatStderr(nil)
	if out != "" {
		t.Errorf("FormatStderr(nil) = %q; want empty", out)
	}
}

func TestClassify_WrappedGoogleAPIError(t *testing.T) {
	inner := &googleapi.Error{Code: 403, Message: "forbidden"}
	wrapped := fmt.Errorf("API call failed: %w", inner)
	c := Classify(wrapped)
	if c == nil {
		t.Fatal("Classify returned nil for wrapped 403")
	}
	if c.Category != CategoryPermission {
		t.Errorf("Category = %q; want %q", c.Category, CategoryPermission)
	}
}
