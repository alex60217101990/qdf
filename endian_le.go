//go:build 386 || amd64 || arm || arm64 || loong64 || mips64le || mipsle || ppc64le || riscv64 || wasm

package qdf

// nativeLittleEndian is true on architectures whose native byte order
// matches the QDF wire's little-endian convention. On those targets the
// QPack raw-LE bulk codec can write/read numeric slices as a single
// memcpy via unsafe slice aliasing.
const nativeLittleEndian = true
