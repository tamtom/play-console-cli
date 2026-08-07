package webdriver

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func promotionsURL(developerID, appID, account string) string {
	return consoleAppURL(developerID, appID, account, "promotions")
}

func promotionCreateURL(developerID, appID, account string) string {
	return consoleAppURL(developerID, appID, account, "promotions/create")
}

// Promotion is one promo-code campaign shown by Play Console.
type Promotion struct {
	Name          string `json:"name"`
	Type          string `json:"type,omitempty"`
	CodesRedeemed string `json:"codesRedeemed,omitempty"`
	StartDate     string `json:"startDate,omitempty"`
	EndDate       string `json:"endDate,omitempty"`
	Status        string `json:"status,omitempty"`
}

// PromotionsState is the promo-code landing page state.
type PromotionsState struct {
	Promotions              []Promotion `json:"promotions"`
	TermsAcceptanceRequired bool        `json:"termsAcceptanceRequired"`
}

// PaidAppPromotionForm creates generated, one-time-use codes for a paid app.
type PaidAppPromotionForm struct {
	Name      string
	StartDate string // YYYY-MM-DD, interpreted by the English Console UI in GMT.
	EndDate   string // YYYY-MM-DD, interpreted by the English Console UI in GMT.
	CodeCount int
}

const promotionHelpers = `
(() => {
  const clean = s => (s || '').replace(/\s+/g, ' ').trim();
  const visible = e => !!(e && e.getClientRects().length);
  const unwrap = e => e && (e.matches('button, input, [role=button], [role=radio]')
    ? e : e.querySelector('button, input, [role=button], [role=radio]'));
  const button = id => unwrap(document.querySelector('[debug-id="' + id + '"]'));
  const input = id => {
    const e = document.querySelector('[debug-id="' + id + '"]');
    return e && (e.matches('input') ? e : e.querySelector('input'));
  };
  const isoDate = s => {
    const value = clean(s);
    const pad = n => String(n).padStart(2, '0');
    let match = value.match(/^(\d{4})-(\d{1,2})-(\d{1,2})(?:\D|$)/);
    if (match) return match[1] + '-' + pad(match[2]) + '-' + pad(match[3]);
    match = value.match(/^([A-Za-z]+)\s+(\d{1,2}),\s*(\d{4})(?:\D|$)/);
    if (match) {
      const month = ['jan', 'feb', 'mar', 'apr', 'may', 'jun',
        'jul', 'aug', 'sep', 'oct', 'nov', 'dec'].indexOf(match[1].slice(0, 3).toLowerCase()) + 1;
      if (month) return match[3] + '-' + pad(month) + '-' + pad(match[2]);
    }
    match = value.match(/^(\d{1,2})\/(\d{1,2})\/(\d{4})(?:\D|$)/);
    return match ? match[3] + '-' + pad(match[1]) + '-' + pad(match[2]) : '';
  };
  const date = id => {
    const field = document.querySelector('[debug-id="' + id + '"]');
    const button = field && field.querySelector('material-datepicker [aria-haspopup=dialog]');
    return isoDate(button && (button.innerText || button.textContent));
  };
  const paidApp = () => {
    const group = document.querySelector('[debug-id=reward-type]');
    if (!group) return null;
    const option = [...group.querySelectorAll('label, material-radio, [role=radio]')].find(e =>
      /^paid app(?:\s|$)/i.test(clean(e.getAttribute('aria-label') || e.innerText || e.textContent)));
    return unwrap(option);
  };
  const selected = e => !!(e && (e.checked === true || e.getAttribute('aria-checked') === 'true'));
  window.__gplayPromotion = { clean, visible, button, input, isoDate, date, paidApp, selected };
  return true;
})()
`

const promotionTermsModalVisibleScript = `(() => {
  const h = document.querySelector('[debug-id=mega-overlay-header-text]');
  return !!(h && h.getClientRects().length && /promo codes terms of service/i.test(h.textContent));
})()`

var promotionTermsModalSettle = 3 * time.Second

