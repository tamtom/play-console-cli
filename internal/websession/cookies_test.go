package websession

import (
	"testing"
	"time"
)

func TestParseCookieHeader_Basic(t *testing.T) {
	cookies, err := ParseCookieHeader("https://play.google.com/", "SID=sid-value; SAPISID=sapisid-value; HSID=hsid-value")
	if err != nil {
		t.Fatalf("ParseCookieHeader: %v", err)
	}
	if len(cookies) != 3 {
		t.Fatalf("got %d cookies, want 3", len(cookies))
	}
	if cookies[0].Name != "SID" || cookies[0].Value != "sid-value" {
		t.Errorf("cookies[0] = %+v", cookies[0])
	}
	if cookies[1].Name != "SAPISID" || cookies[1].Value != "sapisid-value" {
		t.Errorf("cookies[1] = %+v", cookies[1])
	}
	if cookies[2].Name != "HSID" || cookies[2].Value != "hsid-value" {
		t.Errorf("cookies[2] = %+v", cookies[2])
	}
}

func TestParseCookieHeader_WhitespaceAndEmptySegments(t *testing.T) {
	cookies, err := ParseCookieHeader("https://play.google.com/", "  a=1 ; ; b=2;")
	if err != nil {
		t.Fatalf("ParseCookieHeader: %v", err)
	}
	if len(cookies) != 2 {
		t.Fatalf("got %d cookies, want 2: %+v", len(cookies), cookies)
	}
	if cookies[0].Name != "a" || cookies[1].Name != "b" {
		t.Errorf("cookies = %+v", cookies)
	}
}

func TestParseCookieHeader_ValueWithEquals(t *testing.T) {
	cookies, err := ParseCookieHeader("https://play.google.com/", "token=abc=def=")
	if err != nil {
		t.Fatalf("ParseCookieHeader: %v", err)
	}
	if len(cookies) != 1 || cookies[0].Value != "abc=def=" {
		t.Errorf("cookies = %+v, want value abc=def=", cookies)
	}
}

func TestParseCookieHeader_QuotedValue(t *testing.T) {
	cookies, err := ParseCookieHeader("https://play.google.com/", `a="quoted value"; b=plain`)
	if err != nil {
		t.Fatalf("ParseCookieHeader: %v", err)
	}
	if len(cookies) != 2 {
		t.Fatalf("got %d cookies, want 2", len(cookies))
	}
	if cookies[0].Value != "quoted value" {
		t.Errorf("quoted value = %q, want %q", cookies[0].Value, "quoted value")
	}
}

func TestParseCookieHeader_SegmentWithoutEqualsSkipped(t *testing.T) {
	cookies, err := ParseCookieHeader("https://play.google.com/", "a=1; stray; b=2")
	if err != nil {
		t.Fatalf("ParseCookieHeader: %v", err)
	}
	if len(cookies) != 2 {
		t.Fatalf("got %d cookies, want 2: %+v", len(cookies), cookies)
	}
}

func TestParseCookieHeader_Errors(t *testing.T) {
	for _, header := range []string{"", "   ", "stray-only", "; ;"} {
		if _, err := ParseCookieHeader("https://play.google.com/", header); err == nil {
			t.Errorf("expected error for header %q", header)
		}
	}
	if _, err := ParseCookieHeader("https://play.google.com/", "=value"); err == nil {
		t.Error("expected error for empty cookie name")
	}
	if _, err := ParseCookieHeader("::bad-origin::", "a=1"); err == nil {
		t.Error("expected error for invalid origin")
	}
}

