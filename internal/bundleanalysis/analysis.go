// Package bundleanalysis analyzes AAB and APK files by reading their ZIP
// structure and grouping entries into logical buckets (dex, resources, assets,
// native libs, per-module, manifests, etc.). It is used by `gplay bundle`.
package bundleanalysis

import (
	"archive/zip"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
)

// Bucket is a coarse category for bundle entries.
type Bucket string

const (
	BucketDex       Bucket = "dex"
	BucketResources Bucket = "resources"
	BucketAssets    Bucket = "assets"
	BucketNative    Bucket = "native"
	BucketManifest  Bucket = "manifest"
	BucketMeta      Bucket = "meta"
	BucketKotlin    Bucket = "kotlin"
	BucketOther     Bucket = "other"
)

// ModuleSummary is the size footprint of one bundle module (base, dynamic
// feature, etc.).
type ModuleSummary struct {
	Name            string  `json:"name"`
	UncompressedB   int64   `json:"uncompressed_bytes"`
	CompressedB     int64   `json:"compressed_bytes"`
	Files           int     `json:"files"`
	CompressionRate float64 `json:"compression_ratio,omitempty"`
}

// BucketSummary aggregates entries that belong to a logical bucket.
type BucketSummary struct {
	Bucket        Bucket `json:"bucket"`
	UncompressedB int64  `json:"uncompressed_bytes"`
	CompressedB   int64  `json:"compressed_bytes"`
	Files         int    `json:"files"`
}

// Analysis is the full analysis of a bundle.
type Analysis struct {
	Path             string          `json:"path"`
	TotalCompressed  int64           `json:"total_compressed_bytes"`
	TotalUncompressd int64           `json:"total_uncompressed_bytes"`
	TotalFiles       int             `json:"total_files"`
	Modules          []ModuleSummary `json:"modules,omitempty"`
	Buckets          []BucketSummary `json:"buckets"`
	// LargestFiles lists top-N individual entries by uncompressed size.
	LargestFiles []LargeFile `json:"largest_files,omitempty"`
	// Warnings collects non-fatal issues (unknown entries, broken ZIP, etc.).
	Warnings []string `json:"warnings,omitempty"`
}

// LargeFile is a single notable entry in the bundle.
type LargeFile struct {
	Name          string `json:"name"`
	Module        string `json:"module,omitempty"`
	UncompressedB int64  `json:"uncompressed_bytes"`
	CompressedB   int64  `json:"compressed_bytes"`
}

// Options controls Analyze behavior.
type Options struct {
	TopFiles int // how many largest files to capture (0 = none)
}

// Analyze inspects an AAB/APK at path and returns an Analysis.
func Analyze(zipPath string, opts Options) (*Analysis, error) {
	r, err := zip.OpenReader(zipPath) // #nosec G304 -- user-supplied bundle path
	if err != nil {
		return nil, fmt.Errorf("open zip %s: %w", zipPath, err)
	}
	defer func() { _ = r.Close() }()

	res := &Analysis{Path: zipPath}
	modules := map[string]*ModuleSummary{}
	buckets := map[Bucket]*BucketSummary{}
	var all []LargeFile

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		moduleName := moduleForEntry(f.Name)
		bucket := bucketForEntry(f.Name)

		mod, ok := modules[moduleName]
		if !ok {
			mod = &ModuleSummary{Name: moduleName}
			modules[moduleName] = mod
		}
		mod.UncompressedB += int64(f.UncompressedSize64) // #nosec G115 -- ZIP sizes fit in int64 in practice
		mod.CompressedB += int64(f.CompressedSize64)     // #nosec G115
		mod.Files++

		bs, ok := buckets[bucket]
		if !ok {
			bs = &BucketSummary{Bucket: bucket}
			buckets[bucket] = bs
		}
		bs.UncompressedB += int64(f.UncompressedSize64) // #nosec G115
		bs.CompressedB += int64(f.CompressedSize64)     // #nosec G115
		bs.Files++

		res.TotalCompressed += int64(f.CompressedSize64)    // #nosec G115
		res.TotalUncompressd += int64(f.UncompressedSize64) // #nosec G115
		res.TotalFiles++

		if opts.TopFiles > 0 {
			all = append(all, LargeFile{
				Name:          f.Name,
				Module:        moduleName,
				UncompressedB: int64(f.UncompressedSize64), // #nosec G115
				CompressedB:   int64(f.CompressedSize64),   // #nosec G115
			})
		}
	}

	res.Modules = sortedModules(modules)
	res.Buckets = sortedBuckets(buckets)

	if opts.TopFiles > 0 {
		sort.Slice(all, func(i, j int) bool { return all[i].UncompressedB > all[j].UncompressedB })
		if len(all) > opts.TopFiles {
			all = all[:opts.TopFiles]
		}
		res.LargestFiles = all
	}

	return res, nil
}

