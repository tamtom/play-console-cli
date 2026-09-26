package shared

import (
	"net/url"
	"regexp"
)

var textURLPattern = regexp.MustCompile(`https?://[^\s"'<>]+`)

// RedactURLsInText applies the dry-run URL redaction to every http or https
// URL in s. Use it before an error text goes to stderr or to a report file,
// because a *url.Error contains the full request URL with its query.
func RedactURLsInText(s string) string {
	return textURLPattern.ReplaceAllStringFunc(s, func(raw string) string {
		u, err := url.Parse(raw)
		if err != nil {
			return "<redacted-url>"
		}
		return redactURL(*u)
	})
}
