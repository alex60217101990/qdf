package qdf

import "encoding/binary"

// Patch container wire format. A patch is NOT a normal qdf blob: it carries
// its own magic ('QDP') so a patch can never be mistaken for a full value and
// vice versa. Header layout:
//
//	'Q' 'D' 'P'  patchVersion1            4 bytes magic+version
//	flags        (1 byte)                 patch flag bits
//	schemaFP     (8 bytes, little-endian) hash of typeDesc(T)
//	baseFP       (8 bytes, little-endian) present iff flags&flagPatchBaseFP
//	body...                               root patch (rANS-framed iff flagPatchRANS)
const (
	patchMagic0   = 'Q'
	patchMagic1   = 'D'
	patchMagic2   = 'P'
	patchVersion1 = 0x01
)

// Patch flag bits (header byte 4).
const (
	flagPatchDense  byte = 1 << 0 // field names interned in the patch body
	flagPatchRANS   byte = 1 << 1 // body after the header is rANS-compressed
	flagPatchBaseFP byte = 1 << 2 // baseFP field present (8 bytes)
)

// Patch body tags. Disjoint numbering from the value tag space (wire.go) — a
// patch body is parsed by the delta reader, never by the value tag dispatcher.
const (
	tagStructPatch     = 0x01 // varuint(nChanged), nChanged×(varuint(fieldIdx), op)
	tagSlicePatch      = 0x02 // varuint(newLen), varuint(nEntries), nEntries×(varuint(idx), op)
	tagMapPatch        = 0x03 // varuint(nUpdate), nUpdate×(key-value, op), varuint(nDelete), nDelete×(key-value)
	tagKeyedSlicePatch = 0x04 // flags byte, [if orderChanged: varuint(newLen)+newLen×key], varuint(nOps), nOps×(key, op)
	tagColSlicePatch   = 0x05 // varuint(n), varuint(nChangedCols), per col: varuint(colIdx), mode byte, body
)

// Keyed-slice-patch flag bits (the flags byte after tagKeyedSlicePatch).
const (
	flagKeyedOrderChanged = 1 << 0 // keys were added/removed/reordered; the new key order follows
)

// Op bytes. Every changed location is one op byte + payload.
const (
	opUnchanged = 0x00 // never written; sparse encoding omits unchanged locations
	opReplace   = 0x01 // payload is the whole new value (normal td.encode codec)
	opMerge     = 0x02 // payload is a recursive patch body (tagStructPatch/Slice/Map)
)

type patchHeader struct {
	schemaFP uint64
	baseFP   uint64
	flags    byte
}

// writePatchHeader appends the header to dst. baseFP is written only when
// flags&flagPatchBaseFP is set.
func writePatchHeader(dst []byte, flags byte, schemaFP, baseFP uint64) []byte {
	dst = append(dst, patchMagic0, patchMagic1, patchMagic2, patchVersion1, flags)
	dst = binary.LittleEndian.AppendUint64(dst, schemaFP)
	if flags&flagPatchBaseFP != 0 {
		dst = binary.LittleEndian.AppendUint64(dst, baseFP)
	}
	return dst
}

// readPatchHeader parses the header and returns it plus the number of header
// bytes consumed. The body starts at buf[n:].
func readPatchHeader(buf []byte) (patchHeader, int, error) {
	var h patchHeader
	if len(buf) < 13 || buf[0] != patchMagic0 || buf[1] != patchMagic1 ||
		buf[2] != patchMagic2 || buf[3] != patchVersion1 {
		return h, 0, ErrInvalidPatch
	}
	h.flags = buf[4]
	h.schemaFP = binary.LittleEndian.Uint64(buf[5:13])
	n := 13
	if h.flags&flagPatchBaseFP != 0 {
		if len(buf) < 21 {
			return h, 0, ErrInvalidPatch
		}
		h.baseFP = binary.LittleEndian.Uint64(buf[13:21])
		n = 21
	}
	return h, n, nil
}
