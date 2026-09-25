package onetimeproducts

import (
	"strings"
	"testing"

	"google.golang.org/api/androidpublisher/v3"
)

func TestValidateProductID(t *testing.T) {
	t.Parallel()

	for _, id := range []string{"0", "a", "1_product.v2", "product."} {
		if err := validateProductID(id); err != nil {
			t.Errorf("validateProductID(%q): %v", id, err)
		}
	}
	for _, id := range []string{"_product", ".product", "Product", "product-id", "product id"} {
		err := validateProductID(id)
		if err == nil {
			t.Errorf("validateProductID(%q) succeeded, want error", id)
			continue
		}
		if !strings.Contains(err.Error(), "lowercase letter or number") {
			t.Errorf("validateProductID(%q) error = %q", id, err)
		}
	}
	if err := validateProductID(""); err == nil || !strings.Contains(err.Error(), "--product-id") {
		t.Fatalf("empty product ID error = %v", err)
	}
}

func TestValidateBatchProductIDs(t *testing.T) {
	t.Parallel()

	ids := make([]string, maxOneTimeProductBatchSize+1)
	for i := range ids {
		ids[i] = "product" + strings.Repeat("a", i)
	}
	if err := validateBatchProductIDs(ids); err == nil || !strings.Contains(err.Error(), "at most 100") {
		t.Fatalf("oversized batch error = %v", err)
	}
	if err := validateBatchProductIDs([]string{"one", "two", "one"}); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate batch error = %v", err)
	}
}

func TestRequireRegionsVersionForRegionalPricing(t *testing.T) {
	t.Parallel()

	product := &androidpublisher.OneTimeProduct{
		PurchaseOptions: []*androidpublisher.OneTimeProductPurchaseOption{{
			RegionalPricingAndAvailabilityConfigs: []*androidpublisher.OneTimeProductPurchaseOptionRegionalPricingAndAvailabilityConfig{{RegionCode: "US"}},
		}},
	}
	if err := requireRegionsVersion(product, ""); err == nil || !strings.Contains(err.Error(), "--regions-version") {
		t.Fatalf("missing regions version error = %v", err)
	}
	if err := requireRegionsVersion(product, "2025/02"); err != nil {
		t.Fatalf("regions version rejected: %v", err)
	}
}

func TestNormalizeLatencyTolerance(t *testing.T) {
	t.Parallel()

	for _, value := range []string{
		"",
		"PRODUCT_UPDATE_LATENCY_TOLERANCE_LATENCY_SENSITIVE",
		"PRODUCT_UPDATE_LATENCY_TOLERANCE_LATENCY_TOLERANT",
	} {
		if _, err := normalizeLatencyTolerance(value); err != nil {
			t.Errorf("normalizeLatencyTolerance(%q): %v", value, err)
		}
	}
	if _, err := normalizeLatencyTolerance("fast"); err == nil || !strings.Contains(err.Error(), "--latency-tolerance") {
		t.Fatalf("invalid latency tolerance error = %v", err)
	}
}
