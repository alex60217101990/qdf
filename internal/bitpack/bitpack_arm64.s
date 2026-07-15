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

// -----------------------------------------------------------------------
// Variable-width pack kernels: 10 / 12 / 14 / 20-bit
//
// Strategy: for each group of 4 values (2 for 20-bit) load them into two
// 128-bit NEON registers (one for 20-bit), mask each lane, then shift each
// lane to its bit-stream offset using SSHL with a pre-loaded per-lane shift
// vector. VORR combines the two halves; EXT rotates lanes so the upper 64-bit
// lane falls into lane 0; a second VORR produces the fully-packed 64-bit word
// in lane 0; UMOV extracts it to a GP register; byte-granular stores write
// the result. The n%4 (or n%2) tail is handled by the scalar Pack in Go.
//
// WORD-encoded NEON instructions (not natively named in Go's arm64 assembler):
//
//   SSHL Vd.2D, Vn.2D, Vm.2D  — signed shift left each lane by Vm lane amount
//     encoding: 0 Q 0 01110 11 1 Vm 0100 01 Vn Rd  (Q=1, size=11=D)
//     SSHL V0.2D, V0.2D, V6.2D  → WORD $0x4EE64400
//     SSHL V1.2D, V1.2D, V7.2D  → WORD $0x4EE74421
//
//   VORR Vd.16B, Vn.16B, Vm.16B  — bitwise OR (vector ORR)
//     encoding: 0 Q 0 01110 10 1 Rm 000111 Rn Rd  (Q=1)
//     VORR V2.16B, V1.16B, V0.16B  (Rd=V2, Rn=V1, Rm=V0)  → WORD $0x4EA01C22
//     VORR V2.16B, V2.16B, V3.16B  (Rd=V2, Rn=V2, Rm=V3)  → WORD $0x4EA31C42
//     VORR V0.16B, V0.16B, V1.16B  (Rd=V0, Rn=V0, Rm=V1)  → WORD $0x4EA11C00
//
//   EXT Vd.16B, Vn.16B, Vm.16B, #8  — extract/rotate by 8 bytes
//     encoding: 0 Q 1 01110 00 0 Rm 0 1000 0 Rn Rd  (Q=1)
//     EXT V3.16B, V2.16B, V2.16B, #8  (Rd=V3, Rn=V2, Rm=V2)  → WORD $0x6E024043
//     EXT V1.16B, V0.16B, V0.16B, #8  (Rd=V1, Rn=V0, Rm=V0)  → WORD $0x6E004001
//
//   UMOV Xd, Vn.D[0]  — unsigned move lane 0 to GP register (Q=1, imm5=01000)
//     encoding: 0 1 001110 000 01000 001111 1 Rn Rd
//     UMOV X3, V2.D[0]  → WORD $0x4E083C43
//     UMOV X3, V0.D[0]  → WORD $0x4E083C03
// -----------------------------------------------------------------------

// Shift-amount tables: per-lane left-shift amounts for each bit width.
// For 4-value widths (10/12/14): 32 bytes = two 128-bit vectors (V6, V7).
// For 2-value width (20): 16 bytes = one 128-bit vector (V6).

DATA packVar10Shifts<>+0(SB)/8,  $0   // V6.D[0] = lane-0 shift for 10-bit
DATA packVar10Shifts<>+8(SB)/8,  $10  // V6.D[1] = lane-1 shift for 10-bit
DATA packVar10Shifts<>+16(SB)/8, $20  // V7.D[0] = lane-2 shift for 10-bit
DATA packVar10Shifts<>+24(SB)/8, $30  // V7.D[1] = lane-3 shift for 10-bit
GLOBL packVar10Shifts<>(SB), RODATA|NOPTR, $32

