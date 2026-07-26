package preflight

import (
	"bytes"
	"fmt"
	"strings"
)

// scanManifest checks the structural and security-relevant properties of
// AndroidManifest.xml. When the manifest cannot be decoded it falls back to
// substring heuristics so an unusual or obfuscated build still gets coverage.
func scanManifest(c *scanContext) []Finding {
	var out []Finding

	if len(c.manifestRaw) == 0 {
		return append(out, Finding{
			Check:    "manifest",
			Severity: SeverityError,
			Message:  "AndroidManifest.xml not found",
			Hint:     "the archive must contain base/manifest/AndroidManifest.xml (AAB) or AndroidManifest.xml (APK)",
		})
	}

	out = append(out, checkResourceTable(c)...)

	if c.manifest == nil {
		out = append(out, Finding{
			Check:    "manifest_undecodable",
			Severity: SeverityWarning,
			Message:  fmt.Sprintf("AndroidManifest.xml could not be decoded (%v); falling back to heuristic checks", c.manifestErr),
			Hint:     "structured manifest checks are unavailable for this build",
		})
		return append(out, heuristicManifestChecks(c.manifestRaw)...)
	}

	m := c.manifest
	out = append(out, checkManifestIdentity(m)...)
	out = append(out, checkManifestSecurityFlags(m)...)
	out = append(out, checkExportedComponents(m)...)
	out = append(out, checkForegroundServiceTypes(m)...)
	return out
}

// checkResourceTable verifies a compiled resource table is present.
func checkResourceTable(c *scanContext) []Finding {
	if c.hasEntry("base/resources.pb") || c.hasEntry("resources.arsc") {
		return nil
	}
	// Feature-only modules legitimately ship without a base resource table.
	if c.hasEntrySuffix("/resources.pb") {
		return nil
	}
	return []Finding{{
		Check:    "resources",
		Severity: SeverityError,
		Message:  "resource table not found (base/resources.pb or resources.arsc)",
		Hint:     "the upload does not look like a complete app bundle or APK",
	}}
}

// checkManifestIdentity validates package name and version metadata.
func checkManifestIdentity(m *Manifest) []Finding {
	var out []Finding

	if m.Package == "" {
		out = append(out, Finding{
			Check:    "package_name",
			Severity: SeverityError,
			Message:  "manifest declares no package name",
		})
	} else if isPlaceholderPackage(m.Package) {
		out = append(out, Finding{
			Check:    "package_name",
			Severity: SeverityError,
			Message:  fmt.Sprintf("package name %q uses a reserved placeholder prefix", m.Package),
			Hint:     "Play rejects applicationIds under com.example, com.android, or android",
		})
	}

	if !m.Application.Present {
		out = append(out, Finding{
			Check:    "application",
			Severity: SeverityError,
			Message:  "manifest has no <application> element",
		})
	}

	if m.VersionCode <= 0 {
		out = append(out, Finding{
			Check:    "version_code",
			Severity: SeverityWarning,
			Message:  "versionCode is missing or not a positive integer",
			Hint:     "Play requires a versionCode strictly greater than any previously uploaded build",
		})
	}

	if m.MinSdk > 0 && m.TargetSdk > 0 && m.MinSdk > m.TargetSdk {
		out = append(out, Finding{
			Check:    "sdk_range",
			Severity: SeverityError,
			Message:  fmt.Sprintf("minSdkVersion %d is greater than targetSdkVersion %d", m.MinSdk, m.TargetSdk),
		})
	}
	return out
}

// isPlaceholderPackage reports whether a package name uses a prefix Play
// refuses to publish.
func isPlaceholderPackage(pkg string) bool {
	for _, bad := range []string{"com.example.", "com.android.", "android."} {
		if strings.HasPrefix(pkg, bad) {
			return true
		}
	}
	return pkg == "com.example" || pkg == "com.android"
}

