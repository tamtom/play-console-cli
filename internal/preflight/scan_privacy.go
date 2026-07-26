package preflight

import (
	"fmt"
	"strings"
)

// adIDPermission is required to read the advertising ID from Android 13.
const adIDPermission = "com.google.android.gms.permission.AD_ID"

// adIDTargetSDK is the target SDK from which AD_ID must be declared.
const adIDTargetSDK = 33

// scanPrivacy surfaces the tracking and advertising SDKs in the build and the
// Data safety obligations they create.
func scanPrivacy(c *scanContext) []Finding {
	tracking := c.sdksInCategory(categoryTracking)
	ads := c.sdksInCategory(categoryAds)

	var out []Finding

	if len(tracking) > 0 {
		out = append(out, Finding{
			Check:    "tracking_sdk",
			Severity: SeverityInfo,
			Message:  fmt.Sprintf("analytics/attribution SDK detected: %s", strings.Join(tracking, ", ")),
			Hint:     "every data type these SDKs collect must be disclosed in the Data safety form, including data collected by the SDK itself",
			Ref:      refDataSafety,
		})
	}
	if len(ads) > 0 {
		out = append(out, Finding{
			Check:    "ads_sdk",
			Severity: SeverityInfo,
			Message:  fmt.Sprintf("advertising SDK detected: %s", strings.Join(ads, ", ")),
			Hint:     "declare the app as containing ads in the Play Console and disclose ad-related data collection",
			Ref:      refDataSafety,
		})
	}

	out = append(out, checkAdvertisingID(c, len(ads) > 0, len(tracking) > 0)...)
	out = append(out, checkNetworkPrivacy(c, len(tracking)+len(ads) > 0)...)
	return out
}

// checkAdvertisingID reconciles the AD_ID permission with the SDKs present.
func checkAdvertisingID(c *scanContext, hasAds, hasTracking bool) []Finding {
	if c.manifest == nil {
		return nil
	}
	m := c.manifest
	declared := m.HasPermission(adIDPermission)

	switch {
	case !declared && (hasAds || hasTracking) && m.TargetSdk >= adIDTargetSDK:
		return []Finding{{
			Check:    "advertising_id",
			Severity: SeverityWarning,
			Message:  fmt.Sprintf("an ads/attribution SDK is present but %s is not declared (targetSdk %d)", adIDPermission, m.TargetSdk),
			Hint:     "apps targeting Android 13+ must declare this permission to receive the advertising ID; without it the SDK reads zeros",
			Ref:      "https://support.google.com/googleplay/android-developer/answer/6048248",
		}}
	case declared && !hasAds && !hasTracking:
		return []Finding{{
			Check:    "advertising_id",
			Severity: SeverityInfo,
			Message:  fmt.Sprintf("%s is declared but no ads or attribution SDK was detected", adIDPermission),
			Hint:     "remove the permission if the advertising ID is unused; declaring it commits you to a Data safety disclosure",
			Ref:      refDataSafety,
		}}
	}
	return nil
}

// checkNetworkPrivacy flags transport-level exposure that affects the data
// safety story.
func checkNetworkPrivacy(c *scanContext, hasDataSDK bool) []Finding {
	if c.manifest == nil {
		return nil
	}
	m := c.manifest

	var out []Finding
	if hasDataSDK && isTrue(m.Application.UsesCleartextTraffic) {
		out = append(out, Finding{
			Check:    "insecure_transit",
			Severity: SeverityWarning,
			Message:  "cleartext traffic is enabled while data-collecting SDKs are present",
			Hint:     "the Data safety form asks whether data is encrypted in transit; cleartext contradicts that claim",
			Ref:      refDataSafety,
		})
	}
	if hasDataSDK && isTrue(m.Application.AllowBackup) {
		out = append(out, Finding{
			Check:    "backup_exposure",
			Severity: SeverityInfo,
			Message:  "allowBackup is enabled while data-collecting SDKs are present",
			Hint:     "exclude analytics identifiers and tokens from backups via android:dataExtractionRules",
		})
	}
	return out
}
