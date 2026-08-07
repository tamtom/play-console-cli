package webdriver

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func contentRatingURL(developerID, appID, account string) string {
	return consoleAppURL(developerID, appID, account, "app-content/content-rating-overview")
}

// AppliedRating is one regional authority rating shown by Play Console.
type AppliedRating struct {
	Authority string `json:"authority,omitempty"`
	Rating    string `json:"rating"`
}

// ContentRatingState is the current IARC result and any unfinished draft.
type ContentRatingState struct {
	IARCStatus    string          `json:"iarcStatus,omitempty"`
	EmailAddress  string          `json:"emailAddress,omitempty"`
	CertificateID string          `json:"certificateId,omitempty"`
	SubmittedAt   string          `json:"submittedAt,omitempty"`
	DraftStatus   string          `json:"draftStatus,omitempty"`
	CanStart      bool            `json:"canStart"`
	Ratings       []AppliedRating `json:"ratings"`
}

func contentRatingReadyExpr() string {
	return `location.pathname.endsWith('/app-content/content-rating-overview') && ` +
		`/content ratings/i.test(document.body.innerText || '') && ` +
		`!!document.querySelector('[debug-id=start-new-questionnaire-button], [debug-id=iarc-status]')`
}

const readContentRatingScript = `(() => {
  const clean = s => (s || '').replace(/\s+/g, ' ').trim();
  const text = id => clean((document.querySelector('[debug-id="' + id + '"]') || {}).innerText);
  const value = id => {
    const field = document.querySelector('[debug-id="' + id + '"]');
    return clean((field && field.querySelector('[field-value]') || field || {}).innerText);
  };
  const body = clean(document.body.innerText);
  const startWrap = document.querySelector('[debug-id=start-new-questionnaire-button]');
  const start = startWrap && (startWrap.matches('button') ? startWrap : (startWrap.querySelector('button') || startWrap));
  const ratings = [...document.querySelectorAll('[debug-id=rating-icon]')].map(icon => {
    const parts = clean(icon.getAttribute('alt')).split(',').map(clean).filter(Boolean);
    return { rating: parts.shift() || '', authority: parts.join(', ') };
  }).filter(r => r.rating);
  return {
	iarcStatus: value('iarc-status'),
    emailAddress: text('email-address-value'),
    certificateId: value('iarc-id'),
    submittedAt: value('submitted-date'),
    draftStatus: /incomplete questionnaire/i.test(body) ? 'In progress' : '',
    canStart: !!(start && !start.disabled && start.getAttribute('aria-disabled') !== 'true'),
    ratings,
  };
})()`

// ReadContentRating reads the app's submitted IARC rating and draft state.
func ReadContentRating(ctx context.Context, b *Browser, developerID, appID, account string) (*ContentRatingState, error) {
	if strings.TrimSpace(developerID) == "" || strings.TrimSpace(appID) == "" {
		return nil, fmt.Errorf("developer ID and app ID are required")
	}
	if err := b.Navigate(ctx, contentRatingURL(developerID, appID, account)); err != nil {
		return nil, err
	}
	if err := b.EvalUntil(ctx, contentRatingReadyExpr(), 60*time.Second); err != nil {
		return nil, fmt.Errorf("the Content ratings page did not load (is the gplay browser profile signed in?): %w", err)
	}
	var state ContentRatingState
	if err := b.Eval(ctx, readContentRatingScript, &state); err != nil {
		return nil, err
	}
	state.IARCStatus = normalizeIARCStatus(state.IARCStatus)
	if state.Ratings == nil {
		state.Ratings = []AppliedRating{}
	}
	return &state, nil
}

func normalizeIARCStatus(status string) string {
	status = strings.Join(strings.Fields(status), " ")
	lower := strings.ToLower(status)
	switch {
	case strings.HasPrefix(lower, "check_circle"):
		return "Completed"
	case strings.HasPrefix(lower, "timelapse"):
		return "In progress"
	case strings.HasSuffix(lower, " view details"):
		return strings.TrimSpace(status[:len(status)-len(" view details")])
	default:
		return status
	}
}