// checkManifestSecurityFlags inspects the security-relevant application flags.
func checkManifestSecurityFlags(m *Manifest) []Finding {
	var out []Finding
	app := m.Application

	if isTrue(app.Debuggable) {
		out = append(out, Finding{
			Check:    "debuggable",
			Severity: SeverityError,
			Message:  "android:debuggable is true",
			Hint:     "release builds must not be debuggable; Play rejects debuggable uploads",
		})
	}

	if isTrue(m.TestOnly) {
		out = append(out, Finding{
			Check:    "test_only",
			Severity: SeverityError,
			Message:  "android:testOnly is true",
			Hint:     "remove testOnly; it is set by `gradlew installDebug` style builds and blocks Play upload",
		})
	}

	if isTrue(app.UsesCleartextTraffic) {
		sev, hint := SeverityWarning, "prefer HTTPS everywhere; cleartext traffic weakens transport security"
		if app.NetworkSecurityConfig != "" {
			sev = SeverityInfo
			hint = "a network security config is present, which may already restrict cleartext to specific domains"
		}
		out = append(out, Finding{
			Check:    "cleartext_traffic",
			Severity: sev,
			Message:  "android:usesCleartextTraffic is true",
			Hint:     hint,
		})
	}

	if isTrue(app.AllowBackup) {
		out = append(out, Finding{
			Check:    "allow_backup",
			Severity: SeverityWarning,
			Message:  "android:allowBackup is true",
			Hint:     "app data can be extracted via adb backup on some devices; set false or supply a backup rules file",
		})
	}

	if isTrue(app.RequestLegacyExternalStorage) && m.TargetSdk >= 30 {
		out = append(out, Finding{
			Check:    "legacy_storage",
			Severity: SeverityWarning,
			Message:  fmt.Sprintf("requestLegacyExternalStorage is set but targetSdk %d ignores it", m.TargetSdk),
			Hint:     "scoped storage is mandatory from Android 11; migrate off legacy external storage",
		})
	}
	return out
}

// checkExportedComponents flags components reachable by other apps.
func checkExportedComponents(m *Manifest) []Finding {
	var out []Finding

	for _, comp := range m.Application.Components {
		name := comp.Name
		if name == "" {
			name = "(unnamed " + comp.Kind + ")"
		}

		// Android 12 requires an explicit android:exported on any component
		// with an intent filter; the install fails outright without it.
		if comp.HasIntentFilter && comp.Exported == nil && m.TargetSdk >= 31 {
			out = append(out, Finding{
				Check:    "exported_undeclared",
				Severity: SeverityError,
				Message:  fmt.Sprintf("%s %s has an intent filter but no android:exported", comp.Kind, name),
				Entry:    name,
				Hint:     "Android 12+ refuses to install apps whose filtered components omit android:exported",
				Ref:      "https://developer.android.com/guide/topics/manifest/activity-element#exported",
			})
			continue
		}

		if !isTrue(comp.Exported) {
			continue
		}
		// A launcher activity is exported by design.
		if comp.IsLauncher {
			continue
		}
		if comp.Permission != "" {
			continue
		}

		sev := SeverityWarning
		if comp.Kind == "activity" && comp.HasIntentFilter {
			// Deep-link style activities are commonly exported on purpose.
			sev = SeverityInfo
		}
		out = append(out, Finding{
			Check:    "exported_component",
			Severity: sev,
			Message:  fmt.Sprintf("%s %s is exported without a permission guard", comp.Kind, name),
			Entry:    name,
			Hint:     "any installed app can reach this component; add android:permission or set exported=false",
		})

		if comp.Kind == "provider" && isTrue(comp.GrantURIPermissions) {
			out = append(out, Finding{
				Check:    "exported_provider",
				Severity: SeverityError,
				Message:  fmt.Sprintf("provider %s is exported and grants URI permissions", name),
				Entry:    name,
				Hint:     "an exported provider with grantUriPermissions can leak private files to any app",
			})
		}
	}
	return out
}

// foregroundServiceTypesNeedingNoPermission are the types Android does not
// gate behind a dedicated FOREGROUND_SERVICE_* permission.
var foregroundServiceTypesNeedingNoPermission = map[string]bool{
	"shortService": true,
}

