package webdriver

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func policyCenterURL(developerID, appID, account string) string {
	return consoleAppURL(developerID, appID, account, "policy-center")
}

// PolicyIssue is one entry on the Policy status page.
type PolicyIssue struct {
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
}

// PolicyStatus is the Policy status page state.
type PolicyStatus struct {
	State   string        `json:"state"` // "empty" when nothing is reported, "issues" otherwise
	Message string        `json:"message,omitempty"`
	Issues  []PolicyIssue `json:"issues"`
}

const policyStatusReadyScript = `!!document.querySelector('[debug-id=empty-state], [role=row]')`

const readPolicyStatusScript = `(() => {
  const clean = s => (s || '').replace(/\s+/g, ' ').trim();
  const empty = document.querySelector('[debug-id=empty-state]');
  if (empty) {
    return { state: 'empty', message: clean(empty.innerText), issues: [] };
  }
  const rows = [...document.querySelectorAll('[role=row]')]
    .map(r => [...r.querySelectorAll('[role=gridcell]')].map(c => clean(c.innerText)))
    .filter(cells => cells.length && cells[0] && !/^(issue|policy|app)$/i.test(cells[0]));
  return {
    state: rows.length ? 'issues' : 'empty',
    message: rows.length ? '' : clean((document.querySelector('[debug-id=content]') || document.body).innerText).slice(0, 300),
    issues: rows.map(cells => ({ title: cells[0], detail: cells.slice(1).join(' | ') })),
  };
})()`

