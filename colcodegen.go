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

// PeekColStruct reports whether the next byte is a columnar container frame,
// without consuming it. Generated decode uses this to pick the columnar path
// vs the row-major fallback (a tiny slice the encoder kept row-major, or a
// reflect-produced row-major encoding of the same field).
func (d *Decoder) PeekColStruct() bool {
	return d.i < len(d.buf) && d.buf[d.i] == tagColStruct
}

// ReadColStructHeader consumes the columnar container header and returns the
// row count and column shape (names + kind bytes). It mirrors readColShape's
// non-index path. n is bounded by maxColumnarElems.
func (d *Decoder) ReadColStructHeader() (int, []string, []byte, error) {
	cs, err := d.readColShape(maxColumnarElems)
	if err != nil {
		return 0, nil, nil, err
	}
	d.colMaxLen = cs.n
	kinds := make([]byte, len(cs.sh.kinds))
	for i, k := range cs.sh.kinds {
		kinds[i] = byte(k)
	}
	return cs.n, cs.sh.names, kinds, nil
}

// ReadIntColumn decodes one signed-integer column of n values.
func (d *Decoder) ReadIntColumn(n int) ([]int64, error) {
	var s []int64
	if err := decodeSliceInt64Into(d, &s); err != nil {
		return nil, err
	}
	if len(s) != n {
		return nil, ErrTypeMismatch
	}
	return s, nil
}

// ReadUintColumn decodes one unsigned-integer column of n values.
func (d *Decoder) ReadUintColumn(n int) ([]uint64, error) {
	var s []uint64
	if err := decodeSliceUint64Into(d, &s); err != nil {
		return nil, err
	}
	if len(s) != n {
		return nil, ErrTypeMismatch
	}
	return s, nil
}

// ReadFloat64Column decodes one float64 column of n values.
func (d *Decoder) ReadFloat64Column(n int) ([]float64, error) {
	var s []float64
	if err := decodeSliceFloat64Into(d, &s); err != nil {
		return nil, err
	}
	if len(s) != n {
		return nil, ErrTypeMismatch
	}
	return s, nil
}

// ReadFloat32Column decodes one float32 column (32-bit patterns via the
// unsigned codec, mirroring WriteFloat32Column).
func (d *Decoder) ReadFloat32Column(n int) ([]float32, error) {
	var u []uint64
	if err := decodeSliceUint64Into(d, &u); err != nil {
		return nil, err
	}
	if len(u) != n {
		return nil, ErrTypeMismatch
	}
	out := make([]float32, n)
	for i, v := range u {
		out[i] = math.Float32frombits(uint32(v))
	}
	return out, nil
}

// ReadBoolColumn decodes one bool column of n values.
func (d *Decoder) ReadBoolColumn(n int) ([]bool, error) {
	var s []bool
	if err := decodeSliceBoolInto(d, &s); err != nil {
		return nil, err
	}
	if len(s) != n {
		return nil, ErrTypeMismatch
	}
	return s, nil
}
