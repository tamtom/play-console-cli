package webdriver

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

func appPricingURL(developerID, appID, account string) string {
	return consoleAppURL(developerID, appID, account, "paid-app")
}

// pricingHelpers installs App pricing page lookups. Setting a price is a
// wizard: "Set pricing" selects countries (same mat-checkbox table as the
// Countries editor), "Continue" opens a dialog asking for the price in the
// merchant's home currency, and "Save changes" on the page stages the result.
const pricingHelpers = `
(() => {
  const clean = s => (s || '').replace(/\s+/g, ' ').trim();
  const priceType = () => {
    const el = document.querySelector('[debug-id=app-price-type]');
    const t = el ? clean(el.textContent).toLowerCase() : '';
    if (t.includes('paid')) return 'paid';
    if (t.includes('free')) return 'free';
    return '';
  };
  const pricesSet = () => [...document.querySelectorAll('[role=row]')].some(r => {
    const cells = [...r.querySelectorAll('[role=gridcell]')].map(c => clean(c.innerText));
    return cells.length >= 2 && cells[1] !== '' && cells[1] !== '-';
  });
  const setPricingButton = () => {
    const w = document.querySelector('[debug-id=bulk-price-edit-button]');
    return w && (w.querySelector('button') || w);
  };
  const bar = () => document.querySelector('[debug-id=bulk-price-edit-bottom-bar]');
  // The Save/Discard controls render near, but not inside, the bottom-bar
  // element, so they are looked up document-wide by their debug ids/text.
  const saveButton = () => {
    const w = document.querySelector('[debug-id=save-button]');
    return w && (w.querySelector('button') || w);
  };
  // The bottom bar only offers Discard when a pricing draft is staged; it is
  // the reliable staged marker (the price table keeps showing old values).
  const discardButton = () =>
    [...document.querySelectorAll('button')].find(x =>
      /discard changes/i.test(x.textContent) && x.getClientRects().length);
  const staged = () => !!discardButton();
  const enabled = b => !!(b && !b.disabled && b.getAttribute('aria-disabled') !== 'true');
  window.__gplayPricing = { priceType, pricesSet, setPricingButton, bar, saveButton, discardButton, staged, enabled };
  return true;
})()
`

// AppPricingState is the App pricing page state managed by this driver.
type AppPricingState struct {
	PriceType     string `json:"priceType"`
	PricesSet     bool   `json:"pricesSet"`
	StagedChanges bool   `json:"stagedChanges"`
	CanSave       bool   `json:"canSave"`
}

const readAppPricingScript = `(() => {
  const p = window.__gplayPricing;
  return {
    priceType: p.priceType(),
    pricesSet: p.pricesSet(),
    stagedChanges: p.staged(),
    canSave: p.enabled(p.saveButton()),
  };
})()`

const pricingPageReadyScript = formHelpers + ` && ` + pricingHelpers +
	` && !!window.__gplayPricing.setPricingButton()`

// pricingStagedScript reports whether a pricing draft is currently staged;
// pricingUnstagedScript is its negation, used to wait a discard out.
const pricingStagedScript = formHelpers + ` && ` + pricingHelpers + ` && window.__gplayPricing.staged()`

const pricingUnstagedScript = formHelpers + ` && ` + pricingHelpers + ` && !window.__gplayPricing.staged()`

// OpenAppPricing opens the app's App pricing page. It changes nothing.
func OpenAppPricing(ctx context.Context, b *Browser, developerID, appID, account string) error {
	if strings.TrimSpace(developerID) == "" || strings.TrimSpace(appID) == "" {
		return fmt.Errorf("developer ID and app ID are required")
	}
	if err := b.Navigate(ctx, appPricingURL(developerID, appID, account)); err != nil {
		return err
	}
	if err := b.EvalUntil(ctx, pricingPageReadyScript, 60*time.Second); err != nil {
		return fmt.Errorf("the App pricing page did not load (is the gplay browser profile signed in?): %w", err)
	}
	return nil
}