func bucketForEntry(name string) Bucket {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".dex"):
		return BucketDex
	case strings.HasPrefix(lower, "meta-inf/"):
		return BucketMeta
	case strings.HasSuffix(lower, "androidmanifest.xml"):
		return BucketManifest
	case strings.Contains(lower, "/lib/") || strings.HasPrefix(lower, "lib/"):
		return BucketNative
	case strings.Contains(lower, "/res/") || strings.HasPrefix(lower, "res/"):
		return BucketResources
	case strings.HasSuffix(lower, "resources.pb") || strings.HasSuffix(lower, "resources.arsc"):
		return BucketResources
	case strings.Contains(lower, "/assets/") || strings.HasPrefix(lower, "assets/"):
		return BucketAssets
	case strings.HasPrefix(lower, "kotlin/") || strings.Contains(lower, "/kotlin/") || strings.HasSuffix(lower, ".kotlin_module"):
		return BucketKotlin
	default:
		return BucketOther
	}
}

// moduleForEntry extracts the AAB module name from an entry path. AABs store
// each module under a top-level folder: base/, <feature>/, etc. APKs have no
// modules; we label everything "apk".
func moduleForEntry(name string) string {
	// AAB modules use forward slashes.
	p := path.Clean(name)
	if p == "." || p == "/" {
		return "apk"
	}
	// If there's a top-level folder, treat it as a module name.
	slash := strings.Index(p, "/")
	if slash <= 0 {
		return "apk"
	}
	top := p[:slash]
	// Heuristic: meta / manifest-only entries live at root or under META-INF.
	if strings.EqualFold(top, "META-INF") {
		return "meta"
	}
	return top
}

func sortedModules(m map[string]*ModuleSummary) []ModuleSummary {
	out := make([]ModuleSummary, 0, len(m))
	for _, v := range m {
		if v.UncompressedB > 0 {
			v.CompressionRate = float64(v.CompressedB) / float64(v.UncompressedB)
		}
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UncompressedB > out[j].UncompressedB })
	return out
}

func sortedBuckets(m map[Bucket]*BucketSummary) []BucketSummary {
	out := make([]BucketSummary, 0, len(m))
	for _, v := range m {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UncompressedB > out[j].UncompressedB })
	return out
}

// Diff is a delta between two analyses.
type Diff struct {
	Base            string        `json:"base"`
	Candidate       string        `json:"candidate"`
	DeltaCompressed int64         `json:"delta_compressed_bytes"`
	DeltaUncompr    int64         `json:"delta_uncompressed_bytes"`
	PerModule       []ModuleDelta `json:"per_module,omitempty"`
	PerBucket       []BucketDelta `json:"per_bucket,omitempty"`
	Regression      bool          `json:"regression"`
	ThresholdBytes  int64         `json:"threshold_bytes,omitempty"`
}

// ModuleDelta captures per-module size change.
type ModuleDelta struct {
	Module             string `json:"module"`
	DeltaCompressed    int64  `json:"delta_compressed_bytes"`
	DeltaUncompressed  int64  `json:"delta_uncompressed_bytes"`
	BaseUncompressed   int64  `json:"base_uncompressed_bytes"`
	CandUncompressed   int64  `json:"candidate_uncompressed_bytes"`
	AddedInCandidate   bool   `json:"added_in_candidate,omitempty"`
	RemovedInCandidate bool   `json:"removed_in_candidate,omitempty"`
}

// BucketDelta captures per-bucket size change.
type BucketDelta struct {
	Bucket            Bucket `json:"bucket"`
	DeltaCompressed   int64  `json:"delta_compressed_bytes"`
	DeltaUncompressed int64  `json:"delta_uncompressed_bytes"`
	BaseUncompressed  int64  `json:"base_uncompressed_bytes"`
	CandUncompressed  int64  `json:"candidate_uncompressed_bytes"`
}

