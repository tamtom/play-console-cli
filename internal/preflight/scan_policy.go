package preflight

import "fmt"

// currentMinTargetSDK is the minimum target API level Play accepts for new
// apps and updates.
//
// Google raises this every year, roughly each August, to "latest release
// minus one". Review this constant annually; `--min-target-sdk` overrides it
// without a rebuild.
const currentMinTargetSDK = 35

// lowMinSDK is the API level below which device coverage no longer justifies
// the compatibility cost.
const lowMinSDK = 21

const (
	refTargetSDK      = "https://developer.android.com/google/play/requirements/target-sdk"
	refFamiliesPolicy = "https://support.google.com/googleplay/android-developer/answer/9893335"
	refAccessibility  = "https://support.google.com/googleplay/android-developer/answer/10964491"
)

// restrictedServicePermissions map a component's bind permission to the
// policy that governs it.
var restrictedServicePermissions = map[string]struct {
	Label string
	Hint  string
	Ref   string
}{
	"android.permission.BIND_ACCESSIBILITY_SERVICE": {
		"accessibility service",
		"AccessibilityService may only be used to help users with disabilities; any other use is rejected and can suspend the account",
		refAccessibility,
	},
	"android.permission.BIND_VPN_SERVICE": {
		"VPN service",
		"VpnService requires a dedicated policy declaration explaining how traffic is handled",
		refPermissionDeclaration,
	},
	"android.permission.BIND_DEVICE_ADMIN": {
		"device admin receiver",
		"device administration APIs are restricted to enterprise management apps",
		refPermissionDeclaration,
	},
	"android.permission.BIND_NOTIFICATION_LISTENER_SERVICE": {
		"notification listener",
		"reading all notifications is sensitive data access and needs justification",
		refPermissionDeclaration,
	},
}

// scanPolicy checks Play policy deadlines and restricted app behaviors.
func scanPolicy(c *scanContext) []Finding {
	var out []Finding
	out = append(out, checkTargetSDKFloor(c)...)
	out = append(out, checkUploadFormat(c)...)
	out = append(out, checkRestrictedServices(c)...)
	out = append(out, checkFamiliesExposure(c)...)
	return out
}

// checkTargetSDKFloor enforces the Play target API level requirement.
func checkTargetSDKFloor(c *scanContext) []Finding {
	if c.manifest == nil {
		return nil
	}
	m := c.manifest

	floor := c.opts.MinTargetSDK
	if floor <= 0 {
		floor = currentMinTargetSDK
	}

	var out []Finding
	switch {
	case m.TargetSdk == 0:
		out = append(out, Finding{
			Check:    "target_sdk",
			Severity: SeverityWarning,
			Message:  "targetSdkVersion could not be determined from the manifest",
			Hint:     "Play requires an explicit target API level; check your <uses-sdk> or Gradle configuration",
			Ref:      refTargetSDK,
		})
	case m.TargetSdk < floor:
		out = append(out, Finding{
			Check:    "target_sdk",
			Severity: SeverityError,
			Message:  fmt.Sprintf("targetSdkVersion %d is below the Play minimum of %d", m.TargetSdk, floor),
			Hint:     "Play blocks uploads below the current target API requirement; raise targetSdk and retest",
			Ref:      refTargetSDK,
		})
	}

	if m.MinSdk > 0 && m.MinSdk < lowMinSDK {
		out = append(out, Finding{
			Check:    "min_sdk",
			Severity: SeverityInfo,
			Message:  fmt.Sprintf("minSdkVersion %d supports devices below Android 5.0", m.MinSdk),
			Hint:     "these versions have negligible share and no security updates; raising minSdk cuts build and test cost",
		})
	}
	return out
}

// checkUploadFormat notes when an APK is scanned rather than an App Bundle.
func checkUploadFormat(c *scanContext) []Finding {
	if c.format != formatAPK {
		return nil
	}
	return []Finding{{
		Check:    "upload_format",
		Severity: SeverityInfo,
		Message:  "scanned artifact is an APK",
		Hint:     "new apps on Play must be published as Android App Bundles; APKs are accepted only for existing apps and internal sharing",
		Ref:      "https://developer.android.com/guide/app-bundle",
	}}
}

// checkRestrictedServices flags components bound by a restricted permission.
func checkRestrictedServices(c *scanContext) []Finding {
	if c.manifest == nil {
		return nil
	}
	var out []Finding
	for _, comp := range c.manifest.Application.Components {
		rule, ok := restrictedServicePermissions[comp.Permission]
		if !ok {
			continue
		}
		out = append(out, Finding{
			Check:    "restricted_service",
			Severity: SeverityWarning,
			Message:  fmt.Sprintf("%s declares a %s", comp.Name, rule.Label),
			Entry:    comp.Name,
			Hint:     rule.Hint,
			Ref:      rule.Ref,
		})
	}
	return out
}

// checkFamiliesExposure notes the extra obligations that ads create for apps
// that may be targeted at children.
func checkFamiliesExposure(c *scanContext) []Finding {
	if len(c.sdksInCategory(categoryAds)) == 0 {
		return nil
	}
	return []Finding{{
		Check:    "families_policy",
		Severity: SeverityInfo,
		Message:  "app serves ads",
		Hint:     "if the app targets children, ads must come from a Google-certified ad SDK and no personalised ads or AD_ID may be used",
		Ref:      refFamiliesPolicy,
	}}
}
