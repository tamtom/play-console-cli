package preflight

import (
	"bytes"
	"encoding/binary"
	"testing"
	"unicode/utf16"
)

// --- minimal AXML encoder used to build fixtures ---------------------------

type testAttr struct {
	ns       string // "" means no namespace
	name     string
	raw      string // raw string value; "" for none
	dataType byte
	data     uint32
}

type testElem struct {
	name     string
	attrs    []testAttr
	children []testElem
}

type axmlEncoder struct {
	strs []string
	idx  map[string]uint32
	utf8 bool
}

func newAXMLEncoder(utf8 bool) *axmlEncoder {
	return &axmlEncoder{idx: map[string]uint32{}, utf8: utf8}
}

func (e *axmlEncoder) intern(s string) uint32 {
	if i, ok := e.idx[s]; ok {
		return i
	}
	i := uint32(len(e.strs))
	e.strs = append(e.strs, s)
	e.idx[s] = i
	return i
}

func (e *axmlEncoder) ref(s string) uint32 {
	if s == "" {
		return noRef
	}
	return e.intern(s)
}

// collect interns every string the tree needs, before offsets are computed.
func (e *axmlEncoder) collect(el testElem) {
	e.intern(el.name)
	for _, a := range el.attrs {
		if a.ns != "" {
			e.intern(a.ns)
		}
		e.intern(a.name)
		if a.raw != "" {
			e.intern(a.raw)
		}
	}
	for _, c := range el.children {
		e.collect(c)
	}
}

func u16(b *bytes.Buffer, v uint16) { _ = binary.Write(b, binary.LittleEndian, v) }
func u32(b *bytes.Buffer, v uint32) { _ = binary.Write(b, binary.LittleEndian, v) }

func (e *axmlEncoder) stringPool() []byte {
	data := &bytes.Buffer{}
	offsets := make([]uint32, len(e.strs))
	for i, s := range e.strs {
		offsets[i] = uint32(data.Len())
		if e.utf8 {
			data.WriteByte(byte(len([]rune(s))))
			data.WriteByte(byte(len(s)))
			data.WriteString(s)
			data.WriteByte(0)
		} else {
			units := utf16.Encode([]rune(s))
			u16(data, uint16(len(units)))
			for _, u := range units {
				u16(data, u)
			}
			u16(data, 0)
		}
	}
	for data.Len()%4 != 0 {
		data.WriteByte(0)
	}

	headerSize := uint32(28)
	stringsStart := headerSize + uint32(len(e.strs))*4
	total := stringsStart + uint32(data.Len())

	out := &bytes.Buffer{}
	u16(out, chunkStringPool)
	u16(out, uint16(headerSize))
	u32(out, total)
	u32(out, uint32(len(e.strs)))
	u32(out, 0) // styleCount
	var flags uint32
	if e.utf8 {
		flags = stringPoolUTF8Flag
	}
	u32(out, flags)
	u32(out, stringsStart)
	u32(out, 0) // stylesStart
	for _, o := range offsets {
		u32(out, o)
	}
	out.Write(data.Bytes())
	return out.Bytes()
}

func (e *axmlEncoder) element(el testElem) []byte {
	out := &bytes.Buffer{}
	size := uint32(36 + 20*len(el.attrs))
	u16(out, chunkXMLStartElement)
	u16(out, 16)
	u32(out, size)
	u32(out, 1)     // line
	u32(out, noRef) // comment
	u32(out, noRef) // element namespace
	u32(out, e.intern(el.name))
	u16(out, 20) // attributeStart
	u16(out, 20) // attributeSize
	u16(out, uint16(len(el.attrs)))
	u16(out, 0) // idIndex
	u16(out, 0) // classIndex
	u16(out, 0) // styleIndex
	for _, a := range el.attrs {
		u32(out, e.ref(a.ns))
		u32(out, e.intern(a.name))
		u32(out, e.ref(a.raw))
		u16(out, 8) // Res_value size
		out.WriteByte(0)
		out.WriteByte(a.dataType)
		u32(out, a.data)
	}

	for _, c := range el.children {
		out.Write(e.element(c))
	}

	end := &bytes.Buffer{}
	u16(end, chunkXMLEndElement)
	u16(end, 16)
	u32(end, 24)
	u32(end, 1)
	u32(end, noRef)
	u32(end, noRef)
	u32(end, e.intern(el.name))
	out.Write(end.Bytes())
	return out.Bytes()
}

// encodeAXML renders a tree as a binary XML document.
func encodeAXML(t *testing.T, root testElem, utf8 bool) []byte {
	t.Helper()
	e := newAXMLEncoder(utf8)
	e.collect(root)
	body := e.element(root)
	pool := e.stringPool()

	out := &bytes.Buffer{}
	u16(out, chunkXML)
	u16(out, 8)
	u32(out, uint32(8+len(pool)+len(body)))
	out.Write(pool)
	out.Write(body)
	return out.Bytes()
}

// --- tests -----------------------------------------------------------------

