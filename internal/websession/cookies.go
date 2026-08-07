package websession

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// hostRe validates cookie-origin hosts (DNS names, optional port).
var hostRe = regexp.MustCompile(`^[a-zA-Z0-9.-]+(:[0-9]+)?$`)

// NormalizeOrigin canonicalizes an origin for use as a Session.Cookies key:
// a bare host or host-with-scheme becomes "https://host/". A leading dot
// (cookie-domain style) is stripped.
func NormalizeOrigin(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, ".")
	if raw == "" {
		return "", fmt.Errorf("origin is empty")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parsing origin %q: %w", raw, err)
	}
	if u.Host == "" {
		return "", fmt.Errorf("origin %q has no host", raw)
	}
	if !hostRe.MatchString(u.Host) {
		return "", fmt.Errorf("origin %q has invalid host %q", raw, u.Host)
	}
	scheme := u.Scheme
	if scheme == "" {
		scheme = "https"
	}
	return scheme + "://" + u.Host + "/", nil
}

// ParseCookieHeader parses a raw Cookie header value ("a=1; b=2") as copied
// from browser DevTools into cookies for the given origin. Segments without
// "=" are skipped; surrounding double quotes on values are stripped. Note
// that quoted values containing ";" are not supported (browsers almost never
// emit them for Google cookies).
func ParseCookieHeader(origin, header string) ([]Cookie, error) {
	if _, err := NormalizeOrigin(origin); err != nil {
		return nil, err
	}
	var cookies []Cookie
	for _, seg := range strings.Split(header, ";") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		name, value, ok := strings.Cut(seg, "=")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("cookie with empty name in segment %q", seg)
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 && strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
			value = value[1 : len(value)-1]
		}
		cookies = append(cookies, Cookie{Name: name, Value: value})
	}
	if len(cookies) == 0 {
		return nil, fmt.Errorf("no cookies found in header")
	}
	return cookies, nil
}

// cookieExport mirrors the JSON shape emitted by browser cookie exporters
// (Cookie-Editor, EditThisCookie): an array of objects, one per cookie.
type cookieExport struct {
	Name           string  `json:"name"`
	Value          string  `json:"value"`
	Domain         string  `json:"domain"`
	ExpirationDate float64 `json:"expirationDate"` // seconds since epoch
	Expires        string  `json:"expires"`        // RFC3339 alternative
}

// ParseCookieExportJSON parses a cookie-exporter JSON dump (array of cookie
// objects, or a single object) into cookies grouped by normalized origin.
// Entries with empty names or domains are skipped; an error is returned when
// nothing usable remains.
func ParseCookieExportJSON(data []byte) (map[string][]Cookie, error) {
	data = []byte(strings.TrimSpace(string(data)))
	if len(data) == 0 {
		return nil, fmt.Errorf("empty cookie export")
	}
	var entries []cookieExport
	if err := json.Unmarshal(data, &entries); err != nil {
		var single cookieExport
		if err2 := json.Unmarshal(data, &single); err2 != nil {
			return nil, fmt.Errorf("parsing cookie export JSON: %w", err)
		}
		entries = []cookieExport{single}
	}

	byOrigin := make(map[string][]Cookie)
	for _, e := range entries {
		if strings.TrimSpace(e.Name) == "" || strings.TrimSpace(e.Domain) == "" {
			continue
		}
		origin, err := NormalizeOrigin(e.Domain)
		if err != nil {
			continue
		}
		c := Cookie{Name: strings.TrimSpace(e.Name), Value: e.Value}
		if e.ExpirationDate > 0 {
			sec := int64(e.ExpirationDate)
			c.Expires = time.Unix(sec, 0).UTC()
		} else if strings.TrimSpace(e.Expires) != "" {
			if ts, err := time.Parse(time.RFC3339, strings.TrimSpace(e.Expires)); err == nil {
				c.Expires = ts.UTC()
			}
		}
		byOrigin[origin] = append(byOrigin[origin], c)
	}
	if len(byOrigin) == 0 {
		return nil, fmt.Errorf("no usable cookies found in export (need name and domain fields)")
	}
	return byOrigin, nil
}
