package preflight

import (
	"fmt"
	"strings"
)

const refPaymentsPolicy = "https://support.google.com/googleplay/android-developer/answer/10281818"

// billingPermission is the legacy in-app billing permission.
const billingPermission = "com.android.vending.BILLING"

// scanBilling reports how the build takes payment. Play's Payments policy
// requires Play Billing for in-app digital goods; third-party processors are
// allowed only for physical goods and a narrow set of exemptions, so their
// presence is worth surfacing before review rather than after.
func scanBilling(c *scanContext) []Finding {
	var out []Finding

	hasPlayBilling := c.hasSDK(sdkPlayBilling)
	thirdParty := c.sdksInCategory(categoryPayment)
	wrappers := c.sdksInCategory(categoryBilling)

	if len(thirdParty) > 0 {
		sev := SeverityWarning
		hint := "Play requires Google Play Billing for in-app digital goods; third-party processors are permitted only for physical goods or an approved exemption"
		if hasPlayBilling {
			// Shipping both is normal for apps selling physical and digital
			// goods side by side.
			sev = SeverityInfo
			hint = "both Play Billing and a third-party processor are present; ensure digital goods route through Play Billing"
		}
		out = append(out, Finding{
			Check:    "third_party_payments",
			Severity: sev,
			Message:  fmt.Sprintf("third-party payment SDK detected: %s", strings.Join(thirdParty, ", ")),
			Hint:     hint,
			Ref:      refPaymentsPolicy,
		})
	}

	if c.manifest != nil {
		hasPermission := c.manifest.HasPermission(billingPermission)
		switch {
		case hasPermission && !hasPlayBilling && len(wrappers) == 0:
			out = append(out, Finding{
				Check:    "billing_permission",
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("%s is declared but no Play Billing implementation was detected", billingPermission),
				Hint:     "remove the permission if the app no longer sells in-app products",
				Ref:      refPaymentsPolicy,
			})
		case hasPlayBilling:
			out = append(out, Finding{
				Check:    "play_billing",
				Severity: SeverityInfo,
				Message:  "Google Play Billing Library detected",
				Hint:     "Play enforces a minimum Billing Library version for new uploads; keep the dependency current",
				Ref:      "https://developer.android.com/google/play/billing/deprecation-faq",
			})
		}
	}

	// Billing wrappers still use Play Billing underneath, but the underlying
	// library must be present for purchases to work at all.
	for _, w := range wrappers {
		if w == sdkNameByID(sdkPlayBilling) {
			continue
		}
		if hasPlayBilling {
			continue
		}
		out = append(out, Finding{
			Check:    "billing_wrapper",
			Severity: SeverityWarning,
			Message:  fmt.Sprintf("%s was detected without the Google Play Billing Library", w),
			Hint:     "billing wrappers depend on Play Billing; confirm the dependency was not stripped",
		})
	}
	return out
}
