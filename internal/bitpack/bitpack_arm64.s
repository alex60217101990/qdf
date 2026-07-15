//go:build arm64 && qdf_simd

#include "textflag.h"

// func unpackBits8NEON(out []uint64, in []byte, n int)
// out[i] = uint64(in[i]) for i in [0,n); n is a multiple of 8.
// Widens 8 bytes -> 8 uint64 per iteration via a USHLL zero-extend chain.
TEXT ·unpackBits8NEON(SB), NOSPLIT, $0-56
	MOVD out_base+0(FP), R0
	MOVD in_base+24(FP), R1
	MOVD n+48(FP), R2

loop8:
	CBZ     R2, done
	VLD1    (R1), [V0.B8]           // 8 bytes
	VUSHLL  $0, V0.B8, V1.H8        // u8 -> u16 x8
	VUSHLL  $0, V1.H4, V2.S4        // low 4 u16 -> u32
	VUSHLL2 $0, V1.H8, V3.S4        // high 4 u16 -> u32
	VUSHLL  $0, V2.S2, V4.D2        // u32 -> u64
	VUSHLL2 $0, V2.S4, V5.D2
	VUSHLL  $0, V3.S2, V6.D2
	VUSHLL2 $0, V3.S4, V7.D2
	VST1    [V4.D2, V5.D2, V6.D2, V7.D2], (R0)
	ADD     $64, R0
	ADD     $8, R1
	SUB     $8, R2
	B       loop8

done:
	RET

// func unpackBits16NEON(out []uint64, in []byte, n int)
// out[i] = uint64(LE_u16(in[i*2:])) for i in [0,n); n is a multiple of 8.
TEXT ·unpackBits16NEON(SB), NOSPLIT, $0-56
	MOVD out_base+0(FP), R0
	MOVD in_base+24(FP), R1
	MOVD n+48(FP), R2

loop16:
	CBZ     R2, done16
	VLD1    (R1), [V0.H8]           // 8 uint16 (16 bytes)
	VUSHLL  $0, V0.H4, V1.S4        // low 4 u16 -> u32
	VUSHLL2 $0, V0.H8, V2.S4        // high 4 u16 -> u32
	VUSHLL  $0, V1.S2, V3.D2        // u32 -> u64
	VUSHLL2 $0, V1.S4, V4.D2
	VUSHLL  $0, V2.S2, V5.D2
	VUSHLL2 $0, V2.S4, V6.D2
	VST1    [V3.D2, V4.D2, V5.D2, V6.D2], (R0)
	ADD     $64, R0
	ADD     $16, R1
	SUB     $8, R2
	B       loop16

done16:
	RET

// func unpackBits32NEON(out []uint64, in []byte, n int)
// out[i] = uint64(LE_u32(in[i*4:])) for i in [0,n); n is a multiple of 4.
TEXT ·unpackBits32NEON(SB), NOSPLIT, $0-56
	MOVD out_base+0(FP), R0
	MOVD in_base+24(FP), R1
	MOVD n+48(FP), R2

loop32:
	CBZ     R2, done32
	VLD1    (R1), [V0.S4]           // 4 uint32 (16 bytes)
	VUSHLL  $0, V0.S2, V1.D2        // low 2 u32 -> u64
	VUSHLL2 $0, V0.S4, V2.D2        // high 2 u32 -> u64
	VST1    [V1.D2, V2.D2], (R0)
	ADD     $32, R0
	ADD     $16, R1
	SUB     $4, R2
	B       loop32

done32:
	RET

// func unpackVar2NEON(out []uint64, in []byte, pairs int, twoB int, shifts *[16]int64, mask uint64)
// Decodes 2 values per iteration for an arbitrary width b in [1,28]. Each pair
// broadcasts the 8-byte window at the current byte offset to both lanes, shifts
// each lane right by its in-window offset (USHL with the negative shift vector
// for off), and ANDs the width mask. bitOff advances by twoB (=2*b) per pair.
TEXT ·unpackVar2NEON(SB), NOSPLIT, $0-80
	MOVD out_base+0(FP), R0
	MOVD in_base+24(FP), R1
	MOVD pairs+48(FP), R2
	MOVD twoB+56(FP), R3
	MOVD shifts+64(FP), R4
	MOVD mask+72(FP), R5
	VDUP R5, V2.D2                 // mask broadcast to both lanes
	MOVD $0, R6                    // bitOff

