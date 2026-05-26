//go:build mips || mips64 || ppc64 || s390x

package qdf

// On big-endian targets the QPack raw-LE bulk codec falls back to a
// scalar LE-byte emit loop. The wire format is unchanged; only the
// platform-side encode/decode path is slower.
const nativeLittleEndian = false
