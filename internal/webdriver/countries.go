package webdriver

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// trackCountriesURL builds the console route for a track's page. track is a
// console track path segment: "production" for the production track, or the
// numeric track ID for custom/closed tracks (visible in the console URL after
// "Manage track").
func trackCountriesURL(developerID, appID, account, track string) string {
	track = strings.TrimSpace(track)
	if track == "" {
		track = "production"
	}
	return consoleAppURL(developerID, appID, account, "tracks/"+track)
}

// countriesHelpers installs country-table lookups in the page. The console
// renders one mat-checkbox per country/region row, labeled by the country
// name, plus a "Select all rows" header checkbox. Selection state lives in
// aria-checked. The Save/Discard controls sit in the page-level publishing
// bottom bar, exactly like the Store settings editor.
const countriesHelpers = `
(() => {
  const scope = () => document.querySelector('[debug-id=countries-regions-section]') || document;
  const boxes = () => [...scope().querySelectorAll('mat-checkbox[role=checkbox]')]
    .filter(c => (c.getAttribute('aria-label') || '') !== 'Select all rows');
  const selectAll = () => scope().querySelector('mat-checkbox[aria-label="Select all rows"]');
  const selected = () => boxes()
    .filter(c => c.getAttribute('aria-checked') === 'true')
    .map(c => c.getAttribute('aria-label'));
  const bar = () => document.querySelector('[debug-id=publishing-bottom-bar]');
  const mainButton = () => {
    const b = bar();
    return b && (b.querySelector('[debug-id=main-button] button') || b.querySelector('[debug-id=main-button]'));
  };
  const discardButton = () => {
    const b = bar();
    return b && (b.querySelector('[debug-id=discard-button] button') || b.querySelector('[debug-id=discard-button]'));
  };
  window.__gplayCountries = { boxes, selectAll, selected, bar, mainButton, discardButton };
  return true;
})()
`

const countriesTabReadyScript = formHelpers + ` && ` + countriesHelpers +
	` && !! [...document.querySelectorAll('[role=tab]')].find(t => /countries/i.test(t.textContent))`

const countriesEditorReadyScript = formHelpers + ` && ` + countriesHelpers +
	` && !!(document.querySelector('[debug-id=empty-state-include-button]') ||
	  document.querySelector('[debug-id=edit-countries-button]') ||
	  window.__gplayCountries.boxes().length > 0)`

const openCountriesTabScript = `(() => {
  const tab = [...document.querySelectorAll('[role=tab]')].find(t => /countries/i.test(t.textContent));
  if (!tab) return false;
  tab.click();
  return true;
})()`

// openCountriesEditorScript enters the country editor. The empty state has an
// "Add countries / regions" button; a configured track offers an edit button
// instead. When the table is already open there is nothing to do.
const openCountriesEditorScript = `(() => {
  if (window.__gplayCountries.boxes().length > 0) return 'already-open';
  const include = document.querySelector('[debug-id=empty-state-include-button]');
  if (include) { include.click(); return 'opened-empty'; }
  const edit = document.querySelector('[debug-id=edit-countries-button]');
  if (edit) { (edit.querySelector('button') || edit).click(); return 'opened-edit'; }
  return 'no-entry';
})()`

const countriesTableReadyScript = `(() => window.__gplayCountries.boxes().length > 0)()`

// OpenCountriesEditor opens a track's Countries / regions editor in select
// mode. It saves nothing; callers either SubmitCountries or DiscardCountries
// afterwards. track is "production" or a numeric track ID.
func OpenCountriesEditor(ctx context.Context, b *Browser, developerID, appID, account, track string) error {
	if strings.TrimSpace(developerID) == "" || strings.TrimSpace(appID) == "" {
		return fmt.Errorf("developer ID and app ID are required")
	}
	if err := b.Navigate(ctx, trackCountriesURL(developerID, appID, account, track)); err != nil {
		return err
	}
	if err := b.EvalUntil(ctx, countriesTabReadyScript, 60*time.Second); err != nil {
		return fmt.Errorf("the track page did not load (is the gplay browser profile signed in?): %w", err)
	}
	var opened bool
	if err := b.Eval(ctx, openCountriesTabScript, &opened); err != nil {
		return err
	}
	if !opened {
		return fmt.Errorf("no Countries / regions tab was found on the track page")
	}
	if err := b.EvalUntil(ctx, countriesEditorReadyScript, 30*time.Second); err != nil {
		return fmt.Errorf("the Countries / regions section did not load: %w", err)
	}
	var result string
	if err := b.Eval(ctx, openCountriesEditorScript, &result); err != nil {
		return err
	}
	if result == "no-entry" {
		return fmt.Errorf("could not find a way into the Countries / regions editor")
	}
	if err := b.EvalUntil(ctx, countriesTableReadyScript, 30*time.Second); err != nil {
		return fmt.Errorf("country list did not load: %w", err)
	}
	return nil
}

