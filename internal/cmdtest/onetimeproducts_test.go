package cmdtest_test

import (
	"context"
	"strings"
	"testing"
)

func TestOnetimeproductsCreate_EmptyJSON_FailsWithHelpfulError(t *testing.T) {
	root := RootCommand("test")
	var runErr error
	_, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"onetimeproducts", "create", "--product-id", "test", "--json", "{}"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})
	if runErr == nil {
		t.Fatal("expected error for empty JSON")
	}
	errMsg := runErr.Error() + stderr
	if !strings.Contains(errMsg, "no updatable fields") {
		t.Errorf("expected 'no updatable fields' error, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "valid fields") {
		t.Errorf("expected error to list valid fields, got: %s", errMsg)
	}
}

func TestOnetimeproductsCreate_OnlyImmutableFields_FailsWithHelpfulError(t *testing.T) {
	root := RootCommand("test")
	var runErr error
	_, _ = captureOutput(t, func() {
		if err := root.Parse([]string{"onetimeproducts", "create", "--product-id", "test", "--json", `{"packageName":"com.example","productId":"test"}`}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})
	if runErr == nil {
		t.Fatal("expected error for only immutable fields")
	}
	if !strings.Contains(runErr.Error(), "no updatable fields") {
		t.Errorf("expected 'no updatable fields' error, got: %s", runErr.Error())
	}
}

func TestOnetimeproductsCreate_InvalidJSON_FailsBeforeAPICall(t *testing.T) {
	root := RootCommand("test")
	var runErr error
	_, _ = captureOutput(t, func() {
		if err := root.Parse([]string{"onetimeproducts", "create", "--product-id", "test", "--json", "not json"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})
	if runErr == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(runErr.Error(), "invalid JSON") {
		t.Errorf("expected 'invalid JSON' error, got: %s", runErr.Error())
	}
}

func TestOnetimeproductsPatch_EmptyJSON_NoUpdateMask_FailsWithHelpfulError(t *testing.T) {
	root := RootCommand("test")
	var runErr error
	_, _ = captureOutput(t, func() {
		if err := root.Parse([]string{"onetimeproducts", "patch", "--product-id", "test", "--json", "{}"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})
	if runErr == nil {
		t.Fatal("expected error for empty JSON without --update-mask")
	}
	if !strings.Contains(runErr.Error(), "no updatable fields") {
		t.Errorf("expected 'no updatable fields' error, got: %s", runErr.Error())
	}
}

func TestOnetimeproductsPatch_ExplicitUpdateMask_SkipsDeriving(t *testing.T) {
	root := RootCommand("test")
	var runErr error
	_, _ = captureOutput(t, func() {
		if err := root.Parse([]string{"onetimeproducts", "patch", "--product-id", "test", "--json", "{}", "--update-mask", "listings"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})
	// Should NOT get "no updatable fields" — should pass validation and fail later (auth)
	if runErr != nil && strings.Contains(runErr.Error(), "no updatable fields") {
		t.Errorf("explicit --update-mask should skip derive; got: %s", runErr.Error())
	}
}

func TestSubscriptionsUpdate_EmptyJSON_NoUpdateMask_FailsWithHelpfulError(t *testing.T) {
	root := RootCommand("test")
	var runErr error
	_, _ = captureOutput(t, func() {
		if err := root.Parse([]string{"subscriptions", "update", "--product-id", "test", "--json", "{}"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})
	if runErr == nil {
		t.Fatal("expected error for empty JSON without --update-mask")
	}
	if !strings.Contains(runErr.Error(), "no updatable fields") {
		t.Errorf("expected 'no updatable fields' error, got: %s", runErr.Error())
	}
}

func TestOffersUpdate_EmptyJSON_NoUpdateMask_FailsWithHelpfulError(t *testing.T) {
	root := RootCommand("test")
	var runErr error
	_, _ = captureOutput(t, func() {
		if err := root.Parse([]string{"offers", "update", "--product-id", "test", "--base-plan-id", "monthly", "--offer-id", "trial", "--json", "{}"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})
	if runErr == nil {
		t.Fatal("expected error for empty JSON without --update-mask")
	}
	if !strings.Contains(runErr.Error(), "no updatable fields") {
		t.Errorf("expected 'no updatable fields' error, got: %s", runErr.Error())
	}
}
