package webdriver

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// consoleAppURL builds a console route for one app. The authuser and hl
// parameters mirror what the console itself puts in its links.
func consoleAppURL(developerID, appID, account, route string) string {
	u := "https://play.google.com/console/developers/" +
		url.PathEscape(strings.TrimSpace(developerID)) + "/app/" +
		url.PathEscape(strings.TrimSpace(appID)) + "/" + route
	if account != "" {
		u += "?" + url.Values{
			"authuser": {strings.TrimSpace(account)},
			"hl":       {"en"},
		}.Encode()
	}
	return u
}

func publishingOverviewURL(developerID, appID, account string) string {
	return consoleAppURL(developerID, appID, account, "publishing")
}

func appDashboardURL(developerID, appID, account string) string {
	return consoleAppURL(developerID, appID, account, "app-dashboard")
}

// publishHelpers installs Publishing overview lookups in the page. The
// console renders the page asynchronously; every read goes through these so
// the Go side never depends on the Dart app's obfuscated structure.
//
// The changes table uses three row shapes established against the live
// console: a particle-table-header row to skip, subheader rows that name the
// group the following changes belong to (e.g. "Closed testing - Alpha"), and
// change rows whose gridcells are [item, description, actions].
const publishHelpers = `
(() => {
  const g = window.__gplay;
  const clean = s => (s || '').replace(/\s+/g, ' ').trim();
  const section = () => document.querySelector('[debug-id=not-sent-for-review-changes]');
  const sendButton = () => {
    const w = document.querySelector('[debug-id=send-for-review-button]');
    return w && (w.querySelector('button') || w);
  };
  const changes = () => {
    const table = document.querySelector('[debug-id=changes-table]');
    if (!table) return [];
    const out = [];
    let group = '';
    for (const row of table.querySelectorAll('[role=row]')) {
      if (row.classList.contains('particle-table-header')) continue;
      const cells = [...row.querySelectorAll('[role=gridcell]')]
        .map(c => clean(c.innerText))
        .filter(t => t && t !== 'more_vert');
      if (row.classList.contains('subheader')) {
        group = cells[0] || clean(row.innerText);
        continue;
      }
      if (!cells.length) continue;
      out.push({ section: group, item: cells[0] || '', description: cells[1] || '' });
    }
    return out;
  };
  const blockedReason = () => {
    const s = section();
    if (!s) return '';
    // The lock banner is one innerText line ("To send changes for review, …");
    // reading the line is robust to how the banner nests icons and links.
    const line = (s.innerText || '').split('\n')
      .map(l => l.trim())
      .find(l => /^to send changes for review/i.test(l));
    return line || '';
  };
  const managedPublishing = () => {
    const el = document.querySelector('[debug-id=managed-publishing-dropdown]');
    if (!el) return false;
    return !/managed publishing off/i.test(el.textContent || '');
  };
  // When the changes table fails to render rows (observed after saving a
  // release into the overview), the send button's "Submit N changes for
  // review" label is the only pending-count signal.
  const summaryCount = () => {
    const s = section();
    const m = s && /submit (\d+) changes? for review/i.exec(s.innerText || '');
    return m ? Number(m[1]) : 0;
  };
  // While Google reviews, the overview shows a "Changes in review" section;
  // right after sending it also says "N changes sent for review".
  const inReview = () =>
    /changes in review|changes are now in review/i.test(document.body.innerText || '');
  window.__gplayPublish = { section, sendButton, changes, blockedReason, managedPublishing, summaryCount, inReview };
  return true;
})()
`

// PendingChange is one row of the Publishing overview's not-yet-submitted
// changes table.
type PendingChange struct {
	Section     string `json:"section,omitempty"`
	Item        string `json:"item"`
	Description string `json:"description,omitempty"`
}

// PublishingOverview is the state of the Publishing overview page.
type PublishingOverview struct {
	PendingChanges    []PendingChange `json:"pendingChanges"`
	CanSendForReview  bool            `json:"canSendForReview"`
	SendBlockedReason string          `json:"sendBlockedReason,omitempty"`
	ManagedPublishing bool            `json:"managedPublishing"`
	// SummaryPendingCount is the count from the send button's "Submit N
	// changes for review" label, used when the changes table renders no rows.
	SummaryPendingCount int `json:"summaryPendingCount,omitempty"`
	// InReview reports whether the overview shows the changes as in review.
	InReview bool `json:"inReview"`
}