func promotionTermsModalVisible(ctx context.Context, b *Browser) (bool, error) {
	if err := settle(ctx, promotionTermsModalSettle); err != nil {
		return false, err
	}
	var required bool
	if err := b.Eval(ctx, promotionTermsModalVisibleScript, &required); err != nil {
		return false, err
	}
	return required, nil
}

func promotionsReadyExpr() string {
	return `location.pathname.endsWith('/promotions') && !!document.querySelector(` +
		`'[debug-id=create-promotion-button], [debug-id=promotions-table], [debug-id=empty-state], [debug-id=mega-overlay-header-text]')`
}

func promotionCreateReadyExpr() string {
	return `location.pathname.endsWith('/promotions/create') && !!document.querySelector(` +
		`'[debug-id=name-input], [debug-id=mega-overlay-header-text]')`
}

const readPromotionsScript = promotionHelpers + ` && (() => {
  const p = window.__gplayPromotion;
  const clean = p.clean;
  const table = document.querySelector('[debug-id=promotions-table]');
  const promotions = table ? [...table.querySelectorAll('[role=row]')].map(row =>
    [...row.querySelectorAll('[role=gridcell]')].map(cell => clean(cell.innerText)).filter(Boolean)
  ).filter(cells => cells.length && !/^promotion name$/i.test(cells[0])).map(cells => ({
    name: cells[0] || '',
    type: cells[1] || '',
    codesRedeemed: cells[2] || '',
    startDate: p.isoDate(cells[3]) || cells[3] || '',
    endDate: p.isoDate(cells[4]) || cells[4] || '',
    status: cells[5] || '',
  })) : [];
	return {promotions};
})()`

// ReadPromotions opens the promo-code page and returns all visible campaigns.
func ReadPromotions(ctx context.Context, b *Browser, developerID, appID, account string) (*PromotionsState, error) {
	if strings.TrimSpace(developerID) == "" || strings.TrimSpace(appID) == "" {
		return nil, fmt.Errorf("developer ID and app ID are required")
	}
	if err := b.Navigate(ctx, promotionsURL(developerID, appID, account)); err != nil {
		return nil, err
	}
	if err := b.EvalUntil(ctx, promotionsReadyExpr(), 60*time.Second); err != nil {
		return nil, fmt.Errorf("the Promo codes page did not load (is the gplay browser profile signed in?): %w", err)
	}
	var state PromotionsState
	if err := b.Eval(ctx, readPromotionsScript, &state); err != nil {
		return nil, err
	}
	if state.Promotions == nil {
		state.Promotions = []Promotion{}
	}
	return &state, nil
}

func focusPromotionInputScript(id string) string {
	return promotionHelpers + fmt.Sprintf(` && (() => {
  const input = window.__gplayPromotion.input(%s);
  if (!input || input.disabled) return false;
  input.focus();
  input.setSelectionRange(0, (input.value || '').length);
  return document.activeElement === input;
})()`, jsString(id))
}

func typePromotionInput(ctx context.Context, b *Browser, id, value string) error {
	var focused bool
	if err := b.Eval(ctx, focusPromotionInputScript(id), &focused); err != nil {
		return err
	}
	if !focused {
		return fmt.Errorf("promo-code %s input was missing or disabled", id)
	}
	if err := b.InsertText(ctx, value); err != nil {
		return err
	}
	want := promotionHelpers + fmt.Sprintf(` && (() => {
  const input = window.__gplayPromotion.input(%s);
  return !!input && input.value === %s;
})()`, jsString(id), jsString(value))
	if err := b.EvalUntil(ctx, want, 15*time.Second); err != nil {
		return fmt.Errorf("promo-code %s input did not accept %q: %w", id, value, err)
	}
	return nil
}

func promotionDateMatchesScript(id, date string) string {
	return promotionHelpers + fmt.Sprintf(` && window.__gplayPromotion.date(%s) === %s`, jsString(id), jsString(date))
}

