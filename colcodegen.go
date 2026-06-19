package qdf

import (
	"math"
	"unsafe"
)

// This file exposes the no-reflect columnar primitives that qdfgen-generated
// code calls to transpose a []struct field into the tagColStruct wire frame.
// The byte layout is identical to encodeColumnar (columnar.go) so the reflect
// path and generated code interoperate. Generated code never uses reflection.

// WriteColStructHeader writes the columnar container header: tagColStruct,
// the row count n, and the column shape (field names + kind bytes). The shape
// is declared inline the first time this encoder sees it and reused by id
// afterwards, sharing encState's colStruct shape-id space with the reflect
// path. kinds[i] is the colKind byte for column i (see classifyColKind:
// int*->0, uint*->1, float64->2, bool->3, float32->6); it must match what the
// reflect encoder would emit for the same field so cross-decoding works.
func (e *Encoder) WriteColStructHeader(n int, names []string, kinds []byte) {
	if e.state == nil {
		e.state = newEncState()
	}
	st := e.state
	e.buf = append(e.buf, tagColStruct)
	e.buf = appendUvarint(e.buf, uint64(n))
	// colKind is a uint8; reinterpret the kinds byte slice without copying.
	var ck []colKind
	if len(kinds) > 0 {
		ck = unsafe.Slice((*colKind)(unsafe.Pointer(&kinds[0])), len(kinds))
	}
	if id := st.colShapeFor(names, ck); id != 0 {
		e.buf = appendUvarint(e.buf, uint64(id))
		return
	}
	e.buf = appendUvarint(e.buf, 0)
	e.buf = appendUvarint(e.buf, uint64(len(names)))
	for i := range names {
		e.WriteString(names[i])
		e.buf = append(e.buf, kinds[i])
	}
	st.colShapeDeclare(names, ck)
}

// WriteIntColumn encodes a column gathered from a signed-integer field. The
// adaptive picker chooses raw/FOR/Delta/RLE/Dict/PFOR per value range.
func (e *Encoder) WriteIntColumn(s []int64) error { return encodeSliceInt64(e, unsafe.Pointer(&s)) }

// WriteUintColumn encodes an unsigned-integer column.
func (e *Encoder) WriteUintColumn(s []uint64) error { return encodeSliceUint64(e, unsafe.Pointer(&s)) }

// WriteFloat64Column encodes a float64 column.
func (e *Encoder) WriteFloat64Column(s []float64) error {
	return encodeSliceFloat64(e, unsafe.Pointer(&s))
}

// WriteFloat32Column encodes a float32 column as its 32-bit patterns through
// the unsigned codec, bit-exact and matching the reflect colKindFloat32 path.
func (e *Encoder) WriteFloat32Column(s []float32) error {
	u := make([]uint64, len(s))
	for i, v := range s {
		u[i] = uint64(math.Float32bits(v))
	}
	return encodeSliceUint64(e, unsafe.Pointer(&u))
}

// WriteBoolColumn encodes a bool column.
func (e *Encoder) WriteBoolColumn(s []bool) error { return encodeSliceBool(e, unsafe.Pointer(&s)) }
