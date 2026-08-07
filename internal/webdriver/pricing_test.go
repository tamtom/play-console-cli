package webdriver

import (
	"context"
	"strings"
	"testing"
	"time"
)

func withPricingHelpers(script string) string {
	return formHelpers + ` && ` + pricingHelpers + ` && ` + script
}

func TestAppPricingURL(t *testing.T) {
	got := appPricingURL(publishingTestDeveloper, publishingTestApp, testAccount)
	want := "https://play.google.com/console/developers/1234567890/app/9876543210/paid-app?authuser=me%40example.com&hl=en"
	if got != want {
		t.Errorf("url = %q, want %q", got, want)
	}
}

func TestOpenAndReadAppPricing(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(pricingPageReadyScript, true)
	f.setReply(readAppPricingScript, map[string]any{
		"priceType": "paid", "pricesSet": false, "canSave": false,
	})
	b := connectFake(t, f)

	if err := OpenAppPricing(context.Background(), b, publishingTestDeveloper, publishingTestApp, testAccount); err != nil {
		t.Fatalf("OpenAppPricing: %v", err)
	}
	state, err := ReadAppPricing(context.Background(), b)
	if err != nil {
		t.Fatalf("ReadAppPricing: %v", err)
	}
	if state.PriceType != "paid" || state.PricesSet || state.CanSave {
		t.Errorf("state = %+v", state)
	}
}

func scriptPricingWizard(t *testing.T, f *fakeChrome) {
	t.Helper()
	f.setReply(pricingStagedScript, false)
	f.setReply(withPricingHelpers(setPricingClickScript), true)
	f.setReply(pricingCountryTableReadyScript, true)
	f.setReply(pricingSelectAllScript, "clicked")
	f.setReply(pricingSelectionCountedScript, true)
	f.setReply(pricingContinueEnabledScript, true)
	f.setReply(pricingContinueClickScript, true)
	f.setReply(priceInputReadyScript, true)
	f.setReply(focusPriceInputScript, true)
	f.setReply(priceTypedHeldScript("69.99"), true)
	f.setReply(pricingUpdateClickScript, true)
	f.setReply(pricingDialogGoneScript, true)
	f.setReply(pricingWizardDoneScript, true)
}

func TestSetAppPrice(t *testing.T) {
	f := newFakeChrome(t)
	scriptPricingWizard(t, f)
	b := connectFake(t, f)
	shortenPricingTimeouts(t)

	if err := SetAppPrice(context.Background(), b, "69.99"); err != nil {
		t.Fatalf("SetAppPrice: %v", err)
	}
}

func TestSetAppPrice_RefusesDisabledSetPricing(t *testing.T) {
	f := newFakeChrome(t)
	scriptPricingWizard(t, f)
	f.setReply(withPricingHelpers(setPricingClickScript), false)
	b := connectFake(t, f)

	err := SetAppPrice(context.Background(), b, "69.99")
	if err == nil || !strings.Contains(err.Error(), "Set pricing") {
		t.Errorf("err = %v, want Set-pricing error", err)
	}
}

func TestSetAppPrice_ErrorsWhenPriceNotAccepted(t *testing.T) {
	f := newFakeChrome(t)
	scriptPricingWizard(t, f)
	f.setReply(priceTypedHeldScript("69.99"), false)
	b := connectFake(t, f)
	shortenPricingTimeouts(t)

	err := SetAppPrice(context.Background(), b, "69.99")
	if err == nil || !strings.Contains(err.Error(), "did not accept") {
		t.Errorf("err = %v, want price-not-accepted error", err)
	}
}

func shortenPricingTimeouts(t *testing.T) {
	t.Helper()
	origSettle, origStep, origDialog := pricingFormSettleTimeout, pricingStepSettle, pricingDialogSettle
	pricingFormSettleTimeout, pricingStepSettle, pricingDialogSettle = 300*time.Millisecond, 0, 0
	t.Cleanup(func() {
		pricingFormSettleTimeout, pricingStepSettle, pricingDialogSettle = origSettle, origStep, origDialog
	})
}

func TestSetAppPrice_ErrorsWhenInputNotFocusable(t *testing.T) {
	f := newFakeChrome(t)
	scriptPricingWizard(t, f)
	f.setReply(focusPriceInputScript, false)
	b := connectFake(t, f)
	shortenPricingTimeouts(t)

	err := SetAppPrice(context.Background(), b, "69.99")
	if err == nil || !strings.Contains(err.Error(), "could not focus") {
		t.Errorf("err = %v, want focus error", err)
	}
}

func TestSetAppPrice_ErrorsWhenPricesNotFilled(t *testing.T) {
	f := newFakeChrome(t)
	scriptPricingWizard(t, f)
	f.setReply(pricingWizardDoneScript, false)
	b := connectFake(t, f)
	shortenPricingTimeouts(t)

	orig := pricingPostSetTimeout
	pricingPostSetTimeout = 300 * time.Millisecond
	t.Cleanup(func() { pricingPostSetTimeout = orig })

	err := SetAppPrice(context.Background(), b, "69.99")
	if err == nil || !strings.Contains(err.Error(), "was not staged with prices") {
		t.Errorf("err = %v, want not-staged error", err)
	}
}

func TestSetAppPrice_RequiresPrice(t *testing.T) {
	if err := SetAppPrice(context.Background(), nil, "  "); err == nil {
		t.Error("want an error for an empty price")
	}
}

func TestSubmitAppPricing(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(withPricingHelpers(submitPricingClickScript), true)
	f.setReply(pricingSaveDialogPresentScript, true)
	f.setReply(pricingSaveConfirmScript, "confirmed")
	f.setReply(pricingSaveSettledWait, true)
	b := connectFake(t, f)
	shortenPricingTimeouts(t)

	if err := SubmitAppPricing(context.Background(), b, 2*time.Second); err != nil {
		t.Fatalf("SubmitAppPricing: %v", err)
	}
}

func TestSubmitAppPricing_RefusesDisabledSave(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(withPricingHelpers(submitPricingClickScript), false)
	b := connectFake(t, f)

	err := SubmitAppPricing(context.Background(), b, 300*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "missing or disabled") {
		t.Errorf("err = %v, want disabled-save error", err)
	}
}

func TestSubmitAppPricing_ErrorsOnUnknownDialog(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(withPricingHelpers(submitPricingClickScript), true)
	f.setReply(pricingSaveDialogPresentScript, true)
	f.setReply(pricingSaveConfirmScript, "no-button")
	b := connectFake(t, f)
	shortenPricingTimeouts(t)

	err := SubmitAppPricing(context.Background(), b, 300*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "no recognizable confirm button") {
		t.Errorf("err = %v, want unknown-dialog error", err)
	}
}

func TestDiscardAppPricing_NoEditorIsNoOp(t *testing.T) {
	f := newFakeChrome(t)
	// discard script answers "": nothing open.
	b := connectFake(t, f)

	if err := DiscardAppPricing(context.Background(), b); err != nil {
		t.Fatalf("DiscardAppPricing: %v", err)
	}
}

func TestDiscardAppPricing_CancelsBar(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(withPricingHelpers(discardPricingClickScript), "bar")
	f.setReply(pricingUnstagedScript, true)
	b := connectFake(t, f)

	if err := DiscardAppPricing(context.Background(), b); err != nil {
		t.Fatalf("DiscardAppPricing: %v", err)
	}
}
