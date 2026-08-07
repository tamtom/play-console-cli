package webdriver

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func advancedDistributionURL(developerID, appID, account string) string {
	return consoleAppURL(developerID, appID, account, "advanced-distribution")
}

// distributionHelpers installs Form factors page lookups. The left nav is
// custom tab-button elements; each managed form factor shows a "Manage <X>"
// console-button; adding goes through the "Add form factor" dropdown whose
// options are material-select-item menu items with aria-labels.
const distributionHelpers = `
(() => {
  const clean = s => (s || '').replace(/\s+/g, ' ').trim();
  const visible = e => !!(e && e.getClientRects().length);
  const factors = () => [...document.querySelectorAll('[debug-id=manage]')]
    .filter(visible)
    .map(e => {
      const button = e.querySelector('button');
      const row = e.closest('section, article, [role=row], console-card') || e.parentElement;
      return clean(e.getAttribute('aria-label') ||
        (button && button.getAttribute('aria-label')) ||
        (row && row.textContent) || e.innerText).replace(/^manage\s*/i, '').trim();
    })
    .filter(Boolean);
  const tasks = () => [...document.querySelectorAll('[debug-id=task-text]')]
    .map(e => clean(e.innerText)).filter(Boolean);
  const tabButton = name => [...document.querySelectorAll('tab-button')]
    .find(x => clean(x.textContent).toLowerCase() === clean(name).toLowerCase());
  const addButton = () => [...document.querySelectorAll('[debug-id=text-button]')]
    .map(w => w.querySelector('button') || w)
    .find(b => b.getClientRects().length && clean(b.textContent) === 'Add form factor');
  const menuItem = label => [...document.querySelectorAll('material-select-item')]
    .find(x => x.getAttribute('aria-label') === label && x.getClientRects().length);
  window.__gplayDist = { factors, tasks, tabButton, addButton, menuItem };
  return true;
})()
`

const distributionPageReadyScript = formHelpers + ` && ` + distributionHelpers +
	` && !!window.__gplayDist.tabButton('Form factors')`

const formFactorsReadyScript = `!!document.querySelector('[debug-id=manage], [debug-id=text-button]')`

const addFormFactorClickScript = formHelpers + ` && ` + distributionHelpers + ` && (() => {
  const btn = window.__gplayDist.addButton();
  if (!btn) return false;
  btn.click();
  return true;
})()`

// selectFormFactorScript picks a named option out of the opened Add form
// factor menu.
func selectFormFactorScript(factor string) string {
	return distributionHelpers + ` && (() => {
  const item = window.__gplayDist.menuItem(` + jsString(factor) + `);
  if (!item) return 'missing';
  item.click();
  return 'clicked';
})()`
}

const openFormFactorsTabScript = `(() => {
  const tab = window.__gplayDist.tabButton('Form factors');
  if (!tab) return false;
  tab.click();
  return true;
})()`

// DistributionState is the Form factors page state.
type DistributionState struct {
	FormFactors []string `json:"formFactors"`
	Tasks       []string `json:"tasks"`
}

const readDistributionScript = `(() => ({
  formFactors: window.__gplayDist.factors(),
  tasks: window.__gplayDist.tasks(),
}))()`

// ReadDistribution opens the Form factors page and reads the managed form
// factors plus outstanding per-factor requirements.
func ReadDistribution(ctx context.Context, b *Browser, developerID, appID, account string) (*DistributionState, error) {
	if strings.TrimSpace(developerID) == "" || strings.TrimSpace(appID) == "" {
		return nil, fmt.Errorf("developer ID and app ID are required")
	}
	if err := b.Navigate(ctx, advancedDistributionURL(developerID, appID, account)); err != nil {
		return nil, err
	}
	if err := b.EvalUntil(ctx, distributionPageReadyScript, 60*time.Second); err != nil {
		return nil, fmt.Errorf("the Advanced settings page did not load (is the gplay browser profile signed in?): %w", err)
	}
	var opened bool
	if err := b.Eval(ctx, openFormFactorsTabScript, &opened); err != nil {
		return nil, err
	}
	if !opened {
		return nil, fmt.Errorf("no Form factors tab was found")
	}
	if err := b.EvalUntil(ctx, formFactorsReadyScript, 30*time.Second); err != nil {
		return nil, fmt.Errorf("the Form factors page did not load: %w", err)
	}
	var state DistributionState
	if err := b.Eval(ctx, readDistributionScript, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

const distributionMenuPresentScript = `(() =>
  !! [...document.querySelectorAll('material-select-item')].find(x => x.getClientRects().length)
)()`

// AddFormFactor adds a form factor (e.g. "Android TV", "Wear OS", "Android
// Automotive OS", "Android Auto", "Google Play Games on PC") through the Add
// form factor dropdown. Product-policy opt-ins remain explicit follow-up
// tasks; callers must verify the factor appears afterwards.
func AddFormFactor(ctx context.Context, b *Browser, factor string) error {
	factor = strings.TrimSpace(factor)
	if factor == "" {
		return fmt.Errorf("form factor name is required")
	}
	var clicked bool
	if err := b.Eval(ctx, addFormFactorClickScript, &clicked); err != nil {
		return err
	}
	if !clicked {
		return fmt.Errorf(`"Add form factor" button was not found`)
	}
	if err := b.EvalUntil(ctx, distributionMenuPresentScript, 15*time.Second); err != nil {
		return fmt.Errorf("the form factor menu did not open: %w", err)
	}
	var result string
	if err := b.Eval(ctx, selectFormFactorScript(factor), &result); err != nil {
		return err
	}
	if result == "missing" {
		return fmt.Errorf("form factor %q was not found in the menu (already added, or unavailable)", factor)
	}
	return nil
}
