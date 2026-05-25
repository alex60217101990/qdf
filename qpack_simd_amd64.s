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