// ReadAppPricing reads the pricing page state.
func ReadAppPricing(ctx context.Context, b *Browser) (*AppPricingState, error) {
	if err := b.Eval(ctx, formHelpers+` && `+pricingHelpers, nil); err != nil {
		return nil, err
	}
	var state AppPricingState
	if err := b.Eval(ctx, readAppPricingScript, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

const setPricingClickScript = `(() => {
  const p = window.__gplayPricing;
  const btn = p && p.setPricingButton();
  if (!p.enabled(btn)) return false;
  btn.click();
  return true;
})()`

const pricingCountryTableReadyScript = `!!document.querySelector('mat-checkbox[aria-label="Select all rows"]')`

// pricingSelectionCountedScript waits for the console's own "N countries /
// regions selected" readout: the selection model lags the header checkbox, and
// continuing early prices an empty set (proven against the live console).
const pricingSelectionCountedScript = `(() => /\d+ countries \/ regions selected/i.test(document.body.innerText))()`

// pricingWizardDoneScript is the wizard's success condition: a staged draft
// with an enabled Save changes AND computed prices in the table. A submission
// that lost the price fails this instead of looking successful.
const pricingWizardDoneScript = formHelpers + ` && ` + pricingHelpers + ` && (() => {
  const p = window.__gplayPricing;
  return p.staged() && p.enabled(p.saveButton()) && p.pricesSet();
})()`

const pricingSelectAllScript = `(() => {
  const all = document.querySelector('mat-checkbox[aria-label="Select all rows"]');
  if (!all) return 'no-select-all';
  if (all.getAttribute('aria-checked') !== 'true') all.click();
  return 'clicked';
})()`

const pricingContinueEnabledScript = `(() => {
  const btn = [...document.querySelectorAll('button')].find(b =>
    /^continue$/i.test(b.textContent.trim()) && b.getClientRects().length);
  return !!(btn && !btn.disabled && btn.getAttribute('aria-disabled') !== 'true');
})()`

const pricingContinueClickScript = `(() => {
  const btn = [...document.querySelectorAll('button')].find(b =>
    /^continue$/i.test(b.textContent.trim()) && b.getClientRects().length);
  if (!btn || btn.disabled) return false;
  btn.click();
  return true;
})()`

// priceInputReadyScript matches the dialog's "New price in <currency>" input
// without hard-coding the merchant's currency.
const priceInputReadyScript = `(() => {
  const i = document.querySelector('input[aria-label^="New price in"]');
  return !!(i && i.getClientRects().length);
})()`

// focusPriceInputScript focuses the dialog's price field and selects any
// existing content so the typed value replaces it.
const focusPriceInputScript = `(() => {
  const i = document.querySelector('input[aria-label^="New price in"]');
  if (!i) return false;
  i.focus();
  i.setSelectionRange(0, (i.value || '').length);
  return document.activeElement === i;
})()`

// priceTypedHeldScript reports whether the typed price is present and the
// dialog's Update button is enabled.
func priceTypedHeldScript(price string) string {
	return fmt.Sprintf(`(() => {
  const i = document.querySelector('input[aria-label^="New price in"]');
  if (!i || i.value !== %s) return false;
  const popup = document.querySelector('[debug-id=console-dialog-popup]') || document;
  const btn = [...popup.querySelectorAll('button')].find(b => /^update$/i.test(b.textContent.trim()));
  return !!(btn && !btn.disabled);
})()`, jsString(price))
}

const pricingUpdateClickScript = `(() => {
  const popup = document.querySelector('[debug-id=console-dialog-popup]') || document;
  const btn = [...popup.querySelectorAll('button')].find(b =>
    /^update$/i.test(b.textContent.trim()) && b.getClientRects().length);
  if (!btn || btn.disabled) return false;
  btn.click();
  return true;
})()`

const pricingDialogGoneScript = `(() => !document.querySelector('input[aria-label^="New price in"]'))()`

// pricingPostSetTimeout bounds the post-wizard verification wait,
// pricingFormSettleTimeout bounds the wait for the dialog's form to accept
// the price, and pricingStepSettle is the quiet beat after each wizard state
// change. Clicking the moment a control appears (EvalUntil's poll cadence)
// outruns the console's event wiring and silently breaks the wizard — proven
// against the live console. They are vars so tests can shorten them.
var (
	pricingPostSetTimeout    = 30 * time.Second
	pricingFormSettleTimeout = 15 * time.Second
	pricingStepSettle        = 2 * time.Second
	pricingDialogSettle      = 3 * time.Second
)

// settle waits out the console's async wiring after a wizard state change.
func settle(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

// SetAppPrice runs the "Set pricing" wizard: selects every country, enters
// the given price in the merchant's home currency, and confirms. It stops
// before the page-level save (see SubmitAppPricing).
func SetAppPrice(ctx context.Context, b *Browser, price string) error {
	if strings.TrimSpace(price) == "" {
		return fmt.Errorf("price is required")
	}
	dbg := func(format string, args ...any) {
		if os.Getenv("GPLAY_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "[SetAppPrice] "+format+"\n", args...) // #nosec G705 -- debug log
		}
	}
	var clicked bool
	// A previously staged draft would blend into the verification below, so
	// the wizard always starts from a clean slate.
	var staged bool
	if err := b.Eval(ctx, pricingStagedScript, &staged); err != nil {
		return err
	}
	if staged {
		if err := DiscardAppPricing(ctx, b); err != nil {
			return fmt.Errorf("discarding the previously staged pricing change: %w", err)
		}
		dbg("discarded a previously staged draft")
		// The console processes the discard asynchronously; starting the
		// wizard in the same breath races it (proven against the live console).
		if err := settle(ctx, pricingStepSettle); err != nil {
			return err
		}
	}
	if err := b.Eval(ctx, formHelpers+` && `+pricingHelpers+` && `+setPricingClickScript, &clicked); err != nil {
		return err
	}
	dbg("setPricingClick: %v", clicked)
	if !clicked {
		return fmt.Errorf(`"Set pricing" button was missing or disabled`)
	}
	if err := b.EvalUntil(ctx, pricingCountryTableReadyScript, 30*time.Second); err != nil {
		return fmt.Errorf("country selection did not open: %w", err)
	}
	if err := settle(ctx, pricingStepSettle); err != nil {
		return err
	}
	var sel string
	if err := b.Eval(ctx, pricingSelectAllScript, &sel); err != nil {
		return err
	}
	dbg("selectAll: %v", sel)
	if sel != "clicked" {
		return fmt.Errorf(`"Select all rows" control was not found`)
	}
	if err := b.EvalUntil(ctx, pricingSelectionCountedScript, 15*time.Second); err != nil {
		return fmt.Errorf("countries were not selected after Select all: %w", err)
	}
	if err := b.EvalUntil(ctx, pricingContinueEnabledScript, 15*time.Second); err != nil {
		return fmt.Errorf("the Continue button was not enabled after selecting countries: %w", err)
	}
	if err := settle(ctx, pricingStepSettle); err != nil {
		return err
	}
	if err := b.Eval(ctx, pricingContinueClickScript, &clicked); err != nil {
		return err
	}
	dbg("continueClick: %v", clicked)
	if !clicked {
		return fmt.Errorf(`"Continue" button was missing or disabled`)
	}
	if err := b.EvalUntil(ctx, priceInputReadyScript, 30*time.Second); err != nil {
		return fmt.Errorf("price dialog did not open: %w", err)
	}
	// The dialog initializes asynchronously and wipes its form about a second
	// after the input first appears; a value set before that is silently lost
	// (proven against the live console). Let the reset happen before typing.
	if err := settle(ctx, pricingDialogSettle); err != nil {
		return err
	}
	// The console is a compiled Dart app: values set via element.value plus
	// synthetic events are silently lost on submission (proven repeatedly
	// against the live console). The only reliable entry is trusted typing:
	// focus the field, insert the text as a keyboard would, verify, then
	// click Update.
	var focused bool
	if err := b.Eval(ctx, focusPriceInputScript, &focused); err != nil {
		return err
	}
	if !focused {
		return fmt.Errorf("could not focus the price input")
	}
	if err := b.InsertText(ctx, price); err != nil {
		return fmt.Errorf("typing the price: %w", err)
	}
	if err := b.EvalUntil(ctx, priceTypedHeldScript(price), pricingFormSettleTimeout); err != nil {
		return fmt.Errorf("the pricing dialog did not accept the typed price %q: %w", price, err)
	}
	// The Dart app's form model trails the DOM value even for trusted typing;
	// submitting in the same beat sends an empty price (proven against the
	// live console). Give the model a moment before clicking Update.
	if err := settle(ctx, pricingDialogSettle); err != nil {
		return err
	}
	var held bool
	if err := b.Eval(ctx, priceTypedHeldScript(price), &held); err != nil {
		return err
	}
	if !held {
		return fmt.Errorf("the price %q did not stay in the pricing form", price)
	}
	if err := b.Eval(ctx, pricingUpdateClickScript, &clicked); err != nil {
		return err
	}
	dbg("updateClick: %v", clicked)
	if !clicked {
		return fmt.Errorf(`"Update" button was missing or disabled`)
	}
	if err := b.EvalUntil(ctx, pricingDialogGoneScript, 30*time.Second); err != nil {
		return fmt.Errorf("price dialog did not close: %w", err)
	}
	dbg("dialog gone")
	if err := b.EvalUntil(ctx, pricingWizardDoneScript, pricingPostSetTimeout); err != nil {
		return fmt.Errorf("the pricing change was not staged with prices after the wizard: %w", err)
	}
	return nil
}

const submitPricingClickScript = `(() => {
  const p = window.__gplayPricing;
  const btn = p && p.saveButton();
  if (!p.enabled(btn)) return false;
  btn.click();
  return true;
})()`

const pricingSaveSettledScript = `(() => {
  const p = window.__gplayPricing;
  return !p.staged();
})()`

// pricingSaveSettledWait reinstalls the helpers on every poll: clicking Save
// can re-render or navigate the page, wiping window.__gplayPricing and making
// the bare script throw until the wait times out.
const pricingSaveSettledWait = pricingHelpers + ` && ` + pricingSaveSettledScript

const pricingSaveDialogPresentScript = `(() =>
  !!(document.querySelector('.pane.modal.visible') ||
    [...document.querySelectorAll('[role=dialog], [debug-id=console-dialog-popup]')].find(d => d.getClientRects().length))
)()`

const pricingSaveConfirmScript = `(() => {
  const d = document.querySelector('.pane.modal.visible') ||
    [...document.querySelectorAll('[role=dialog], [debug-id=console-dialog-popup]')].find(d => d.getClientRects().length);
  if (!d) return 'none';
  const btn = [...d.querySelectorAll('button')].find(b =>
    !b.disabled && /save|confirm|yes|^ok$|got it|send/i.test(b.textContent));
  if (!btn) return 'no-button';
  btn.click();
  return 'confirmed';
})()`

// SubmitAppPricing clicks "Save changes", confirms the save dialog when one
// appears, and waits for the staged draft to be consumed. Callers must reload
// and verify the persisted state.
func SubmitAppPricing(ctx context.Context, b *Browser, timeout time.Duration) error {
	// Same wiring race as the wizard: the click must not land in the same
	// breath as the bar's activation.
	if err := settle(ctx, pricingStepSettle); err != nil {
		return err
	}
	var clicked bool
	if err := b.Eval(ctx, formHelpers+` && `+pricingHelpers+` && `+submitPricingClickScript, &clicked); err != nil {
		return err
	}
	if !clicked {
		return fmt.Errorf(`"Save changes" button was missing or disabled`)
	}
	// The confirmation dialog appears asynchronously, or not at all.
	_ = b.EvalUntil(ctx, pricingSaveDialogPresentScript, 10*time.Second) //nolint:errcheck // absence is fine
	var confirm string
	if err := b.Eval(ctx, pricingSaveConfirmScript, &confirm); err != nil {
		return err
	}
	if confirm == "no-button" {
		return fmt.Errorf("the save confirmation dialog has no recognizable confirm button; nothing was confirmed")
	}
	if err := b.EvalUntil(ctx, pricingSaveSettledWait, timeout); err != nil {
		return fmt.Errorf("saving the price did not complete: %w", err)
	}
	return nil
}

const discardPricingClickScript = `(() => {
  const popup = document.querySelector('[debug-id=console-dialog-popup]');
  if (popup && popup.getClientRects().length) {
    const btn = [...popup.querySelectorAll('button')].find(b => /cancel|close/i.test(b.textContent));
    if (btn) { btn.click(); return 'dialog'; }
  }
  const d = window.__gplayPricing.discardButton();
  if (d) { d.click(); return 'bar'; }
  return '';
})()`

// DiscardAppPricing leaves any open pricing editor without saving, dropping a
// staged pricing draft. It is a no-op when nothing is open or staged.
func DiscardAppPricing(ctx context.Context, b *Browser) error {
	var result string
	if err := b.Eval(ctx, formHelpers+` && `+pricingHelpers+` && `+discardPricingClickScript, &result); err != nil {
		return err
	}
	if result == "" {
		return nil
	}
	if result == "dialog" {
		if err := b.EvalUntil(ctx, pricingDialogGoneScript, 15*time.Second); err != nil {
			return fmt.Errorf("discarding price edits did not complete: %w", err)
		}
		return nil
	}
	if err := b.EvalUntil(ctx, pricingUnstagedScript, 15*time.Second); err != nil {
		return fmt.Errorf("discarding the staged pricing change did not complete: %w", err)
	}
	return nil
}
