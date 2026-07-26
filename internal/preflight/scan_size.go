package preflight

import (
	"archive/zip"
	"fmt"
	"sort"
	"strings"
)

// Play download-size budget. The hard limit applies to the generated APK set
// a device downloads; the soft limit is where it is worth intervening.
const (
	defaultMaxBundleBytes  = 200 * 1024 * 1024
	softBundleWarningBytes = 150 * 1024 * 1024
	defaultMaxDexBytes     = 64 * 1024 * 1024
	maxDexFiles            = 20
	topEntriesReported     = 5
)

// scanSize reports download-size pressure and payload bloat.
func scanSize(c *scanContext) []Finding {
	limit := c.opts.MaxBundleBytes
	if limit <= 0 {
		limit = defaultMaxBundleBytes
	}
	maxDex := c.opts.MaxDexBytes
	if maxDex <= 0 {
		maxDex = defaultMaxDexBytes
	}

	var out []Finding
	out = append(out, checkBundleSize(c, limit)...)
	out = append(out, checkDexBudget(c, maxDex)...)
	out = append(out, checkPayloadBreakdown(c, limit)...)
	return out
}

// checkBundleSize compares the archive against the Play download budget.
func checkBundleSize(c *scanContext, limit int64) []Finding {
	total := c.totalCompressed

	// The number Play enforces is the size of the split APKs it generates, so
	// treat the archive size as an approximation and say so.
	approx := "this is the archive's compressed size, an approximation of the generated download"

	switch {
	case total > limit:
		return []Finding{{
			Check:    "bundle_size",
			Severity: SeverityError,
			Message:  fmt.Sprintf("compressed size %s exceeds the %s limit", humanBytes(total), humanBytes(limit)),
			Hint:     "move large assets to Play Asset Delivery or split them into dynamic feature modules; " + approx,
			Ref:      "https://developer.android.com/guide/playcore/asset-delivery",
		}}
	case total > softBundleWarningBytes && limit > softBundleWarningBytes:
		return []Finding{{
			Check:    "bundle_size",
			Severity: SeverityWarning,
			Message:  fmt.Sprintf("compressed size %s is approaching the %s limit", humanBytes(total), humanBytes(limit)),
			Hint:     "install rates fall as download size grows; " + approx,
		}}
	}
	return nil
}

// checkDexBudget reports dex fragmentation and oversized dex files.
func checkDexBudget(c *scanContext, maxDex int64) []Finding {
	dexes := c.dexFiles()
	if len(dexes) == 0 {
		return nil
	}

	var out []Finding
	for _, f := range dexes {
		if int64(f.UncompressedSize64) > maxDex { // #nosec G115
			out = append(out, Finding{
				Check:    "dex",
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("%s is %s uncompressed, over the %s budget", f.Name, humanBytes(int64(f.UncompressedSize64)), humanBytes(maxDex)), // #nosec G115
				Entry:    f.Name,
			})
		}
	}
	if len(dexes) > maxDexFiles {
		out = append(out, Finding{
			Check:    "dex",
			Severity: SeverityWarning,
			Message:  fmt.Sprintf("%d dex files in the build", len(dexes)),
			Hint:     "enable R8 full mode and check for unused dependencies; heavy multidex slows cold start on older devices",
		})
	}
	return out
}

// checkPayloadBreakdown summarizes where the bytes are when the build is
// large enough for it to matter.
func checkPayloadBreakdown(c *scanContext, limit int64) []Finding {
	if c.totalCompressed <= softBundleWarningBytes && c.totalCompressed <= limit/2 {
		return nil
	}

	buckets := map[string]int64{}
	for _, f := range c.files {
		buckets[sizeBucket(f.Name)] += int64(f.CompressedSize64) // #nosec G115
	}

	type kv struct {
		name string
		size int64
	}
	var ordered []kv
	for name, size := range buckets {
		ordered = append(ordered, kv{name, size})
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].size > ordered[j].size })

	var parts []string
	for _, b := range ordered {
		parts = append(parts, fmt.Sprintf("%s %s", b.name, humanBytes(b.size)))
	}

	out := []Finding{{
		Check:    "size_breakdown",
		Severity: SeverityInfo,
		Message:  fmt.Sprintf("compressed size by category: %s", strings.Join(parts, ", ")),
	}}

	if largest := largestEntries(c.files, topEntriesReported); len(largest) > 0 {
		var names []string
		for _, f := range largest {
			names = append(names, fmt.Sprintf("%s (%s)", f.Name, humanBytes(int64(f.CompressedSize64)))) // #nosec G115
		}
		out = append(out, Finding{
			Check:    "largest_entries",
			Severity: SeverityInfo,
			Message:  "largest entries: " + strings.Join(names, ", "),
			Hint:     "compress or lazily download these assets to cut install size",
		})
	}
	return out
}

// largestEntries returns the n biggest entries by compressed size.
func largestEntries(files []*zip.File, n int) []*zip.File {
	sorted := make([]*zip.File, len(files))
	copy(sorted, files)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].CompressedSize64 > sorted[j].CompressedSize64
	})
	if len(sorted) > n {
		sorted = sorted[:n]
	}
	return sorted
}

// sizeBucket classifies an entry into a reporting category.
func sizeBucket(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".dex"):
		return "dex"
	case strings.HasSuffix(lower, ".so"):
		return "native"
	case strings.Contains(lower, "/res/") || strings.HasPrefix(lower, "res/"):
		return "resources"
	case strings.Contains(lower, "/assets/") || strings.HasPrefix(lower, "assets/"):
		return "assets"
	case strings.HasSuffix(lower, "resources.arsc") || strings.HasSuffix(lower, "resources.pb"):
		return "resource-table"
	case strings.HasPrefix(lower, "meta-inf/"):
		return "meta-inf"
	default:
		return "other"
	}
}

// humanBytes renders a byte count using binary units.
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit && exp < 3; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGT"[exp])
}
