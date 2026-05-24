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
type Unmarshaler interface {
	UnmarshalQDF(src []byte) (n int, err error)
}
