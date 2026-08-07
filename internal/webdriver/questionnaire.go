package webdriver

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// questionnaireHelpers installs wizard-step lookups. The console's policy
// questionnaires (health, finance, target audience, data safety) share a
// stepper: choice checkboxes/radios live in containers keyed by
// POLICY_RESPONSE_CHOICE_ID_* debug ids, with Next/Back/Save footer buttons.
const questionnaireHelpers = `
(() => {
  const clean = s => (s || '').replace(/\s+/g, ' ').trim();
  const choices = () => [...document.querySelectorAll('[debug-id^="POLICY_RESPONSE_CHOICE_ID_"]')]
    .map(w => {
      const input = w.querySelector('[role=checkbox], [role=radio], input[type=checkbox], input[type=radio]');
      const checked = input ? (input.getAttribute('aria-checked') === 'true' || input.checked === true) : false;
      return { id: w.getAttribute('debug-id'), selected: checked, el: input || w };
    });
  const footerButton = re => [...document.querySelectorAll('button')].find(b =>
    b.getClientRects().length && re.test(clean(b.textContent)));
  const next = () => footerButton(/^(next|get started)$/i);
  const save = () => footerButton(/^(save|save changes|submit)$/i);
  const stepLabel = () => {
    const m = /step (\d+) of (\d+)/i.exec(document.body.innerText || '');
    return m ? m[1] + '/' + m[2] : '';
  };
  window.__gplayQuest = { choices, next, save, stepLabel };
  return true;
})()
`

// QuestionnaireChoice is one selectable choice on a questionnaire step.
type QuestionnaireChoice struct {
	ID       string `json:"id"`
	Selected bool   `json:"selected"`
}

// QuestionnaireStep is the questionnaire's current step.
type QuestionnaireStep struct {
	StepLabel string                `json:"stepLabel"`
	Choices   []QuestionnaireChoice `json:"choices"`
	HasNext   bool                  `json:"hasNext"`
	CanSave   bool                  `json:"canSave"`
}

const questionnaireReadyScript = formHelpers + ` && ` + questionnaireHelpers +
	` && !!document.querySelector('[debug-id=stepper]')`

// questionnaireGoneScript reports that the wizard closed, which is how both a
// save and a discard settle.
const questionnaireGoneScript = `(() => !document.querySelector('[debug-id=stepper]'))()`

// choiceClickScript toggles one questionnaire choice; choiceCheckedScript
// reports whether it reached the wanted state.
func choiceClickScript(id string) string {
	return fmt.Sprintf(`(() => {
  const w = document.querySelector('[debug-id=%s]');
  if (!w) return false;
  const input = w.querySelector('[role=checkbox], [role=radio], input[type=checkbox], input[type=radio]') || w;
  input.click();
  return true;
})()`, jsString(id))
}

func choiceCheckedScript(id string, want bool) string {
	return fmt.Sprintf(`(() => {
  const w = document.querySelector('[debug-id=%s]');
  if (!w) return false;
  const input = w.querySelector('[role=checkbox], [role=radio], input[type=checkbox], input[type=radio]') || w;
  const checked = input.getAttribute('aria-checked') === 'true' || input.checked === true;
  return checked === %v;
})()`, jsString(id), want)
}

const readQuestionnaireStepScript = `(() => {
  const q = window.__gplayQuest;
  const next = q.next();
  const save = q.save();
  return {
    stepLabel: q.stepLabel(),
    choices: q.choices().map(c => ({ id: c.id, selected: c.selected })),
    hasNext: !!(next && !next.disabled),
    canSave: !!(save && !save.disabled),
  };
})()`

// OpenQuestionnaire opens a questionnaire page and waits for its stepper.
// page is the console app-content route (e.g. "health").
func OpenQuestionnaire(ctx context.Context, b *Browser, developerID, appID, account, page string) error {
	if strings.TrimSpace(developerID) == "" || strings.TrimSpace(appID) == "" {
		return fmt.Errorf("developer ID and app ID are required")
	}
	if err := b.Navigate(ctx, appContentURL(developerID, appID, account, page)); err != nil {
		return err
	}
	if err := b.EvalUntil(ctx, questionnaireReadyScript, 60*time.Second); err != nil {
		return fmt.Errorf("questionnaire %q did not load (is the gplay browser profile signed in?): %w", page, err)
	}
	return nil
}

// ReadQuestionnaireStep reads the current step's choices and footer state.
func ReadQuestionnaireStep(ctx context.Context, b *Browser) (*QuestionnaireStep, error) {
	if err := b.Eval(ctx, formHelpers+` && `+questionnaireHelpers, nil); err != nil {
		return nil, err
	}
	var step QuestionnaireStep
	if err := b.Eval(ctx, readQuestionnaireStepScript, &step); err != nil {
		return nil, err
	}
	return &step, nil
}

