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
