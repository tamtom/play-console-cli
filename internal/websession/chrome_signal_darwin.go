//go:build darwin

package websession

import (
	"os"
	"syscall"
)

// terminateChrome asks Chrome to quit gracefully, so the next launch does not
// show the "didn't shut down correctly" restore prompt.
func terminateChrome(p *os.Process) error {
	return p.Signal(syscall.SIGTERM)
}