DATA packVar12Shifts<>+0(SB)/8,  $0   // V6.D[0] = lane-0 shift for 12-bit
DATA packVar12Shifts<>+8(SB)/8,  $12  // V6.D[1] = lane-1 shift for 12-bit
DATA packVar12Shifts<>+16(SB)/8, $24  // V7.D[0] = lane-2 shift for 12-bit
DATA packVar12Shifts<>+24(SB)/8, $36  // V7.D[1] = lane-3 shift for 12-bit
GLOBL packVar12Shifts<>(SB), RODATA|NOPTR, $32

DATA packVar14Shifts<>+0(SB)/8,  $0   // V6.D[0] = lane-0 shift for 14-bit
DATA packVar14Shifts<>+8(SB)/8,  $14  // V6.D[1] = lane-1 shift for 14-bit
DATA packVar14Shifts<>+16(SB)/8, $28  // V7.D[0] = lane-2 shift for 14-bit
DATA packVar14Shifts<>+24(SB)/8, $42  // V7.D[1] = lane-3 shift for 14-bit
GLOBL packVar14Shifts<>(SB), RODATA|NOPTR, $32

DATA packVar20Shifts<>+0(SB)/8, $0   // V6.D[0] = lane-0 shift for 20-bit
DATA packVar20Shifts<>+8(SB)/8, $20  // V6.D[1] = lane-1 shift for 20-bit
GLOBL packVar20Shifts<>(SB), RODATA|NOPTR, $16

// Mask tables: (1<<width)-1 replicated across all 64-bit lanes used.
// 10-bit mask = 0x3FF, 12-bit = 0xFFF, 14-bit = 0x3FFF, 20-bit = 0xFFFFF.

DATA packVar10Mask<>+0(SB)/8, $0x3FF
DATA packVar10Mask<>+8(SB)/8, $0x3FF
GLOBL packVar10Mask<>(SB), RODATA|NOPTR, $16

DATA packVar12Mask<>+0(SB)/8, $0xFFF
DATA packVar12Mask<>+8(SB)/8, $0xFFF
GLOBL packVar12Mask<>(SB), RODATA|NOPTR, $16

DATA packVar14Mask<>+0(SB)/8, $0x3FFF
DATA packVar14Mask<>+8(SB)/8, $0x3FFF
GLOBL packVar14Mask<>(SB), RODATA|NOPTR, $16

DATA packVar20Mask<>+0(SB)/8, $0xFFFFF
DATA packVar20Mask<>+8(SB)/8, $0xFFFFF
GLOBL packVar20Mask<>(SB), RODATA|NOPTR, $16

// func packBits10NEON(out []byte, vals []uint64, groups int)
//
// Packs `groups` chunks of four 10-bit values into byte-aligned 40-bit chunks
// (5 bytes each). Caller guarantees len(out) >= 5*groups, len(vals) >= 4*groups.
//
// Register allocation:
//   R0 = out ptr, R1 = vals ptr, R2 = groups count
//   V4 = mask (0x3FF in both lanes), V6 = shifts [0,10], V7 = shifts [20,30]
//   V0,V1 = loaded values; V2,V3 = temporaries
//   R3,R4 = scalar GP temporaries for store
TEXT ·packBits10NEON(SB), NOSPLIT, $0-56
	MOVD out_base+0(FP), R0
	MOVD vals_base+24(FP), R1
	MOVD groups+48(FP), R2
	// Load per-lane shift vectors and mask vector.
	MOVD $packVar10Shifts<>(SB), R10
	VLD1 (R10), [V6.D2, V7.D2]           // V6=[0,10] V7=[20,30]
	MOVD $packVar10Mask<>(SB), R10
	VLD1 (R10), [V4.D2]                   // V4=[0x3FF,0x3FF]

