package onetimeproducts

import (
	"fmt"
	"regexp"
	"strings"

	"google.golang.org/api/androidpublisher/v3"
)

const (
	maxOneTimeProductBatchSize = 100
	latencySensitive           = "PRODUCT_UPDATE_LATENCY_TOLERANCE_LATENCY_SENSITIVE"
	latencyTolerant            = "PRODUCT_UPDATE_LATENCY_TOLERANCE_LATENCY_TOLERANT"
)

var productIDPattern = regexp.MustCompile(`^[0-9a-z][0-9a-z_.]*$`)

func validateProductID(productID string) error {
	if strings.TrimSpace(productID) == "" {
		return fmt.Errorf("--product-id is required")
	}
	if !productIDPattern.MatchString(productID) {
		return fmt.Errorf("invalid product ID %q: must start with a lowercase letter or number and contain only lowercase letters, numbers, underscores, and periods", productID)
	}
	return nil
}

func validateBatchProductIDs(productIDs []string) error {
	if len(productIDs) == 0 {
		return fmt.Errorf("at least one product ID is required")
	}
	if len(productIDs) > maxOneTimeProductBatchSize {
		return fmt.Errorf("batch accepts at most %d product IDs (got %d)", maxOneTimeProductBatchSize, len(productIDs))
	}
	seen := make(map[string]struct{}, len(productIDs))
	for index, productID := range productIDs {
		productID = strings.TrimSpace(productID)
		if err := validateProductID(productID); err != nil {
			return fmt.Errorf("product ID %d: %w", index+1, err)
		}
		if _, exists := seen[productID]; exists {
			return fmt.Errorf("duplicate product ID %q: every item in a batch must be different", productID)
		}
		seen[productID] = struct{}{}
		productIDs[index] = productID
	}
	return nil
}

func requireRegionsVersion(product *androidpublisher.OneTimeProduct, regionsVersion string) error {
	if product == nil || strings.TrimSpace(regionsVersion) != "" {
		return nil
	}
	for _, purchaseOption := range product.PurchaseOptions {
		if purchaseOption != nil && (len(purchaseOption.RegionalPricingAndAvailabilityConfigs) > 0 || purchaseOption.NewRegionsConfig != nil) {
			return fmt.Errorf("--regions-version is required when regional pricing is set; use `gplay pricing regions-version` to discover the current version")
		}
	}
	return nil
}

func normalizeLatencyTolerance(value string) (string, error) {
	value = strings.TrimSpace(value)
	switch value {
	case "", latencySensitive, latencyTolerant:
		return value, nil
	default:
		return "", fmt.Errorf("invalid --latency-tolerance %q: expected %s or %s", value, latencySensitive, latencyTolerant)
	}
}

func validateBatchUpdateRequest(req *androidpublisher.BatchUpdateOneTimeProductsRequest, latencyTolerance string) error {
	if req == nil || len(req.Requests) == 0 {
		return fmt.Errorf("batch update requires at least one request")
	}
	if len(req.Requests) > maxOneTimeProductBatchSize {
		return fmt.Errorf("batch update accepts at most %d requests (got %d)", maxOneTimeProductBatchSize, len(req.Requests))
	}
	productIDs := make([]string, 0, len(req.Requests))
	for index, item := range req.Requests {
		if item == nil || item.OneTimeProduct == nil {
			return fmt.Errorf("request %d: oneTimeProduct is required", index+1)
		}
		productIDs = append(productIDs, item.OneTimeProduct.ProductId)
		if !item.AllowMissing && strings.TrimSpace(item.UpdateMask) == "" {
			return fmt.Errorf("request %d: updateMask is required when allowMissing is false", index+1)
		}
		if err := requireRegionsVersion(item.OneTimeProduct, regionsVersionValue(item.RegionsVersion)); err != nil {
			return fmt.Errorf("request %d: regionsVersion is required when regional pricing is set; use `gplay pricing regions-version` to discover the current version", index+1)
		}
		if latencyTolerance != "" {
			item.LatencyTolerance = latencyTolerance
		}
	}
	return validateBatchProductIDs(productIDs)
}

func validateBatchDeleteRequest(req *androidpublisher.BatchDeleteOneTimeProductsRequest, latencyTolerance string) error {
	if req == nil || len(req.Requests) == 0 {
		return fmt.Errorf("batch delete requires at least one request")
	}
	if len(req.Requests) > maxOneTimeProductBatchSize {
		return fmt.Errorf("batch delete accepts at most %d requests (got %d)", maxOneTimeProductBatchSize, len(req.Requests))
	}
	productIDs := make([]string, 0, len(req.Requests))
	for index, item := range req.Requests {
		if item == nil {
			return fmt.Errorf("request %d must not be null", index+1)
		}
		productIDs = append(productIDs, item.ProductId)
		if latencyTolerance != "" {
			item.LatencyTolerance = latencyTolerance
		}
	}
	return validateBatchProductIDs(productIDs)
}

func regionsVersionValue(version *androidpublisher.RegionsVersion) string {
	if version == nil {
		return ""
	}
	return version.Version
}
