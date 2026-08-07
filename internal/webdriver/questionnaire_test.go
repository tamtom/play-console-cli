package webdriver

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestReadQuestionnaireStep(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(readQuestionnaireStepScript, map[string]any{
		"stepLabel": "1/2",
		"choices": []map[string]any{
			{"id": "POLICY_RESPONSE_CHOICE_ID_HEALTH_MEDICAL", "selected": true},
			{"id": "POLICY_RESPONSE_CHOICE_ID_HEALTH_OTHER", "selected": false},
		},
		"hasNext": true, "canSave": false,
	})
	b := connectFake(t, f)

	step, err := ReadQuestionnaireStep(context.Background(), b)
	if err != nil {
		t.Fatalf("ReadQuestionnaireStep: %v", err)
	}
	if step.StepLabel != "1/2" || len(step.Choices) != 2 || !step.Choices[0].Selected || !step.HasNext {
		t.Errorf("step = %+v", step)
	}
}

func TestSetStepChoices_RejectsUnknownChoice(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(readQuestionnaireStepScript, map[string]any{
		"stepLabel": "1/2",
		"choices":   []map[string]any{{"id": "CHOICE_A", "selected": false}},
	})
	b := connectFake(t, f)

	err := SetStepChoices(context.Background(), b, []string{"CHOICE_NOPE"})
	if err == nil || !strings.Contains(err.Error(), `"CHOICE_NOPE" is not on this step`) {
		t.Errorf("err = %v, want unknown-choice error", err)
	}
}

func TestSetStepChoices_TogglesOnlyMismatches(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(readQuestionnaireStepScript, map[string]any{
		"stepLabel": "1/2",
		"choices": []map[string]any{
			{"id": "CHOICE_A", "selected": true},
			{"id": "CHOICE_B", "selected": false},
		},
	})
	// checked-state waits after toggles.
	f.setReply(choiceCheckedScript("CHOICE_A", false), true)
	f.setReply(choiceCheckedScript("CHOICE_B", true), true)
	b := connectFake(t, f)

	if err := SetStepChoices(context.Background(), b, []string{"CHOICE_B"}); err != nil {
		t.Fatalf("SetStepChoices: %v", err)
	}
	clicks := 0
	for _, expr := range f.seen {
		if strings.Contains(expr, "input.click()") {
			clicks++
		}
	}
	if clicks != 2 {
		t.Errorf("clicks = %d, want 2 (one per mismatched choice)", clicks)
	}
}

func TestQuestionnaireNext_LastStep(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(readQuestionnaireStepScript, map[string]any{
		"stepLabel": "2/2", "choices": []map[string]any{}, "hasNext": false, "canSave": true,
	})
	b := connectFake(t, f)

	next, err := QuestionnaireNext(context.Background(), b)
	if err != nil {
		t.Fatalf("QuestionnaireNext: %v", err)
	}
	if next {
		t.Error("next = true on the last step, want false")
	}
}

func TestQuestionnaireSave_RefusesMissingSave(t *testing.T) {
	f := newFakeChrome(t)
	// save click composite answers false.
	b := connectFake(t, f)

	err := QuestionnaireSave(context.Background(), b, 300*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "Save button was missing or disabled") {
		t.Errorf("err = %v, want missing-save error", err)
	}
}

func TestImportDataSafetyCSV_RequiresFile(t *testing.T) {
	err := ImportDataSafetyCSV(context.Background(), nil, "dev", "app", "", "/nonexistent/file.csv")
	if err == nil || !strings.Contains(err.Error(), "CSV file") {
		t.Errorf("err = %v, want missing-file error", err)
	}
}

func TestImportDataSafetyCSV(t *testing.T) {
	csv := t.TempDir() + "/answers.csv"
	if err := os.WriteFile(csv, []byte("a,b\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f := newFakeChrome(t)
	f.setReply(dataSafetyPageReadyScript, true)
	f.setReply(importCSVClickScript, true)
	f.setReply(importDialogReadyWait, true)
	f.setDOMReply(t, "DOM.getDocument", map[string]any{"root": map[string]any{"nodeId": 1}})
	f.setDOMReply(t, "DOM.querySelector", map[string]any{"nodeId": 2})
	f.setDOMReply(t, "DOM.setFileInputFiles", map[string]any{})
	f.setReply(importDialogClickImportScript, "clicked")
	f.setReply(importDialogGoneScript, true)
	b := connectFake(t, f)

	if err := ImportDataSafetyCSV(context.Background(), b, publishingTestDeveloper, publishingTestApp, testAccount, csv); err != nil {
		t.Fatalf("ImportDataSafetyCSV: %v", err)
	}
}