func TestParseAXMLRejectsNonAXML(t *testing.T) {
	if _, err := parseAXML([]byte("not xml at all")); err == nil {
		t.Fatal("expected error for non-AXML input")
	}
	if _, err := parseAXML(nil); err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestParseAXMLTypedValues(t *testing.T) {
	root := testElem{
		name: "manifest",
		attrs: []testAttr{
			{name: "package", raw: "com.example.app", dataType: typeString},
			{ns: AndroidNS, name: "versionCode", dataType: typeIntDec, data: 42},
			{ns: AndroidNS, name: "testOnly", dataType: typeIntBoolean, data: 1},
		},
		children: []testElem{
			{name: "uses-sdk", attrs: []testAttr{
				{ns: AndroidNS, name: "minSdkVersion", dataType: typeIntDec, data: 24},
				{ns: AndroidNS, name: "targetSdkVersion", dataType: typeIntDec, data: 34},
			}},
			{name: "application", attrs: []testAttr{
				{ns: AndroidNS, name: "debuggable", dataType: typeIntBoolean, data: 0},
				{ns: AndroidNS, name: "label", dataType: typeReference, data: 0x7f100001},
			}},
		},
	}

	for _, utf8 := range []bool{true, false} {
		node, err := parseAXML(encodeAXML(t, root, utf8))
		if err != nil {
			t.Fatalf("utf8=%v: %v", utf8, err)
		}
		if node.Name != "manifest" {
			t.Fatalf("utf8=%v: root = %q, want manifest", utf8, node.Name)
		}
		if a, ok := node.Plain("package"); !ok || a.Value != "com.example.app" {
			t.Errorf("utf8=%v: package = %q ok=%v", utf8, a.Value, ok)
		}
		if v, ok := androidInt(node, "versionCode"); !ok || v != 42 {
			t.Errorf("utf8=%v: versionCode = %d ok=%v", utf8, v, ok)
		}
		if v, ok := androidBool(node, "testOnly"); !ok || !v {
			t.Errorf("utf8=%v: testOnly = %v ok=%v", utf8, v, ok)
		}

		sdk := node.Child("uses-sdk")
		if sdk == nil {
			t.Fatalf("utf8=%v: uses-sdk missing", utf8)
		}
		if v, ok := androidInt(sdk, "targetSdkVersion"); !ok || v != 34 {
			t.Errorf("utf8=%v: targetSdk = %d ok=%v", utf8, v, ok)
		}

		app := node.Child("application")
		if app == nil {
			t.Fatalf("utf8=%v: application missing", utf8)
		}
		if v, ok := androidBool(app, "debuggable"); !ok || v {
			t.Errorf("utf8=%v: debuggable = %v ok=%v, want false/known", utf8, v, ok)
		}
		// A resource reference is not statically determinable.
		ref, _ := app.Android("label")
		if _, ok := ref.BoolValue(); ok {
			t.Errorf("utf8=%v: reference should not resolve to a bool", utf8)
		}
		if ref.Type != AttrReference {
			t.Errorf("utf8=%v: label type = %v, want AttrReference", utf8, ref.Type)
		}
	}
}

func TestParseAXMLNestedChildren(t *testing.T) {
	root := testElem{
		name: "manifest",
		children: []testElem{
			{name: "application", children: []testElem{
				{name: "activity", attrs: []testAttr{
					{ns: AndroidNS, name: "name", raw: ".MainActivity", dataType: typeString},
				}},
				{name: "service", attrs: []testAttr{
					{ns: AndroidNS, name: "name", raw: ".SyncService", dataType: typeString},
				}},
			}},
		},
	}
	node, err := parseAXML(encodeAXML(t, root, true))
	if err != nil {
		t.Fatal(err)
	}
	app := node.Child("application")
	if app == nil || len(app.Children) != 2 {
		t.Fatalf("expected 2 application children, got %+v", app)
	}
	if got := androidString(app.Children[0], "name"); got != ".MainActivity" {
		t.Errorf("activity name = %q", got)
	}
	if got := androidString(app.Children[1], "name"); got != ".SyncService" {
		t.Errorf("service name = %q", got)
	}
}

func TestParseAXMLTruncatedChunkIsRejected(t *testing.T) {
	data := encodeAXML(t, testElem{name: "manifest"}, true)
	// Corrupt the outer chunk size so parsing must bail rather than panic.
	binary.LittleEndian.PutUint32(data[4:8], 0xFFFFFF)
	if _, err := parseAXML(data[:16]); err == nil {
		t.Fatal("expected error on truncated input")
	}
}

func TestParseStringPoolRejectsAbsurdCount(t *testing.T) {
	chunk := make([]byte, 40)
	binary.LittleEndian.PutUint16(chunk[0:2], chunkStringPool)
	binary.LittleEndian.PutUint16(chunk[2:4], 28)
	binary.LittleEndian.PutUint32(chunk[4:8], 40)
	binary.LittleEndian.PutUint32(chunk[8:12], 1<<30) // bogus string count
	if _, err := parseStringPool(chunk); err == nil {
		t.Fatal("expected error for absurd string count")
	}
}