const readPublishingOverviewScript = `(() => {
  const p = window.__gplayPublish;
  const btn = p.sendButton();
  return {
    pendingChanges: p.changes(),
    canSendForReview: !!(btn && !btn.disabled && btn.getAttribute('aria-disabled') !== 'true'),
    sendBlockedReason: p.blockedReason(),
    managedPublishing: p.managedPublishing(),
    summaryPendingCount: p.summaryCount(),
    inReview: p.inReview(),
  };
})()`

// publishingOverviewReadyScript gates reads on the overview page being loaded
// at all. The not-sent section is absent when everything is already in
// review, so the managed-publishing dropdown is the page marker.
const publishingOverviewReadyScript = `(() => {
  const p = window.__gplayPublish;
  return !!p && !!(p.section() || document.querySelector('[debug-id=managed-publishing-dropdown]'));
})()`

// publishingOverviewSettledScript reports the definitive send state: either
// the button is enabled, the console has explained why sending is blocked, or
// the overview shows the changes are already in review.
const publishingOverviewSettledScript = `(() => {
  const p = window.__gplayPublish;
  if (!p) return false;
  const btn = p.sendButton();
  const enabled = !!(btn && !btn.disabled && btn.getAttribute('aria-disabled') !== 'true');
  return enabled || p.blockedReason() !== '' || /changes in review/i.test(document.body.innerText || '');
})()`

// The *Wait forms reinstall the helpers on every poll, since a re-render wipes
// window.__gplayPublish and would make the bare script throw until timeout.
const publishingOverviewReadyWait = formHelpers + ` && ` + publishHelpers + ` && ` + publishingOverviewReadyScript

const publishingOverviewSettledWait = formHelpers + ` && ` + publishHelpers + ` && ` + publishingOverviewSettledScript

// ReadPublishingOverview opens the app's Publishing overview and reads its
// state. It changes nothing.
func ReadPublishingOverview(ctx context.Context, b *Browser, developerID, appID, account string) (*PublishingOverview, error) {
	if strings.TrimSpace(developerID) == "" || strings.TrimSpace(appID) == "" {
		return nil, fmt.Errorf("developer ID and app ID are required")
	}
	if err := b.Navigate(ctx, publishingOverviewURL(developerID, appID, account)); err != nil {
		return nil, err
	}
	if err := b.EvalUntil(ctx, publishingOverviewReadyWait, 60*time.Second); err != nil {
		return nil, fmt.Errorf("the Publishing overview did not load (is the gplay browser profile signed in?): %w", err)
	}
	var overview PublishingOverview
	if err := b.Eval(ctx, readPublishingOverviewScript, &overview); err != nil {
		return nil, err
	}
	if overview.CanSendForReview || overview.SendBlockedReason != "" {
		return &overview, nil
	}
	// Disabled with no explanation means the reviewability evaluation was
	// still in flight. Wait for the definitive state, then read once more.
	if err := b.EvalUntil(ctx, publishingOverviewSettledWait, 60*time.Second); err != nil {
		return &overview, nil // best effort: treat as blocked without a reason
	}
	if err := b.Eval(ctx, readPublishingOverviewScript, &overview); err != nil {
		return nil, err
	}
	return &overview, nil
}

const sendForReviewClickScript = `(() => {
  const p = window.__gplayPublish;
  const btn = p && p.sendButton();
  if (!btn || btn.disabled || btn.getAttribute('aria-disabled') === 'true') return false;
  btn.click();
  return true;
})()`

const sendForReviewDialogPresentScript = `(() => {
  const p = window.__gplayPublish;
  if (p && !p.section()) return true; // page already settled, no dialog coming
  return !!document.querySelector('.pane.modal.visible');
})()`

// The confirmation is a .pane.modal dialog ("Send N changes for review?")
// with Cancel and "Send changes for review" buttons — not a [role=dialog].
const sendForReviewConfirmScript = `(() => {
  const d = document.querySelector('.pane.modal.visible');
  if (!d) return 'none';
  const btn = [...d.querySelectorAll('button')].find(b =>
    !b.disabled && /send|confirm|got it|^ok$/i.test(b.textContent));
  if (!btn) return 'no-button';
  btn.click();
  return 'confirmed';
})()`

const sendForReviewClickWait = formHelpers + ` && ` + publishHelpers + ` && ` + sendForReviewClickScript

const sendForReviewSettledScript = `(() => {
  const p = window.__gplayPublish;
  if (!p) return false;
  const s = p.section();
  return !s || p.changes().length === 0;
})()`