func TestParseCookieExportJSON_Array(t *testing.T) {
	data := `[
		{"name":"SAPISID","value":"sapisid-value","domain":".play.google.com","path":"/","expirationDate":1800000000},
		{"name":"SID","value":"sid-value","domain":".play.google.com","path":"/"},
		{"name":"ACCOUNT_CHOOSER","value":"acc","domain":"accounts.google.com","hostOnly":true}
	]`
	byOrigin, err := ParseCookieExportJSON([]byte(data))
	if err != nil {
		t.Fatalf("ParseCookieExportJSON: %v", err)
	}
	play := byOrigin["https://play.google.com/"]
	if len(play) != 2 {
		t.Fatalf("play.google.com cookies = %d, want 2 (origins: %v)", len(play), keys(byOrigin))
	}
	if play[0].Expires.IsZero() {
		t.Error("expirationDate should map to Expires")
	} else if play[0].Expires.Unix() != 1800000000 {
		t.Errorf("Expires = %d, want 1800000000", play[0].Expires.Unix())
	}
	if play[1].Expires.IsZero() != true {
		t.Error("missing expirationDate should leave Expires zero")
	}
	accts := byOrigin["https://accounts.google.com/"]
	if len(accts) != 1 || accts[0].Name != "ACCOUNT_CHOOSER" {
		t.Errorf("accounts.google.com cookies = %+v", accts)
	}
}

func TestParseCookieExportJSON_SingleObject(t *testing.T) {
	data := `{"name":"SAPISID","value":"v","domain":"play.google.com"}`
	byOrigin, err := ParseCookieExportJSON([]byte(data))
	if err != nil {
		t.Fatalf("ParseCookieExportJSON: %v", err)
	}
	if got := byOrigin["https://play.google.com/"]; len(got) != 1 {
		t.Errorf("cookies = %+v", byOrigin)
	}
}

func TestParseCookieExportJSON_SkipsEmptyNames(t *testing.T) {
	data := `[{"name":"","value":"x","domain":"a.com"},{"name":"ok","value":"y","domain":"a.com"}]`
	byOrigin, err := ParseCookieExportJSON([]byte(data))
	if err != nil {
		t.Fatalf("ParseCookieExportJSON: %v", err)
	}
	if got := byOrigin["https://a.com/"]; len(got) != 1 || got[0].Name != "ok" {
		t.Errorf("cookies = %+v", byOrigin)
	}
}

func TestParseCookieExportJSON_Errors(t *testing.T) {
	for _, data := range []string{"", "not json", "[]", "[{}]", `[{"name":"a"}]`} {
		if _, err := ParseCookieExportJSON([]byte(data)); err == nil {
			t.Errorf("expected error for input %q", data)
		}
	}
}

func TestParseCookieExportJSON_ExpiresString(t *testing.T) {
	// Some exporters emit an RFC3339 "expires" string instead of a timestamp.
	expires := time.Date(2027, 1, 2, 3, 4, 5, 0, time.UTC)
	data := `[{"name":"a","value":"1","domain":"x.com","expires":"` + expires.Format(time.RFC3339) + `"}]`
	byOrigin, err := ParseCookieExportJSON([]byte(data))
	if err != nil {
		t.Fatalf("ParseCookieExportJSON: %v", err)
	}
	got := byOrigin["https://x.com/"]
	if len(got) != 1 || !got[0].Expires.Equal(expires) {
		t.Errorf("Expires = %v, want %v", got, expires)
	}
}

func TestNormalizeOrigin(t *testing.T) {
	cases := map[string]string{
		"play.google.com":          "https://play.google.com/",
		"https://play.google.com":  "https://play.google.com/",
		"https://play.google.com/": "https://play.google.com/",
		".play.google.com":         "https://play.google.com/",
	}
	for in, want := range cases {
		got, err := NormalizeOrigin(in)
		if err != nil {
			t.Errorf("NormalizeOrigin(%q): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("NormalizeOrigin(%q) = %q, want %q", in, got, want)
		}
	}
	if _, err := NormalizeOrigin(""); err == nil {
		t.Error("expected error for empty origin")
	}
	if _, err := NormalizeOrigin("https://"); err == nil {
		t.Error("expected error for missing host")
	}
}

// keys returns the origin keys of a cookie map for test diagnostics.
func keys(m map[string][]Cookie) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