loop_pk10:
	CBZ  R2, done_pk10
	VLD1 (R1), [V0.D2, V1.D2]            // V0=[v0,v1], V1=[v2,v3]
	ADD  $32, R1
	WORD $0x4E241C00                       // VAND V0.16B = V0.16B & V4.16B (Rm=V4,Rn=V0,Rd=V0)
	WORD $0x4E241C21                       // VAND V1.16B = V1.16B & V4.16B (Rm=V4,Rn=V1,Rd=V1)
	WORD $0x4EE64400                       // SSHL V0.2D, V0.2D, V6.2D → [v0<<0, v1<<10]
	WORD $0x4EE74421                       // SSHL V1.2D, V1.2D, V7.2D → [v2<<20, v3<<30]
	WORD $0x4EA01C22                       // VORR V2.16B = V0.16B | V1.16B → [v0|v2, v1|v3]
	WORD $0x6E024043                       // EXT  V3.16B, V2.16B, V2.16B, #8 → [v1|v3, v0|v2]
	WORD $0x4EA31C42                       // VORR V2.16B |= V3.16B → V2.D[0] = all 4 ORed
	WORD $0x4E083C43                       // UMOV X3, V2.D[0]
	MOVW R3, (R0)                          // store bytes 0-3
	LSR  $32, R3, R4
	MOVB R4, 4(R0)                         // store byte 4
	ADD  $5, R0
	SUB  $1, R2
	B    loop_pk10

done_pk10:
	RET

// func packBits12NEON(out []byte, vals []uint64, groups int)
//
// Packs `groups` chunks of four 12-bit values into byte-aligned 48-bit chunks
// (6 bytes each). Caller guarantees len(out) >= 6*groups, len(vals) >= 4*groups.
TEXT ·packBits12NEON(SB), NOSPLIT, $0-56
	MOVD out_base+0(FP), R0
	MOVD vals_base+24(FP), R1
	MOVD groups+48(FP), R2
	MOVD $packVar12Shifts<>(SB), R10
	VLD1 (R10), [V6.D2, V7.D2]           // V6=[0,12] V7=[24,36]
	MOVD $packVar12Mask<>(SB), R10
	VLD1 (R10), [V4.D2]                   // V4=[0xFFF,0xFFF]

loop_pk12:
	CBZ  R2, done_pk12
	VLD1 (R1), [V0.D2, V1.D2]            // V0=[v0,v1], V1=[v2,v3]
	ADD  $32, R1
	WORD $0x4E241C00                       // VAND V0.16B = V0.16B & V4.16B (Rm=V4,Rn=V0,Rd=V0)
	WORD $0x4E241C21                       // VAND V1.16B = V1.16B & V4.16B (Rm=V4,Rn=V1,Rd=V1)
	WORD $0x4EE64400                       // SSHL V0.2D, V0.2D, V6.2D → [v0<<0, v1<<12]
	WORD $0x4EE74421                       // SSHL V1.2D, V1.2D, V7.2D → [v2<<24, v3<<36]
	WORD $0x4EA01C22                       // VORR V2.16B = V0.16B | V1.16B → [v0|v2, v1|v3]
	WORD $0x6E024043                       // EXT  V3.16B, V2.16B, V2.16B, #8 → [v1|v3, v0|v2]
	WORD $0x4EA31C42                       // VORR V2.16B |= V3.16B → V2.D[0] = all 4 ORed
	WORD $0x4E083C43                       // UMOV X3, V2.D[0]
	MOVW R3, (R0)                          // store bytes 0-3
	LSR  $32, R3, R4
	MOVH R4, 4(R0)                         // store bytes 4-5
	ADD  $6, R0
	SUB  $1, R2
	B    loop_pk12

done_pk12:
	RET

// func packBits14NEON(out []byte, vals []uint64, groups int)
//
// Packs `groups` chunks of four 14-bit values into byte-aligned 56-bit chunks
// (7 bytes each). Caller guarantees len(out) >= 7*groups, len(vals) >= 4*groups.
TEXT ·packBits14NEON(SB), NOSPLIT, $0-56
	MOVD out_base+0(FP), R0
	MOVD vals_base+24(FP), R1
	MOVD groups+48(FP), R2
	MOVD $packVar14Shifts<>(SB), R10
	VLD1 (R10), [V6.D2, V7.D2]           // V6=[0,14] V7=[28,42]
	MOVD $packVar14Mask<>(SB), R10
	VLD1 (R10), [V4.D2]                   // V4=[0x3FFF,0x3FFF]