// SetStepChoices makes exactly the given choice IDs selected on the current
// step: selected-but-unwanted choices are clicked off, wanted ones on. It
// errors when a requested choice is not on this step.
func SetStepChoices(ctx context.Context, b *Browser, ids []string) error {
	step, err := ReadQuestionnaireStep(ctx, b)
	if err != nil {
		return err
	}
	visible := map[string]bool{}
	for _, c := range step.Choices {
		visible[c.ID] = true
	}
	for _, id := range ids {
		if !visible[id] {
			return fmt.Errorf("choice %q is not on this step (visible: %s)", id, strings.Join(stepChoiceIDs(step), ", "))
		}
	}
	for _, c := range step.Choices {
		want := false
		for _, id := range ids {
			if id == c.ID {
				want = true
				break
			}
		}
		if c.Selected == want {
			continue
		}
		if err := b.Eval(ctx, choiceClickScript(c.ID), nil); err != nil {
			return fmt.Errorf("toggling %q: %w", c.ID, err)
		}
		if err := b.EvalUntil(ctx, choiceCheckedScript(c.ID, want), 5*time.Second); err != nil {
			return fmt.Errorf("toggling %q: %w", c.ID, err)
		}
	}
	return nil
}

func stepChoiceIDs(step *QuestionnaireStep) []string {
	ids := make([]string, 0, len(step.Choices))
	for _, c := range step.Choices {
		ids = append(ids, c.ID)
	}
	return ids
}

const questionnaireNextClickScript = formHelpers + ` && ` + questionnaireHelpers + ` && (() => {
  const q = window.__gplayQuest;
  const n = q.next();
  if (!n || n.disabled) return false;
  n.click();
  return true;
})()`

const questionnaireSaveClickScript = formHelpers + ` && ` + questionnaireHelpers + ` && (() => {
  const q = window.__gplayQuest;
  const s = q.save();
  if (!s || s.disabled) return false;
  s.click();
  return true;
})()`

// questionnaireAdvancedScript reports a step transition: the step label or the
// choices set changes when the wizard moves on.
func questionnaireAdvancedScript(before string) string {
	return formHelpers + ` && ` + questionnaireHelpers + fmt.Sprintf(` && (() => {
  const q = window.__gplayQuest;
  return q.stepLabel() !== %s || q.choices().length === 0;
})()`, jsString(before))
}

// QuestionnaireNext advances to the next step, returning false when the
// wizard is already on its last step.
func QuestionnaireNext(ctx context.Context, b *Browser) (bool, error) {
	step, err := ReadQuestionnaireStep(ctx, b)
	if err != nil {
		return false, err
	}
	if !step.HasNext {
		return false, nil
	}
	before := step.StepLabel
	var clicked bool
	if err := b.Eval(ctx, questionnaireNextClickScript, &clicked); err != nil {
		return false, err
	}
	if !clicked {
		return false, fmt.Errorf("the Next button was missing or disabled")
	}
	if err := b.EvalUntil(ctx, questionnaireAdvancedScript(before), 30*time.Second); err != nil {
		return false, fmt.Errorf("the questionnaire did not advance: %w", err)
	}
	return true, nil
}

// QuestionnaireSave clicks the wizard's final Save and waits for the stepper
// to disappear. Callers must re-open the questionnaire to verify persistence.
func QuestionnaireSave(ctx context.Context, b *Browser, timeout time.Duration) error {
	var clicked bool
	if err := b.Eval(ctx, questionnaireSaveClickScript, &clicked); err != nil {
		return err
	}
	if !clicked {
		return fmt.Errorf("the questionnaire's Save button was missing or disabled")
	}
	if err := b.EvalUntil(ctx, questionnaireGoneScript, timeout); err != nil {
		return fmt.Errorf("saving the questionnaire did not complete: %w", err)
	}
	return nil
}

const discardQuestionnaireClickScript = `(() => {
  const btn = [...document.querySelectorAll('button')].find(x =>
    x.getClientRects().length && /^discard/i.test(x.textContent.trim()));
  if (!btn || btn.disabled) return false;
  btn.click();
  return true;
})()`

const discardConfirmScript = `(() => {
  const d = document.querySelector('.pane.modal.visible') ||
    [...document.querySelectorAll('[role=dialog]')].find(d => d.getClientRects().length);
  if (!d) return 'none';
  const btn = [...d.querySelectorAll('button')].find(b =>
    !b.disabled && /discard|yes|confirm|^ok$/i.test(b.textContent));
  if (!btn) return 'no-button';
  btn.click();
  return 'confirmed';
})()`

// QuestionnaireDiscard leaves the wizard without saving, confirming the
// discard dialog when one appears. When the discard control is disabled the
// wizard has no unsaved changes, so there is nothing to do.
func QuestionnaireDiscard(ctx context.Context, b *Browser) error {
	var clicked bool
	if err := b.Eval(ctx, discardQuestionnaireClickScript, &clicked); err != nil {
		return err
	}
	if !clicked {
		return nil
	}
	// A "discard changes?" confirmation may appear; tolerate its absence.
	_ = b.EvalUntil(ctx, sendForReviewDialogPresentScript, 5*time.Second) //nolint:errcheck // optional dialog
	var confirm string
	if err := b.Eval(ctx, discardConfirmScript, &confirm); err != nil {
		return err
	}
	if confirm == "no-button" {
		return fmt.Errorf("the discard confirmation dialog has no recognizable confirm button")
	}
	if err := b.EvalUntil(ctx, questionnaireGoneScript, 15*time.Second); err != nil {
		return fmt.Errorf("discarding the questionnaire did not complete: %w", err)
	}
	return nil
}
