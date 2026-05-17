package cmdtest_test

import (
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/cmdtest"
	"github.com/tamtom/play-console-cli/internal/testutil"
)

func TestOnetimeproductsCreate_DryRunWiresUpsertRequest(t *testing.T) {
	t.Setenv("GPLAY_SERVICE_ACCOUNT_JSON", testutil.MockServiceAccount(t))
	cmdtest.Build(t)

	productJSON := `{
		"listings": [
			{
				"languageCode": "en-US",
				"title": "100 Coins",
				"description": "A pack of 100 coins"
			}
		],
		"purchaseOptions": [
			{
				"purchaseOptionId": "default",
				"buyOption": {},
				"regionalPricingAndAvailabilityConfigs": [
					{
						"regionCode": "US",
						"availability": "AVAILABLE",
						"price": {
							"currencyCode": "USD",
							"units": "1",
							"nanos": 990000000
						}
					}
				]
			}
		]
	}`

	r := cmdtest.Run(t,
		"--dry-run",
		"onetimeproducts",
		"create",
		"--package", "com.example.app",
		"--product-id", "coins_100_test",
		"--json", productJSON,
		"--regions-version", "2022/02",
	)

	cmdtest.AssertExitCode(t, r.ExitCode, 0)
	assertContainsAll(t, r.Stderr, []string{
		"[DRY RUN] PATCH",
		"/androidpublisher/v3/applications/com.example.app/onetimeproducts/coins_100_test",
		"allowMissing=true",
		"regionsVersion.version=2022%2F02",
		"updateMask=listings%2CpurchaseOptions",
		`"packageName":"com.example.app"`,
		`"productId":"coins_100_test"`,
	})
}

func assertContainsAll(t *testing.T, got string, substrings []string) {
	t.Helper()
	for _, substring := range substrings {
		if !strings.Contains(got, substring) {
			t.Fatalf("expected output to contain %q\noutput:\n%s", substring, got)
		}
	}
}
