package preflight

import (
	"fmt"
	"strings"
)

// Manifest is the decoded, format-independent view of an AndroidManifest.xml.
//
// Optional booleans are pointers so scanners can tell "absent" and "not
// statically determinable" apart from an explicit false. A nil pointer means
// the attribute was missing or resolved to a resource reference.
type Manifest struct {
	Package     string               `json:"package,omitempty"`
	VersionCode int64                `json:"version_code,omitempty"`
	VersionName string               `json:"version_name,omitempty"`
	MinSdk      int                  `json:"min_sdk,omitempty"`
	TargetSdk   int                  `json:"target_sdk,omitempty"`
	TestOnly    *bool                `json:"test_only,omitempty"`
	Permissions []ManifestPermission `json:"permissions,omitempty"`
	Features    []ManifestFeature    `json:"features,omitempty"`
	Application Application          `json:"application"`

	// Root is the raw decoded tree, kept for checks that need to look at
	// elements the typed model does not model explicitly.
	Root *Node `json:"-"`
}

// ManifestPermission is a <uses-permission> declaration.
type ManifestPermission struct {
	Name   string `json:"name"`
	MaxSdk int    `json:"max_sdk,omitempty"`
}

// ManifestFeature is a <uses-feature> declaration.
type ManifestFeature struct {
	Name     string `json:"name"`
	Required *bool  `json:"required,omitempty"`
}

// Application is the decoded <application> element.
type Application struct {
	Present                      bool            `json:"present"`
	Name                         string          `json:"name,omitempty"`
	Debuggable                   *bool           `json:"debuggable,omitempty"`
	AllowBackup                  *bool           `json:"allow_backup,omitempty"`
	UsesCleartextTraffic         *bool           `json:"uses_cleartext_traffic,omitempty"`
	NetworkSecurityConfig        string          `json:"network_security_config,omitempty"`
	RequestLegacyExternalStorage *bool           `json:"request_legacy_external_storage,omitempty"`
	ExtractNativeLibs            *bool           `json:"extract_native_libs,omitempty"`
	MetaData                     []MetaDataEntry `json:"meta_data,omitempty"`
	Components                   []Component     `json:"components,omitempty"`
}

// MetaDataEntry is an <meta-data> child of <application>.
type MetaDataEntry struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

// Component is an activity, activity-alias, service, receiver, or provider.
type Component struct {
	Kind                   string   `json:"kind"`
	Name                   string   `json:"name"`
	Exported               *bool    `json:"exported,omitempty"`
	Permission             string   `json:"permission,omitempty"`
	HasIntentFilter        bool     `json:"has_intent_filter,omitempty"`
	IsLauncher             bool     `json:"is_launcher,omitempty"`
	ForegroundServiceTypes []string `json:"foreground_service_types,omitempty"`
	GrantURIPermissions    *bool    `json:"grant_uri_permissions,omitempty"`
}

// componentKinds are the manifest elements treated as app components.
var componentKinds = []string{"activity", "activity-alias", "service", "receiver", "provider"}

// parseManifestBytes decodes either an APK binary manifest or an AAB protobuf
// manifest into the typed model.
func parseManifestBytes(data []byte) (*Manifest, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("manifest: empty")
	}
	var (
		root *Node
		err  error
	)
	if isAXML(data) {
		root, err = parseAXML(data)
	} else {
		root, err = parseProtoXML(data)
	}
	if err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}
	if root.Name != "manifest" {
		return nil, fmt.Errorf("manifest: root element is %q, want manifest", root.Name)
	}
	return manifestFromNode(root), nil
}

