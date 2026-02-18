package shared

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// brailleFrames are the spinner animation frames.
var brailleFrames = []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}

const (
	spinnerTickRate = 120 * time.Millisecond
	spinnerEnvVar   = "GPLAY_SPINNER_DISABLED"
)

// spinner displays a braille animation on stderr during long-running operations.
type spinner struct {
	label  string
	writer io.Writer // nil disables output
	tick   time.Duration
	delay  time.Duration // if > 0, wait this long before showing spinner
	mu sync.Mutex
}

// newSpinner creates a spinner. If the env var GPLAY_SPINNER_DISABLED is set
// or GPLAY_DEBUG is set (debug/retry logs would collide), the writer is forced
// to nil (no output).
func newSpinner(label string, w io.Writer, delay time.Duration) *spinner {
	if os.Getenv(spinnerEnvVar) != "" || os.Getenv("GPLAY_DEBUG") != "" {
		w = nil
	}
	return &spinner{
		label:  label,
		writer: w,
		tick:   spinnerTickRate,
		delay:  delay,
	}
}

// stderrWriter returns os.Stderr if it is a TTY, nil otherwise.
func stderrWriter() io.Writer {
	if info, err := os.Stderr.Stat(); err == nil && info.Mode()&os.ModeCharDevice != 0 {
		return os.Stderr
	}
	return nil
}

// WithSpinner wraps fn with a braille spinner on stderr.
// If fn is nil, it returns nil immediately.
// If fn returns an error, WithSpinner returns it.
// If fn panics, the spinner stops and the panic is re-raised.
func WithSpinner(label string, fn func() error) error {
	if fn == nil {
		return nil
	}
	s := newSpinner(label, stderrWriter(), 0)
	return s.run(fn)
}

// WithSpinnerDelayed wraps fn with a spinner that only appears after delay.
// Useful for operations that may complete quickly — no visual noise for fast calls.
func WithSpinnerDelayed(label string, delay time.Duration, fn func() error) error {
	if fn == nil {
		return nil
	}
	s := newSpinner(label, stderrWriter(), delay)
	return s.run(fn)
}

// run executes fn while displaying the spinner animation.
func (s *spinner) run(fn func() error) error {
	if fn == nil {
		return nil
	}

	done := make(chan struct{})
	var fnErr error
	var panicked bool
	var panicVal interface{}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				panicVal = r
			}
			close(done)
		}()
		fnErr = fn()
	}()

	s.animate(done)

	if panicked {
		panic(panicVal)
	}
	return fnErr
}

// animate runs the ticker loop until done is closed.
func (s *spinner) animate(done <-chan struct{}) {
	if s.writer == nil {
		<-done
		return
	}

	frame := 0

	if s.delay > 0 {
		// Wait for delay or completion, whichever comes first.
		select {
		case <-done:
			return
		case <-time.After(s.delay):
			// Delay elapsed; start animating.
		}
	}

	ticker := time.NewTicker(s.tick)
	defer ticker.Stop()

	// Print the first frame immediately.
	s.printFrame(frame)
	frame++

	for {
		select {
		case <-done:
			s.clearLine()
			return
		case <-ticker.C:
			s.printFrame(frame % len(brailleFrames))
			frame++
		}
	}
}

// printFrame writes a single spinner frame to the writer.
func (s *spinner) printFrame(idx int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.writer == nil {
		return
	}
	fmt.Fprintf(s.writer, "\r%s %s", brailleFrames[idx], s.label)
}

// clearLine overwrites the spinner line with spaces and returns the cursor.
func (s *spinner) clearLine() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.writer == nil {
		return
	}
	// Overwrite with spaces: frame char (up to 3 bytes rendered as 1 cell) + space + label.
	width := 2 + len(s.label)
	fmt.Fprintf(s.writer, "\r%s\r", strings.Repeat(" ", width))
}
