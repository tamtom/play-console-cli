package webdriver

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

func appSettingsURL(developerID, appID, account string) string {
	u := "https://play.google.com/console/developers/" +
		url.PathEscape(strings.TrimSpace(developerID)) + "/app/" +
		url.PathEscape(strings.TrimSpace(appID)) + "/store-settings"
	if account != "" {
		u += "?" + url.Values{
			"authuser": {strings.TrimSpace(account)},
			"hl":       {"en"},
		}.Encode()
	}
	return u
}

const appSettingsHelpers = `
(() => {
  const g = window.__gplay;
  const dialog = () => [...document.querySelectorAll('[role=dialog]')].find(d =>
    d.getClientRects().length > 0 &&
    g.norm((d.querySelector('[debug-id=mega-overlay-header-text]') || {}).textContent) === 'app category');
  const dropdown = id => {
    const d = dialog();
    return d && d.querySelector('material-dropdown-select[debug-id="' + id + '"]');
  };
  const value = id => {
    const el = dropdown(id);
    return ((el && el.querySelector('.button-text')) || {}).textContent || '';
  };
  const kind = () => {
    const value_ = g.norm(value('type-dropdown'));
    return value_ === 'app' || value_ === 'game' ? value_ : '';
  };
  const category = () => {
    const value_ = value('category-dropdown').replace(/\s+/g, ' ').trim();
    return g.norm(value_) === 'select a category' ? '' : value_;
  };
  const save = () => {
    const d = dialog();
    const bar = d && d.querySelector(
      'publishing-bottom-bar[debug-id=app-category-publishing-bottom-bar]');
    return bar && bar.querySelector('button[debug-id=main-button]');
  };
  window.__gplaySettings = { dialog, dropdown, value, kind, category, save };
  return true;
})()
`

const openAppSettingsEditorScript = `(() => {
  const edit = document.querySelector(
    'console-button[debug-id=edit-app-category-section-button] button'
  );
  if (!edit || edit.disabled || edit.getAttribute('aria-disabled') === 'true') return false;
  edit.click();
  return true;
})()`

// OpenAppSettings opens the requested app's App category editor. It does not
// change any values.
func OpenAppSettings(ctx context.Context, b *Browser, developerID, appID, account string) error {
	if strings.TrimSpace(developerID) == "" {
		return fmt.Errorf("developer ID is required")
	}
	if strings.TrimSpace(appID) == "" {
		return fmt.Errorf("app ID is required")
	}
	if strings.TrimSpace(account) == "" {
		return fmt.Errorf("browser account email is required")
	}
	if err := b.Navigate(ctx, appSettingsURL(developerID, appID, account)); err != nil {
		return err
	}
	expectedPath := "/developers/" + url.PathEscape(strings.TrimSpace(developerID)) +
		"/app/" + url.PathEscape(strings.TrimSpace(appID)) + "/store-settings"
	ready := fmt.Sprintf(`location.pathname.endsWith(%s) && !!document.querySelector(
	  'console-button[debug-id=edit-app-category-section-button] button'
	)`, jsString(expectedPath))
	if err := b.EvalUntil(ctx, ready, 60*time.Second); err != nil {
		return fmt.Errorf("app Store settings did not load (is the gplay browser profile signed in?): %w", err)
	}
	var opened bool
	if err := b.Eval(ctx, openAppSettingsEditorScript, &opened); err != nil {
		return err
	}
	if !opened {
		return fmt.Errorf("the App category Edit button was missing or disabled")
	}
	editorReady := formHelpers + ` && ` + appSettingsHelpers +
		` && !!window.__gplaySettings.dropdown('type-dropdown')` +
		` && !!window.__gplaySettings.dropdown('category-dropdown')`
	if err := b.EvalUntil(ctx, editorReady, 30*time.Second); err != nil {
		return fmt.Errorf("the App category editor did not open: %w", err)
	}
	return nil
}

// AppSettingsState is the App category state managed by this driver.
type AppSettingsState struct {
	Kind      string `json:"kind"`
	Category  string `json:"category"`
	CanSubmit bool   `json:"canSubmit"`
}

const readAppSettingsScript = `(() => {
  const s = window.__gplaySettings;
  const save = s && s.save();
  return {
    kind: s ? s.kind() : '',
    category: s ? s.category() : '',
    canSubmit: !!(save && !save.disabled && save.getAttribute('aria-disabled') !== 'true'),
  };
})()`