// checkForegroundServiceTypes verifies each declared foreground service type
// has its matching runtime permission. Android 14 throws
// SecurityException at startForeground() when the permission is absent.
func checkForegroundServiceTypes(m *Manifest) []Finding {
	var out []Finding
	if m.TargetSdk > 0 && m.TargetSdk < 34 {
		return nil
	}

	for _, comp := range m.Application.Components {
		if comp.Kind != "service" {
			continue
		}
		for _, t := range comp.ForegroundServiceTypes {
			if foregroundServiceTypesNeedingNoPermission[t] {
				continue
			}
			perm := "android.permission.FOREGROUND_SERVICE_" + camelToScreamingSnake(t)
			if m.HasPermission(perm) {
				continue
			}
			out = append(out, Finding{
				Check:    "foreground_service_type",
				Severity: SeverityError,
				Message:  fmt.Sprintf("service %s declares foregroundServiceType %q without %s", comp.Name, t, perm),
				Entry:    comp.Name,
				Hint:     "Android 14 throws SecurityException at startForeground() when the type permission is missing",
				Ref:      "https://developer.android.com/about/versions/14/changes/fgs-types-required",
			})
		}
	}

	// A declared FGS type permission with no service using it is dead weight
	// and, more importantly, expands the Play declaration surface.
	declared := map[string]bool{}
	for _, comp := range m.Application.Components {
		for _, t := range comp.ForegroundServiceTypes {
			declared[strings.ToLower(t)] = true
		}
	}
	for _, p := range m.Permissions {
		const prefix = "android.permission.FOREGROUND_SERVICE_"
		if !strings.HasPrefix(p.Name, prefix) {
			continue
		}
		suffix := strings.TrimPrefix(p.Name, prefix)
		if declared[strings.ToLower(strings.ReplaceAll(suffix, "_", ""))] {
			continue
		}
		out = append(out, Finding{
			Check:    "foreground_service_unused",
			Severity: SeverityInfo,
			Message:  fmt.Sprintf("%s is declared but no service uses that foregroundServiceType", p.Name),
			Hint:     "remove the permission or add the matching android:foregroundServiceType",
		})
	}
	return out
}

// camelToScreamingSnake converts "mediaPlayback" to "MEDIA_PLAYBACK".
func camelToScreamingSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r)
			continue
		}
		if r >= 'a' && r <= 'z' {
			b.WriteRune(r - 32)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// heuristicManifestChecks runs substring checks used only when the manifest
// could not be decoded. Binary manifests store booleans as typed values, so a
// literal `="true"` match means the payload is text or unusual — either way,
// worth reporting.
func heuristicManifestChecks(manifest []byte) []Finding {
	var out []Finding

	type probe struct {
		attr     string
		check    string
		severity Severity
		message  string
		hint     string
	}
	probes := []probe{
		{"debuggable", "debuggable", SeverityError, "android:debuggable appears to be true", "release builds must not be debuggable"},
		{"testOnly", "test_only", SeverityError, "manifest appears to set android:testOnly=true", "remove testOnly before uploading"},
		{"usesCleartextTraffic", "cleartext_traffic", SeverityWarning, "android:usesCleartextTraffic appears to be true", "prefer HTTPS"},
	}

	for _, p := range probes {
		if !bytes.Contains(manifest, []byte(p.attr)) {
			continue
		}
		literal := []byte(p.attr + "=\"true\"")
		if bytes.Contains(manifest, literal) || containsNearby(manifest, []byte(p.attr), []byte{0x01}, 4) {
			out = append(out, Finding{
				Check:    p.check,
				Severity: p.severity,
				Message:  p.message,
				Hint:     p.hint,
			})
		}
	}
	return out
}

// containsNearby reports whether needle appears within window bytes after a
// matching context marker.
func containsNearby(haystack, context, needle []byte, window int) bool {
	if len(context) == 0 || len(needle) == 0 {
		return false
	}
	idx := bytes.Index(haystack, context)
	if idx < 0 {
		return false
	}
	end := idx + len(context) + window + len(needle)
	if end > len(haystack) {
		end = len(haystack)
	}
	return bytes.Contains(haystack[idx:end], needle)
}
