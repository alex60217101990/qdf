package qdf_test

import (
	"fmt"

	"github.com/alex60217101990/qdf"
)

// Color is a 24-bit RGB value stored as a single uint32 on the
// wire. It implements Marshaler / Unmarshaler so qdf encodes it
// as a primitive integer instead of recursing into reflect.
type Color struct{ R, G, B uint8 }

// MarshalQDF packs the three bytes into a uint32 and writes it
// using the encoder primitives from a temporary qdf.Encoder
// over the caller's buffer. This shape — temporary encoder over
// the caller's dst slice — is what qdfgen produces by default.
func (c Color) MarshalQDF(dst []byte) ([]byte, error) {
	enc := qdf.NewEncoderOnBuf(dst, qdf.Fast)
	enc.MarkHeaderWritten() // dst already contains the parent header
	enc.WriteUint(uint64(c.R)<<16 | uint64(c.G)<<8 | uint64(c.B))
	return enc.Bytes(), nil
}

// UnmarshalQDF reverses the pack and returns the number of bytes
// consumed from src.
func (c *Color) UnmarshalQDF(src []byte) (int, error) {
	dec := qdf.NewDecoderOnBuf(src)
	dec.MarkHeaderRead()
	v, err := dec.ReadUint()
	if err != nil {
		return 0, err
	}
	c.R = uint8(v >> 16)
	c.G = uint8(v >> 8)
	c.B = uint8(v)
	return dec.Pos(), nil
}

// ExampleMarshaler shows a custom MarshalQDF / UnmarshalQDF pair.
// Use the interface when you want a compact wire form for a
// type — e.g. packing a small struct into a single primitive,
// versioning a payload, or pre-validating bytes before they hit
// the wire.
func ExampleMarshaler() {
	in := Color{R: 0x12, G: 0x34, B: 0x56}
	buf, _ := qdf.MarshalDirect(&in)

	var out Color
	_ = qdf.UnmarshalDirect(buf, &out)
	fmt.Printf("R=%02x G=%02x B=%02x\n", out.R, out.G, out.B)
	// Output: R=12 G=34 B=56
}
