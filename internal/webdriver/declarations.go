package webdriver

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func appContentURL(developerID, appID, account, page string) string {
	return consoleAppURL(developerID, appID, account, "app-content/"+page)
}

// Declaration is one App content policy declaration's summary state.
type Declaration struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	LastEdited  string `json:"lastEdited,omitempty"`
	NeedsAction bool   `json:"needsAction"`
}

// DeclarationsState is the App content overview: declarations needing
// attention plus the completed (actioned) ones.
type DeclarationsState struct {
	NeedAttention []Declaration `json:"needAttention"`
	Actioned      []Declaration `json:"actioned"`
}

// appContentReadyScript waits for the App content overview's tab strip,
// appContentLoadedScript for the Need attention tab's content, and
// actionedDeclarationsReadyScript for the Actioned tab's sections.
const appContentReadyScript = formHelpers + ` && !!document.querySelector('[debug-id=tab-strip]') &&
  !! [...document.querySelectorAll('[role=tab]')].find(t => /actioned/i.test(t.textContent))`

const appContentLoadedScript = `(/all caught up/i.test(document.body.innerText || '')) ||
  !!document.querySelector('[debug-id^=policy-declaration-id-][debug-id$=-section]')`

const actionedDeclarationsReadyScript = `!!document.querySelector('[debug-id^=policy-declaration-id-][debug-id$=-section]')`

const openActionedTabScript = `(() => {
  const tab = [...document.querySelectorAll('[role=tab]')].find(t => /actioned/i.test(t.textContent));
  if (!tab) return false;
  tab.click();
  return true;
})()`

const readDeclarationsScript = `(() => {
  const clean = s => (s || '').replace(/\s+/g, ' ').trim();
  const sections = sel => [...document.querySelectorAll(sel)].map(s => {
    const key = s.getAttribute('debug-id')
      .replace('policy-declaration-id-', '').replace('-section', '');
    const labeled = s.querySelector('[aria-label^="Show details for "]');
    const title = labeled
      ? labeled.getAttribute('aria-label').replace('Show details for ', '')
      : clean((s.querySelector('[debug-id=title-text]') || {}).innerText);
    const text = clean(s.innerText);
    const m = /(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) \d{1,2}, \d{4}/.exec(text);
    return { key, title, lastEdited: m ? m[0] : '' };
  });
  const actioned = sections('[debug-id^=policy-declaration-id-][debug-id$=-section]');
  return { actioned };
})()`

const readAttentionScript = `(() => {
  const clean = s => (s || '').replace(/\s+/g, ' ').trim();
  const text = clean((document.querySelector('[debug-id=content]') || document.body).innerText);
  if (/all caught up/i.test(text)) return [];
  const rows = [...document.querySelectorAll('[debug-id^=policy-declaration-id-][debug-id$=-section]')].map(s => {
    const key = s.getAttribute('debug-id')
      .replace('policy-declaration-id-', '').replace('-section', '');
    const title = clean(
      (s.querySelector('[debug-id=title-text]') || {}).innerText ||
      (s.querySelector('[debug-id=section-title]') || {}).innerText);
    return { key, title: title || key, needsAction: true };
  });
  return rows;
})()`

// ReadDeclarations opens the App content overview and reads the completed
// declarations plus anything needing attention.
func ReadDeclarations(ctx context.Context, b *Browser, developerID, appID, account string) (*DeclarationsState, error) {
	if strings.TrimSpace(developerID) == "" || strings.TrimSpace(appID) == "" {
		return nil, fmt.Errorf("developer ID and app ID are required")
	}
	if err := b.Navigate(ctx, appContentURL(developerID, appID, account, "overview")); err != nil {
		return nil, err
	}
	if err := b.EvalUntil(ctx, appContentReadyScript, 60*time.Second); err != nil {
		return nil, fmt.Errorf("the App content page did not load (is the gplay browser profile signed in?): %w", err)
	}
	// The Need attention tab is the default view; read it before switching.
	if err := b.EvalUntil(ctx, appContentLoadedScript, 30*time.Second); err != nil {
		return nil, fmt.Errorf("declarations did not load: %w", err)
	}
	var attention []Declaration
	if err := b.Eval(ctx, readAttentionScript, &attention); err != nil {
		return nil, err
	}
	var opened bool
	if err := b.Eval(ctx, openActionedTabScript, &opened); err != nil {
		return nil, err
	}
	if !opened {
		return nil, fmt.Errorf("no Actioned tab was found on the App content page")
	}
	if err := b.EvalUntil(ctx, actionedDeclarationsReadyScript, 30*time.Second); err != nil {
		return nil, fmt.Errorf("completed declarations did not load: %w", err)
	}
	var wire struct {
		Actioned []Declaration `json:"actioned"`
	}
	if err := b.Eval(ctx, readDeclarationsScript, &wire); err != nil {
		return nil, err
	}
	state := &DeclarationsState{Actioned: wire.Actioned}
	for i := range attention {
		attention[i].NeedsAction = true
	}
	state.NeedAttention = attention
	return state, nil
}

