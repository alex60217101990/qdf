// Package customcodec is a generator fixture: a []struct field whose element
// has a hand-written codec must NOT be columnar-transposed, while a plain
// all-scalar element must be.
package customcodec

// CustomElem has a both-direction hand-written codec whose wire form differs
// from the structural field layout.
type CustomElem struct {
	A int64
	B int64
}

func (c CustomElem) MarshalQDF(dst []byte) ([]byte, error) {
	return append(dst, byte(c.A), byte(c.B)), nil
}

func (c *CustomElem) UnmarshalQDF(src []byte) (int, error) {
	if len(src) < 2 {
		return 0, nil
	}
	c.A, c.B = int64(src[0]), int64(src[1])
	return 2, nil
}

// CustomHolder slices the custom-codec element.
type CustomHolder struct {
	Items []CustomElem
}

// PlainElem is an all-scalar struct with NO codec → columnar-eligible.
type PlainElem struct {
	A int64
	B int64
}

// PlainHolder slices the plain element.
type PlainHolder struct {
	Items []PlainElem
}

// NamedByte is a defined byte element type. A slice of it ([]NamedByte) must NOT
// be columnar-transposed as a Bytes column — that emit assumes a literal []byte
// and would generate non-compiling code (unsafe.SliceData → *NamedByte, and an
// illegal []NamedByte([]byte(...)) conversion). It must fall through to the
// generic row-major per-element encoder instead.
type NamedByte byte

// NamedByteElem mixes a scalar column with a defined-byte-element slice.
type NamedByteElem struct {
	ID   int64
	Data []NamedByte
}

// NamedByteHolder slices the element with the defined-byte-element field.
type NamedByteHolder struct {
	Rows []NamedByteElem
}

// NamedByteArray carries the ARRAY form of the same hazard. [16]NamedByte is not
// [16]byte: the flat-blob path would emit WriteBytes(v.K[:]) — a []NamedByte
// where []byte is wanted — and copy(v.K[:], b) between different element types.
// Both are compile errors, so this type is the array-side twin of NamedByteElem.
type NamedByteArray struct {
	ID int64
	K  [16]NamedByte
}

// Tag is a defined STRING type with a hand-written codec (not a struct). A field
// of this type must route through its MarshalQDF/UnmarshalQDF, NOT be emitted
// structurally as a bare string (which bypasses the codec).
type Tag string

func (t Tag) MarshalQDF(dst []byte) ([]byte, error) {
	return append(dst, "T:"+string(t)...), nil
}

func (t *Tag) UnmarshalQDF(src []byte) (int, error) {
	*t = Tag(string(src)[2:]) // strip "T:"
	return len(src), nil
}

// NamedCodecHolder uses the named-non-struct codec type as a field.
type NamedCodecHolder struct {
	Label Tag `qdf:"label"`
}

// PtrCodecHolder holds a POINTER to the named-non-struct codec type. The encode
// side must route *Tag through its MarshalQDF (EncodeNested), exactly as the
// value-field path and the decode side (DecodeNested) do; otherwise encode writes
// a bare string while decode reads it back through UnmarshalQDF — a corruption.
type PtrCodecHolder struct {
	Label *Tag `qdf:"label"`
}
