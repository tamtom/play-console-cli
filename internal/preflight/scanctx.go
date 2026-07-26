package preflight

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"sync"
)

// Bundle format identifiers.
const (
	formatAAB     = "aab"
	formatAPK     = "apk"
	formatUnknown = "unknown"
)

// scanContext carries everything the scanners need for one bundle. It is
// built once and shared, so expensive work (dex scanning in particular)
// happens at most once per run.
type scanContext struct {
	path   string
	format string
	opts   Options

	files   []*zip.File
	entries map[string]*zip.File

	totalCompressed   int64
	totalUncompressed int64

	// manifestRaw is the undecoded AndroidManifest.xml payload.
	manifestRaw []byte
	// manifest is nil when the payload could not be decoded.
	manifest *Manifest
	// manifestErr records why decoding failed.
	manifestErr error

	sdkOnce sync.Once
	sdkHits map[string]bool
}

// newScanContext indexes the archive and decodes the manifest.
func newScanContext(path string, r *zip.ReadCloser, opts Options) *scanContext {
	ctx := &scanContext{
		path:    path,
		opts:    opts,
		files:   r.File,
		entries: make(map[string]*zip.File, len(r.File)),
	}
	for _, f := range r.File {
		ctx.entries[f.Name] = f
		ctx.totalCompressed += int64(f.CompressedSize64)     // #nosec G115
		ctx.totalUncompressed += int64(f.UncompressedSize64) // #nosec G115
	}

	ctx.format = detectFormat(ctx.entries)
	ctx.manifestRaw = ctx.manifestBytes()
	if len(ctx.manifestRaw) > 0 {
		m, err := parseManifestBytes(ctx.manifestRaw)
		if err != nil {
			ctx.manifestErr = err
		} else {
			ctx.manifest = m
		}
	}
	return ctx
}

// detectFormat classifies the archive as an App Bundle or an APK.
func detectFormat(entries map[string]*zip.File) string {
	if _, ok := entries["BundleConfig.pb"]; ok {
		return formatAAB
	}
	if _, ok := entries["base/manifest/AndroidManifest.xml"]; ok {
		return formatAAB
	}
	if _, ok := entries["AndroidManifest.xml"]; ok {
		return formatAPK
	}
	return formatUnknown
}

// manifestBytes returns the raw manifest payload for either format.
func (c *scanContext) manifestBytes() []byte {
	if b := c.read(c.entries["base/manifest/AndroidManifest.xml"], c.maxEntryBytes()); len(b) > 0 {
		return b
	}
	return c.read(c.entries["AndroidManifest.xml"], c.maxEntryBytes())
}

func (c *scanContext) maxEntryBytes() int64 {
	if c.opts.MaxScanBytesPerEntry > 0 {
		return c.opts.MaxScanBytesPerEntry
	}
	return 5 * 1024 * 1024
}

func (c *scanContext) maxDexScanBytes() int64 {
	if c.opts.MaxDexScanBytes > 0 {
		return c.opts.MaxDexScanBytes
	}
	return 96 * 1024 * 1024
}

// read returns up to maxBytes of an entry's contents, or nil.
func (c *scanContext) read(f *zip.File, maxBytes int64) []byte {
	if f == nil {
		return nil
	}
	rc, err := f.Open()
	if err != nil {
		return nil
	}
	defer func() { _ = rc.Close() }()
	data, err := io.ReadAll(io.LimitReader(rc, maxBytes))
	if err != nil {
		return nil
	}
	return data
}

// hasEntry reports whether an exact entry name exists.
func (c *scanContext) hasEntry(name string) bool {
	_, ok := c.entries[name]
	return ok
}

