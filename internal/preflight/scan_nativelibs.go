package preflight

import (
	"archive/zip"
	"bytes"
	"debug/elf"
	"fmt"
	"sort"
	"strings"
)

// requiredPageSize is the memory page size Android 15+ devices may use.
// Shared libraries must be built with segments aligned to at least this
// boundary or they cannot be loaded on 16 KB page-size devices.
const requiredPageSize = 16384

// maxELFBytes caps how large a shared library may be before it is skipped for
// ELF inspection, keeping memory use bounded.
const maxELFBytes = 64 * 1024 * 1024

// sixteenKBTargetSDK is the target SDK from which Play requires 16 KB page
// size support for apps shipping native code.
const sixteenKBTargetSDK = 35

// abi64Bit lists the 64-bit ABIs Play cares about for page-size compliance.
var abi64Bit = map[string]bool{
	"arm64-v8a": true,
	"x86_64":    true,
	"riscv64":   true,
}

// scanNativeLibs checks 64-bit coverage, ABI hygiene, page-size alignment,
// and debug-symbol bloat in shipped shared libraries.
func scanNativeLibs(c *scanContext) []Finding {
	libs := c.nativeLibs()
	if len(libs) == 0 {
		return nil
	}

	var out []Finding
	out = append(out, checkABICoverage(libs)...)
	out = append(out, checkPageAlignment(c, libs)...)
	out = append(out, checkStrippedSymbols(c, libs)...)
	out = append(out, checkExtractNativeLibs(c)...)
	return out
}

// checkABICoverage verifies the shipped ABI set meets Play requirements.
func checkABICoverage(libs map[string][]*zip.File) []Finding {
	var out []Finding

	abis := make([]string, 0, len(libs))
	for abi := range libs {
		abis = append(abis, abi)
	}
	sort.Strings(abis)

	if libs["arm64-v8a"] == nil {
		out = append(out, Finding{
			Check:    "native_libs",
			Severity: SeverityError,
			Message:  fmt.Sprintf("native libraries present (%s) but arm64-v8a is missing", strings.Join(abis, ", ")),
			Hint:     "Play requires 64-bit support; add arm64-v8a to your ABI splits",
			Ref:      "https://developer.android.com/google/play/requirements/64-bit",
		})
	}
	if libs["x86"] != nil && libs["x86_64"] == nil {
		out = append(out, Finding{
			Check:    "native_abi",
			Severity: SeverityInfo,
			Message:  "x86 is shipped without x86_64",
			Hint:     "64-bit x86 emulators and Chromebooks cannot use the 32-bit variant",
		})
	}
	if libs["armeabi"] != nil {
		out = append(out, Finding{
			Check:    "native_abi",
			Severity: SeverityWarning,
			Message:  "armeabi (ARMv5) libraries are shipped",
			Hint:     "the ABI has been unsupported since NDK r17; drop it to save download size",
		})
	}
	return out
}

// checkPageAlignment verifies 64-bit shared libraries are built for 16 KB
// memory pages, which Android 15 devices may use.
func checkPageAlignment(c *scanContext, libs map[string][]*zip.File) []Finding {
	var misaligned []string
	var skipped int

	for abi, files := range libs {
		if !abi64Bit[abi] {
			continue
		}
		for _, f := range files {
			if !strings.HasSuffix(f.Name, ".so") {
				continue
			}
			if int64(f.UncompressedSize64) > maxELFBytes { // #nosec G115
				skipped++
				continue
			}
			data := c.read(f, maxELFBytes)
			ok, err := isPageAligned(data, requiredPageSize)
			if err != nil {
				skipped++
				continue
			}
			if !ok {
				misaligned = append(misaligned, f.Name)
			}
		}
	}

	var out []Finding
	if len(misaligned) > 0 {
		sort.Strings(misaligned)
		sev := SeverityWarning
		hint := "rebuild with NDK r28+, or pass -Wl,-z,max-page-size=16384 to the linker"
		if c.manifest != nil && c.manifest.TargetSdk >= sixteenKBTargetSDK {
			sev = SeverityError
			hint = "required for apps targeting Android 15+; rebuild with NDK r28+ or -Wl,-z,max-page-size=16384"
		}
		for _, name := range misaligned {
			out = append(out, Finding{
				Check:    "page_alignment",
				Severity: sev,
				Message:  "shared library is not aligned for 16 KB memory pages",
				Entry:    name,
				Hint:     hint,
				Ref:      "https://developer.android.com/guide/practices/page-sizes",
			})
		}
	}
	if skipped > 0 {
		out = append(out, Finding{
			Check:    "page_alignment",
			Severity: SeverityInfo,
			Message:  fmt.Sprintf("%d shared librar(y/ies) could not be inspected for page alignment", skipped),
			Hint:     "the file was too large to read or is not a valid ELF object",
		})
	}
	return out
}

// isPageAligned reports whether every loadable ELF segment is aligned to at
// least pageSize.
func isPageAligned(data []byte, pageSize uint64) (bool, error) {
	if len(data) == 0 {
		return false, fmt.Errorf("empty file")
	}
	f, err := elf.NewFile(bytes.NewReader(data))
	if err != nil {
		return false, err
	}
	defer func() { _ = f.Close() }()

	// 32-bit objects never run on 16 KB page devices, so the rule is moot.
	if f.Class == elf.ELFCLASS32 {
		return true, nil
	}

	sawLoad := false
	for _, p := range f.Progs {
		if p.Type != elf.PT_LOAD {
			continue
		}
		sawLoad = true
		if p.Align < pageSize {
			return false, nil
		}
	}
	if !sawLoad {
		return false, fmt.Errorf("no PT_LOAD segments")
	}
	return true, nil
}

// checkStrippedSymbols flags libraries still carrying debug information.
func checkStrippedSymbols(c *scanContext, libs map[string][]*zip.File) []Finding {
	var out []Finding
	for _, files := range libs {
		for _, f := range files {
			if !strings.HasSuffix(f.Name, ".so") {
				continue
			}
			if int64(f.UncompressedSize64) > maxELFBytes { // #nosec G115
				continue
			}
			data := c.read(f, maxELFBytes)
			if !hasDebugSections(data) {
				continue
			}
			out = append(out, Finding{
				Check:    "unstripped_library",
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("shared library ships debug symbols (%d bytes uncompressed)", f.UncompressedSize64),
				Entry:    f.Name,
				Hint:     "run the NDK strip tool, or upload a native debug symbols file separately instead",
			})
		}
	}
	return out
}

// hasDebugSections reports whether an ELF object retains debug sections.
func hasDebugSections(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	f, err := elf.NewFile(bytes.NewReader(data))
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	for _, s := range f.Sections {
		if strings.HasPrefix(s.Name, ".debug_") || s.Name == ".symtab" {
			return true
		}
	}
	return false
}

// checkExtractNativeLibs flags the legacy extraction mode, which doubles the
// on-device footprint of native code.
func checkExtractNativeLibs(c *scanContext) []Finding {
	if c.manifest == nil || !isTrue(c.manifest.Application.ExtractNativeLibs) {
		return nil
	}
	return []Finding{{
		Check:    "extract_native_libs",
		Severity: SeverityWarning,
		Message:  "android:extractNativeLibs is true",
		Hint:     "set it false so libraries are loaded from the APK; this cuts install size and speeds updates",
		Ref:      "https://developer.android.com/topic/performance/reduce-apk-size",
	}}
}