// ReadAppSettings reads the current editor values and Save state.
func ReadAppSettings(ctx context.Context, b *Browser) (*AppSettingsState, error) {
	if err := b.Eval(ctx, formHelpers+` && `+appSettingsHelpers, nil); err != nil {
		return nil, err
	}
	var state AppSettingsState
	if err := b.Eval(ctx, readAppSettingsScript, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func setDropdownScript(debugID, value string) string {
	missing := "category control not found"
	if debugID == "type-dropdown" {
		missing = "application type control not found"
	}
	return fmt.Sprintf(`(() => {
  const s = window.__gplaySettings;
  const dropdown = s && s.dropdown(%s);
  const button = dropdown && dropdown.querySelector('[role=button]');
  if (!button) return %s;
  if (window.__gplay.norm(s.value(%s)) === window.__gplay.norm(%s)) return 'selected';
  button.click();
  return 'opened';
})()`, jsString(debugID), jsString(missing), jsString(debugID), jsString(value))
}

func dropdownOptionScript(value string, click bool) string {
	action := "return !!option"
	if click {
		action = "if (!option) return false; option.click(); return true"
	}
	return fmt.Sprintf(`(() => {
  const norm = window.__gplay.norm;
  const desired = norm(%s);
  const option = [...document.querySelectorAll('[role=option]')].find(el =>
    el.getClientRects().length > 0 &&
    el.getAttribute('aria-disabled') !== 'true' &&
    norm(el.textContent) === desired);
  %s;
})()`, jsString(value), action)
}

func dropdownHasValueScript(debugID, value string) string {
	return fmt.Sprintf(`(() => {
  const s = window.__gplaySettings;
  return !!s && window.__gplay.norm(s.value(%s)) === window.__gplay.norm(%s);
})()`, jsString(debugID), jsString(value))
}

func setDropdown(ctx context.Context, b *Browser, debugID, label, value string) error {
	var result string
	if err := b.Eval(ctx, setDropdownScript(debugID, value), &result); err != nil {
		return err
	}
	switch result {
	case "selected":
		return nil
	case "opened":
		if err := b.EvalUntil(ctx, dropdownOptionScript(value, false), 10*time.Second); err != nil {
			return fmt.Errorf("waiting for %s option %q: %w", label, value, err)
		}
		var clicked bool
		if err := b.Eval(ctx, dropdownOptionScript(value, true), &clicked); err != nil {
			return err
		}
		if !clicked {
			return fmt.Errorf("%s option %q disappeared before it could be selected", label, value)
		}
		if err := b.EvalUntil(ctx, dropdownHasValueScript(debugID, value), 10*time.Second); err != nil {
			return fmt.Errorf("selecting %s %q: %w", label, value, err)
		}
		return nil
	default:
		return fmt.Errorf("setting %s: %s", label, result)
	}
}

// SetAppClassification selects the application type and its required category
// without saving.
func SetAppClassification(ctx context.Context, b *Browser, kind, category string) error {
	if err := setDropdown(ctx, b, "type-dropdown", "application type", kind); err != nil {
		return err
	}
	return setDropdown(ctx, b, "category-dropdown", "category", category)
}

const submitAppSettingsScript = `(() => {
  const s = window.__gplaySettings;
  const save = s && s.save();
  if (!save || save.disabled || save.getAttribute('aria-disabled') === 'true') return false;
  save.click();
  return true;
})()`

const appSettingsSaveSettledScript = `(() => {
  const s = window.__gplaySettings;
  const save = s && s.save();
  return !!s && (!s.dialog() ||
    !!(save && (save.disabled || save.getAttribute('aria-disabled') === 'true')));
})()`

// SubmitAppSettings saves App category and waits for the editor to settle.
// Callers must reload and verify the persisted values.
func SubmitAppSettings(ctx context.Context, b *Browser, timeout time.Duration) error {
	var clicked bool
	if err := b.Eval(ctx, submitAppSettingsScript, &clicked); err != nil {
		return err
	}
	if !clicked {
		return fmt.Errorf(`"Save" button was missing or disabled`)
	}
	if err := b.EvalUntil(ctx, appSettingsSaveSettledScript, timeout); err != nil {
		return fmt.Errorf("app settings update did not complete: %w", err)
	}
	return nil
}
