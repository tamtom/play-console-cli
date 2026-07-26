package preflight

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// permissionRule describes how a permission should be reported.
type permissionRule struct {
	Severity Severity
	Reason   string
	Hint     string
	Ref      string
}

const (
	refPermissionDeclaration = "https://support.google.com/googleplay/android-developer/answer/9214102"
	refAllFilesAccess        = "https://support.google.com/googleplay/android-developer/answer/10467955"
	refBackgroundLocation    = "https://support.google.com/googleplay/android-developer/answer/9799150"
	refPackageVisibility     = "https://support.google.com/googleplay/android-developer/answer/10158779"
	refDataSafety            = "https://support.google.com/googleplay/android-developer/answer/10787469"
)

// restrictedPermissions require an approved Play declaration. Shipping them
// without one is the single most common cause of a policy rejection.
var restrictedPermissions = map[string]permissionRule{
	"android.permission.READ_SMS":                           {SeverityWarning, "SMS permission group", "requires an approved Permissions Declaration; only default SMS handlers qualify", refPermissionDeclaration},
	"android.permission.SEND_SMS":                           {SeverityWarning, "SMS permission group", "requires an approved Permissions Declaration; only default SMS handlers qualify", refPermissionDeclaration},
	"android.permission.RECEIVE_SMS":                        {SeverityWarning, "SMS permission group", "requires an approved Permissions Declaration; only default SMS handlers qualify", refPermissionDeclaration},
	"android.permission.RECEIVE_MMS":                        {SeverityWarning, "SMS permission group", "requires an approved Permissions Declaration", refPermissionDeclaration},
	"android.permission.RECEIVE_WAP_PUSH":                   {SeverityWarning, "SMS permission group", "requires an approved Permissions Declaration", refPermissionDeclaration},
	"android.permission.WRITE_SMS":                          {SeverityWarning, "SMS permission group", "requires an approved Permissions Declaration", refPermissionDeclaration},
	"android.permission.READ_CALL_LOG":                      {SeverityWarning, "Call Log permission group", "requires an approved Permissions Declaration; only default phone/assistant handlers qualify", refPermissionDeclaration},
	"android.permission.WRITE_CALL_LOG":                     {SeverityWarning, "Call Log permission group", "requires an approved Permissions Declaration", refPermissionDeclaration},
	"android.permission.PROCESS_OUTGOING_CALLS":             {SeverityWarning, "Call Log permission group", "deprecated and restricted; migrate to CallRedirectionService", refPermissionDeclaration},
	"android.permission.MANAGE_EXTERNAL_STORAGE":            {SeverityWarning, "All files access", "requires an approved All Files Access declaration; most apps should use scoped storage or the photo picker", refAllFilesAccess},
	"android.permission.ACCESS_BACKGROUND_LOCATION":         {SeverityWarning, "Background location", "requires an approved Location Permissions declaration and a demo video", refBackgroundLocation},
	"android.permission.QUERY_ALL_PACKAGES":                 {SeverityWarning, "Broad package visibility", "requires justification; prefer a <queries> element listing the packages you need", refPackageVisibility},
	"android.permission.REQUEST_INSTALL_PACKAGES":           {SeverityWarning, "Unknown app install", "restricted to apps whose core purpose is installing other apps", refPermissionDeclaration},
	"android.permission.SYSTEM_ALERT_WINDOW":                {SeverityWarning, "Display over other apps", "overlays are heavily restricted and a common rejection reason", refPermissionDeclaration},
	"android.permission.PACKAGE_USAGE_STATS":                {SeverityWarning, "Usage access", "requires justification; usage data is treated as sensitive", refPermissionDeclaration},
	"android.permission.BIND_ACCESSIBILITY_SERVICE":         {SeverityWarning, "Accessibility service", "AccessibilityService may only be used for accessibility; other uses are rejected", refPermissionDeclaration},
	"android.permission.BIND_DEVICE_ADMIN":                  {SeverityWarning, "Device admin", "device admin APIs are restricted to enterprise management use cases", refPermissionDeclaration},
	"android.permission.USE_FULL_SCREEN_INTENT":             {SeverityWarning, "Full-screen intent", "from Android 14 this is granted only to calling and alarm apps", refPermissionDeclaration},
	"android.permission.SCHEDULE_EXACT_ALARM":               {SeverityWarning, "Exact alarms", "requires justification; prefer setAndAllowWhileIdle or WorkManager", refPermissionDeclaration},
	"android.permission.USE_EXACT_ALARM":                    {SeverityWarning, "Exact alarms", "restricted to alarm clock and calendar apps", refPermissionDeclaration},
	"android.permission.MANAGE_MEDIA":                       {SeverityWarning, "Manage media", "requires justification for bulk media modification", refPermissionDeclaration},
	"android.permission.BIND_NOTIFICATION_LISTENER_SERVICE": {SeverityWarning, "Notification access", "reading all notifications requires justification and is treated as sensitive data", refPermissionDeclaration},
	"android.permission.BIND_VPN_SERVICE":                   {SeverityWarning, "VPN service", "VpnService use is restricted and requires a specific policy declaration", refPermissionDeclaration},
}

