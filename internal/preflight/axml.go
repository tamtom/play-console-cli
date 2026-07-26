package preflight

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"unicode/utf16"
)

// Binary XML (AXML) decoder for APK manifests.
//
// APKs store AndroidManifest.xml in the platform's binary resource-chunk
// format rather than as text. This decoder reads the string pool and the
// element chunks so scanners can inspect real, typed attribute values.
//
// Reference: frameworks/base/libs/androidfw/include/androidfw/ResourceTypes.h

const (
	chunkNull            = 0x0000
	chunkStringPool      = 0x0001
	chunkXML             = 0x0003
	chunkXMLStartNS      = 0x0100
	chunkXMLEndNS        = 0x0101
	chunkXMLStartElement = 0x0102
	chunkXMLEndElement   = 0x0103
	chunkXMLCData        = 0x0104
	chunkXMLResourceMap  = 0x0180
)

// Res_value data types we care about.
const (
	typeNull       = 0x00
	typeReference  = 0x01
	typeAttribute  = 0x02
	typeString     = 0x03
	typeFloat      = 0x04
	typeDimension  = 0x05
	typeFraction   = 0x06
	typeIntDec     = 0x10
	typeIntHex     = 0x11
	typeIntBoolean = 0x12
)

const stringPoolUTF8Flag = 1 << 8

// noRef is the sentinel used for "no string" in AXML string references.
const noRef = 0xFFFFFFFF

// errNotAXML signals the payload is not binary XML, letting callers fall back
// to another decoder.
var errNotAXML = errors.New("not binary AXML")

// isAXML reports whether data looks like a binary XML chunk.
func isAXML(data []byte) bool {
	if len(data) < 8 {
		return false
	}
	return binary.LittleEndian.Uint16(data[0:2]) == chunkXML
}

// parseAXML decodes a binary AndroidManifest.xml into a Node tree.
func parseAXML(data []byte) (*Node, error) {
	if !isAXML(data) {
		return nil, errNotAXML
	}
	headerSize := binary.LittleEndian.Uint16(data[2:4])
	totalSize := binary.LittleEndian.Uint32(data[4:8])
	if int(headerSize) < 8 || int(headerSize) > len(data) {
		return nil, fmt.Errorf("axml: bad header size %d", headerSize)
	}
	// Trust the smaller of declared size and actual length; some tools pad.
	end := len(data)
	if int(totalSize) <= len(data) && totalSize > 0 {
		end = int(totalSize)
	}

	var pool []string
	var root *Node
	stack := []*Node{}

	off := int(headerSize)
	for off+8 <= end {
		cType := binary.LittleEndian.Uint16(data[off : off+2])
		cHeader := binary.LittleEndian.Uint16(data[off+2 : off+4])
		cSize := binary.LittleEndian.Uint32(data[off+4 : off+8])
		if cSize < 8 || int(cSize) > end-off {
			return nil, fmt.Errorf("axml: bad chunk size %d at offset %d", cSize, off)
		}
		chunk := data[off : off+int(cSize)]

		switch cType {
		case chunkStringPool:
			p, err := parseStringPool(chunk)
			if err != nil {
				return nil, err
			}
			pool = p
		case chunkXMLStartElement:
			node, err := parseStartElement(chunk, int(cHeader), pool)
			if err != nil {
				return nil, err
			}
			if len(stack) == 0 {
				if root == nil {
					root = node
				}
			} else {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, node)
			}
			stack = append(stack, node)
		case chunkXMLEndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case chunkXMLStartNS, chunkXMLEndNS, chunkXMLCData, chunkXMLResourceMap, chunkNull:
			// Not needed: attribute namespaces are stored as full URIs.
		}
		off += int(cSize)
	}

	if root == nil {
		return nil, errors.New("axml: no root element")
	}
	return root, nil
}

// parseStringPool decodes a RES_STRING_POOL chunk into its string values.
func parseStringPool(chunk []byte) ([]string, error) {
	if len(chunk) < 28 {
		return nil, errors.New("axml: string pool too small")
	}
	count := binary.LittleEndian.Uint32(chunk[8:12])
	flags := binary.LittleEndian.Uint32(chunk[16:20])
	stringsStart := binary.LittleEndian.Uint32(chunk[20:24])
	utf8 := flags&stringPoolUTF8Flag != 0

	// Guard against absurd counts from malformed input before allocating.
	if int(count) > len(chunk)/4 {
		return nil, fmt.Errorf("axml: string count %d exceeds chunk capacity", count)
	}
	offsetsAt := 28
	if offsetsAt+int(count)*4 > len(chunk) {
		return nil, errors.New("axml: truncated string offset table")
	}

	out := make([]string, count)
	for i := 0; i < int(count); i++ {
		rel := binary.LittleEndian.Uint32(chunk[offsetsAt+i*4 : offsetsAt+i*4+4])
		at := int(stringsStart) + int(rel)
		if at < 0 || at >= len(chunk) {
			continue // tolerate individual bad offsets
		}
		if utf8 {
			out[i] = decodeUTF8String(chunk, at)
		} else {
			out[i] = decodeUTF16String(chunk, at)
		}
	}
	return out, nil
}

