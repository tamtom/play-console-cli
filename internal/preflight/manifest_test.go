package preflight

import "testing"

// richManifestProto builds an AAB-style manifest exercising the whole model.
func richManifestProto() []byte {
	return pbNode(pbElem{
		name: "manifest",
		attrs: []pbAttr{
			{name: "package", value: "com.example.app"},
			{ns: AndroidNS, name: "versionCode", compiled: pbPrimInt(7)},
			{ns: AndroidNS, name: "versionName", value: "2.0.0"},
		},
		children: []pbElem{
			{name: "uses-sdk", attrs: []pbAttr{
				{ns: AndroidNS, name: "minSdkVersion", compiled: pbPrimInt(23)},
				{ns: AndroidNS, name: "targetSdkVersion", compiled: pbPrimInt(35)},
			}},
			{name: "uses-permission", attrs: []pbAttr{
				{ns: AndroidNS, name: "name", value: "android.permission.INTERNET"},
			}},
			{name: "uses-permission", attrs: []pbAttr{
				{ns: AndroidNS, name: "name", value: "android.permission.WRITE_EXTERNAL_STORAGE"},
				{ns: AndroidNS, name: "maxSdkVersion", compiled: pbPrimInt(28)},
			}},
			{name: "uses-permission-sdk-23", attrs: []pbAttr{
				{ns: AndroidNS, name: "name", value: "android.permission.CAMERA"},
			}},
			{name: "uses-feature", attrs: []pbAttr{
				{ns: AndroidNS, name: "name", value: "android.hardware.camera"},
				{ns: AndroidNS, name: "required", compiled: pbPrimBool(false)},
			}},
			{name: "application", attrs: []pbAttr{
				{ns: AndroidNS, name: "name", value: ".App"},
				{ns: AndroidNS, name: "allowBackup", compiled: pbPrimBool(true)},
				{ns: AndroidNS, name: "networkSecurityConfig", value: "@xml/net_config"},
			}, children: []pbElem{
				{name: "meta-data", attrs: []pbAttr{
					{ns: AndroidNS, name: "name", value: "com.google.android.gms.ads.APPLICATION_ID"},
					{ns: AndroidNS, name: "value", value: "ca-app-pub-000~111"},
				}},
				{name: "activity", attrs: []pbAttr{
					{ns: AndroidNS, name: "name", value: ".MainActivity"},
					{ns: AndroidNS, name: "exported", compiled: pbPrimBool(true)},
				}, children: []pbElem{
					{name: "intent-filter", children: []pbElem{
						{name: "action", attrs: []pbAttr{
							{ns: AndroidNS, name: "name", value: "android.intent.action.MAIN"},
						}},
						{name: "category", attrs: []pbAttr{
							{ns: AndroidNS, name: "name", value: "android.intent.category.LAUNCHER"},
						}},
					}},
				}},
				{name: "service", attrs: []pbAttr{
					{ns: AndroidNS, name: "name", value: ".SyncService"},
					{ns: AndroidNS, name: "foregroundServiceType", value: "dataSync|location"},
				}},
				{name: "provider", attrs: []pbAttr{
					{ns: AndroidNS, name: "name", value: ".FileProvider"},
					{ns: AndroidNS, name: "exported", compiled: pbPrimBool(false)},
					{ns: AndroidNS, name: "grantUriPermissions", compiled: pbPrimBool(true)},
				}},
			}},
		},
	})
}

