package checks

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	checksapi "google.golang.org/api/checks/v1alpha"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/preflight"
)

const (
	maxRepoSourceSnippetBytes = 1 << 20
	maxRepoSourceTotalBytes   = 10 << 20
)

type repoUploadManifest struct {
	Version           int      `json:"version"`
	Provider          string   `json:"provider"`
	Account           string   `json:"account"`
	Repository        string   `json:"repository"`
	CLIVersion        string   `json:"cliVersion"`
	LocalScanPath     string   `json:"localScanPath"`
	SourceCodePaths   []string `json:"sourceCodePaths"`
	SourceCodeCount   int      `json:"sourceCodeCount"`
	SourceCodeBytes   int      `json:"sourceCodeBytes"`
	CodeExcerptCount  int      `json:"codeExcerptCount"`
	CodeExcerptBytes  int      `json:"codeExcerptBytes"`
	DetectedSources   int      `json:"detectedSources"`
	RemoteHost        string   `json:"remoteHost"`
	Branch            string   `json:"branch"`
	RevisionID        string   `json:"revisionId"`
	RequestSHA256     string   `json:"requestSha256"`
	CredentialMatches []string `json:"credentialMatches"`
	ManifestHash      string   `json:"manifestHash"`
}

func buildRepoUploadManifest(account, repository string, request *checksapi.GoogleChecksRepoScanV1alphaGenerateScanRequest) (*repoUploadManifest, error) {
	if request == nil || request.CliAnalysis == nil || request.ScmMetadata == nil {
		return nil, fmt.Errorf("repository scan request is incomplete")
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("encode repository scan request: %w", err)
	}
	credentialMatches := preflight.DetectCredentialPatterns(encoded)
	if len(credentialMatches) > 0 {
		return nil, fmt.Errorf("repository scan payload contains credential-shaped data (%s); remove and rotate it before upload", strings.Join(credentialMatches, ", "))
	}
	remoteHost, err := safeRemoteHost(request.ScmMetadata.RemoteUri)
	if err != nil {
		return nil, err
	}
	manifest := &repoUploadManifest{
		Version: 1, Provider: "official-checks-api", Account: account, Repository: strings.TrimSpace(repository),
		CLIVersion: strings.TrimSpace(request.CliVersion), LocalScanPath: strings.TrimSpace(request.LocalScanPath),
		DetectedSources: len(request.CliAnalysis.Sources), RemoteHost: remoteHost,
		Branch: request.ScmMetadata.Branch, RevisionID: request.ScmMetadata.RevisionId,
		RequestSHA256: repoSHA256(encoded), CredentialMatches: []string{}, SourceCodePaths: []string{},
	}
	for _, scan := range request.CliAnalysis.CodeScans {
		if scan == nil || scan.SourceCode == nil {
			return nil, fmt.Errorf("every cliAnalysis.codeScans entry must include sourceCode")
		}
		code := []byte(scan.SourceCode.Code)
		if err := validateRepoSourceText(scan.SourceCode.Path, code); err != nil {
			return nil, err
		}
		manifest.SourceCodeCount++
		manifest.SourceCodeBytes += len(code)
		manifest.SourceCodePaths = append(manifest.SourceCodePaths, filepath.ToSlash(strings.TrimSpace(scan.SourceCode.Path)))
	}
	for _, source := range request.CliAnalysis.Sources {
		if source == nil || source.CodeAttribution == nil {
			continue
		}
		excerpt := []byte(source.CodeAttribution.CodeExcerpt)
		if err := validateRepoSourceText(source.CodeAttribution.Path, excerpt); err != nil {
			return nil, err
		}
		if len(excerpt) > 0 {
			manifest.CodeExcerptCount++
			manifest.CodeExcerptBytes += len(excerpt)
		}
	}
	if manifest.SourceCodeBytes+manifest.CodeExcerptBytes > maxRepoSourceTotalBytes {
		return nil, fmt.Errorf("repository scan payload contains %d source bytes, above the %d-byte safety limit", manifest.SourceCodeBytes+manifest.CodeExcerptBytes, maxRepoSourceTotalBytes)
	}
	sort.Strings(manifest.SourceCodePaths)
	manifest.ManifestHash, err = repoManifestHash(manifest)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func validateRepoSourceText(path string, data []byte) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("repository source entry is missing its path")
	}
	if len(data) > maxRepoSourceSnippetBytes {
		return fmt.Errorf("repository source %q exceeds the %d-byte per-snippet limit", path, maxRepoSourceSnippetBytes)
	}
	if !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
		return fmt.Errorf("repository source %q appears binary and will not be uploaded", path)
	}
	return nil
}

func safeRemoteHost(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("scmMetadata.remoteUri is required")
	}
	parsed, err := url.Parse(value)
	if err == nil && parsed.Host != "" {
		if parsed.User != nil {
			return "", fmt.Errorf("scmMetadata.remoteUri must not contain credentials")
		}
		return parsed.Hostname(), nil
	}
	if at := strings.LastIndex(value, "@"); at >= 0 {
		if colon := strings.Index(value[at+1:], ":"); colon > 0 {
			return value[at+1 : at+1+colon], nil
		}
	}
	return "", fmt.Errorf("scmMetadata.remoteUri must be an HTTPS/SSH repository URI without credentials")
}

func repoManifestHash(manifest *repoUploadManifest) (string, error) {
	copyManifest := *manifest
	copyManifest.ManifestHash = ""
	encoded, err := json.Marshal(copyManifest)
	if err != nil {
		return "", fmt.Errorf("encode repository upload manifest: %w", err)
	}
	return repoSHA256(encoded), nil
}

func writeRepoUploadManifest(filesystem shared.Filesystem, path string, manifest *repoUploadManifest) error {
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode repository upload manifest: %w", err)
	}
	if err := filesystem.AtomicWriteFile(path, encoded, 0o600, 0o700); err != nil {
		return fmt.Errorf("write repository upload manifest: %w", err)
	}
	return nil
}

func repoSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
