//go:build windows

package rootfs

import "os"

func syncDirectory(_ *os.Root, _ string) error { return nil }
