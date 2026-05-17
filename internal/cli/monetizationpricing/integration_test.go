//go:build integration

package monetizationpricing

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"google.golang.org/api/androidpublisher/v3"

	"github.com/tamtom/play-console-cli/internal/playclient"
)

const defaultIntegrationPackage = "com.itdeveapps.stepsshare"

func TestIntegration_OneTimeProductCreateWithConvertedRegionalPrices(t *testing.T) {
	skipUnlessIntegration(t)
	skipUnlessMutatingIntegration(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	service, err := playclient.NewService(ctx)
	if err != nil {
		t.Fatalf("creating service: %v", err)
	}

	pkg := integrationPackage()
	basePrice := &androidpublisher.Money{
		CurrencyCode: "USD",
		Units:        1,
		Nanos:        990000000,
	}
	converted, err := ConvertRegionPrices(ctx, service, pkg, basePrice, "")
	if err != nil {
		t.Fatalf("converting region prices: %v", err)
	}
	regionVersion, err := RegionVersion(converted)
	if err != nil {
		t.Fatalf("reading region version: %v", err)
	}

	productID := fmt.Sprintf("ci_otp_%s", time.Now().UTC().Format("20060102150405"))
	product := &androidpublisher.OneTimeProduct{
		PackageName: pkg,
		ProductId:   productID,
		Listings: []*androidpublisher.OneTimeProductListing{
			{
				LanguageCode: "en-US",
				Title:        "CI one-time product",
				Description:  "Created by gplay integration tests.",
			},
		},
		PurchaseOptions: []*androidpublisher.OneTimeProductPurchaseOption{
			{
				PurchaseOptionId:                      "buy",
				BuyOption:                             &androidpublisher.OneTimeProductBuyPurchaseOption{},
				NewRegionsConfig:                      OneTimeProductNewRegionsConfig(converted, DefaultRegionalAvailability),
				RegionalPricingAndAvailabilityConfigs: OneTimeProductRegionalConfigs(converted, DefaultRegionalAvailability),
			},
		},
	}

	created, err := service.API.Monetization.Onetimeproducts.Patch(pkg, productID, product).
		Context(ctx).
		AllowMissing(true).
		RegionsVersionVersion(regionVersion).
		UpdateMask("listings,purchaseOptions").
		Do()
	if err != nil {
		t.Fatalf("creating one-time product %s with regionVersion %s: %v", productID, regionVersion, err)
	}
	if created.ProductId != productID {
		t.Fatalf("created product id = %q, want %q", created.ProductId, productID)
	}
	if len(created.PurchaseOptions) == 0 {
		t.Fatal("expected purchase options in response")
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if err := service.API.Monetization.Onetimeproducts.Delete(pkg, productID).Context(cleanupCtx).Do(); err != nil {
			t.Logf("cleanup delete for %s failed: %v", productID, err)
		}
	})
}

func skipUnlessIntegration(t *testing.T) {
	t.Helper()
	v := os.Getenv("GPLAY_INTEGRATION_TEST")
	if v != "1" && v != "true" {
		t.Skip("skipping integration test; set GPLAY_INTEGRATION_TEST=1")
	}
}

func skipUnlessMutatingIntegration(t *testing.T) {
	t.Helper()
	v := os.Getenv("GPLAY_MUTATING_INTEGRATION_TEST")
	if v != "1" && v != "true" {
		t.Skip("skipping mutating integration test; set GPLAY_MUTATING_INTEGRATION_TEST=1")
	}
}

func integrationPackage() string {
	if v := os.Getenv("GPLAY_INTEGRATION_PACKAGE"); v != "" {
		return v
	}
	return defaultIntegrationPackage
}
