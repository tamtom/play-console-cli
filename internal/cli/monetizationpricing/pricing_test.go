package monetizationpricing

import (
	"testing"

	"google.golang.org/api/androidpublisher/v3"
)

func TestSummary_SortsRegionsAndUsesMapKeyFallback(t *testing.T) {
	resp := convertedFixture()
	resp.ConvertedRegionPrices["TH"] = androidpublisher.ConvertedRegionPrice{
		Price: &androidpublisher.Money{CurrencyCode: "THB", Units: 29},
	}

	summary, err := Summary(resp)
	if err != nil {
		t.Fatal(err)
	}

	if summary.RegionVersion != "2026/05" {
		t.Fatalf("region version = %q, want 2026/05", summary.RegionVersion)
	}
	if summary.RegionCount != 3 {
		t.Fatalf("region count = %d, want 3", summary.RegionCount)
	}
	got := []string{
		summary.Regions[0].RegionCode + ":" + summary.Regions[0].CurrencyCode,
		summary.Regions[1].RegionCode + ":" + summary.Regions[1].CurrencyCode,
		summary.Regions[2].RegionCode + ":" + summary.Regions[2].CurrencyCode,
	}
	want := []string{"BG:BGN", "TH:THB", "US:USD"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("region %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBasePlanRegionalConfigs_UsesConvertedPrices(t *testing.T) {
	configs := BasePlanRegionalConfigs(convertedFixture())

	if len(configs) != 2 {
		t.Fatalf("configs len = %d, want 2", len(configs))
	}
	if configs[0].RegionCode != "BG" || configs[0].Price.CurrencyCode != "BGN" {
		t.Fatalf("first config = %#v", configs[0])
	}
	if !configs[0].NewSubscriberAvailability {
		t.Fatal("expected new subscriber availability")
	}
	if configs[1].RegionCode != "US" || configs[1].Price.CurrencyCode != "USD" {
		t.Fatalf("second config = %#v", configs[1])
	}
}

func TestOneTimeProductRegionalConfigs_UsesConvertedPrices(t *testing.T) {
	configs := OneTimeProductRegionalConfigs(convertedFixture(), "")

	if len(configs) != 2 {
		t.Fatalf("configs len = %d, want 2", len(configs))
	}
	if configs[0].RegionCode != "BG" || configs[0].Price.CurrencyCode != "BGN" || configs[0].Availability != "AVAILABLE" {
		t.Fatalf("first config = %#v", configs[0])
	}
}

func TestRegionVersion_MissingReturnsError(t *testing.T) {
	if _, err := RegionVersion(&androidpublisher.ConvertRegionPricesResponse{}); err == nil {
		t.Fatal("expected error")
	}
}

func convertedFixture() *androidpublisher.ConvertRegionPricesResponse {
	return &androidpublisher.ConvertRegionPricesResponse{
		RegionVersion: &androidpublisher.RegionsVersion{Version: "2026/05"},
		ConvertedRegionPrices: map[string]androidpublisher.ConvertedRegionPrice{
			"US": {
				RegionCode: "US",
				Price:      &androidpublisher.Money{CurrencyCode: "USD", Units: 9, Nanos: 990000000},
			},
			"BG": {
				RegionCode: "BG",
				Price:      &androidpublisher.Money{CurrencyCode: "BGN", Units: 18, Nanos: 990000000},
			},
		},
		ConvertedOtherRegionsPrice: &androidpublisher.ConvertedOtherRegionsPrice{
			UsdPrice: &androidpublisher.Money{CurrencyCode: "USD", Units: 9, Nanos: 990000000},
			EurPrice: &androidpublisher.Money{CurrencyCode: "EUR", Units: 8, Nanos: 990000000},
		},
	}
}
