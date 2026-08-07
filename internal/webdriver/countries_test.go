package webdriver

import (
	"context"
	"strings"
	"testing"
	"time"
)

func withCountryHelpers(script string) string {
	return formHelpers + ` && ` + countriesHelpers + ` && ` + script
}

func TestTrackCountriesURL(t *testing.T) {
	got := trackCountriesURL(publishingTestDeveloper, publishingTestApp, testAccount, "production")
	want := "https://play.google.com/console/developers/1234567890/app/9876543210/tracks/production?authuser=me%40example.com&hl=en"
	if got != want {
		t.Errorf("url = %q, want %q", got, want)
	}
}

func scriptOpenCountriesEditor(t *testing.T, f *fakeChrome) {
	t.Helper()
	f.setReply(countriesTabReadyScript, true)
	f.setReply(openCountriesTabScript, true)
	f.setReply(countriesEditorReadyScript, true)
	f.setReply(openCountriesEditorScript, "opened-empty")
	f.setReply(countriesTableReadyScript, true)
}

func TestOpenCountriesEditor(t *testing.T) {
	f := newFakeChrome(t)
	scriptOpenCountriesEditor(t, f)
	b := connectFake(t, f)

	if err := OpenCountriesEditor(context.Background(), b, publishingTestDeveloper, publishingTestApp, testAccount, "production"); err != nil {
		t.Fatalf("OpenCountriesEditor: %v", err)
	}
}

func TestOpenCountriesEditor_NoEntryPoint(t *testing.T) {
	f := newFakeChrome(t)
	scriptOpenCountriesEditor(t, f)
	f.setReply(openCountriesEditorScript, "no-entry")
	b := connectFake(t, f)

	err := OpenCountriesEditor(context.Background(), b, publishingTestDeveloper, publishingTestApp, testAccount, "production")
	if err == nil || !strings.Contains(err.Error(), "could not find a way into") {
		t.Errorf("err = %v, want no-entry error", err)
	}
}

func TestOpenCountriesEditor_MissingTab(t *testing.T) {
	f := newFakeChrome(t)
	scriptOpenCountriesEditor(t, f)
	f.setReply(openCountriesTabScript, false)
	b := connectFake(t, f)

	err := OpenCountriesEditor(context.Background(), b, publishingTestDeveloper, publishingTestApp, testAccount, "production")
	if err == nil || !strings.Contains(err.Error(), "no Countries / regions tab was found") {
		t.Errorf("err = %v, want missing-tab error", err)
	}
}

func TestReadCountries(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(readCountriesScript, map[string]any{
		"selected":  []string{"Slovenia", "United States"},
		"canSubmit": true,
	})
	b := connectFake(t, f)

	state, err := ReadCountries(context.Background(), b)
	if err != nil {
		t.Fatalf("ReadCountries: %v", err)
	}
	if len(state.Selected) != 2 || state.Selected[0] != "Slovenia" || !state.CanSubmit {
		t.Errorf("state = %+v", state)
	}
}

func TestSetCountries_SelectsRequested(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(countryKnownScript("Slovenia"), true)
	f.setReply(countryCheckedScript("Slovenia"), true)
	b := connectFake(t, f)

	if err := SetCountries(context.Background(), b, []string{"Slovenia"}); err != nil {
		t.Fatalf("SetCountries: %v", err)
	}
}

func TestSetCountries_RejectsUnknownBeforeClicking(t *testing.T) {
	f := newFakeChrome(t)
	// "Narnia" unknown; "Slovenia" would be known but must never be checked
	// because validation of the whole list happens first.
	f.setReply(countryKnownScript("Slovenia"), true)
	b := connectFake(t, f)

	err := SetCountries(context.Background(), b, []string{"Slovenia", "Narnia"})
	if err == nil || !strings.Contains(err.Error(), `"Narnia" was not found`) {
		t.Fatalf("err = %v, want unknown-country error", err)
	}
	for _, expr := range f.seen {
		if strings.Contains(expr, "box.click()") {
			t.Errorf("no checkbox should have been clicked; saw %q", expr)
		}
	}
}

func TestSetAllCountries(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(withCountryHelpers(setAllCountriesScript), "clicked")
	f.setReply(allCountriesCheckedScript, true)
	b := connectFake(t, f)

	if err := SetAllCountries(context.Background(), b); err != nil {
		t.Fatalf("SetAllCountries: %v", err)
	}
}

func TestSetAllCountries_MissingControl(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(withCountryHelpers(setAllCountriesScript), "no-select-all")
	b := connectFake(t, f)

	err := SetAllCountries(context.Background(), b)
	if err == nil || !strings.Contains(err.Error(), "Select all rows") {
		t.Errorf("err = %v, want missing-control error", err)
	}
}

func TestSubmitCountries(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(withCountryHelpers(submitCountriesClickScript), true)
	f.setReply(countriesSaveSettledScript, true)
	b := connectFake(t, f)

	if err := SubmitCountries(context.Background(), b, 2*time.Second); err != nil {
		t.Fatalf("SubmitCountries: %v", err)
	}
}

func TestSubmitCountries_RefusesDisabledSave(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(withCountryHelpers(submitCountriesClickScript), false)
	b := connectFake(t, f)

	err := SubmitCountries(context.Background(), b, 300*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "missing or disabled") {
		t.Errorf("err = %v, want disabled-save error", err)
	}
}

func TestDiscardCountries_NoEditsIsNoOp(t *testing.T) {
	f := newFakeChrome(t)
	// No discard button at all: answers false.
	b := connectFake(t, f)

	if err := DiscardCountries(context.Background(), b); err != nil {
		t.Fatalf("DiscardCountries: %v", err)
	}
}

func TestDiscardCountries_Discards(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(withCountryHelpers(discardCountriesClickScript), true)
	f.setReply(countriesDiscardSettledScript, true)
	b := connectFake(t, f)

	if err := DiscardCountries(context.Background(), b); err != nil {
		t.Fatalf("DiscardCountries: %v", err)
	}
}

func TestUnsetCountries_RemovesRequested(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(countryKnownScript("Sudan"), true)
	f.setReply(countryUncheckedScript("Sudan"), true)
	b := connectFake(t, f)

	if err := UnsetCountries(context.Background(), b, []string{"Sudan"}); err != nil {
		t.Fatalf("UnsetCountries: %v", err)
	}
}

func TestUnsetCountries_RejectsUnknownBeforeClicking(t *testing.T) {
	f := newFakeChrome(t)
	b := connectFake(t, f)

	err := UnsetCountries(context.Background(), b, []string{"Narnia"})
	if err == nil || !strings.Contains(err.Error(), `"Narnia" was not found`) {
		t.Fatalf("err = %v, want unknown-country error", err)
	}
	for _, expr := range f.seen {
		if strings.Contains(expr, "box.click()") {
			t.Errorf("no checkbox should have been clicked; saw %q", expr)
		}
	}
}
