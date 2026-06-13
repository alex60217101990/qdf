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
