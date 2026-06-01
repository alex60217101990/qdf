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