// SendForReview clicks "Send app for review" on the Publishing overview,
// confirms the dialog when one appears, and waits for the pending-changes
// section to empty. The browser must already be on the Publishing overview
// (see ReadPublishingOverview); callers must re-read the page afterwards to
// verify the outcome.
func SendForReview(ctx context.Context, b *Browser, timeout time.Duration) error {
	var clicked bool
	if err := b.Eval(ctx, sendForReviewClickWait, &clicked); err != nil {
		return err
	}
	if !clicked {
		return fmt.Errorf(`"Send app for review" button was missing or disabled`)
	}
	// The confirmation dialog appears asynchronously, or not at all when the
	// console submits directly. Wait briefly for it but tolerate its absence.
	_ = b.EvalUntil(ctx, sendForReviewDialogPresentScript, 10*time.Second) //nolint:errcheck // absence is fine
	var confirm string
	if err := b.Eval(ctx, sendForReviewConfirmScript, &confirm); err != nil {
		return err
	}
	if confirm == "no-button" {
		return fmt.Errorf("the send-for-review confirmation dialog has no recognizable confirm button; nothing was confirmed")
	}
	if err := b.EvalUntil(ctx, sendForReviewSettledScript, timeout); err != nil {
		return fmt.Errorf("sending changes for review did not complete: %w", err)
	}
	return nil
}

// SetupGoal is one app-dashboard setup checklist (e.g. "Set up your app").
type SetupGoal struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Complete     int      `json:"complete"`
	Total        int      `json:"total"`
	PendingTasks []string `json:"pendingTasks"`
}

// AppSetup is the app dashboard's setup state: per-goal progress plus the
// app-level state chip ("draft" while the app has never been live).
type AppSetup struct {
	AppState string      `json:"appState,omitempty"`
	Goals    []SetupGoal `json:"goals"`
}

// dashboardHelpers installs app-dashboard checklist lookups. A task is
// complete when its task-text span carries the "Completed task." aria-label;
// goal progress comes from the "N of M complete" header text.
const dashboardHelpers = `
(() => {
  const clean = s => (s || '').replace(/\s+/g, ' ').trim();
  const goals = () => [...document.querySelectorAll('[debug-id$=-goal]')].map(g => {
    // The bold goal title ("Set up your app") is the first line of the goal's
    // own text; header elements inside it hold the longer description instead.
    const lines = (g.innerText || '').split('\n').map(clean).filter(Boolean);
    const m = /(\d+)\s+of\s+(\d+)\s+complete/i.exec(g.innerText || '');
    const tasks = [...g.querySelectorAll('[debug-id=task-text]')].map(t => ({
      text: clean(t.textContent),
      done: /^completed task\./i.test(t.getAttribute('aria-label') || ''),
    }));
    return {
      id: g.getAttribute('debug-id'),
      title: lines[0] || '',
      complete: m ? Number(m[1]) : tasks.filter(t => t.done).length,
      total: m ? Number(m[2]) : tasks.length,
      pendingTasks: tasks.filter(t => !t.done).map(t => t.text),
    };
  });
  const appState = () => {
    const h = document.querySelector('[debug-id=app-header]');
    return h && /draft app/i.test(h.innerText || '') ? 'draft' : '';
  };
  window.__gplayDash = { goals, appState };
  return true;
})()
`

const appDashboardReadyScript = formHelpers + ` && ` + dashboardHelpers +
	` && window.__gplayDash.goals().length > 0`

const readAppSetupScript = `(() => ({
  appState: window.__gplayDash.appState(),
  goals: window.__gplayDash.goals(),
}))()`

// ReadAppSetup opens the app dashboard and reads the setup checklists. It
// changes nothing.
func ReadAppSetup(ctx context.Context, b *Browser, developerID, appID, account string) (*AppSetup, error) {
	if strings.TrimSpace(developerID) == "" || strings.TrimSpace(appID) == "" {
		return nil, fmt.Errorf("developer ID and app ID are required")
	}
	if err := b.Navigate(ctx, appDashboardURL(developerID, appID, account)); err != nil {
		return nil, err
	}
	if err := b.EvalUntil(ctx, appDashboardReadyScript, 60*time.Second); err != nil {
		return nil, fmt.Errorf("the app dashboard did not load (is the gplay browser profile signed in?): %w", err)
	}
	var setup AppSetup
	if err := b.Eval(ctx, readAppSetupScript, &setup); err != nil {
		return nil, err
	}
	return &setup, nil
}