func setPromotionDate(ctx context.Context, b *Browser, id string, date time.Time) error {
	open := fmt.Sprintf(`(() => {
  const field = document.querySelector('[debug-id=%s]');
  const button = field && [...field.querySelectorAll('button, [role=button]')].find(b =>
    b.getClientRects().length && /select a date/i.test(b.getAttribute('aria-label') || b.textContent));
  if (!button || button.disabled) return false;
  button.click();
  return true;
})()`, jsString(id))
	var opened bool
	if err := b.Eval(ctx, open, &opened); err != nil {
		return err
	}
	if !opened {
		return fmt.Errorf("promo-code %s date picker was missing or disabled", id)
	}
	const dateInputReady = `(() => !![...document.querySelectorAll('input[aria-label="Enter date"]')]
  .find(input => input.getClientRects().length))()`
	if err := b.EvalUntil(ctx, dateInputReady, 15*time.Second); err != nil {
		return fmt.Errorf("promo-code %s date input did not open: %w", id, err)
	}
	const focusDate = `(() => {
  const input = [...document.querySelectorAll('input[aria-label="Enter date"]')]
    .find(input => input.getClientRects().length);
  if (!input) return false;
  input.focus();
  input.setSelectionRange(0, (input.value || '').length);
  return document.activeElement === input;
})()`
	var focused bool
	if err := b.Eval(ctx, focusDate, &focused); err != nil {
		return err
	}
	if !focused {
		return fmt.Errorf("promo-code %s date input could not be focused", id)
	}
	if err := b.InsertText(ctx, date.Format("01/02/2006")); err != nil {
		return err
	}
	if err := b.PressEnter(ctx); err != nil {
		return err
	}
	if err := b.EvalUntil(ctx, promotionDateMatchesScript(id, date.Format("2006-01-02")), 15*time.Second); err != nil {
		return fmt.Errorf("promo-code %s date did not accept %s: %w", id, date.Format("2006-01-02"), err)
	}
	return nil
}

func promotionFormReadyScript(form PaidAppPromotionForm) string {
	return promotionHelpers + fmt.Sprintf(` && (() => {
  const p = window.__gplayPromotion;
  const name = p.input('name-input');
  const count = p.input('codes-limit-input');
  const paid = p.paidApp();
  const save = p.button('save-button');
  return !!(name && name.value === %s && count && count.value === %s &&
    p.date('start-date-picker') === %s && p.date('end-date-picker') === %s &&
    p.selected(paid) && save && !save.disabled && save.getAttribute('aria-disabled') !== 'true');
})()`, jsString(form.Name), jsString(strconv.Itoa(form.CodeCount)), jsString(form.StartDate), jsString(form.EndDate))
}

const promotionSubmitScript = `(() => {
  const p = window.__gplayPromotion;
  const button = p && p.button('save-button');
  if (!button || button.disabled || button.getAttribute('aria-disabled') === 'true') return false;
  button.click();
  return true;
})()`

const promotionConfirmReadyScript = `(() => {
  const dialog = document.querySelector('.pane.modal.visible, [role=dialog]');
  return !!(dialog && /create promo code\?/i.test(dialog.innerText || dialog.textContent));
})()`

const promotionConfirmScript = `(() => {
  const dialog = document.querySelector('.pane.modal.visible, [role=dialog]');
  const button = dialog && [...dialog.querySelectorAll('button')].find(b =>
    b.getClientRects().length && !b.disabled && /^create$/i.test(b.textContent.trim()));
  if (!button) return false;
  button.click();
  return true;
})()`