loop_v2:
	CBZ   R2, done_v2
	LSR   $3, R6, R7              // byteOff = bitOff>>3
	AND   $7, R6, R8             // off = bitOff&7
	ADD   R7, R1, R9             // ptr = in_base + byteOff
	VLD1R (R9), [V0.D2]          // broadcast 8 bytes -> both lanes
	LSL   $4, R8, R10            // off*16 (16 bytes per shift vector)
	ADD   R10, R4, R11
	VLD1  (R11), [V1.D2]         // shift vector [-off, -(off+b)]
	// 0x4EE14400 is SSHL V0.2D,V0.2D,V1.2D (signed; bit29 U=0), NOT USHL — Go asm
	// lacks both vector mnemonics so it is WORD-encoded. The shift vector is
	// negative (a right shift); SSHL sign-fills the high bits, but the VAND mask
	// below keeps only the low b bits, and for this 2-at-a-time kernel b<=28, so
	// off+b<=35 and the sign fill (bits >= 64-35 = 29) never reaches the masked
	// region (bits < b <= 28). SAFE ONLY while b<=28: if this kernel is ever
	// widened, switch to USHL (0x6EE14400, U=1) for an explicit zero fill or the
	// sign bits will corrupt the result.
	WORD  $0x4EE14400            // SSHL V0.2D, V0.2D, V1.2D  (see note above)
	VAND  V2.B16, V0.B16, V0.B16 // mask
	VST1  [V0.D2], (R0)
	ADD   $16, R0
	ADD   R3, R6, R6             // bitOff += 2*b
	SUB   $1, R2, R2
	B     loop_v2

done_v2:
	RET

// packBoolWeights: bit-position weights for packing 8 bools → 1 byte.
// In memory (little-endian) this is [0x01,0x02,0x04,0x08,0x10,0x20,0x40,0x80]
// so VLD1 into B8 loads weight[i] = 1<<i into lane i.
DATA packBoolWeights<>+0(SB)/8, $0x8040201008040201
GLOBL packBoolWeights<>(SB), RODATA|NOPTR, $8

// func packBoolsNEON8(src *bool, dst *byte)
// Packs 8 bools from src[0..7] into one byte at *dst, LSB-first:
//   bool[i] → bit i of *dst.
// Strategy: load 8 bool bytes, AND with 1 to isolate the LSB per lane,
// multiply each lane by its bit-position weight (1,2,4,8,16,32,64,128),
// horizontally sum all 8 lanes → single byte, store to *dst.
// Caller guarantees len(src) >= 8 and *dst is zero-initialized.
//
// Instructions not in Go's arm64 assembler are WORD-encoded:
//   WORD $0x0E229C00 = MUL  V0.8B, V0.8B, V2.8B
//   WORD $0x0E31B803 = ADDV B3, V0.8B            (result → B3 = V3.B[0])
//   WORD $0x0D000023 = ST1  {V3.B}[0], [X1]      (store B3 to *R1)
TEXT ·packBoolsNEON8(SB),NOSPLIT,$0-16
	MOVD src+0(FP), R0
	MOVD dst+8(FP), R1
	MOVD $packBoolWeights<>(SB), R2
	VLD1 (R0), [V0.B8]              // V0[i] = src[i] (bool byte, 0 or 1)
	VLD1 (R2), [V2.B8]              // V2[i] = 1<<i  (bit-position weights)
	MOVD $1, R2
	VDUP R2, V1.B8                  // V1 = [1,1,1,1,1,1,1,1] (LSB mask)
	VAND V0.B8, V1.B8, V0.B8       // V0[i] &= 1 (sanitize non-canonical bools)
	WORD $0x0E229C00                // MUL  V0.8B, V0.8B, V2.8B
	WORD $0x0E31B803                // ADDV B3, V0.8B   → B3 = packed byte
	WORD $0x0D000023                // ST1  {V3.B}[0], [X1] → *R1 = B3
	RET

