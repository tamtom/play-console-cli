package preflight

import (
	"errors"
	"fmt"
	"strconv"

	"google.golang.org/protobuf/encoding/protowire"
)

// aapt2 protobuf XML decoder for AAB manifests.
//
// Unlike APKs, an App Bundle stores base/manifest/AndroidManifest.xml as a
// serialized aapt2 `XmlNode` protobuf. We decode the wire format directly
// rather than generating Go types from Resources.proto, which keeps the
// dependency surface to protowire and avoids vendoring the schema.
//
// Field numbers below come from
// frameworks/base/tools/aapt2/Resources.proto.

// Field numbers for XmlNode.
const (
	fieldXMLNodeElement = 1
	fieldXMLNodeText    = 2
)

// Field numbers for XmlElement.
const (
	fieldXMLElementNamespaceURI = 2
	fieldXMLElementName         = 3
	fieldXMLElementAttribute    = 4
	fieldXMLElementChild        = 5
)

// Field numbers for XmlAttribute.
const (
	fieldXMLAttrNamespaceURI = 1
	fieldXMLAttrName         = 2
	fieldXMLAttrValue        = 3
	fieldXMLAttrCompiledItem = 6
)

// Field numbers for Item.
const (
	fieldItemRef       = 1
	fieldItemStr       = 2
	fieldItemRawStr    = 3
	fieldItemStyledStr = 4
	fieldItemPrim      = 7
)

// Field numbers for Primitive's oneof.
const (
	fieldPrimIntDecimal     = 6
	fieldPrimIntHexadecimal = 7
	fieldPrimBoolean        = 8
)

// fieldStringValue is the `value` field on String/RawString/StyledString.
const fieldStringValue = 1

// maxProtoDepth bounds recursion so a malformed or hostile bundle cannot
// exhaust the stack.
const maxProtoDepth = 100

// errNotProtoXML signals the payload is not a decodable aapt2 XmlNode.
var errNotProtoXML = errors.New("not aapt2 protobuf XML")

// parseProtoXML decodes a serialized aapt2 XmlNode into a Node tree.
func parseProtoXML(data []byte) (*Node, error) {
	if len(data) == 0 {
		return nil, errNotProtoXML
	}
	node, err := decodeXMLNode(data, 0)
	if err != nil {
		return nil, err
	}
	if node == nil {
		return nil, errNotProtoXML
	}
	return node, nil
}