// decodeLen8 reads a UTF-8 pool length, which uses a high-bit continuation.
func decodeLen8(b []byte, at int) (length, next int) {
	if at >= len(b) {
		return 0, at
	}
	v := int(b[at])
	at++
	if v&0x80 != 0 {
		if at >= len(b) {
			return 0, at
		}
		v = ((v & 0x7F) << 8) | int(b[at])
		at++
	}
	return v, at
}

// decodeLen16 reads a UTF-16 pool length, which uses a high-bit continuation.
func decodeLen16(b []byte, at int) (length, next int) {
	if at+2 > len(b) {
		return 0, at
	}
	v := int(binary.LittleEndian.Uint16(b[at : at+2]))
	at += 2
	if v&0x8000 != 0 {
		if at+2 > len(b) {
			return 0, at
		}
		v = ((v & 0x7FFF) << 16) | int(binary.LittleEndian.Uint16(b[at:at+2]))
		at += 2
	}
	return v, at
}

func decodeUTF8String(b []byte, at int) string {
	// UTF-8 entries carry the character count first, then the byte count.
	_, at = decodeLen8(b, at)
	byteLen, at := decodeLen8(b, at)
	if at+byteLen > len(b) || byteLen < 0 {
		return ""
	}
	return string(b[at : at+byteLen])
}

func decodeUTF16String(b []byte, at int) string {
	unitLen, at := decodeLen16(b, at)
	if unitLen < 0 || at+unitLen*2 > len(b) {
		return ""
	}
	units := make([]uint16, unitLen)
	for i := 0; i < unitLen; i++ {
		units[i] = binary.LittleEndian.Uint16(b[at+i*2 : at+i*2+2])
	}
	return string(utf16.Decode(units))
}

func poolAt(pool []string, idx uint32) string {
	if idx == noRef || int(idx) >= len(pool) {
		return ""
	}
	return pool[idx]
}

// parseStartElement decodes a RES_XML_START_ELEMENT chunk and its attributes.
func parseStartElement(chunk []byte, headerSize int, pool []string) (*Node, error) {
	// Layout after the node header: ns, name, attrStart, attrSize, attrCount, ...
	if headerSize+20 > len(chunk) {
		return nil, errors.New("axml: truncated start element")
	}
	base := headerSize
	nameIdx := binary.LittleEndian.Uint32(chunk[base+4 : base+8])
	attrStart := binary.LittleEndian.Uint16(chunk[base+8 : base+10])
	attrSize := binary.LittleEndian.Uint16(chunk[base+10 : base+12])
	attrCount := binary.LittleEndian.Uint16(chunk[base+12 : base+14])

	node := &Node{Name: poolAt(pool, nameIdx)}
	if attrSize == 0 {
		attrSize = 20
	}

	at := base + int(attrStart)
	for i := 0; i < int(attrCount); i++ {
		if at+int(attrSize) > len(chunk) {
			break // tolerate truncation rather than failing the whole parse
		}
		rec := chunk[at : at+int(attrSize)]
		nsIdx := binary.LittleEndian.Uint32(rec[0:4])
		nIdx := binary.LittleEndian.Uint32(rec[4:8])
		rawIdx := binary.LittleEndian.Uint32(rec[8:12])
		dataType := rec[15]
		data := binary.LittleEndian.Uint32(rec[16:20])

		a := Attr{
			Namespace: poolAt(pool, nsIdx),
			Name:      poolAt(pool, nIdx),
			Value:     poolAt(pool, rawIdx),
		}
		applyResValue(&a, dataType, data, pool)
		node.Attrs = append(node.Attrs, a)
		at += int(attrSize)
	}
	return node, nil
}

// applyResValue fills in the typed portion of an attribute from a Res_value.
func applyResValue(a *Attr, dataType byte, data uint32, pool []string) {
	switch dataType {
	case typeIntBoolean:
		a.Type = AttrBool
		a.Bool = data != 0
		a.Int = int64(int32(data)) // #nosec G115 -- intentional signed reinterpretation
		if a.Value == "" {
			a.Value = strconv.FormatBool(a.Bool)
		}
	case typeIntDec:
		a.Type = AttrInt
		a.Int = int64(int32(data)) // #nosec G115
		if a.Value == "" {
			a.Value = strconv.FormatInt(a.Int, 10)
		}
	case typeIntHex:
		a.Type = AttrInt
		a.Int = int64(int32(data)) // #nosec G115
		if a.Value == "" {
			a.Value = "0x" + strconv.FormatUint(uint64(data), 16)
		}
	case typeString:
		a.Type = AttrString
		if a.Value == "" {
			a.Value = poolAt(pool, data)
		}
	case typeReference, typeAttribute:
		a.Type = AttrReference
		if a.Value == "" {
			a.Value = fmt.Sprintf("@0x%08x", data)
		}
	case typeNull:
		a.Type = AttrUnknown
	case typeFloat, typeDimension, typeFraction:
		a.Type = AttrUnknown
	default:
		if a.Type == AttrUnknown && a.Value != "" {
			a.Type = AttrString
		}
	}
}
