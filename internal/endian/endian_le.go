//go:build 386 || amd64 || arm || arm64 || loong64 || mips64le || mipsle || ppc64le || riscv64 || wasm

// Package endian exposes a build-time constant for native byte order.
// QDF's wire format is little-endian; on little-endian targets the
// QPack raw-LE bulk codec can write/read numeric slices as a single
// memcpy via unsafe slice aliasing. On big-endian targets the codec
// falls back to per-element shuffles.
package endian

// NativeIsLittle is true on architectures whose native byte order is
// little-endian. Pin a single build-tagged constant in this package
// rather than dual-defined consts scattered across the codec so the
// branch decision lives in one place.
const NativeIsLittle = true