// hasEntrySuffix reports whether any entry name ends with suffix.
func (c *scanContext) hasEntrySuffix(suffix string) bool {
	for name := range c.entries {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

// dexFiles returns every .dex entry in the archive.
func (c *scanContext) dexFiles() []*zip.File {
	var out []*zip.File
	for _, f := range c.files {
		if strings.HasSuffix(f.Name, ".dex") {
			out = append(out, f)
		}
	}
	return out
}

// nativeLibs returns entries under lib/<abi>/ or base/lib/<abi>/ keyed by ABI.
func (c *scanContext) nativeLibs() map[string][]*zip.File {
	out := map[string][]*zip.File{}
	for _, f := range c.files {
		abi, ok := nativeLibABI(f.Name)
		if !ok {
			continue
		}
		out[abi] = append(out[abi], f)
	}
	return out
}

// nativeLibABI extracts the ABI directory from a native library entry name.
func nativeLibABI(name string) (string, bool) {
	var prefix string
	switch {
	case strings.HasPrefix(name, "lib/"):
		prefix = "lib/"
	case strings.HasPrefix(name, "base/lib/"):
		prefix = "base/lib/"
	default:
		// Dynamic feature modules use <module>/lib/<abi>/.
		idx := strings.Index(name, "/lib/")
		if idx < 0 {
			return "", false
		}
		prefix = name[:idx+len("/lib/")]
	}
	rest := name[len(prefix):]
	slash := strings.Index(rest, "/")
	if slash <= 0 {
		return "", false
	}
	return rest[:slash], true
}

// sdkIndex scans dex bytecode and entry paths once for every known SDK
// signature, returning the set of matched signature IDs.
func (c *scanContext) sdkIndex() map[string]bool {
	c.sdkOnce.Do(func() {
		c.sdkHits = map[string]bool{}

		// Path-based signals are cheap; check them first.
		for _, sig := range sdkSignatures {
			for _, frag := range sig.Paths {
				if c.anyEntryContains(frag) {
					c.sdkHits[sig.ID] = true
					break
				}
			}
		}

		// Collect the dex markers still worth searching for.
		var needles [][]byte
		var owners []string
		for _, sig := range sdkSignatures {
			if c.sdkHits[sig.ID] {
				continue
			}
			for _, m := range sig.Markers {
				needles = append(needles, []byte(m))
				owners = append(owners, sig.ID)
			}
		}
		if len(needles) == 0 {
			return
		}

		for _, f := range c.dexFiles() {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			hits := searchStream(rc, needles, c.maxDexScanBytes())
			_ = rc.Close()
			for i := range hits {
				c.sdkHits[owners[i]] = true
			}
		}
	})
	return c.sdkHits
}

// anyEntryContains reports whether any entry name contains the fragment.
func (c *scanContext) anyEntryContains(frag string) bool {
	for name := range c.entries {
		if strings.Contains(name, frag) {
			return true
		}
	}
	return false
}

// hasSDK reports whether a signature ID matched.
func (c *scanContext) hasSDK(id string) bool { return c.sdkIndex()[id] }

// sdksInCategory returns the display names of every matched SDK in a category.
func (c *scanContext) sdksInCategory(category string) []string {
	idx := c.sdkIndex()
	var out []string
	for _, sig := range sdkSignatures {
		if sig.Category == category && idx[sig.ID] {
			out = append(out, sig.Name)
		}
	}
	return out
}

// searchStream scans a reader for any of the needles, returning the set of
// matched needle indices. It reads in bounded chunks with an overlap so
// matches spanning a chunk boundary are still found, keeping memory flat
// regardless of how large the entry is.
func searchStream(r io.Reader, needles [][]byte, limit int64) map[int]bool {
	found := map[int]bool{}
	if len(needles) == 0 {
		return found
	}

	maxNeedle := 0
	for _, n := range needles {
		if len(n) > maxNeedle {
			maxNeedle = len(n)
		}
	}
	if maxNeedle == 0 {
		return found
	}

	const chunkSize = 1 << 20
	overlap := maxNeedle - 1
	buf := make([]byte, 0, chunkSize+overlap)
	tmp := make([]byte, chunkSize)

	var read int64
	for read < limit {
		toRead := int64(chunkSize)
		if remaining := limit - read; remaining < toRead {
			toRead = remaining
		}
		n, err := r.Read(tmp[:toRead])
		if n > 0 {
			read += int64(n)
			buf = append(buf, tmp[:n]...)
			for i, needle := range needles {
				if found[i] {
					continue
				}
				if bytes.Contains(buf, needle) {
					found[i] = true
				}
			}
			if len(found) == len(needles) {
				return found
			}
			// Keep only the tail so a match can span the next boundary.
			if len(buf) > overlap {
				keep := buf[len(buf)-overlap:]
				buf = append(buf[:0], keep...)
			}
		}
		if err != nil {
			break
		}
	}
	return found
}