// manifestFromNode projects a decoded tree onto the typed model.
func manifestFromNode(root *Node) *Manifest {
	m := &Manifest{Root: root}

	if a, ok := root.Plain("package"); ok {
		m.Package = a.Value
	}
	if v, ok := androidInt(root, "versionCode"); ok {
		m.VersionCode = v
	}
	m.VersionName = androidString(root, "versionName")
	m.TestOnly = optAndroidBool(root, "testOnly")

	if sdk := root.Child("uses-sdk"); sdk != nil {
		if v, ok := androidInt(sdk, "minSdkVersion"); ok {
			m.MinSdk = int(v)
		}
		if v, ok := androidInt(sdk, "targetSdkVersion"); ok {
			m.TargetSdk = int(v)
		}
	}

	for _, p := range root.ChildrenNamed("uses-permission") {
		perm := ManifestPermission{Name: androidString(p, "name")}
		if v, ok := androidInt(p, "maxSdkVersion"); ok {
			perm.MaxSdk = int(v)
		}
		if perm.Name != "" {
			m.Permissions = append(m.Permissions, perm)
		}
	}
	// <uses-permission-sdk-23> grants the same capability on newer platforms.
	for _, p := range root.ChildrenNamed("uses-permission-sdk-23") {
		if name := androidString(p, "name"); name != "" {
			m.Permissions = append(m.Permissions, ManifestPermission{Name: name})
		}
	}

	for _, f := range root.ChildrenNamed("uses-feature") {
		name := androidString(f, "name")
		if name == "" {
			continue
		}
		m.Features = append(m.Features, ManifestFeature{
			Name:     name,
			Required: optAndroidBool(f, "required"),
		})
	}

	if app := root.Child("application"); app != nil {
		m.Application = applicationFromNode(app)
	}
	return m
}

// applicationFromNode projects the <application> subtree onto the typed model.
func applicationFromNode(app *Node) Application {
	out := Application{
		Present:                      true,
		Name:                         androidString(app, "name"),
		Debuggable:                   optAndroidBool(app, "debuggable"),
		AllowBackup:                  optAndroidBool(app, "allowBackup"),
		UsesCleartextTraffic:         optAndroidBool(app, "usesCleartextTraffic"),
		NetworkSecurityConfig:        androidString(app, "networkSecurityConfig"),
		RequestLegacyExternalStorage: optAndroidBool(app, "requestLegacyExternalStorage"),
		ExtractNativeLibs:            optAndroidBool(app, "extractNativeLibs"),
	}

	for _, md := range app.ChildrenNamed("meta-data") {
		name := androidString(md, "name")
		if name == "" {
			continue
		}
		out.MetaData = append(out.MetaData, MetaDataEntry{Name: name, Value: androidString(md, "value")})
	}

	for _, kind := range componentKinds {
		for _, c := range app.ChildrenNamed(kind) {
			out.Components = append(out.Components, componentFromNode(kind, c))
		}
	}
	return out
}

// componentFromNode projects a single component element onto the typed model.
func componentFromNode(kind string, c *Node) Component {
	comp := Component{
		Kind:                kind,
		Name:                androidString(c, "name"),
		Exported:            optAndroidBool(c, "exported"),
		Permission:          androidString(c, "permission"),
		GrantURIPermissions: optAndroidBool(c, "grantUriPermissions"),
	}

	filters := c.ChildrenNamed("intent-filter")
	comp.HasIntentFilter = len(filters) > 0
	for _, f := range filters {
		hasMain, hasLauncher := false, false
		for _, a := range f.ChildrenNamed("action") {
			if androidString(a, "name") == "android.intent.action.MAIN" {
				hasMain = true
			}
		}
		for _, cat := range f.ChildrenNamed("category") {
			if androidString(cat, "name") == "android.intent.category.LAUNCHER" {
				hasLauncher = true
			}
		}
		if hasMain && hasLauncher {
			comp.IsLauncher = true
		}
	}

	if raw := androidString(c, "foregroundServiceType"); raw != "" {
		for _, t := range strings.Split(raw, "|") {
			if t = strings.TrimSpace(t); t != "" {
				comp.ForegroundServiceTypes = append(comp.ForegroundServiceTypes, t)
			}
		}
	}
	return comp
}

// optAndroidBool returns a pointer to the boolean value of an android:
// attribute, or nil when absent or not statically determinable.
func optAndroidBool(n *Node, name string) *bool {
	v, ok := androidBool(n, name)
	if !ok {
		return nil
	}
	return &v
}

// isTrue reports whether an optional boolean is present and true.
func isTrue(b *bool) bool { return b != nil && *b }

// isFalse reports whether an optional boolean is present and false.
func isFalse(b *bool) bool { return b != nil && !*b }

// HasPermission reports whether the manifest declares the named permission.
func (m *Manifest) HasPermission(name string) bool {
	for _, p := range m.Permissions {
		if p.Name == name {
			return true
		}
	}
	return false
}

// MetaDataValue returns the value of an <application> meta-data entry.
func (m *Manifest) MetaDataValue(name string) (string, bool) {
	for _, md := range m.Application.MetaData {
		if md.Name == name {
			return md.Value, true
		}
	}
	return "", false
}