// declarationFormReady waits for a simple declaration form: radios or the URL
// input, plus the publishing bottom bar.
const declarationFormHelpers = `
(() => {
  const bar = () => document.querySelector('[debug-id=publishing-bottom-bar]');
  const mainButton = () => {
    const b = bar();
    return b && (b.querySelector('[debug-id=main-button] button') || b.querySelector('[debug-id=main-button]'));
  };
  const radios = () => [...document.querySelectorAll('[role=radio], input[type=radio]')]
    .filter(r => r.getClientRects().length);
  const enabled = b => !!(b && !b.disabled && b.getAttribute('aria-disabled') !== 'true');
  window.__gplayDecl = { bar, mainButton, radios, enabled };
  return true;
})()
`

const declarationFormReadyScript = declarationFormHelpers + ` && !!window.__gplayDecl.bar()`

// radioClickScript clicks the declaration form's radio at index (0 = yes,
// 1 = no); radioCheckedScript reports whether that radio ended up selected.
func radioClickScript(index int) string {
	return declarationFormHelpers + fmt.Sprintf(` && (() => {
  const r = window.__gplayDecl.radios();
  if (r.length < 2) return 'no-radios';
  r[%d].click();
  return 'clicked';
})()`, index)
}

func radioCheckedScript(index int) string {
	return declarationFormHelpers + fmt.Sprintf(` && (() => {
  const r = window.__gplayDecl.radios();
  if (r.length < 2) return false;
  const c = r[%d].getAttribute('aria-checked');
  return c === 'true' || r[%d].checked === true;
})()`, index, index)
}

// RadioDeclarationState is a yes/no declaration form's state.
type RadioDeclarationState struct {
	Answer   string `json:"answer"` // "yes", "no", or "" when unreadable
	CanSave  bool   `json:"canSave"`
	Saveable bool   `json:"-"`
}

const readRadioDeclarationWait = declarationFormHelpers + ` && ` + readRadioDeclarationScript

const readRadioDeclarationScript = `(() => {
  const d = window.__gplayDecl;
  const radios = d.radios();
  let answer = '';
  if (radios.length >= 2) {
    const checked = i => (radios[i].getAttribute('aria-checked') ?? radios[i].checked) === true ||
      (radios[i].getAttribute('aria-checked') === 'true');
    if (checked(0)) answer = 'yes';
    else if (checked(1)) answer = 'no';
  }
  return { answer, canSave: d.enabled(d.mainButton()) };
})()`

