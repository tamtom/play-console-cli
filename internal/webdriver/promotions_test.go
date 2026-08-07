package webdriver

import (
	"context"
	"strings"
	"testing"
	"time"
)

func skipPromotionTermsSettle(t *testing.T) {
	t.Helper()
	original := promotionTermsModalSettle
	promotionTermsModalSettle = 0
	t.Cleanup(func() { promotionTermsModalSettle = original })
}

func TestReadPromotions_EmptyState(t *testing.T) {
	skipPromotionTermsSettle(t)
	f := newFakeChrome(t)
	f.setReply(promotionsReadyExpr(), true)
	f.setReply(readPromotionsScript, map[string]any{"promotions": []map[string]any{}})
	b := connectFake(t, f)

	state, err := ReadPromotions(context.Background(), b, publishingTestDeveloper, publishingTestApp, testAccount)
	if err != nil {
		t.Fatal(err)
	}
	if state.Promotions == nil || len(state.Promotions) != 0 || state.TermsAcceptanceRequired {
		t.Errorf("state = %+v", state)
	}
}

func TestPromotionTermsModalVisible_WaitsForDelayedModal(t *testing.T) {
	original := promotionTermsModalSettle
	promotionTermsModalSettle = 50 * time.Millisecond
	t.Cleanup(func() { promotionTermsModalSettle = original })
	f := newFakeChrome(t)
	f.setReply(promotionTermsModalVisibleScript, false)
	b := connectFake(t, f)
	go func() {
		time.Sleep(10 * time.Millisecond)
		f.setReply(promotionTermsModalVisibleScript, true)
	}()

	required, err := promotionTermsModalVisible(context.Background(), b)
	if err != nil || !required {
		t.Fatalf("required = %v, err = %v", required, err)
	}
}

func TestCreatePaidAppPromotion_FailsClosedOnTerms(t *testing.T) {
	skipPromotionTermsSettle(t)
	f := newFakeChrome(t)
	b := connectFake(t, f)

	_, err := CreatePaidAppPromotion(context.Background(), b, publishingTestDeveloper, publishingTestApp, testAccount, false, PaidAppPromotionForm{
		Name: "Launch", StartDate: "2026-08-02", EndDate: "2026-08-03", CodeCount: 10,
	}, time.Second)
	if err == nil || !strings.Contains(err.Error(), "accepted manually") {
		t.Fatalf("err = %v", err)
	}
	if len(f.seen) != 0 {
		t.Fatalf("browser used without authoritative Terms acceptance: %q", f.seen)
	}
}

func TestCreatePaidAppPromotion_VisibleTermsModalVetoesAcceptedStatus(t *testing.T) {
	skipPromotionTermsSettle(t)
	f := newFakeChrome(t)
	f.setReply(promotionCreateReadyExpr(), true)
	f.setReply(promotionTermsModalVisibleScript, true)
	b := connectFake(t, f)

	_, err := CreatePaidAppPromotion(context.Background(), b, publishingTestDeveloper, publishingTestApp, testAccount, true, PaidAppPromotionForm{
		Name: "Launch", StartDate: "2026-08-02", EndDate: "2026-08-03", CodeCount: 10,
	}, time.Second)
	if err == nil || !strings.Contains(err.Error(), "accepted manually") {
		t.Fatalf("err = %v", err)
	}
	want := []string{"", promotionCreateReadyExpr(), promotionTermsModalVisibleScript}
	if strings.Join(f.seen, "\n") != strings.Join(want, "\n") {
		t.Fatalf("expressions after modal veto = %q, want %q", f.seen, want)
	}
}

func TestCreatePaidAppPromotion_VerifiesExactPromotion(t *testing.T) {
	skipPromotionTermsSettle(t)
	form := PaidAppPromotionForm{Name: "Launch", StartDate: "2026-08-02", EndDate: "2026-08-03", CodeCount: 10}
	for _, tt := range []struct {
		name    string
		result  Promotion
		wantErr bool
	}{
		{name: "exact", result: Promotion{Name: "Launch", Type: "Paid app", StartDate: "2026-08-02", EndDate: "2026-08-03"}},
		{name: "wrong date", result: Promotion{Name: "Launch", Type: "Paid app", StartDate: "2026-08-02", EndDate: "2026-08-04"}, wantErr: true},
		{name: "wrong type", result: Promotion{Name: "Launch", Type: "Subscription", StartDate: "2026-08-02", EndDate: "2026-08-03"}, wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeChrome(t)
			f.defaultReply = true
			f.setReply(promotionTermsModalVisibleScript, false)
			f.setReply(readPromotionsScript, map[string]any{"promotions": []Promotion{tt.result}})
			b := connectFake(t, f)

			_, err := CreatePaidAppPromotion(context.Background(), b, publishingTestDeveloper, publishingTestApp, testAccount, true, form, time.Second)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			for _, expression := range []string{
				promotionDateMatchesScript("start-date-picker", form.StartDate),
				promotionDateMatchesScript("end-date-picker", form.EndDate),
			} {
				if !seenExpression(f.seen, expression) {
					t.Fatalf("exact date verification did not run: %s", expression)
				}
			}
		})
	}
}

func seenExpression(expressions []string, want string) bool {
	for _, expression := range expressions {
		if expression == want {
			return true
		}
	}
	return false
}

func TestPromotionScriptsNeverAcceptTerms(t *testing.T) {
	for _, script := range []string{promotionHelpers, promotionTermsModalVisibleScript, readPromotionsScript, promotionSubmitScript} {
		if strings.Contains(script, "accept-button") {
			t.Fatal("promotion automation must never accept legal terms")
		}
	}
	if strings.Contains(promotionHelpers, "|| group.querySelector") {
		t.Fatal("promotion automation must never fall back to an arbitrary reward")
	}
}

func TestCreatePaidAppPromotion_ValidatesConsoleLimits(t *testing.T) {
	for _, form := range []PaidAppPromotionForm{
		{Name: strings.Repeat("x", 61), StartDate: "2026-08-02", EndDate: "2026-08-03", CodeCount: 1},
		{Name: "Too long", StartDate: "2026-08-02", EndDate: "2027-08-03", CodeCount: 1},
	} {
		if _, err := CreatePaidAppPromotion(context.Background(), nil, publishingTestDeveloper, publishingTestApp, "", true, form, time.Second); err == nil {
			t.Fatalf("CreatePaidAppPromotion accepted %+v", form)
		}
	}
}
