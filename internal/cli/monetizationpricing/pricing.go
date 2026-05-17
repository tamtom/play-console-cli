package monetizationpricing

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"google.golang.org/api/androidpublisher/v3"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

const (
	DefaultRegionalAvailability = "AVAILABLE"
)

type RegionCurrency struct {
	RegionCode   string `json:"regionCode"`
	CurrencyCode string `json:"currencyCode"`
}

type RegionsVersionSummary struct {
	RegionVersion      string                                       `json:"regionVersion"`
	RegionCount        int                                          `json:"regionCount"`
	Regions            []RegionCurrency                             `json:"regions"`
	OtherRegionsPrices *androidpublisher.ConvertedOtherRegionsPrice `json:"otherRegionsPrices,omitempty"`
}

func LoadMoney(value string) (*androidpublisher.Money, error) {
	var price androidpublisher.Money
	if err := shared.LoadJSONArg(value, &price); err != nil {
		return nil, err
	}
	if strings.TrimSpace(price.CurrencyCode) == "" {
		return nil, fmt.Errorf("price currencyCode is required")
	}
	return &price, nil
}

func ConvertRegionPrices(ctx context.Context, service *playclient.Service, pkg string, price *androidpublisher.Money, productTaxCategoryCode string) (*androidpublisher.ConvertRegionPricesResponse, error) {
	if price == nil {
		return nil, fmt.Errorf("base price is required")
	}
	req := &androidpublisher.ConvertRegionPricesRequest{
		Price: price,
	}
	if strings.TrimSpace(productTaxCategoryCode) != "" {
		req.ProductTaxCategoryCode = strings.TrimSpace(productTaxCategoryCode)
	}

	resp, err := service.API.Monetization.ConvertRegionPrices(pkg, req).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	if _, err := RegionVersion(resp); err != nil {
		return nil, err
	}
	if len(resp.ConvertedRegionPrices) == 0 {
		return nil, fmt.Errorf("Google Play returned no converted region prices")
	}
	return resp, nil
}

func RegionVersion(resp *androidpublisher.ConvertRegionPricesResponse) (string, error) {
	if resp == nil || resp.RegionVersion == nil || strings.TrimSpace(resp.RegionVersion.Version) == "" {
		return "", fmt.Errorf("Google Play did not return a regionVersion")
	}
	return strings.TrimSpace(resp.RegionVersion.Version), nil
}

func Summary(resp *androidpublisher.ConvertRegionPricesResponse) (*RegionsVersionSummary, error) {
	version, err := RegionVersion(resp)
	if err != nil {
		return nil, err
	}
	regions := make([]RegionCurrency, 0, len(resp.ConvertedRegionPrices))
	for key, converted := range resp.ConvertedRegionPrices {
		regionCode := strings.TrimSpace(converted.RegionCode)
		if regionCode == "" {
			regionCode = key
		}
		if strings.TrimSpace(regionCode) == "" || converted.Price == nil {
			continue
		}
		regions = append(regions, RegionCurrency{
			RegionCode:   regionCode,
			CurrencyCode: converted.Price.CurrencyCode,
		})
	}
	sort.Slice(regions, func(i, j int) bool {
		return regions[i].RegionCode < regions[j].RegionCode
	})
	return &RegionsVersionSummary{
		RegionVersion:      version,
		RegionCount:        len(regions),
		Regions:            regions,
		OtherRegionsPrices: resp.ConvertedOtherRegionsPrice,
	}, nil
}

func BasePlanRegionalConfigs(resp *androidpublisher.ConvertRegionPricesResponse) []*androidpublisher.RegionalBasePlanConfig {
	keys := sortedRegionKeys(resp)
	configs := make([]*androidpublisher.RegionalBasePlanConfig, 0, len(keys))
	for _, key := range keys {
		converted := resp.ConvertedRegionPrices[key]
		regionCode := convertedRegionCode(key, converted)
		if regionCode == "" || converted.Price == nil {
			continue
		}
		configs = append(configs, &androidpublisher.RegionalBasePlanConfig{
			RegionCode:                regionCode,
			Price:                     converted.Price,
			NewSubscriberAvailability: true,
			ForceSendFields:           []string{"NewSubscriberAvailability"},
		})
	}
	return configs
}

func OtherRegionsBasePlanConfig(resp *androidpublisher.ConvertRegionPricesResponse) *androidpublisher.OtherRegionsBasePlanConfig {
	if resp == nil || resp.ConvertedOtherRegionsPrice == nil {
		return nil
	}
	return &androidpublisher.OtherRegionsBasePlanConfig{
		EurPrice:                  resp.ConvertedOtherRegionsPrice.EurPrice,
		UsdPrice:                  resp.ConvertedOtherRegionsPrice.UsdPrice,
		NewSubscriberAvailability: true,
		ForceSendFields:           []string{"NewSubscriberAvailability"},
	}
}

func OneTimeProductRegionalConfigs(resp *androidpublisher.ConvertRegionPricesResponse, availability string) []*androidpublisher.OneTimeProductPurchaseOptionRegionalPricingAndAvailabilityConfig {
	availability = strings.TrimSpace(availability)
	if availability == "" {
		availability = DefaultRegionalAvailability
	}
	keys := sortedRegionKeys(resp)
	configs := make([]*androidpublisher.OneTimeProductPurchaseOptionRegionalPricingAndAvailabilityConfig, 0, len(keys))
	for _, key := range keys {
		converted := resp.ConvertedRegionPrices[key]
		regionCode := convertedRegionCode(key, converted)
		if regionCode == "" || converted.Price == nil {
			continue
		}
		configs = append(configs, &androidpublisher.OneTimeProductPurchaseOptionRegionalPricingAndAvailabilityConfig{
			RegionCode:   regionCode,
			Availability: availability,
			Price:        converted.Price,
		})
	}
	return configs
}

func OneTimeProductNewRegionsConfig(resp *androidpublisher.ConvertRegionPricesResponse, availability string) *androidpublisher.OneTimeProductPurchaseOptionNewRegionsConfig {
	if resp == nil || resp.ConvertedOtherRegionsPrice == nil {
		return nil
	}
	availability = strings.TrimSpace(availability)
	if availability == "" {
		availability = DefaultRegionalAvailability
	}
	return &androidpublisher.OneTimeProductPurchaseOptionNewRegionsConfig{
		Availability: availability,
		EurPrice:     resp.ConvertedOtherRegionsPrice.EurPrice,
		UsdPrice:     resp.ConvertedOtherRegionsPrice.UsdPrice,
	}
}

func sortedRegionKeys(resp *androidpublisher.ConvertRegionPricesResponse) []string {
	if resp == nil {
		return nil
	}
	keys := make([]string, 0, len(resp.ConvertedRegionPrices))
	for key := range resp.ConvertedRegionPrices {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func convertedRegionCode(key string, converted androidpublisher.ConvertedRegionPrice) string {
	if strings.TrimSpace(converted.RegionCode) != "" {
		return strings.TrimSpace(converted.RegionCode)
	}
	return strings.TrimSpace(key)
}