func TestParseManifestBytesProtobuf(t *testing.T) {
	m, err := parseManifestBytes(richManifestProto())
	if err != nil {
		t.Fatal(err)
	}
	if m.Package != "com.example.app" {
		t.Errorf("package = %q", m.Package)
	}
	if m.VersionCode != 7 || m.VersionName != "2.0.0" {
		t.Errorf("version = %d/%q", m.VersionCode, m.VersionName)
	}
	if m.MinSdk != 23 || m.TargetSdk != 35 {
		t.Errorf("sdk = %d/%d", m.MinSdk, m.TargetSdk)
	}
	if len(m.Permissions) != 3 {
		t.Fatalf("permissions = %d, want 3: %+v", len(m.Permissions), m.Permissions)
	}
	if !m.HasPermission("android.permission.CAMERA") {
		t.Error("uses-permission-sdk-23 should contribute a permission")
	}
	if m.Permissions[1].MaxSdk != 28 {
		t.Errorf("maxSdkVersion = %d, want 28", m.Permissions[1].MaxSdk)
	}
	if len(m.Features) != 1 || !isFalse(m.Features[0].Required) {
		t.Errorf("features = %+v", m.Features)
	}
	if !m.Application.Present || m.Application.Name != ".App" {
		t.Errorf("application = %+v", m.Application)
	}
	if !isTrue(m.Application.AllowBackup) {
		t.Error("allowBackup should be true")
	}
	if m.Application.Debuggable != nil {
		t.Error("absent debuggable should stay nil")
	}
	if v, ok := m.MetaDataValue("com.google.android.gms.ads.APPLICATION_ID"); !ok || v != "ca-app-pub-000~111" {
		t.Errorf("ads meta-data = %q ok=%v", v, ok)
	}

	if len(m.Application.Components) != 3 {
		t.Fatalf("components = %d, want 3", len(m.Application.Components))
	}
	act := m.Application.Components[0]
	if act.Kind != "activity" || !act.IsLauncher || !act.HasIntentFilter || !isTrue(act.Exported) {
		t.Errorf("activity = %+v", act)
	}
	svc := m.Application.Components[1]
	if len(svc.ForegroundServiceTypes) != 2 || svc.ForegroundServiceTypes[0] != "dataSync" {
		t.Errorf("fgs types = %+v", svc.ForegroundServiceTypes)
	}
	prov := m.Application.Components[2]
	if !isFalse(prov.Exported) || !isTrue(prov.GrantURIPermissions) {
		t.Errorf("provider = %+v", prov)
	}
}

func TestParseManifestBytesBinaryAXML(t *testing.T) {
	data := encodeAXML(t, testElem{
		name: "manifest",
		attrs: []testAttr{
			{name: "package", raw: "com.example.apk", dataType: typeString},
		},
		children: []testElem{
			{name: "uses-sdk", attrs: []testAttr{
				{ns: AndroidNS, name: "targetSdkVersion", dataType: typeIntDec, data: 33},
			}},
			{name: "application", attrs: []testAttr{
				{ns: AndroidNS, name: "debuggable", dataType: typeIntBoolean, data: 1},
			}},
		},
	}, true)

	m, err := parseManifestBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	if m.Package != "com.example.apk" {
		t.Errorf("package = %q", m.Package)
	}
	if m.TargetSdk != 33 {
		t.Errorf("targetSdk = %d", m.TargetSdk)
	}
	if !isTrue(m.Application.Debuggable) {
		t.Error("debuggable should be true")
	}
}

func TestParseManifestBytesErrors(t *testing.T) {
	if _, err := parseManifestBytes(nil); err == nil {
		t.Error("expected error for empty manifest")
	}
	if _, err := parseManifestBytes([]byte("plain text manifest")); err == nil {
		t.Error("expected error for undecodable manifest")
	}
	// A well-formed tree whose root is not <manifest> must be rejected.
	notManifest := pbNode(pbElem{name: "resources"})
	if _, err := parseManifestBytes(notManifest); err == nil {
		t.Error("expected error for non-manifest root")
	}
}

func TestOptionalBoolHelpers(t *testing.T) {
	tr, fa := true, false
	if !isTrue(&tr) || isTrue(&fa) || isTrue(nil) {
		t.Error("isTrue")
	}
	if !isFalse(&fa) || isFalse(&tr) || isFalse(nil) {
		t.Error("isFalse")
	}
}
