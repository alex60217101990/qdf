//go:build amd64 && qdf_simd

#include "textflag.h"

// func unpackBits32AVX2(out []uint64, in []byte, n int)
//
// Unpacks n little-endian uint32 values from in into n uint64 elements
// of out, zero-extending. Equivalent to:
//
//	for i := 0; i < n; i++ {
//	    out[i] = uint64(LE_u32(in[i*4:]))
//	}
//
// Uses AVX2 VPMOVZXDQ to lift four uint32s into four uint64s per
// 256-bit store. Caller guarantees len(out) >= n and len(in) >= 4*n
// and that AVX2 is available.
TEXT ·unpackBits32AVX2(SB), NOSPLIT, $0-56
	MOVQ out_base+0(FP), DI
	MOVQ in_base+24(FP), SI
	MOVQ n+48(FP), CX

	// Process 4 uint32 -> 4 uint64 per iter.
loop4:
	CMPQ CX, $4
	JL   tail
	VPMOVZXDQ (SI), Y0
	VMOVDQU   Y0, (DI)
	ADDQ      $16, SI
	ADDQ      $32, DI
	SUBQ      $4, CX
	JMP       loop4

tail:
	TESTQ CX, CX
	JZ    done

tailloop:
	MOVL  (SI), AX
	MOVQ  AX, (DI)
	ADDQ  $4, SI
	ADDQ  $8, DI
	DECQ  CX
	JNZ   tailloop

done:
	VZEROUPPER
	RET

// func unpackBits16AVX2(out []uint64, in []byte, n int)
//
// Unpacks n little-endian uint16 values from in into n uint64 elements
// of out, zero-extending. Uses AVX2 VPMOVZXWQ (4 u16 -> 4 u64 per
// 256-bit store).
TEXT ·unpackBits16AVX2(SB), NOSPLIT, $0-56
	MOVQ out_base+0(FP), DI
	MOVQ in_base+24(FP), SI
	MOVQ n+48(FP), CX

loop4_16:
	CMPQ CX, $4
	JL   tail_16
	VPMOVZXWQ (SI), Y0
	VMOVDQU   Y0, (DI)
	ADDQ      $8, SI
	ADDQ      $32, DI
	SUBQ      $4, CX
	JMP       loop4_16

tail_16:
	TESTQ CX, CX
	JZ    done_16

tailloop_16:
	MOVWQZX (SI), AX
	MOVQ    AX, (DI)
	ADDQ    $2, SI
	ADDQ    $8, DI
	DECQ    CX
	JNZ     tailloop_16

done_16:
	VZEROUPPER
	RET

// func unpackBits8AVX2(out []uint64, in []byte, n int)
//
// Unpacks n unsigned 8-bit values from in into n uint64 elements of
// out, zero-extending. Uses AVX2 VPMOVZXBQ (4 u8 -> 4 u64 per 256-bit
// store).
TEXT ·unpackBits8AVX2(SB), NOSPLIT, $0-56
	MOVQ out_base+0(FP), DI
	MOVQ in_base+24(FP), SI
	MOVQ n+48(FP), CX

loop4_8:
	CMPQ CX, $4
	JL   tail_8
	VPMOVZXBQ (SI), Y0
	VMOVDQU   Y0, (DI)
	ADDQ      $4, SI
	ADDQ      $32, DI
	SUBQ      $4, CX
	JMP       loop4_8

tail_8:
	TESTQ CX, CX
	JZ    done_8

tailloop_8:
	MOVBQZX (SI), AX
	MOVQ    AX, (DI)
	ADDQ    $1, SI
	ADDQ    $8, DI
	DECQ    CX
	JNZ     tailloop_8

done_8:
	VZEROUPPER
	RET

// func packBoolsAVX2Block32(out []byte, in []bool, blocks int)
//
// Packs `blocks * 32` booleans from in (each one Go byte, value 0 or
// 1) into `blocks * 4` bytes of out, LSB-first: bool i -> bit (i%8)
// of out[i/8].
//
// Strategy: load 32 bytes at a time, shift every 16-bit lane left
// by 7 so each bool's bit 0 moves to bit 7 of its byte (the byte
// stays either 0x00 or 0x80 because only bit 0 was ever set), then
// VPMOVMSKB extracts the 32 high bits into a 32-bit GP register and
// the resulting mask matches the wire ordering. 32 bools per
// iteration. Tail (n%32 stragglers) is handled in Go.
//
// Caller guarantees len(out) >= blocks*4, len(in) >= blocks*32,
// AVX2 available, and every byte of in is strictly 0 or 1.
TEXT ·packBoolsAVX2Block32(SB), NOSPLIT, $0-56
	MOVQ out_base+0(FP), DI
	MOVQ in_base+24(FP), SI
	MOVQ blocks+48(FP), CX

loop32_bp:
	TESTQ CX, CX
	JZ    done_bp
	VMOVDQU   (SI), Y0
	VPSLLW    $7, Y0, Y0
	VPMOVMSKB Y0, AX
	MOVL      AX, (DI)
	ADDQ      $32, SI
	ADDQ      $4, DI
	DECQ      CX
	JMP       loop32_bp

done_bp:
	VZEROUPPER
	RET
