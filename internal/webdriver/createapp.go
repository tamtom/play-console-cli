package webdriver

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// AppForm mirrors the console's "Create app" dialog.
type AppForm struct {
	Title       string
	PackageName string
	Language    string // BCP 47; only the console default (en-US) is driveable
	Game        bool
	Paid        bool
}

// formHelpers installs label-aware element lookup in the page. The console is
// a compiled Dart app with obfuscated class names, so elements are addressed
// by their visible/accessible labels rather than CSS structure.
const formHelpers = `
(() => {
  const norm = s => (s || '').replace(/\s+/g, ' ').trim().toLowerCase();
  const labelOf = el => {
    const aria = el.getAttribute('aria-label');
    if (aria) return aria;
    if (el.id) {
      const l = document.querySelector('label[for="' + CSS.escape(el.id) + '"]');
      if (l) return l.textContent;
    }
    const wrap = el.closest('label');
    if (wrap) return wrap.textContent;
    const row = el.closest('div');
    if (row && norm(row.textContent)) return row.textContent;
    // Some console controls (the declaration checkboxes) carry no label, id or
    // nearby text; their wording sits a few levels up. Climb until we find it.
    let n = el;
    for (let i = 0; i < 6 && n.parentElement; i++) {
      n = n.parentElement;
      if (norm(n.textContent)) return n.textContent;
    }
    return '';
  };
  window.__gplay = {
    norm: norm,
    labelOf: labelOf,
    find: (sel, text, exact) => [...document.querySelectorAll(sel)].find(el => {
      const l = norm(labelOf(el));
      return exact ? l === norm(text) : l.includes(norm(text));
    }),
    setText: (el, value) => {
      const proto = Object.getPrototypeOf(el);
      const desc = Object.getOwnPropertyDescriptor(proto, 'value');
      (desc && desc.set ? desc.set : (v => { el.value = v; })).call(el, value);
      el.dispatchEvent(new Event('input', { bubbles: true }));
      el.dispatchEvent(new Event('change', { bubbles: true }));
    },
    button: text => [...document.querySelectorAll('button, [role=button]')]
      .find(b => norm(b.textContent) === norm(text)),
  };
  return true;
})()
`

// jsString renders a Go string as a JavaScript literal. On top of JSON
// escaping it also escapes single quotes: values can land inside
// single-quoted JS contexts (e.g. attribute selectors), which JSON leaves
// unprotected, and \' is valid inside any JavaScript string literal.
func jsString(s string) string {
	b, _ := json.Marshal(s)
	return strings.ReplaceAll(string(b), "'", `\'`)
}

// createAppURL is the console's create-app route for a developer account.
func createAppURL(developerID string) string {
	return "https://play.google.com/console/developers/" + developerID + "/create-new-app"
}

// FillAppForm navigates to the create-app page and fills every field. It
// deliberately stops short of submitting so callers can verify the form (and
// so a mistake never creates an app).
func FillAppForm(ctx context.Context, b *Browser, developerID string, form AppForm) error {
	if lang := strings.TrimSpace(form.Language); lang != "" && !strings.EqualFold(lang, "en-US") {
		return fmt.Errorf("browser-driven creation currently only supports the console default language en-US, got %q", lang)
	}
	if err := b.Navigate(ctx, createAppURL(developerID)); err != nil {
		return err
	}
	// The form is rendered asynchronously by the Dart app.
	if err := b.EvalUntil(ctx, formHelpers+` && !!window.__gplay.find('input', 'app name')`, 60*time.Second); err != nil {
		return fmt.Errorf("create-app form did not load (is the profile signed in?): %w", err)
	}

	kind := "app"
	if form.Game {
		kind = "game"
	}
	pricing := "free"
	if form.Paid {
		pricing = "paid"
	}

	fill := fmt.Sprintf(`(() => {
  const g = window.__gplay;
  const name = g.find('input', 'app name');
  const pkg  = g.find('input', 'package name');
  if (!name || !pkg) return 'inputs not found';
  g.setText(name, %s);
  g.setText(pkg, %s);

  const kind = g.find('input[type=radio]', %s, true);
  if (!kind) return 'app/game radio not found';
  if (!kind.checked) kind.click();

  const price = g.find('input[type=radio]', %s, true);
  if (!price) return 'free/paid radio not found';
  if (!price.checked) price.click();

  const policies = g.find('input[type=checkbox]', 'developer program policies');
  const export_ = g.find('input[type=checkbox]', 'us export laws');
  if (!policies || !export_) return 'declaration checkboxes not found';
  if (!policies.checked) policies.click();
  if (!export_.checked) export_.click();
  return 'ok';
})()`, jsString(form.Title), jsString(form.PackageName), jsString(kind), jsString(pricing))

	var result string
	if err := b.Eval(ctx, fill, &result); err != nil {
		return err
	}
	if result != "ok" {
		return fmt.Errorf("filling create-app form: %s", result)
	}
	return nil
}

// FormState reports what the page currently holds, so a caller can verify the
// form before submitting it.
type FormState struct {
	Title       string `json:"title"`
	PackageName string `json:"packageName"`
	Game        bool   `json:"game"`
	Paid        bool   `json:"paid"`
	Policies    bool   `json:"policies"`
	Export      bool   `json:"export"`
	CanSubmit   bool   `json:"canSubmit"`
}

// ReadForm reads the current create-app form values back out of the page.
func ReadForm(ctx context.Context, b *Browser) (*FormState, error) {
	const script = `(() => {
  const g = window.__gplay;
  const radio = t => { const r = g.find('input[type=radio]', t, true); return !!(r && r.checked); };
  const check = t => { const c = g.find('input[type=checkbox]', t); return !!(c && c.checked); };
  const btn = g.button('create app');
  return {
    title: (g.find('input', 'app name') || {}).value || '',
    packageName: (g.find('input', 'package name') || {}).value || '',
    game: radio('game'),
    paid: radio('paid'),
    policies: check('developer program policies'),
    export: check('us export laws'),
    canSubmit: !!(btn && !btn.disabled && btn.getAttribute('aria-disabled') !== 'true'),
  };
})()`
	var state FormState
	if err := b.Eval(ctx, script, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// appIDFromURL pulls the numeric app ID out of a console app URL.
var appIDFromURL = regexp.MustCompile(`/app/(\d+)/`)

// SubmitAppForm clicks "Create app" and waits for the console to land on the
// new app's dashboard, returning its app ID.
func SubmitAppForm(ctx context.Context, b *Browser, timeout time.Duration) (string, error) {
	var clicked bool
	if err := b.Eval(ctx, `(() => { const b = window.__gplay.button('create app');
	  if (!b || b.disabled) return false; b.click(); return true; })()`, &clicked); err != nil {
		return "", err
	}
	if !clicked {
		return "", fmt.Errorf(`"Create app" button was missing or disabled`)
	}
	if err := b.EvalUntil(ctx, `/\/app\/\d+\//.test(location.pathname)`, timeout); err != nil {
		return "", fmt.Errorf("app creation did not complete: %w", err)
	}
	var href string
	if err := b.Eval(ctx, "location.pathname", &href); err != nil {
		return "", err
	}
	m := appIDFromURL.FindStringSubmatch(href)
	if m == nil {
		return "", fmt.Errorf("could not read the new app ID from %s", href)
	}
	return m[1], nil
}
