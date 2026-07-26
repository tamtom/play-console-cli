package preflight

import (
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
)

// secretPattern is one credential-shaped signature.
type secretPattern struct {
	ID       string
	Severity Severity
	Hint     string
	Re       *regexp.Regexp
}

// secretPatterns are deliberately conservative: each anchors on a vendor
// prefix rather than on entropy, so false positives stay rare.
//
// Severity reflects blast radius. A private key or live server-side token is
// an error. A Google API key is a warning: Android apps legitimately embed
// Maps and Firebase keys, and the real defense is key restriction, not
// absence.
//
// These patterns are intentionally unanchored: they search for a credential
// anywhere inside an arbitrary blob, rather than validating that a whole
// string is a trusted URL. Where a credential is carried in a URL, the pattern
// keys on the secret-bearing path rather than on the hostname, so the token is
// still found when the URL is assembled at runtime.
var secretPatterns = []secretPattern{
	{
		"private_key_block", SeverityError, "remove the key from the build; never ship private keys to clients",
		regexp.MustCompile(`-----BEGIN (?:RSA |EC |DSA |OPENSSH |PGP )?PRIVATE KEY-----`),
	},
	{
		"aws_access_key", SeverityError, "rotate the key and move AWS access behind your backend",
		regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	},
	{
		"stripe_secret_key", SeverityError, "rotate immediately; secret keys must never leave your server",
		regexp.MustCompile(`sk_live_[0-9A-Za-z]{24,}`),
	},
	{
		"github_token", SeverityError, "revoke the token; it grants repository access",
		regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{36}`),
	},
	{
		"slack_token", SeverityError, "revoke the token in Slack admin",
		regexp.MustCompile(`xox[baprs]-[0-9A-Za-z\-]{10,}`),
	},
	{
		"slack_webhook", SeverityError, "revoke the webhook; anyone can post to the channel with it",
		// Matches the credential path itself rather than the hooks.slack.com
		// host, so the token is still found when the URL is assembled at
		// runtime or the host is held in a separate constant.
		regexp.MustCompile(`/services/T[A-Za-z0-9]{6,}/B[A-Za-z0-9]{6,}/[A-Za-z0-9]{20,}`),
	},
	{
		"sendgrid_key", SeverityError, "revoke the key in SendGrid",
		regexp.MustCompile(`SG\.[A-Za-z0-9_\-]{20,}\.[A-Za-z0-9_\-]{40,}`),
	},
	{
		"google_oauth_client_secret", SeverityError, "rotate the client secret in Google Cloud Console",
		regexp.MustCompile(`GOCSPX-[A-Za-z0-9_\-]{20,}`),
	},
	{
		"anthropic_api_key", SeverityError, "revoke the key; model API keys must stay server-side",
		regexp.MustCompile(`sk-ant-[A-Za-z0-9_\-]{20,}`),
	},
	{
		"openai_api_key", SeverityError, "revoke the key; model API keys must stay server-side",
		regexp.MustCompile(`sk-proj-[A-Za-z0-9_\-]{20,}`),
	},
	{
		"gcp_service_account", SeverityError, "a service account key grants server-side access; remove it from the build",
		regexp.MustCompile(`"type"\s*:\s*"service_account"`),
	},
	{
		"google_api_key", SeverityWarning, "restrict the key to your package name and signing certificate in Cloud Console",
		regexp.MustCompile(`AIza[0-9A-Za-z_\-]{35}`),
	},
	{
		"jwt_token", SeverityWarning, "confirm this is not a long-lived credential baked into the build",
		regexp.MustCompile(`eyJ[A-Za-z0-9_=\-]{10,}\.[A-Za-z0-9_=\-]{10,}\.[A-Za-z0-9_.=\-]{10,}`),
	},
	{
		"firebase_url", SeverityInfo, "confirm your Realtime Database security rules are not left open",
		regexp.MustCompile(`[a-z0-9\-]+\.firebaseio\.com`),
	},
}

// credentialFileSuffixes are file types that should never ship in a build.
var credentialFileSuffixes = map[string]string{
	".jks":      "Java keystore",
	".keystore": "keystore",
	".p12":      "PKCS#12 keystore",
	".pfx":      "PKCS#12 keystore",
	".pem":      "PEM key or certificate",
	".ppk":      "PuTTY private key",
}

// developerArtifacts are paths that indicate an unclean build input.
var developerArtifacts = []string{
	".DS_Store", "__MACOSX/", ".git/", ".gitignore", "Thumbs.db",
	".env", ".idea/", ".gradle/", "local.properties",
}

// binaryEntrySuffixes are skipped by the regex scan; they are large and
// compressed, so pattern matching is expensive and unreliable there.
var binaryEntrySuffixes = []string{
	".png", ".jpg", ".jpeg", ".webp", ".gif", ".mp3", ".mp4", ".ogg",
	".wav", ".otf", ".ttf", ".zip", ".so", ".kotlin_module", ".version",
}

// secretScanOverlap is the number of bytes carried across chunk boundaries so
// a credential split across a boundary is still matched.
const secretScanOverlap = 512

// scanSecrets looks for credentials and developer artifacts in the build.
func scanSecrets(c *scanContext) []Finding {
	var out []Finding
	out = append(out, scanEntriesForSecrets(c)...)
	out = append(out, checkCredentialFiles(c)...)
	out = append(out, checkDeveloperArtifacts(c)...)
	return out
}

// scanEntriesForSecrets applies the pattern set to resource entries and to
// dex bytecode, where hardcoded string literals also live.
func scanEntriesForSecrets(c *scanContext) []Finding {
	var out []Finding

	for _, f := range c.files {
		if f.UncompressedSize64 == 0 {
			continue
		}
		isDex := strings.HasSuffix(f.Name, ".dex")
		if !isDex && skipForSecretScan(f.Name) {
			continue
		}

		limit := c.maxEntryBytes()
		if isDex {
			limit = c.maxDexScanBytes()
		} else if int64(f.UncompressedSize64) > limit { // #nosec G115
			continue
		}

		rc, err := f.Open()
		if err != nil {
			continue
		}
		hits := scanReaderForSecrets(rc, limit)
		_ = rc.Close()

		ids := make([]string, 0, len(hits))
		for id := range hits {
			ids = append(ids, id)
		}
		sort.Strings(ids)

		for _, id := range ids {
			p := secretPatternByID(id)
			out = append(out, Finding{
				Check:    "secrets",
				Severity: p.Severity,
				Message:  fmt.Sprintf("%s pattern matched", id),
				Entry:    f.Name,
				Hint:     p.Hint,
			})
		}
	}
	return out
}

// scanReaderForSecrets streams a reader in bounded chunks, returning the set
// of matched pattern IDs.
func scanReaderForSecrets(r io.Reader, limit int64) map[string]bool {
	found := map[string]bool{}

	const chunkSize = 1 << 20
	buf := make([]byte, 0, chunkSize+secretScanOverlap)
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
			for _, p := range secretPatterns {
				if found[p.ID] {
					continue
				}
				if p.Re.Match(buf) {
					found[p.ID] = true
				}
			}
			if len(buf) > secretScanOverlap {
				keep := buf[len(buf)-secretScanOverlap:]
				buf = append(buf[:0], keep...)
			}
		}
		if err != nil {
			break
		}
	}
	return found
}

// secretPatternByID resolves a pattern ID back to its definition.
func secretPatternByID(id string) secretPattern {
	for _, p := range secretPatterns {
		if p.ID == id {
			return p
		}
	}
	return secretPattern{ID: id, Severity: SeverityWarning}
}

// skipForSecretScan reports whether an entry is a binary type worth skipping.
func skipForSecretScan(name string) bool {
	lower := strings.ToLower(name)
	for _, suffix := range binaryEntrySuffixes {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

// checkCredentialFiles flags keystores and key material shipped in the build.
func checkCredentialFiles(c *scanContext) []Finding {
	var out []Finding
	for _, f := range c.files {
		lower := strings.ToLower(f.Name)
		for suffix, label := range credentialFileSuffixes {
			if !strings.HasSuffix(lower, suffix) {
				continue
			}
			// Play signing metadata legitimately lives under META-INF.
			if strings.HasPrefix(f.Name, "META-INF/") {
				continue
			}
			out = append(out, Finding{
				Check:    "credential_file",
				Severity: SeverityError,
				Message:  fmt.Sprintf("build ships a %s", label),
				Entry:    f.Name,
				Hint:     "remove it from the packaged assets and rotate the key if it was ever released",
			})
			break
		}
	}
	return out
}

// checkDeveloperArtifacts flags files that leaked in from the dev environment.
func checkDeveloperArtifacts(c *scanContext) []Finding {
	var out []Finding
	seen := map[string]bool{}

	for _, f := range c.files {
		for _, artifact := range developerArtifacts {
			if !strings.Contains(f.Name, artifact) || seen[f.Name] {
				continue
			}
			seen[f.Name] = true
			sev := SeverityWarning
			if strings.Contains(f.Name, ".git/") || strings.Contains(f.Name, ".env") {
				// Repository metadata and dotenv files routinely carry secrets.
				sev = SeverityError
			}
			out = append(out, Finding{
				Check:    "misplaced_files",
				Severity: sev,
				Message:  "build contains a developer-environment artifact",
				Entry:    f.Name,
				Hint:     "exclude it from your assets/resources source sets",
			})
			break
		}
	}
	return out
}
