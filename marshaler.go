package qdf

// Marshaler is implemented by types that know how to serialize themselves
// into the QDF wire format. Implementations should append exactly one
// value to dst and return the extended slice.
type Marshaler interface {
	MarshalQDF(dst []byte) ([]byte, error)
}

// Unmarshaler is implemented by types that know how to deserialize themselves
// from a QDF wire-format slice. Implementations should consume exactly one
// value from src and return the number of bytes consumed.
//
// A custom UnmarshalQDF reads the Fast wire format. Marshal honours this: a
// type implementing Marshaler always emits its Fast-format body and is framed
// as Fast regardless of the Options passed, so Marshaler+Unmarshaler types
// round-trip under any Options. A type that implements Unmarshaler WITHOUT
// also implementing Marshaler is encoded structurally; under a Dense/QPack
// tier that produces a Dense body its Fast-only UnmarshalQDF cannot read.
// Implement both interfaces (or neither) to avoid this — generated code from
// cmd/qdfgen always implements both.
type Unmarshaler interface {
	UnmarshalQDF(src []byte) (n int, err error)
}

// UnmarshalerOpts is an optional extension of Unmarshaler that accepts the
// noCopy flag. When noCopy is true, the implementation should decode string and
// []byte fields as aliases of src (see WithNoCopy) instead of copying. Generated
// code from cmd/qdfgen implements this; the plain UnmarshalQDF delegates to it
// with noCopy=false. Decoders honor it only when the caller opted into noCopy.
type UnmarshalerOpts interface {
	Unmarshaler
	UnmarshalQDFOpts(src []byte, noCopy bool) (n int, err error)
}

// UnmarshalNested decodes one nested Unmarshaler value from src, honoring noCopy
// when u also implements UnmarshalerOpts. External Unmarshalers without the Opts
// method fall back to a copying decode. Used by decodeUnmarshaler and by
// cmd/qdfgen-generated code; exported for the latter.
func UnmarshalNested(u Unmarshaler, src []byte, noCopy bool) (int, error) {
	if noCopy {
		if uo, ok := u.(UnmarshalerOpts); ok {
			return uo.UnmarshalQDFOpts(src, true)
		}
	}
	return u.UnmarshalQDF(src)
}
