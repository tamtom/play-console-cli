package shared

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestWithSpinner_NilFn(t *testing.T) {
	err := WithSpinner("loading", nil)
	if err != nil {
		t.Errorf("WithSpinner(nil) = %v, want nil", err)
	}
}

func TestWithSpinner_ErrorPropagation(t *testing.T) {
	want := errors.New("api failure")
	got := WithSpinner("loading", func() error {
		return want
	})
	if !errors.Is(got, want) {
		t.Errorf("WithSpinner error = %v, want %v", got, want)
	}
}

func TestWithSpinner_PanicReRaise(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic to be re-raised")
		}
		if r != "boom" {
			t.Errorf("recovered %v, want %q", r, "boom")
		}
	}()

	WithSpinner("loading", func() error {
		panic("boom")
	})
}

func TestWithSpinner_SuccessReturnsNil(t *testing.T) {
	err := WithSpinner("loading", func() error {
		return nil
	})
	if err != nil {
		t.Errorf("WithSpinner = %v, want nil", err)
	}
}

func TestSpinner_WritesToWriter(t *testing.T) {
	var buf bytes.Buffer
	s := &spinner{
		label:  "fetching",
		writer: &buf,
		tick:   20 * time.Millisecond,
	}
	err := s.run(func() error {
		time.Sleep(60 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Fatalf("run() = %v, want nil", err)
	}
	output := buf.String()
	if output == "" {
		t.Error("expected spinner output, got empty string")
	}
	if !strings.Contains(output, "fetching") {
		t.Errorf("output should contain label, got: %q", output)
	}
}

func TestSpinner_BrailleFrames(t *testing.T) {
	var buf bytes.Buffer
	s := &spinner{
		label:  "test",
		writer: &buf,
		tick:   10 * time.Millisecond,
	}
	err := s.run(func() error {
		time.Sleep(30 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Fatalf("run() = %v, want nil", err)
	}
	output := buf.String()
	// At least one braille frame should appear.
	frames := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	found := false
	for _, f := range frames {
		if strings.Contains(output, f) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("output should contain a braille frame, got: %q", output)
	}
}

func TestSpinner_NilWriterNoOutput(t *testing.T) {
	s := &spinner{
		label:  "test",
		writer: nil,
		tick:   10 * time.Millisecond,
	}
	err := s.run(func() error {
		time.Sleep(30 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Fatalf("run() = %v, want nil", err)
	}
	// No panic from nil writer.
}

func TestSpinner_EnvVarDisabled(t *testing.T) {
	t.Setenv("GPLAY_SPINNER_DISABLED", "1")

	var buf bytes.Buffer
	s := newSpinner("loading", &buf, 0)
	if s.writer != nil {
		t.Error("spinner writer should be nil when GPLAY_SPINNER_DISABLED=1")
	}
}

func TestSpinner_EnvVarNotSet(t *testing.T) {
	os.Unsetenv("GPLAY_SPINNER_DISABLED")

	var buf bytes.Buffer
	s := newSpinner("loading", &buf, 0)
	if s.writer == nil {
		t.Error("spinner writer should not be nil when env var is unset")
	}
}

func TestSpinner_ErrorPropagation(t *testing.T) {
	want := errors.New("something broke")
	s := &spinner{
		label:  "test",
		writer: nil,
		tick:   10 * time.Millisecond,
	}
	got := s.run(func() error {
		return want
	})
	if !errors.Is(got, want) {
		t.Errorf("run() = %v, want %v", got, want)
	}
}

func TestSpinner_PanicReRaise(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic to be re-raised")
		}
		if r != "kaboom" {
			t.Errorf("recovered %v, want %q", r, "kaboom")
		}
	}()

	s := &spinner{
		label:  "test",
		writer: nil,
		tick:   10 * time.Millisecond,
	}
	s.run(func() error {
		panic("kaboom")
	})
}

func TestSpinner_ClearsLineOnFinish(t *testing.T) {
	var buf bytes.Buffer
	s := &spinner{
		label:  "loading",
		writer: &buf,
		tick:   10 * time.Millisecond,
	}
	err := s.run(func() error {
		time.Sleep(30 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Fatalf("run() = %v, want nil", err)
	}
	output := buf.String()
	// Should end with a clear-line sequence (\r + spaces + \r).
	if !strings.HasSuffix(output, "\r") {
		t.Errorf("output should end with \\r to clear the line, got: %q", output)
	}
}

func TestWithSpinnerDelayed_FastOperationNoOutput(t *testing.T) {
	var buf bytes.Buffer
	s := &spinner{
		label:  "fast",
		writer: &buf,
		tick:   10 * time.Millisecond,
		delay:  200 * time.Millisecond,
	}
	err := s.run(func() error {
		// Return immediately (before delay).
		return nil
	})
	if err != nil {
		t.Fatalf("run() = %v, want nil", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output for fast operation, got: %q", buf.String())
	}
}

func TestWithSpinnerDelayed_SlowOperationShowsSpinner(t *testing.T) {
	var buf bytes.Buffer
	s := &spinner{
		label:  "slow",
		writer: &buf,
		tick:   10 * time.Millisecond,
		delay:  20 * time.Millisecond,
	}
	err := s.run(func() error {
		time.Sleep(80 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Fatalf("run() = %v, want nil", err)
	}
	output := buf.String()
	if output == "" {
		t.Error("expected spinner output for slow operation, got empty string")
	}
	if !strings.Contains(output, "slow") {
		t.Errorf("output should contain label 'slow', got: %q", output)
	}
}

func TestWithSpinnerDelayed_ErrorPropagation(t *testing.T) {
	want := errors.New("delayed error")
	got := WithSpinnerDelayed("test", 50*time.Millisecond, func() error {
		return want
	})
	if !errors.Is(got, want) {
		t.Errorf("WithSpinnerDelayed error = %v, want %v", got, want)
	}
}

func TestWithSpinnerDelayed_NilFn(t *testing.T) {
	err := WithSpinnerDelayed("test", 50*time.Millisecond, nil)
	if err != nil {
		t.Errorf("WithSpinnerDelayed(nil) = %v, want nil", err)
	}
}

func TestSpinner_NilFnDirectRun(t *testing.T) {
	s := &spinner{
		label:  "test",
		writer: nil,
		tick:   10 * time.Millisecond,
	}
	err := s.run(nil)
	if err != nil {
		t.Errorf("run(nil) = %v, want nil", err)
	}
}

func TestSpinner_ThreadSafety(t *testing.T) {
	// Run the spinner multiple times concurrently to check for races.
	// Use `go test -race` to detect issues.
	for i := 0; i < 10; i++ {
		var buf bytes.Buffer
		s := &spinner{
			label:  "concurrent",
			writer: &buf,
			tick:   5 * time.Millisecond,
		}
		err := s.run(func() error {
			time.Sleep(20 * time.Millisecond)
			return nil
		})
		if err != nil {
			t.Fatalf("run() = %v, want nil", err)
		}
	}
}
