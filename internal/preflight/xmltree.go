package preflight

import (
	"strconv"
	"strings"
)

// AndroidNS is the XML namespace URI for framework attributes.
const AndroidNS = "http://schemas.android.com/apk/res/android"

// AttrType describes how an attribute value was encoded in the source manifest.
//
// Binary manifests carry typed values, so `android:debuggable` is a real
// boolean rather than the string "true". Keeping the type lets scanners
// distinguish "absent", "false", and "not statically determinable" (e.g. a
// value that is a resource reference resolved at build time).
type AttrType int

const (
	// AttrUnknown means the value could not be typed (raw string only).
	AttrUnknown AttrType = iota
	// AttrString is a literal string value.
	AttrString
	// AttrBool is a typed boolean.
	AttrBool
	// AttrInt is a typed integer (decimal or hex).
	AttrInt
	// AttrReference is a reference to a resource (@id/foo); the concrete value
	// lives in the resource table and is not resolvable from the manifest.
	AttrReference
)

// Attr is a single decoded manifest attribute.
type Attr struct {
	Namespace string
	Name      string
	// Value is the raw string form when available. For typed values with no
	// raw string, it holds a rendered form (e.g. "true", "34").
	Value string
	Int   int64
	Bool  bool
	Type  AttrType
}

// Node is a decoded XML element. Both the binary AXML decoder (APK) and the
// aapt2 protobuf decoder (AAB) produce this shape so scanners stay
// format-agnostic.
type Node struct {
	Name     string
	Attrs    []Attr
	Children []*Node
}

// Attribute returns the attribute matching namespace and name.
func (n *Node) Attribute(ns, name string) (Attr, bool) {
	if n == nil {
		return Attr{}, false
	}
	for _, a := range n.Attrs {
		if a.Name == name && a.Namespace == ns {
			return a, true
		}
	}
	return Attr{}, false
}

// Android returns the `android:`-namespaced attribute with the given name.
func (n *Node) Android(name string) (Attr, bool) { return n.Attribute(AndroidNS, name) }

// Plain returns an attribute with no namespace (e.g. `package` on <manifest>).
func (n *Node) Plain(name string) (Attr, bool) { return n.Attribute("", name) }

// Child returns the first direct child with the given element name.
func (n *Node) Child(name string) *Node {
	for _, c := range n.ChildrenNamed(name) {
		return c
	}
	return nil
}

// ChildrenNamed returns all direct children with the given element name.
func (n *Node) ChildrenNamed(name string) []*Node {
	if n == nil {
		return nil
	}
	var out []*Node
	for _, c := range n.Children {
		if c.Name == name {
			out = append(out, c)
		}
	}
	return out
}

// Walk invokes fn for this node and every descendant, depth first.
func (n *Node) Walk(fn func(*Node)) {
	if n == nil {
		return
	}
	fn(n)
	for _, c := range n.Children {
		c.Walk(fn)
	}
}

// BoolValue reports the attribute's boolean value. The second return is false
// when the value is not statically determinable (missing, a resource
// reference, or an unparseable string).
func (a Attr) BoolValue() (bool, bool) {
	switch a.Type {
	case AttrBool:
		return a.Bool, true
	case AttrInt:
		return a.Int != 0, true
	case AttrReference:
		return false, false
	}
	switch strings.ToLower(strings.TrimSpace(a.Value)) {
	case "true", "1":
		return true, true
	case "false", "0":
		return false, true
	}
	return false, false
}

// IntValue reports the attribute's integer value. The second return is false
// when the value is not statically determinable.
func (a Attr) IntValue() (int64, bool) {
	if a.Type == AttrInt {
		return a.Int, true
	}
	if a.Type == AttrReference {
		return 0, false
	}
	v := strings.TrimSpace(a.Value)
	if v == "" {
		return 0, false
	}
	if n, err := strconv.ParseInt(v, 10, 64); err == nil {
		return n, true
	}
	return 0, false
}

// StringValue returns the string form of the attribute.
func (a Attr) StringValue() string { return a.Value }

// androidBool is a helper for the common "read an android: boolean" case.
func androidBool(n *Node, name string) (val bool, known bool) {
	a, ok := n.Android(name)
	if !ok {
		return false, false
	}
	return a.BoolValue()
}

// androidInt is a helper for the common "read an android: integer" case.
func androidInt(n *Node, name string) (val int64, known bool) {
	a, ok := n.Android(name)
	if !ok {
		return 0, false
	}
	return a.IntValue()
}

// androidString returns the string form of an android: attribute, or "".
func androidString(n *Node, name string) string {
	a, ok := n.Android(name)
	if !ok {
		return ""
	}
	return a.Value
}