// CountriesState is the production track's country selection as shown in the
// open editor.
type CountriesState struct {
	Selected  []string `json:"selected"`
	CanSubmit bool     `json:"canSubmit"`
}

const readCountriesScript = `(() => {
  const c = window.__gplayCountries;
  const main = c.mainButton();
  return {
    selected: c.selected(),
    canSubmit: !!(main && !main.disabled && main.getAttribute('aria-disabled') !== 'true'),
  };
})()`

// ReadCountries reads the current selection from the open editor.
func ReadCountries(ctx context.Context, b *Browser) (*CountriesState, error) {
	if err := b.Eval(ctx, formHelpers+` && `+countriesHelpers, nil); err != nil {
		return nil, err
	}
	var state CountriesState
	if err := b.Eval(ctx, readCountriesScript, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// countryKnownScript reports whether the country list contains the name.
func countryKnownScript(name string) string {
	return formHelpers + ` && ` + countriesHelpers + ` && (() =>
	  window.__gplayCountries.boxes().some(c => c.getAttribute('aria-label') === ` + jsString(name) + `))()`
}

// countryClickScript clicks the named country's checkbox unless it is already
// targeted.
func countryClickScript(name string) string {
	return formHelpers + ` && ` + countriesHelpers + ` && (() => {
	  const box = window.__gplayCountries.boxes().find(c => c.getAttribute('aria-label') === ` + jsString(name) + `);
	  if (!box || box.getAttribute('aria-checked') === 'true') return true;
	  box.click();
	  return true;
	})()`
}

// countryCheckedScript reports whether the named country ended up targeted.
func countryCheckedScript(name string) string {
	return formHelpers + ` && ` + countriesHelpers + ` && (() => {
	  const box = window.__gplayCountries.boxes().find(c => c.getAttribute('aria-label') === ` + jsString(name) + `);
	  return !!box && box.getAttribute('aria-checked') === 'true';
	})()`
}

// SetCountries targets the named countries (console display names, e.g.
// "Slovenia"). Already-targeted countries are left alone. Unknown names are an
// error before anything is clicked.
func SetCountries(ctx context.Context, b *Browser, names []string) error {
	// Validate every name first so a typo never leaves a partial selection.
	for _, name := range names {
		var known bool
		if err := b.Eval(ctx, countryKnownScript(name), &known); err != nil {
			return err
		}
		if !known {
			return fmt.Errorf("country %q was not found in the console list", name)
		}
	}
	for _, name := range names {
		if err := b.Eval(ctx, countryClickScript(name), nil); err != nil {
			return fmt.Errorf("selecting %q: %w", name, err)
		}
		if err := b.EvalUntil(ctx, countryCheckedScript(name), 5*time.Second); err != nil {
			return fmt.Errorf("selecting %q: %w", name, err)
		}
	}
	return nil
}

// countryUnclickScript clicks the named country's checkbox when it is
// currently targeted, untargeting it.
func countryUnclickScript(name string) string {
	return formHelpers + ` && ` + countriesHelpers + ` && (() => {
	  const box = window.__gplayCountries.boxes().find(c => c.getAttribute('aria-label') === ` + jsString(name) + `);
	  if (!box || box.getAttribute('aria-checked') !== 'true') return true;
	  box.click();
	  return true;
	})()`
}

// countryUncheckedScript reports whether the named country ended up
// untargeted.
func countryUncheckedScript(name string) string {
	return formHelpers + ` && ` + countriesHelpers + ` && (() => {
	  const box = window.__gplayCountries.boxes().find(c => c.getAttribute('aria-label') === ` + jsString(name) + `);
	  return !!box && box.getAttribute('aria-checked') !== 'true';
	})()`
}

// UnsetCountries stops targeting the named countries (console display names,
// e.g. "Sudan"). Unknown names are an error before anything is clicked.
func UnsetCountries(ctx context.Context, b *Browser, names []string) error {
	for _, name := range names {
		var known bool
		if err := b.Eval(ctx, countryKnownScript(name), &known); err != nil {
			return err
		}
		if !known {
			return fmt.Errorf("country %q was not found in the console list", name)
		}
	}
	for _, name := range names {
		if err := b.Eval(ctx, countryUnclickScript(name), nil); err != nil {
			return fmt.Errorf("removing %q: %w", name, err)
		}
		if err := b.EvalUntil(ctx, countryUncheckedScript(name), 5*time.Second); err != nil {
			return fmt.Errorf("removing %q: %w", name, err)
		}
	}
	return nil
}

const setAllCountriesScript = `(() => {
  const all = window.__gplayCountries.selectAll();
  if (!all) return 'no-select-all';
  if (all.getAttribute('aria-checked') === 'true') return 'already';
  all.click();
  return 'clicked';
})()`

const allCountriesCheckedScript = `(() => {
  const boxes = window.__gplayCountries.boxes();
  return boxes.length > 0 && boxes.every(c => c.getAttribute('aria-checked') === 'true');
})()`

// SetAllCountries targets every country/region in the list.
func SetAllCountries(ctx context.Context, b *Browser) error {
	var result string
	if err := b.Eval(ctx, formHelpers+` && `+countriesHelpers+` && `+setAllCountriesScript, &result); err != nil {
		return err
	}
	switch result {
	case "clicked", "already":
	case "no-select-all":
		return fmt.Errorf(`"Select all rows" control was not found`)
	default:
		return fmt.Errorf("selecting all countries: unexpected page state %q", result)
	}
	if err := b.EvalUntil(ctx, allCountriesCheckedScript, 30*time.Second); err != nil {
		return fmt.Errorf("selecting all countries: %w", err)
	}
	return nil
}

const submitCountriesClickScript = `(() => {
  const main = window.__gplayCountries.mainButton();
  if (!main || main.disabled || main.getAttribute('aria-disabled') === 'true') return false;
  main.click();
  return true;
})()`

const countriesSaveSettledScript = `(() => {
  const c = window.__gplayCountries;
  const bar = c.bar();
  const main = c.mainButton();
  return !bar || !!(main && (main.disabled || main.getAttribute('aria-disabled') === 'true'));
})()`

// SubmitCountries saves the selection and waits for the bottom bar to settle.
// Callers must re-open the editor and verify the persisted selection.
func SubmitCountries(ctx context.Context, b *Browser, timeout time.Duration) error {
	var clicked bool
	if err := b.Eval(ctx, formHelpers+` && `+countriesHelpers+` && `+submitCountriesClickScript, &clicked); err != nil {
		return err
	}
	if !clicked {
		return fmt.Errorf(`"Save" button was missing or disabled`)
	}
	if err := b.EvalUntil(ctx, countriesSaveSettledScript, timeout); err != nil {
		return fmt.Errorf("saving countries did not complete: %w", err)
	}
	return nil
}

const discardCountriesClickScript = `(() => {
  const d = window.__gplayCountries.discardButton();
  if (!d) return false;
  d.click();
  return true;
})()`

const countriesDiscardSettledScript = `(() => !window.__gplayCountries.bar())()`

// DiscardCountries leaves the editor without saving.
func DiscardCountries(ctx context.Context, b *Browser) error {
	var clicked bool
	if err := b.Eval(ctx, formHelpers+` && `+countriesHelpers+` && `+discardCountriesClickScript, &clicked); err != nil {
		return err
	}
	if !clicked {
		return nil // no pending edits, nothing to discard
	}
	if err := b.EvalUntil(ctx, countriesDiscardSettledScript, 15*time.Second); err != nil {
		return fmt.Errorf("discarding country edits did not complete: %w", err)
	}
	return nil
}