// CreatePaidAppPromotion creates generated one-time promo codes for a paid app.
// Legal terms are never accepted: the user must review them in Play Console.
func CreatePaidAppPromotion(ctx context.Context, b *Browser, developerID, appID, account string, termsAccepted bool, form PaidAppPromotionForm, timeout time.Duration) (*PromotionsState, error) {
	if strings.TrimSpace(developerID) == "" || strings.TrimSpace(appID) == "" {
		return nil, fmt.Errorf("developer ID and app ID are required")
	}
	start, err := time.Parse("2006-01-02", form.StartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start date: %w", err)
	}
	end, err := time.Parse("2006-01-02", form.EndDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end date: %w", err)
	}
	form.Name = strings.TrimSpace(form.Name)
	if form.Name == "" || len([]rune(form.Name)) > 60 || form.CodeCount < 1 || form.CodeCount > 500 ||
		end.Before(start) || end.After(start.AddDate(1, 0, 0)) {
		return nil, fmt.Errorf("invalid paid-app promotion form")
	}
	if !termsAccepted {
		return nil, fmt.Errorf("the Promo codes Terms of Service must be reviewed and accepted manually in Play Console; no promotion was created")
	}
	if err := b.Navigate(ctx, promotionCreateURL(developerID, appID, account)); err != nil {
		return nil, err
	}
	if err := b.EvalUntil(ctx, promotionCreateReadyExpr(), 60*time.Second); err != nil {
		return nil, fmt.Errorf("the Create promo code form did not load (is the gplay browser profile signed in?): %w", err)
	}
	// A visible modal vetoes the positive API check in case Google published a
	// newer policy between the check and this form load. Never click Accept.
	terms, err := promotionTermsModalVisible(ctx, b)
	if err != nil {
		return nil, err
	}
	if terms {
		return nil, fmt.Errorf("the Promo codes Terms of Service must be reviewed and accepted manually in Play Console; no promotion was created")
	}
	if err := typePromotionInput(ctx, b, "name-input", form.Name); err != nil {
		return nil, err
	}
	if err := setPromotionDate(ctx, b, "start-date-picker", start); err != nil {
		return nil, err
	}
	if err := setPromotionDate(ctx, b, "end-date-picker", end); err != nil {
		return nil, err
	}
	var selected bool
	if err := b.Eval(ctx, promotionHelpers+` && (() => {
  const radio = window.__gplayPromotion.paidApp();
  if (!radio || radio.disabled || radio.getAttribute('aria-disabled') === 'true') return false;
  if (!window.__gplayPromotion.selected(radio)) radio.click();
  return true;
})()`, &selected); err != nil {
		return nil, err
	}
	if !selected {
		return nil, fmt.Errorf("paid app promotion is unavailable for this app")
	}
	if err := typePromotionInput(ctx, b, "codes-limit-input", strconv.Itoa(form.CodeCount)); err != nil {
		return nil, err
	}
	if err := b.EvalUntil(ctx, promotionFormReadyScript(form), 30*time.Second); err != nil {
		return nil, fmt.Errorf("the promo-code form did not match the request: %w", err)
	}
	var clicked bool
	if err := b.Eval(ctx, promotionSubmitScript, &clicked); err != nil {
		return nil, err
	}
	if !clicked {
		return nil, fmt.Errorf("the Create promo code button was missing or disabled")
	}
	if err := b.EvalUntil(ctx, promotionConfirmReadyScript, 15*time.Second); err != nil {
		return nil, fmt.Errorf("promo-code confirmation did not open: %w", err)
	}
	if err := b.Eval(ctx, promotionConfirmScript, &clicked); err != nil {
		return nil, err
	}
	if !clicked {
		return nil, fmt.Errorf("promo-code confirmation had no enabled Create button")
	}
	expectedPath := "/developers/" + url.PathEscape(strings.TrimSpace(developerID)) +
		"/app/" + url.PathEscape(strings.TrimSpace(appID)) + "/promotions"
	settled := fmt.Sprintf(`location.pathname.endsWith(%s)`, jsString(expectedPath))
	if err := b.EvalUntil(ctx, settled, timeout); err != nil {
		return nil, fmt.Errorf("promo-code creation did not complete: %w", err)
	}
	state, err := ReadPromotions(ctx, b, developerID, appID, account)
	if err != nil {
		return nil, err
	}
	for _, promotion := range state.Promotions {
		if promotion.Name == form.Name && strings.EqualFold(strings.TrimSpace(promotion.Type), "Paid app") &&
			promotion.StartDate == form.StartDate && promotion.EndDate == form.EndDate {
			return state, nil
		}
	}
	return nil, fmt.Errorf("promo code %q does not appear with the requested reward and dates after creation", form.Name)
}