// sensitivePermissions are legitimate but must be disclosed in Data safety.
var sensitivePermissions = map[string]permissionRule{
	"android.permission.CAMERA":                               {SeverityInfo, "Camera access", "disclose photo/video collection in the Data safety form", refDataSafety},
	"android.permission.RECORD_AUDIO":                         {SeverityInfo, "Microphone access", "disclose audio collection in the Data safety form", refDataSafety},
	"android.permission.ACCESS_FINE_LOCATION":                 {SeverityInfo, "Precise location", "disclose location collection in the Data safety form", refDataSafety},
	"android.permission.ACCESS_COARSE_LOCATION":               {SeverityInfo, "Approximate location", "disclose location collection in the Data safety form", refDataSafety},
	"android.permission.READ_CONTACTS":                        {SeverityInfo, "Contacts access", "disclose contact collection in the Data safety form", refDataSafety},
	"android.permission.WRITE_CONTACTS":                       {SeverityInfo, "Contacts modification", "disclose contact access in the Data safety form", refDataSafety},
	"android.permission.GET_ACCOUNTS":                         {SeverityInfo, "Account list access", "disclose account access in the Data safety form", refDataSafety},
	"android.permission.BODY_SENSORS":                         {SeverityInfo, "Body sensors", "health data must be disclosed and may need extra policy review", refDataSafety},
	"android.permission.ACTIVITY_RECOGNITION":                 {SeverityInfo, "Physical activity", "disclose fitness data collection in the Data safety form", refDataSafety},
	"android.permission.READ_PHONE_STATE":                     {SeverityInfo, "Phone state", "device identifiers are sensitive; avoid using them for tracking", refDataSafety},
	"android.permission.READ_PHONE_NUMBERS":                   {SeverityInfo, "Phone number", "disclose phone number collection in the Data safety form", refDataSafety},
	"android.permission.READ_MEDIA_IMAGES":                    {SeverityInfo, "Photo access", "consider the Android photo picker, which needs no permission", refDataSafety},
	"android.permission.READ_MEDIA_VIDEO":                     {SeverityInfo, "Video access", "consider the Android photo picker, which needs no permission", refDataSafety},
	"android.permission.READ_MEDIA_AUDIO":                     {SeverityInfo, "Audio file access", "disclose audio file access in the Data safety form", refDataSafety},
	"android.permission.POST_NOTIFICATIONS":                   {SeverityInfo, "Notifications", "must be requested at runtime from Android 13", ""},
	"com.google.android.gms.permission.AD_ID":                 {SeverityInfo, "Advertising ID", "declare advertising ID use in the Data safety form", refDataSafety},
	"android.permission.RECEIVE_BOOT_COMPLETED":               {SeverityInfo, "Start at boot", "background startup is scrutinized for battery impact", ""},
	"android.permission.FOREGROUND_SERVICE":                   {SeverityInfo, "Foreground service", "from Android 14 each service also needs a typed FOREGROUND_SERVICE_* permission", ""},
	"android.permission.REQUEST_IGNORE_BATTERY_OPTIMIZATIONS": {SeverityWarning, "Battery optimisation exemption", "allowed only for a narrow set of app categories", refPermissionDeclaration},
}

// deprecatedPermissions no longer do anything on modern Android.
var deprecatedPermissions = map[string]string{
	"android.permission.GET_TASKS":                  "removed in API 21; use ActivityManager.getAppTasks",
	"android.permission.PERSISTENT_ACTIVITY":        "removed in API 15",
	"android.permission.RESTART_PACKAGES":           "no-op since API 15",
	"android.permission.SET_PREFERRED_APPLICATIONS": "no-op since API 15",
	"android.permission.SMS_FINANCIAL_TRANSACTIONS": "removed in API 31",
}