loop_pk14:
	CBZ  R2, done_pk14
	VLD1 (R1), [V0.D2, V1.D2]            // V0=[v0,v1], V1=[v2,v3]
	ADD  $32, R1
	WORD $0x4E241C00                       // VAND V0.16B = V0.16B & V4.16B (Rm=V4,Rn=V0,Rd=V0)
	WORD $0x4E241C21                       // VAND V1.16B = V1.16B & V4.16B (Rm=V4,Rn=V1,Rd=V1)
	WORD $0x4EE64400                       // SSHL V0.2D, V0.2D, V6.2D → [v0<<0, v1<<14]
	WORD $0x4EE74421                       // SSHL V1.2D, V1.2D, V7.2D → [v2<<28, v3<<42]
	WORD $0x4EA01C22                       // VORR V2.16B = V0.16B | V1.16B → [v0|v2, v1|v3]
	WORD $0x6E024043                       // EXT  V3.16B, V2.16B, V2.16B, #8 → [v1|v3, v0|v2]
	WORD $0x4EA31C42                       // VORR V2.16B |= V3.16B → V2.D[0] = all 4 ORed
	WORD $0x4E083C43                       // UMOV X3, V2.D[0]
	MOVW R3, (R0)                          // store bytes 0-3
	LSR  $32, R3, R4
	MOVH R4, 4(R0)                         // store bytes 4-5
	LSR  $48, R3, R5
	MOVB R5, 6(R0)                         // store byte 6
	ADD  $7, R0
	SUB  $1, R2
	B    loop_pk14

done_pk14:
	RET

// func packBits20NEON(out []byte, vals []uint64, pairs int)
//
// Packs `pairs` chunks of two 20-bit values into byte-aligned 40-bit chunks
// (5 bytes each). Caller guarantees len(out) >= 5*pairs, len(vals) >= 2*pairs.
//
// Two values fit in one 128-bit register (V0): [v0, v1].
// After SSHL with [0,20]: V0=[v0<<0, v1<<20].
// EXT V1, V0, V0, #8 → V1=[v1<<20, v0<<0].
// VORR V0 |= V1 → V0.D[0] = (v0<<0)|(v1<<20) = all packed.
TEXT ·packBits20NEON(SB), NOSPLIT, $0-56
	MOVD out_base+0(FP), R0
	MOVD vals_base+24(FP), R1
	MOVD pairs+48(FP), R2
	MOVD $packVar20Shifts<>(SB), R10
	VLD1 (R10), [V6.D2]                   // V6=[0,20]
	MOVD $packVar20Mask<>(SB), R10
	VLD1 (R10), [V4.D2]                   // V4=[0xFFFFF,0xFFFFF]

loop_pk20:
	CBZ  R2, done_pk20
	VLD1 (R1), [V0.D2]                    // V0=[v0,v1]
	ADD  $16, R1
	WORD $0x4E241C00                       // VAND V0.16B = V0.16B & V4.16B (Rm=V4,Rn=V0,Rd=V0)
	WORD $0x4EE64400                       // SSHL V0.2D, V0.2D, V6.2D → [v0<<0, v1<<20]
	WORD $0x6E004001                       // EXT  V1.16B, V0.16B, V0.16B, #8 → [v1<<20, v0<<0]
	WORD $0x4EA11C00                       // VORR V0.16B |= V1.16B → V0.D[0] = all 2 ORed
	WORD $0x4E083C03                       // UMOV X3, V0.D[0]
	MOVW R3, (R0)                          // store bytes 0-3
	LSR  $32, R3, R4
	MOVB R4, 4(R0)                         // store byte 4
	ADD  $5, R0
	SUB  $1, R2
	B    loop_pk20

done_pk20:
	RET
