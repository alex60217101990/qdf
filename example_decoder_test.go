package qdf_test

import (
	"fmt"

	"github.com/alex60217101990/qdf"
)

// ExampleDecoder_SetNoCopy returns string / []byte values that
// alias the input buffer instead of allocating per-value copies.
// Use this for read-heavy hot paths where the input buffer
// outlives every value the decoder hands back; do not retain
// the values past the buffer's lifetime.
func ExampleDecoder_SetNoCopy() {
	in := []string{"alpha", "beta", "gamma"}
	buf, _ := qdf.Marshal(in, qdf.OptSpeed)

	dec := qdf.NewDecoderOnBuf(buf)
	dec.SetNoCopy(true)

	n, _ := dec.ReadArrayHeader()
	for i := 0; i < n; i++ {
		s, _ := dec.ReadString() // alias into buf
		fmt.Println(s)
	}
	// Output:
	// alpha
	// beta
	// gamma
}

// ExampleDecoder_PeekTag inspects the next wire tag without
// consuming it. Handy when implementing a dispatch loop over a
// schemaless payload, or when an Unmarshaler needs to branch on
// the upcoming type.
func ExampleDecoder_PeekTag() {
	buf, _ := qdf.Marshal(int64(-42), qdf.OptSpeed)
	dec := qdf.NewDecoderOnBuf(buf)
	tag, _ := dec.PeekTag()
	if tag == qdf.TagNil {
		fmt.Println("nil value")
		return
	}
	v, _ := dec.ReadInt()
	fmt.Printf("int=%d (tag=0x%02x)\n", v, tag)
	// Output: int=-42 (tag=0xc7)
}

// ExampleDecoder_IsNil consumes a nil tag if present, returning
// false (without advancing) for any other tag. Useful when a
// field is optional on the wire and the caller wants to keep
// the read cursor stable on a non-nil branch.
func ExampleDecoder_IsNil() {
	buf, _ := qdf.Marshal([]any{nil, "hello"}, qdf.OptSpeed)
	dec := qdf.NewDecoderOnBuf(buf)

	n, _ := dec.ReadArrayHeader()
	for i := 0; i < n; i++ {
		isNil, _ := dec.IsNil()
		if isNil {
			fmt.Println("<nil>")
			continue
		}
		s, _ := dec.ReadString()
		fmt.Println(s)
	}
	// Output:
	// <nil>
	// hello
}
