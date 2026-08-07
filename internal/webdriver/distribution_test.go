package webdriver

import (
	"context"
	"strings"
	"testing"
)

func TestAdvancedDistributionURL(t *testing.T) {
	got := advancedDistributionURL(publishingTestDeveloper, publishingTestApp, testAccount)
	want := "https://play.google.com/console/developers/1234567890/app/9876543210/advanced-distribution?authuser=me%40example.com&hl=en"
	if got != want {
		t.Errorf("url = %q, want %q", got, want)
	}
}

func TestReadDistribution(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(distributionPageReadyScript, true)
	f.setReply(openFormFactorsTabScript, true)
	f.setReply(formFactorsReadyScript, true)
	f.setReply(readDistributionScript, map[string]any{
		"formFactors": []string{"Android TV", "Wear OS"},
		"tasks":       []string{"Accept the Android TV terms"},
	})
	b := connectFake(t, f)

	state, err := ReadDistribution(context.Background(), b, publishingTestDeveloper, publishingTestApp, testAccount)
	if err != nil {
		t.Fatalf("ReadDistribution: %v", err)
	}
	if len(state.FormFactors) != 2 || state.FormFactors[0] != "Android TV" {
		t.Errorf("formFactors = %v", state.FormFactors)
	}
	if len(state.Tasks) != 1 {
		t.Errorf("tasks = %v", state.Tasks)
	}
}

func TestReadDistribution_MissingFormFactorsTab(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(distributionPageReadyScript, true)
	f.setReply(openFormFactorsTabScript, false)
	b := connectFake(t, f)

	_, err := ReadDistribution(context.Background(), b, publishingTestDeveloper, publishingTestApp, testAccount)
	if err == nil || !strings.Contains(err.Error(), "no Form factors tab") {
		t.Errorf("err = %v, want missing-tab error", err)
	}
}

func TestReadDistribution_RequiresIDs(t *testing.T) {
	if _, err := ReadDistribution(context.Background(), nil, "", "app", ""); err == nil {
		t.Error("want an error when developer ID is empty")
	}
}

func TestAddFormFactor(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(addFormFactorClickScript, true)
	f.setReply(distributionMenuPresentScript, true)
	f.setReply(selectFormFactorScript("Android TV"), "clicked")
	b := connectFake(t, f)

	if err := AddFormFactor(context.Background(), b, "Android TV"); err != nil {
		t.Fatalf("AddFormFactor: %v", err)
	}
}

func TestAddFormFactor_UnknownFactor(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(addFormFactorClickScript, true)
	f.setReply(distributionMenuPresentScript, true)
	f.setReply(selectFormFactorScript("Narnia OS"), "missing")
	b := connectFake(t, f)

	err := AddFormFactor(context.Background(), b, "Narnia OS")
	if err == nil || !strings.Contains(err.Error(), `"Narnia OS" was not found in the menu`) {
		t.Errorf("err = %v, want unknown-factor error", err)
	}
}

func TestAddFormFactor_RefusesMissingAddButton(t *testing.T) {
	f := newFakeChrome(t)
	// The fake answers nil (false) for the add-button click.
	b := connectFake(t, f)

	err := AddFormFactor(context.Background(), b, "Android TV")
	if err == nil || !strings.Contains(err.Error(), "Add form factor") {
		t.Errorf("err = %v, want missing-button error", err)
	}
}

func TestAddFormFactor_RequiresName(t *testing.T) {
	if err := AddFormFactor(context.Background(), nil, "  "); err == nil {
		t.Error("want an error for an empty form factor name")
	}
}

func scriptSection(t *testing.T, script, start, end string) string {
	t.Helper()
	_, section, ok := strings.Cut(script, start)
	if !ok {
		t.Fatalf("script is missing %q", start)
	}
	section, _, ok = strings.Cut(section, end)
	if !ok {
		t.Fatalf("script section %q is missing terminator %q", start, end)
	}
	return section
}

func TestDistributionHelpers_FallsBackToManageRowText(t *testing.T) {
	factors := scriptSection(t, distributionHelpers, "const factors", "const tasks")
	readsAncestor := strings.Contains(factors, ".closest(") || strings.Contains(factors, ".parentElement")
	readsText := strings.Contains(factors, ".textContent") || strings.Contains(factors, ".innerText")
	if !readsAncestor || !readsText {
		t.Error("factor lookup must fall back to the manage button row text when aria-label is absent")
	}
}

func TestDistributionHelpers_SelectsNamedVisibleAddButton(t *testing.T) {
	add := scriptSection(t, distributionHelpers, "const addButton", "const menuItem")
	if !strings.Contains(add, "querySelectorAll('[debug-id=text-button]')") ||
		!strings.Contains(add, "Add form factor") ||
		!strings.Contains(add, "getClientRects().length") {
		t.Error("add button lookup must select the visible Add form factor control")
	}
}

func TestDistributionHelpers_DoesNotAcceptLegalTerms(t *testing.T) {
	if strings.Contains(distributionHelpers, "accept-button") {
		t.Error("adding a form factor must leave legal opt-in terms for explicit follow-up")
	}
}
