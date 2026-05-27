//go:build mips || mips64 || ppc64 || s390x

package endian

// NativeIsLittle on big-endian targets — see endian_le.go for the
// contract. The QPack raw-LE bulk codec falls back to a per-element
// shuffle when this is false. Wire format is unchanged.
const NativeIsLittle = false
