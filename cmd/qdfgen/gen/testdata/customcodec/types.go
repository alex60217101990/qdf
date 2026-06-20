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