// decodeXMLNode decodes an XmlNode message, returning nil for text nodes.
func decodeXMLNode(b []byte, depth int) (*Node, error) {
	if depth > maxProtoDepth {
		return nil, errors.New("protoxml: nesting too deep")
	}
	var out *Node
	err := eachField(b, func(num protowire.Number, typ protowire.Type, bytesVal []byte, _ uint64) error {
		if num == fieldXMLNodeElement && typ == protowire.BytesType {
			el, err := decodeXMLElement(bytesVal, depth+1)
			if err != nil {
				return err
			}
			out = el
		}
		// fieldXMLNodeText is ignored: manifests carry no meaningful text nodes.
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// decodeXMLElement decodes an XmlElement message into a Node.
func decodeXMLElement(b []byte, depth int) (*Node, error) {
	if depth > maxProtoDepth {
		return nil, errors.New("protoxml: nesting too deep")
	}
	node := &Node{}
	err := eachField(b, func(num protowire.Number, typ protowire.Type, bytesVal []byte, _ uint64) error {
		if typ != protowire.BytesType {
			return nil
		}
		switch num {
		case fieldXMLElementName:
			node.Name = string(bytesVal)
		case fieldXMLElementNamespaceURI:
			// Element namespace is unused; attributes carry their own URI.
		case fieldXMLElementAttribute:
			a, err := decodeXMLAttribute(bytesVal, depth+1)
			if err != nil {
				return err
			}
			node.Attrs = append(node.Attrs, a)
		case fieldXMLElementChild:
			child, err := decodeXMLNode(bytesVal, depth+1)
			if err != nil {
				return err
			}
			if child != nil {
				node.Children = append(node.Children, child)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return node, nil
}

// decodeXMLAttribute decodes an XmlAttribute, preferring the compiled (typed)
// value over the raw string when both are present.
func decodeXMLAttribute(b []byte, depth int) (Attr, error) {
	var a Attr
	err := eachField(b, func(num protowire.Number, typ protowire.Type, bytesVal []byte, _ uint64) error {
		if typ != protowire.BytesType {
			return nil
		}
		switch num {
		case fieldXMLAttrNamespaceURI:
			a.Namespace = string(bytesVal)
		case fieldXMLAttrName:
			a.Name = string(bytesVal)
		case fieldXMLAttrValue:
			a.Value = string(bytesVal)
		case fieldXMLAttrCompiledItem:
			if err := decodeItem(bytesVal, &a, depth+1); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Attr{}, err
	}
	if a.Type == AttrUnknown && a.Value != "" {
		a.Type = AttrString
	}
	return a, nil
}

// decodeItem fills the typed portion of an attribute from an Item message.
func decodeItem(b []byte, a *Attr, depth int) error {
	if depth > maxProtoDepth {
		return errors.New("protoxml: nesting too deep")
	}
	return eachField(b, func(num protowire.Number, typ protowire.Type, bytesVal []byte, _ uint64) error {
		if typ != protowire.BytesType {
			return nil
		}
		switch num {
		case fieldItemPrim:
			return decodePrimitive(bytesVal, a)
		case fieldItemStr, fieldItemRawStr, fieldItemStyledStr:
			s, err := decodeStringWrapper(bytesVal)
			if err != nil {
				return err
			}
			a.Type = AttrString
			if a.Value == "" {
				a.Value = s
			}
		case fieldItemRef:
			// A resource reference is resolved from the resource table at
			// runtime, so its value is not statically determinable here.
			a.Type = AttrReference
		}
		return nil
	})
}

// decodePrimitive reads the Primitive oneof into the attribute.
func decodePrimitive(b []byte, a *Attr) error {
	return eachField(b, func(num protowire.Number, typ protowire.Type, _ []byte, v uint64) error {
		if typ != protowire.VarintType {
			return nil
		}
		switch num {
		case fieldPrimBoolean:
			a.Type = AttrBool
			a.Bool = v != 0
			a.Int = int64(v) // #nosec G115 -- bool encodes as 0 or 1
			if a.Value == "" {
				a.Value = strconv.FormatBool(a.Bool)
			}
		case fieldPrimIntDecimal:
			a.Type = AttrInt
			a.Int = int64(int32(v)) // #nosec G115 -- proto int32 field
			if a.Value == "" {
				a.Value = strconv.FormatInt(a.Int, 10)
			}
		case fieldPrimIntHexadecimal:
			a.Type = AttrInt
			a.Int = int64(uint32(v)) // #nosec G115 -- proto uint32 field
			if a.Value == "" {
				a.Value = fmt.Sprintf("0x%x", uint32(v)) // #nosec G115
			}
		}
		return nil
	})
}

// decodeStringWrapper reads the `value` field of String/RawString/StyledString.
func decodeStringWrapper(b []byte) (string, error) {
	var s string
	err := eachField(b, func(num protowire.Number, typ protowire.Type, bytesVal []byte, _ uint64) error {
		if num == fieldStringValue && typ == protowire.BytesType {
			s = string(bytesVal)
		}
		return nil
	})
	return s, err
}

// eachField walks a protobuf message, invoking fn per field. Length-delimited
// fields arrive in bytesVal; varints arrive in varintVal.
func eachField(b []byte, fn func(num protowire.Number, typ protowire.Type, bytesVal []byte, varintVal uint64) error) error {
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return fmt.Errorf("protoxml: bad tag: %w", protowire.ParseError(n))
		}
		b = b[n:]

		switch typ {
		case protowire.BytesType:
			v, n := protowire.ConsumeBytes(b)
			if n < 0 {
				return fmt.Errorf("protoxml: bad length-delimited field %d: %w", num, protowire.ParseError(n))
			}
			if err := fn(num, typ, v, 0); err != nil {
				return err
			}
			b = b[n:]
		case protowire.VarintType:
			v, n := protowire.ConsumeVarint(b)
			if n < 0 {
				return fmt.Errorf("protoxml: bad varint field %d: %w", num, protowire.ParseError(n))
			}
			if err := fn(num, typ, nil, v); err != nil {
				return err
			}
			b = b[n:]
		default:
			n := protowire.ConsumeFieldValue(num, typ, b)
			if n < 0 {
				return fmt.Errorf("protoxml: bad field %d: %w", num, protowire.ParseError(n))
			}
			b = b[n:]
		}
	}
	return nil
}