// ReadRadioDeclaration opens a yes/no declaration form and reads the current
// answer. page is the console app-content route (e.g. "government-apps").
func ReadRadioDeclaration(ctx context.Context, b *Browser, developerID, appID, account, page string) (*RadioDeclarationState, error) {
	if err := openDeclarationForm(ctx, b, developerID, appID, account, page); err != nil {
		return nil, err
	}
	var state RadioDeclarationState
	if err := b.Eval(ctx, readRadioDeclarationWait, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func openDeclarationForm(ctx context.Context, b *Browser, developerID, appID, account, page string) error {
	if strings.TrimSpace(developerID) == "" || strings.TrimSpace(appID) == "" {
		return fmt.Errorf("developer ID and app ID are required")
	}
	if err := b.Navigate(ctx, appContentURL(developerID, appID, account, page)); err != nil {
		return err
	}
	if err := b.EvalUntil(ctx, declarationFormReadyScript, 60*time.Second); err != nil {
		return fmt.Errorf("declaration form %q did not load (is the gplay browser profile signed in?): %w", page, err)
	}
	return nil
}

// SetRadioDeclaration sets a yes/no declaration and saves it. It verifies the
// radio state before saving and re-reads the persisted answer afterwards.
// It reports whether anything actually changed: when the requested answer is
// already set, it saves nothing.
func SetRadioDeclaration(ctx context.Context, b *Browser, developerID, appID, account, page string, yes bool) (bool, error) {
	current, err := ReadRadioDeclaration(ctx, b, developerID, appID, account, page)
	if err != nil {
		return false, err
	}
	want := "no"
	if yes {
		want = "yes"
	}
	if current.Answer == want {
		return false, nil
	}
	if current.Answer == "" {
		return false, fmt.Errorf("could not read the current %s answer; nothing was changed", page)
	}
	index := 1
	if yes {
		index = 0
	}
	var result string
	if err := b.Eval(ctx, radioClickScript(index), &result); err != nil {
		return false, err
	}
	if result != "clicked" {
		return false, fmt.Errorf("the %s form has no yes/no radios to set", page)
	}
	if err := b.EvalUntil(ctx, radioCheckedScript(index), 10*time.Second); err != nil {
		return false, fmt.Errorf("selecting the answer on the %s form: %w", page, err)
	}
	if err := submitDeclarationForm(ctx, b); err != nil {
		return false, err
	}
	// Verify the persisted answer.
	saved, err := ReadRadioDeclaration(ctx, b, developerID, appID, account, page)
	if err != nil {
		return false, fmt.Errorf("verifying the saved declaration: %w", err)
	}
	if saved.Answer != want {
		return false, fmt.Errorf("the %s declaration was not saved (got %q, want %q)", page, saved.Answer, want)
	}
	return true, nil
}

const submitDeclarationClickScript = `(() => {
  const d = window.__gplayDecl;
  const btn = d && d.mainButton();
  if (!d.enabled(btn)) return false;
  btn.click();
  return true;
})()`

const submitDeclarationClickWait = declarationFormHelpers + ` && ` + submitDeclarationClickScript

const declarationSaveSettledScript = `(() => {
  const d = window.__gplayDecl;
  const bar = d.bar();
  const main = d.mainButton();
  return !bar || !!(main && (main.disabled || main.getAttribute('aria-disabled') === 'true'));
})()`

func submitDeclarationForm(ctx context.Context, b *Browser) error {
	if err := settle(ctx, 2*time.Second); err != nil {
		return err
	}
	var clicked bool
	if err := b.Eval(ctx, submitDeclarationClickWait, &clicked); err != nil {
		return err
	}
	if !clicked {
		return fmt.Errorf(`"Save" button was missing or disabled`)
	}
	if err := b.EvalUntil(ctx, declarationSaveSettledScript, 2*time.Minute); err != nil {
		return fmt.Errorf("saving the declaration did not complete: %w", err)
	}
	return nil
}

const privacyPolicyInputReadyScript = `(() => {
  const i = document.querySelector('input[aria-label="Privacy policy URL"], [debug-id=privacy-policy-url-input] input');
  return !!(i && i.getClientRects().length);
})()`

const readPrivacyPolicyURLScript = `(() => {
  const i = document.querySelector('input[aria-label="Privacy policy URL"], [debug-id=privacy-policy-url-input] input');
  return i ? i.value : '';
})()`

const focusPrivacyPolicyURLScript = `(() => {
  const i = document.querySelector('input[aria-label="Privacy policy URL"], [debug-id=privacy-policy-url-input] input');
  if (!i) return false;
  i.focus();
  i.setSelectionRange(0, (i.value || '').length);
  return document.activeElement === i;
})()`

func privacyPolicyURLHeldScript(url string) string {
	return fmt.Sprintf(`(() => {
  const i = document.querySelector('input[aria-label="Privacy policy URL"], [debug-id=privacy-policy-url-input] input');
  return !!i && i.value === %s;
})()`, jsString(url))
}

// SetPrivacyPolicyURL sets the privacy policy URL and saves it, verifying the
// persisted value afterwards. Uses trusted typing like the pricing wizard.
// It reports whether anything actually changed: when the URL is already set
// to the requested value, it saves nothing.
func SetPrivacyPolicyURL(ctx context.Context, b *Browser, developerID, appID, account, url string) (bool, error) {
	if err := openDeclarationForm(ctx, b, developerID, appID, account, "privacy-policy"); err != nil {
		return false, err
	}
	if err := b.EvalUntil(ctx, privacyPolicyInputReadyScript, 30*time.Second); err != nil {
		return false, fmt.Errorf("privacy policy URL input did not load: %w", err)
	}
	var current string
	if err := b.Eval(ctx, readPrivacyPolicyURLScript, &current); err != nil {
		return false, err
	}
	if strings.TrimSpace(current) == url {
		return false, nil
	}
	if err := settle(ctx, pricingDialogSettle); err != nil {
		return false, err
	}
	var focused bool
	if err := b.Eval(ctx, focusPrivacyPolicyURLScript, &focused); err != nil {
		return false, err
	}
	if !focused {
		return false, fmt.Errorf("could not focus the privacy policy URL input")
	}
	if err := b.InsertText(ctx, url); err != nil {
		return false, fmt.Errorf("typing the privacy policy URL: %w", err)
	}
	if err := b.EvalUntil(ctx, privacyPolicyURLHeldScript(url), 10*time.Second); err != nil {
		return false, fmt.Errorf("the privacy policy URL did not stay in the form: %w", err)
	}
	if err := submitDeclarationForm(ctx, b); err != nil {
		return false, err
	}
	// Verify the persisted URL.
	if err := openDeclarationForm(ctx, b, developerID, appID, account, "privacy-policy"); err != nil {
		return false, fmt.Errorf("reopening the privacy policy form: %w", err)
	}
	var saved string
	if err := b.Eval(ctx, readPrivacyPolicyURLScript, &saved); err != nil {
		return false, err
	}
	if saved != url {
		return false, fmt.Errorf("the privacy policy URL was not saved (got %q, want %q)", saved, url)
	}
	return true, nil
}