// Compare diffs two analyses. If threshold > 0, regression flag is set when
// the overall uncompressed delta exceeds it.
func Compare(base, cand *Analysis, threshold int64) Diff {
	d := Diff{
		Base:            base.Path,
		Candidate:       cand.Path,
		DeltaCompressed: cand.TotalCompressed - base.TotalCompressed,
		DeltaUncompr:    cand.TotalUncompressd - base.TotalUncompressd,
		ThresholdBytes:  threshold,
	}
	if threshold > 0 {
		d.Regression = d.DeltaUncompr > threshold
	}

	d.PerModule = diffModules(base.Modules, cand.Modules)
	d.PerBucket = diffBuckets(base.Buckets, cand.Buckets)
	return d
}

func diffModules(baseList, candList []ModuleSummary) []ModuleDelta {
	baseIdx := map[string]ModuleSummary{}
	for _, m := range baseList {
		baseIdx[m.Name] = m
	}
	candIdx := map[string]ModuleSummary{}
	for _, m := range candList {
		candIdx[m.Name] = m
	}
	names := map[string]bool{}
	for k := range baseIdx {
		names[k] = true
	}
	for k := range candIdx {
		names[k] = true
	}
	out := make([]ModuleDelta, 0, len(names))
	for n := range names {
		b := baseIdx[n]
		c := candIdx[n]
		out = append(out, ModuleDelta{
			Module:             n,
			DeltaCompressed:    c.CompressedB - b.CompressedB,
			DeltaUncompressed:  c.UncompressedB - b.UncompressedB,
			BaseUncompressed:   b.UncompressedB,
			CandUncompressed:   c.UncompressedB,
			AddedInCandidate:   b.Files == 0 && c.Files > 0,
			RemovedInCandidate: b.Files > 0 && c.Files == 0,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return abs(out[i].DeltaUncompressed) > abs(out[j].DeltaUncompressed)
	})
	return out
}

func diffBuckets(baseList, candList []BucketSummary) []BucketDelta {
	baseIdx := map[Bucket]BucketSummary{}
	for _, b := range baseList {
		baseIdx[b.Bucket] = b
	}
	candIdx := map[Bucket]BucketSummary{}
	for _, b := range candList {
		candIdx[b.Bucket] = b
	}
	names := map[Bucket]bool{}
	for k := range baseIdx {
		names[k] = true
	}
	for k := range candIdx {
		names[k] = true
	}
	out := make([]BucketDelta, 0, len(names))
	for n := range names {
		b := baseIdx[n]
		c := candIdx[n]
		out = append(out, BucketDelta{
			Bucket:            n,
			DeltaCompressed:   c.CompressedB - b.CompressedB,
			DeltaUncompressed: c.UncompressedB - b.UncompressedB,
			BaseUncompressed:  b.UncompressedB,
			CandUncompressed:  c.UncompressedB,
		})
	}
	sort.Slice(out, func(i, j int) bool { return abs(out[i].DeltaUncompressed) > abs(out[j].DeltaUncompressed) })
	return out
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// ParseSizeThreshold parses strings like "500K", "2M", "1G", or plain bytes.
// Returns 0 with no error for the empty string.
func ParseSizeThreshold(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return 0, nil
	}
	mul := int64(1)
	switch {
	case strings.HasSuffix(s, "KB"), strings.HasSuffix(s, "K"):
		mul = 1024
		s = strings.TrimSuffix(strings.TrimSuffix(s, "KB"), "K")
	case strings.HasSuffix(s, "MB"), strings.HasSuffix(s, "M"):
		mul = 1024 * 1024
		s = strings.TrimSuffix(strings.TrimSuffix(s, "MB"), "M")
	case strings.HasSuffix(s, "GB"), strings.HasSuffix(s, "G"):
		mul = 1024 * 1024 * 1024
		s = strings.TrimSuffix(strings.TrimSuffix(s, "GB"), "G")
	case strings.HasSuffix(s, "B"):
		s = strings.TrimSuffix(s, "B")
	}
	if s == "" {
		return 0, errors.New("empty size value")
	}
	var n int64
	for _, c := range s {
		if c == ' ' {
			continue
		}
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid size %q", s)
		}
		n = n*10 + int64(c-'0')
	}
	return n * mul, nil
}