// scanPermissions audits declared permissions against Play policy.
func scanPermissions(c *scanContext) []Finding {
	if c.manifest == nil {
		return heuristicPermissionChecks(c.manifestRaw)
	}
	m := c.manifest

	var out []Finding
	seen := map[string]int{}

	for _, p := range m.Permissions {
		seen[p.Name]++

		if rule, ok := restrictedPermissions[p.Name]; ok {
			out = append(out, Finding{
				Check:    "dangerous_permissions",
				Severity: rule.Severity,
				Message:  fmt.Sprintf("%s (%s)", p.Name, rule.Reason),
				Entry:    p.Name,
				Hint:     rule.Hint,
				Ref:      rule.Ref,
			})
			continue
		}
		if rule, ok := sensitivePermissions[p.Name]; ok {
			out = append(out, Finding{
				Check:    "dangerous_permissions",
				Severity: rule.Severity,
				Message:  fmt.Sprintf("%s (%s)", p.Name, rule.Reason),
				Entry:    p.Name,
				Hint:     rule.Hint,
				Ref:      rule.Ref,
			})
			continue
		}
		if why, ok := deprecatedPermissions[p.Name]; ok {
			out = append(out, Finding{
				Check:    "deprecated_permission",
				Severity: SeverityInfo,
				Message:  fmt.Sprintf("%s is deprecated: %s", p.Name, why),
				Entry:    p.Name,
				Hint:     "remove it; it grants nothing on supported Android versions",
			})
		}
	}

	out = append(out, checkStoragePermissions(m)...)
	out = append(out, checkDuplicatePermissions(seen)...)
	return out
}

// checkStoragePermissions flags legacy storage permissions that no longer
// apply on modern target SDKs.
func checkStoragePermissions(m *Manifest) []Finding {
	var out []Finding
	for _, p := range m.Permissions {
		switch p.Name {
		case "android.permission.WRITE_EXTERNAL_STORAGE":
			if m.TargetSdk >= 30 && p.MaxSdk == 0 {
				out = append(out, Finding{
					Check:    "legacy_storage_permission",
					Severity: SeverityWarning,
					Message:  fmt.Sprintf("WRITE_EXTERNAL_STORAGE is declared without android:maxSdkVersion on targetSdk %d", m.TargetSdk),
					Entry:    p.Name,
					Hint:     `the permission does nothing from Android 11; add android:maxSdkVersion="28" or remove it`,
				})
			}
		case "android.permission.READ_EXTERNAL_STORAGE":
			if m.TargetSdk >= 33 && p.MaxSdk == 0 {
				out = append(out, Finding{
					Check:    "legacy_storage_permission",
					Severity: SeverityWarning,
					Message:  fmt.Sprintf("READ_EXTERNAL_STORAGE is declared without android:maxSdkVersion on targetSdk %d", m.TargetSdk),
					Entry:    p.Name,
					Hint:     `superseded by READ_MEDIA_* from Android 13; add android:maxSdkVersion="32"`,
				})
			}
		}
	}
	return out
}

// checkDuplicatePermissions reports permissions declared more than once.
func checkDuplicatePermissions(seen map[string]int) []Finding {
	var dupes []string
	for name, n := range seen {
		if n > 1 {
			dupes = append(dupes, name)
		}
	}
	if len(dupes) == 0 {
		return nil
	}
	sort.Strings(dupes)
	return []Finding{{
		Check:    "duplicate_permission",
		Severity: SeverityInfo,
		Message:  fmt.Sprintf("declared more than once: %s", strings.Join(dupes, ", ")),
		Hint:     "usually a merged-manifest artifact from a library; harmless but worth tidying",
	}}
}

// heuristicPermissionChecks scans raw manifest bytes when decoding failed.
// Permission names are stored as plain UTF-8 strings in both manifest
// encodings, so substring matching stays reliable here.
func heuristicPermissionChecks(raw []byte) []Finding {
	if len(raw) == 0 {
		return nil
	}
	var out []Finding

	names := make([]string, 0, len(restrictedPermissions)+len(sensitivePermissions))
	for name := range restrictedPermissions {
		names = append(names, name)
	}
	for name := range sensitivePermissions {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if !bytes.Contains(raw, []byte(name)) {
			continue
		}
		rule, ok := restrictedPermissions[name]
		if !ok {
			rule = sensitivePermissions[name]
		}
		out = append(out, Finding{
			Check:    "dangerous_permissions",
			Severity: rule.Severity,
			Message:  fmt.Sprintf("%s (%s)", name, rule.Reason),
			Entry:    name,
			Hint:     rule.Hint,
			Ref:      rule.Ref,
		})
	}
	return out
}