// ReadPolicyStatus opens the Policy status page and reads reported issues.
// Before the first review completes, the page shows an empty state.
func ReadPolicyStatus(ctx context.Context, b *Browser, developerID, appID, account string) (*PolicyStatus, error) {
	if strings.TrimSpace(developerID) == "" || strings.TrimSpace(appID) == "" {
		return nil, fmt.Errorf("developer ID and app ID are required")
	}
	if err := b.Navigate(ctx, policyCenterURL(developerID, appID, account)); err != nil {
		return nil, err
	}
	if err := b.EvalUntil(ctx, policyStatusReadyScript, 60*time.Second); err != nil {
		return nil, fmt.Errorf("the Policy status page did not load (is the gplay browser profile signed in?): %w", err)
	}
	var status PolicyStatus
	if err := b.Eval(ctx, readPolicyStatusScript, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// managedPublishingReadyScript waits for the Publishing overview's
// managed-publishing dropdown, the entry point for both the toggle and the
// Publish action.
const managedPublishingReadyScript = formHelpers + ` && ` + publishHelpers +
	` && !!document.querySelector('[debug-id=managed-publishing-dropdown]')`

// managedPublishingChoices clicks the managed-publishing dropdown and lists or
// picks the on/off options.
const managedPublishingMenuScript = `(() => {
  const dd = document.querySelector('[debug-id=managed-publishing-dropdown]');
  const btn = dd && (dd.querySelector('button, [role=button]') || dd);
  if (!btn) return false;
  btn.click();
  return true;
})()`

const managedPublishingOptionPresentScript = `(() => {
  const opts = [...document.querySelectorAll('[role=option], [role=menuitem], mat-option')]
    .filter(o => o.getClientRects().length);
  return opts.length > 0;
})()`

func managedPublishingOptionScript(on bool) string {
	want := "turn on"
	if !on {
		want = "turn off"
	}
	return fmt.Sprintf(`(() => {
  const opts = [...document.querySelectorAll('[role=option], [role=menuitem], mat-option')]
    .filter(o => o.getClientRects().length);
  const opt = opts.find(o => /%s managed publishing/i.test(o.textContent)) ||
    opts.find(o => /%s/i.test(o.textContent));
  if (!opt) return null;
  opt.click();
  return true;
})()`, want, want)
}

// SetManagedPublishing toggles managed publishing on the Publishing overview
// and verifies the new state. Requires a confirmation dialog when the console
// shows one.
func SetManagedPublishing(ctx context.Context, b *Browser, developerID, appID, account string, on bool) error {
	if strings.TrimSpace(developerID) == "" || strings.TrimSpace(appID) == "" {
		return fmt.Errorf("developer ID and app ID are required")
	}
	if err := b.Navigate(ctx, publishingOverviewURL(developerID, appID, account)); err != nil {
		return err
	}
	if err := b.EvalUntil(ctx, managedPublishingReadyScript, 60*time.Second); err != nil {
		return fmt.Errorf("the Publishing overview did not load (is the gplay browser profile signed in?): %w", err)
	}
	var opened bool
	if err := b.Eval(ctx, managedPublishingMenuScript, &opened); err != nil {
		return err
	}
	if !opened {
		return fmt.Errorf("managed publishing dropdown was not found")
	}
	if err := b.EvalUntil(ctx, managedPublishingOptionPresentScript, 15*time.Second); err != nil {
		return fmt.Errorf("managed publishing options did not open: %w", err)
	}
	var chosen bool
	if err := b.Eval(ctx, managedPublishingOptionScript(on), &chosen); err != nil {
		return err
	}
	if !chosen {
		return fmt.Errorf("the requested managed publishing option was not found")
	}
	// Confirm if a dialog appears; tolerate none.
	_ = b.EvalUntil(ctx, sendForReviewDialogPresentScript, 5*time.Second) //nolint:errcheck // optional
	var confirm string
	if err := b.Eval(ctx, sendForReviewConfirmScript, &confirm); err != nil {
		return err
	}
	// Verify the state flipped.
	want := formHelpers + ` && ` + publishHelpers + fmt.Sprintf(` && (() => window.__gplayPublish.managedPublishing() === %v)()`, on)
	if err := b.EvalUntil(ctx, want, 30*time.Second); err != nil {
		return fmt.Errorf("managed publishing did not switch to %v: %w", on, err)
	}
	return nil
}

// publishSettledScript reports that the overview no longer offers a Publish
// action, i.e. the click was accepted and the changes went live.
const publishSettledScript = `(() => ![...document.querySelectorAll('button')].find(b =>
  b.getClientRects().length && /^publish/i.test(b.textContent.trim())))()`

const publishNowClickScript = `(() => {
  const btn = [...document.querySelectorAll('button')].find(b =>
    b.getClientRects().length && !b.disabled && /^publish/i.test(b.textContent.trim()));
  if (!btn) return false;
  btn.click();
  return true;
})()`

// PublishApprovedChanges clicks the overview's Publish action (present once
// reviewed changes are approved) and confirms the dialog. It errors when
// there is nothing to publish.
func PublishApprovedChanges(ctx context.Context, b *Browser, developerID, appID, account string, timeout time.Duration) error {
	if strings.TrimSpace(developerID) == "" || strings.TrimSpace(appID) == "" {
		return fmt.Errorf("developer ID and app ID are required")
	}
	if err := b.Navigate(ctx, publishingOverviewURL(developerID, appID, account)); err != nil {
		return err
	}
	if err := b.EvalUntil(ctx, managedPublishingReadyScript, 60*time.Second); err != nil {
		return fmt.Errorf("the Publishing overview did not load (is the gplay browser profile signed in?): %w", err)
	}
	var clicked bool
	if err := b.Eval(ctx, publishNowClickScript, &clicked); err != nil {
		return err
	}
	if !clicked {
		return fmt.Errorf("there is nothing to publish: no approved changes with a Publish action")
	}
	_ = b.EvalUntil(ctx, sendForReviewDialogPresentScript, 10*time.Second) //nolint:errcheck // optional
	var confirm string
	if err := b.Eval(ctx, sendForReviewConfirmScript, &confirm); err != nil {
		return err
	}
	if confirm == "no-button" {
		return fmt.Errorf("the publish confirmation dialog has no recognizable confirm button; nothing was published")
	}
	if err := b.EvalUntil(ctx, publishSettledScript, timeout); err != nil {
		return fmt.Errorf("publishing did not complete: %w", err)
	}
	return nil
}