// func decodeHex4NEON(dst []byte, src []byte, lut *[16]byte, blocks int)
//
// Expands a 4-bit nibble stream src into bytes via the 16-entry lut, two output
// bytes per src byte (dst[2i]=lut[src[i]&0xf], dst[2i+1]=lut[src[i]>>4]). Each
// iteration loads 16 src bytes and writes 32 dst bytes; `blocks` counts the
// 16-byte groups. The caller handles the sub-16 / odd tail scalar.
//
// V3 = lut, V4 = 0x0f mask. Per chunk: V1 = low nibbles (AND), V2 = high nibbles
// (USHR #4), TBL maps each through the lut, ZIP1/ZIP2 interleave lo,hi,lo,hi.
TEXT ·decodeHex4NEON(SB), NOSPLIT, $0-64
	MOVD    dst_base+0(FP), R0
	MOVD    src_base+24(FP), R1
	MOVD    lut+48(FP), R2
	MOVD    blocks+56(FP), R3
	VLD1    (R2), [V3.B16]
	MOVD    $0x0f0f0f0f0f0f0f0f, R4
	VDUP    R4, V4.D2

loop_dh4:
	CBZ     R3, done_dh4
	VLD1    (R1), [V0.B16]
	VAND    V4.B16, V0.B16, V1.B16
	VUSHR   $4, V0.B16, V2.B16
	VTBL    V1.B16, [V3.B16], V5.B16
	VTBL    V2.B16, [V3.B16], V6.B16
	VZIP1   V6.B16, V5.B16, V7.B16
	VZIP2   V6.B16, V5.B16, V8.B16
	VST1    [V7.B16, V8.B16], (R0)
	ADD     $32, R0
	ADD     $16, R1
	SUB     $1, R3
	B       loop_dh4

done_dh4:
	RET

// func packBits8NEON(out []byte, vals []uint64, n int)
//
// Packs the low byte of each of n uint64 values into n contiguous bytes of
// out. n must be a multiple of 8. Caller ensures len(out) >= n and
// len(vals) >= n.
//
// Strategy: load 8 uint64s (64 bytes) into V0-V3, then apply a three-step
// XTN (extract-narrow) chain to reduce each 64-bit lane to 8 bits:
//   Step 1: XTN/XTN2  V4.2S←V0.2D, V4.4S←V1.2D  (64→32-bit per lane)
//           XTN/XTN2  V5.2S←V2.2D, V5.4S←V3.2D
//   Step 2: XTN/XTN2  V6.4H←V4.4S, V6.8H←V5.4S  (32→16-bit per lane)
//   Step 3: XTN       V7.8B←V6.8H                 (16→8-bit per lane)
// Store 8 result bytes; repeat.
//
// XTN/XTN2 are WORD-encoded (not named in Go's arm64 assembler):
//   XTN  Vd.2S, Vn.2D  → 0x0EA128XX  (size=10, Q=0)
//   XTN2 Vd.4S, Vn.2D  → 0x4EA128XX  (size=10, Q=1)
//   XTN  Vd.4H, Vn.4S  → 0x0E6128XX  (size=01, Q=0)
//   XTN2 Vd.8H, Vn.4S  → 0x4E6128XX  (size=01, Q=1)
//   XTN  Vd.8B, Vn.8H  → 0x0E2128XX  (size=00, Q=0)
// where XX = Rn[4:0]<<5 | Rd[4:0].
TEXT ·packBits8NEON(SB), NOSPLIT, $0-56
	MOVD out_base+0(FP), R1
	MOVD vals_base+24(FP), R0
	MOVD n+48(FP), R2

