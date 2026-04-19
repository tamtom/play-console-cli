//go:build windows

package doctor

import (
	"syscall"
	"unsafe"
)

// diskFree returns free bytes on the filesystem containing path using the
// Win32 GetDiskFreeSpaceExW API.
func diskFree(path string) (uint64, error) {
	kernel32, err := syscall.LoadLibrary("kernel32.dll")
	if err != nil {
		return 0, err
	}
	defer syscall.FreeLibrary(kernel32)

	proc, err := syscall.GetProcAddress(kernel32, "GetDiskFreeSpaceExW")
	if err != nil {
		return 0, err
	}

	ptr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}

	var freeBytesAvailable, totalBytes, totalFreeBytes uint64
	r1, _, callErr := syscall.SyscallN(uintptr(proc),
		uintptr(unsafe.Pointer(ptr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)
	if r1 == 0 {
		if callErr != 0 {
			return 0, callErr
		}
		return 0, syscall.EINVAL
	}
	return freeBytesAvailable, nil
}
