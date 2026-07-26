package preflight

import (
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
)

// --- minimal aapt2 XmlNode encoder used to build fixtures ------------------

func pbBytes(num protowire.Number, v []byte) []byte {
	b := protowire.AppendTag(nil, num, protowire.BytesType)
	return protowire.AppendBytes(b, v)
}

func pbString(num protowire.Number, s string) []byte {
	return pbBytes(num, []byte(s))
}

func pbVarint(num protowire.Number, v uint64) []byte {
	b := protowire.AppendTag(nil, num, protowire.VarintType)
	return protowire.AppendVarint(b, v)
}

// pbPrimBool builds Item{prim{boolean_value}}.
func pbPrimBool(v bool) []byte {
	var n uint64
	if v {
		n = 1
	}
	prim := pbVarint(fieldPrimBoolean, n)
	return pbBytes(fieldItemPrim, prim)
}

// pbPrimInt builds Item{prim{int_decimal_value}}.
func pbPrimInt(v int32) []byte {
	prim := pbVarint(fieldPrimIntDecimal, uint64(uint32(v)))
	return pbBytes(fieldItemPrim, prim)
}

// pbPrimRef builds Item{ref{}}, i.e. an unresolved resource reference.
func pbPrimRef() []byte {
	return pbBytes(fieldItemRef, pbVarint(2, 0x7f100001))
}

type pbAttr struct {
	ns       string
	name     string
	value    string
	compiled []byte // optional Item payload
}

func pbAttribute(a pbAttr) []byte {
	var body []byte
	if a.ns != "" {
		body = append(body, pbString(fieldXMLAttrNamespaceURI, a.ns)...)
	}
	body = append(body, pbString(fieldXMLAttrName, a.name)...)
	if a.value != "" {
		body = append(body, pbString(fieldXMLAttrValue, a.value)...)
	}
	if a.compiled != nil {
		body = append(body, pbBytes(fieldXMLAttrCompiledItem, a.compiled)...)
	}
	return body
}

type pbElem struct {
	name     string
	attrs    []pbAttr
	children []pbElem
}

// pbNode renders a pbElem as a serialized XmlNode message.
func pbNode(el pbElem) []byte {
	var body []byte
	body = append(body, pbString(fieldXMLElementName, el.name)...)
	for _, a := range el.attrs {
		body = append(body, pbBytes(fieldXMLElementAttribute, pbAttribute(a))...)
	}
	for _, c := range el.children {
		body = append(body, pbBytes(fieldXMLElementChild, pbNode(c))...)
	}
	return pbBytes(fieldXMLNodeElement, body)
}

// --- tests -----------------------------------------------------------------

func TestParseProtoXMLTypedValues(t *testing.T) {
	root := pbElem{
		name: "manifest",
		attrs: []pbAttr{
			{name: "package", value: "com.example.app"},
			{ns: AndroidNS, name: "versionCode", value: "42", compiled: pbPrimInt(42)},
			{ns: AndroidNS, name: "versionName", value: "1.2.3"},
		},
		children: []pbElem{
			{name: "uses-sdk", attrs: []pbAttr{
				{ns: AndroidNS, name: "minSdkVersion", compiled: pbPrimInt(24)},
				{ns: AndroidNS, name: "targetSdkVersion", compiled: pbPrimInt(34)},
			}},
			{name: "uses-permission", attrs: []pbAttr{
				{ns: AndroidNS, name: "name", value: "android.permission.INTERNET"},
			}},
			{name: "application", attrs: []pbAttr{
				{ns: AndroidNS, name: "debuggable", compiled: pbPrimBool(true)},
				{ns: AndroidNS, name: "label", compiled: pbPrimRef()},
			}, children: []pbElem{
				{name: "activity", attrs: []pbAttr{
					{ns: AndroidNS, name: "name", value: ".MainActivity"},
					{ns: AndroidNS, name: "exported", compiled: pbPrimBool(true)},
				}},
			}},
		},
	}

	node, err := parseProtoXML(pbNode(root))
	if err != nil {
		t.Fatal(err)
	}
	if node.Name != "manifest" {
		t.Fatalf("root = %q, want manifest", node.Name)
	}
	if a, _ := node.Plain("package"); a.Value != "com.example.app" {
		t.Errorf("package = %q", a.Value)
	}
	if v, ok := androidInt(node, "versionCode"); !ok || v != 42 {
		t.Errorf("versionCode = %d ok=%v", v, ok)
	}

	sdk := node.Child("uses-sdk")
	if v, ok := androidInt(sdk, "minSdkVersion"); !ok || v != 24 {
		t.Errorf("minSdk = %d ok=%v", v, ok)
	}
	if v, ok := androidInt(sdk, "targetSdkVersion"); !ok || v != 34 {
		t.Errorf("targetSdk = %d ok=%v", v, ok)
	}

	app := node.Child("application")
	if v, ok := androidBool(app, "debuggable"); !ok || !v {
		t.Errorf("debuggable = %v ok=%v, want true", v, ok)
	}
	label, _ := app.Android("label")
	if label.Type != AttrReference {
		t.Errorf("label type = %v, want AttrReference", label.Type)
	}
	if _, ok := label.BoolValue(); ok {
		t.Error("resource reference should not resolve to a bool")
	}

	act := app.Child("activity")
	if v, ok := androidBool(act, "exported"); !ok || !v {
		t.Errorf("exported = %v ok=%v", v, ok)
	}
}

func TestParseProtoXMLCompiledItemBeatsEmptyValue(t *testing.T) {
	// aapt2 often omits the raw string and keeps only the compiled value.
	root := pbElem{name: "manifest", children: []pbElem{
		{name: "application", attrs: []pbAttr{
			{ns: AndroidNS, name: "testOnly", compiled: pbPrimBool(true)},
		}},
	}}
	node, err := parseProtoXML(pbNode(root))
	if err != nil {
		t.Fatal(err)
	}
	v, ok := androidBool(node.Child("application"), "testOnly")
	if !ok || !v {
		t.Fatalf("testOnly = %v ok=%v, want true", v, ok)
	}
}

func TestParseProtoXMLEmptyInput(t *testing.T) {
	if _, err := parseProtoXML(nil); err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestParseProtoXMLMalformedInput(t *testing.T) {
	// A truncated length-delimited field must error, not panic.
	bad := []byte{0x0a, 0x7f, 0x01}
	if _, err := parseProtoXML(bad); err == nil {
		t.Fatal("expected error for malformed protobuf")
	}
}

func TestParseProtoXMLTextNodeIsSkipped(t *testing.T) {
	// An XmlNode carrying only text yields no element.
	if _, err := parseProtoXML(pbString(fieldXMLNodeText, "hello")); err == nil {
		t.Fatal("expected error when no element is present")
	}
}
