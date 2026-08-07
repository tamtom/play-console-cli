//go:build !darwin

package websession

import "os"

// terminateChrome kills the process. Interactive launches only happen on
// macOS; this fallback exists so tests and cross-builds work elsewhere.
func terminateChrome(p *os.Process) error {
	return p.Kill()
}