loop_pk8:
	CBZ  R2, done_pk8
	VLD1 (R0), [V0.D2, V1.D2, V2.D2, V3.D2] // load 8 uint64 (64 bytes)
	ADD  $64, R0
	WORD $0x0EA12804 // XTN  V4.2S, V0.2D  (Rn=V0=0,  Rd=V4=4)
	WORD $0x4EA12824 // XTN2 V4.4S, V1.2D  (Rn=V1=1,  Rd=V4=4)
	WORD $0x0EA12845 // XTN  V5.2S, V2.2D  (Rn=V2=2,  Rd=V5=5)
	WORD $0x4EA12865 // XTN2 V5.4S, V3.2D  (Rn=V3=3,  Rd=V5=5)
	WORD $0x0E612886 // XTN  V6.4H, V4.4S  (Rn=V4=4,  Rd=V6=6)
	WORD $0x4E6128A6 // XTN2 V6.8H, V5.4S  (Rn=V5=5,  Rd=V6=6)
	WORD $0x0E2128C7 // XTN  V7.8B, V6.8H  (Rn=V6=6,  Rd=V7=7)
	VST1 [V7.B8], (R1)                       // store 8 bytes
	ADD  $8, R1
	SUB  $8, R2
	B    loop_pk8

done_pk8:
	RET

// func packBits16NEON(out []byte, vals []uint64, n int)
//
// Packs the low 2 bytes (LE) of each of n uint64 values into 2*n contiguous
// bytes of out. n must be a multiple of 8. Caller ensures len(out) >= 2*n and
// len(vals) >= n.
//
// Strategy: same XTN chain as packBits8NEON but stops after the 32→16-bit
// step — V6.H8 already holds 8 little-endian uint16 values (16 bytes total).
TEXT ·packBits16NEON(SB), NOSPLIT, $0-56
	MOVD out_base+0(FP), R1
	MOVD vals_base+24(FP), R0
	MOVD n+48(FP), R2

loop_pk16:
	CBZ  R2, done_pk16
	VLD1 (R0), [V0.D2, V1.D2, V2.D2, V3.D2] // load 8 uint64 (64 bytes)
	ADD  $64, R0
	WORD $0x0EA12804 // XTN  V4.2S, V0.2D
	WORD $0x4EA12824 // XTN2 V4.4S, V1.2D
	WORD $0x0EA12845 // XTN  V5.2S, V2.2D
	WORD $0x4EA12865 // XTN2 V5.4S, V3.2D
	WORD $0x0E612886 // XTN  V6.4H, V4.4S
	WORD $0x4E6128A6 // XTN2 V6.8H, V5.4S  → V6 = 8 × uint16 (16 bytes)
	VST1 [V6.B16], (R1)                      // store 16 bytes
	ADD  $16, R1
	SUB  $8, R2
	B    loop_pk16

done_pk16:
	RET

// func packBits32NEON(out []byte, vals []uint64, n int)
//
// Packs the low 4 bytes (LE) of each of n uint64 values into 4*n contiguous
// bytes of out. n must be a multiple of 4. Caller ensures len(out) >= 4*n and
// len(vals) >= n.
//
// Strategy: load 4 uint64s into V0-V1 (32 bytes), apply a single XTN step
// to produce V2.S4 holding 4 little-endian uint32 values (16 bytes), store.
TEXT ·packBits32NEON(SB), NOSPLIT, $0-56
	MOVD out_base+0(FP), R1
	MOVD vals_base+24(FP), R0
	MOVD n+48(FP), R2

loop_pk32:
	CBZ  R2, done_pk32
	VLD1 (R0), [V0.D2, V1.D2]               // load 4 uint64 (32 bytes)
	ADD  $32, R0
	WORD $0x0EA12802 // XTN  V2.2S, V0.2D  (Rn=V0=0, Rd=V2=2)
	WORD $0x4EA12822 // XTN2 V2.4S, V1.2D  (Rn=V1=1, Rd=V2=2) → V2 = 4 × uint32
	VST1 [V2.B16], (R1)                      // store 16 bytes
	ADD  $16, R1
	SUB  $4, R2
	B    loop_pk32

done_pk32:
	RET
