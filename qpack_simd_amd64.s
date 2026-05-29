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

// packlow8mask is the per-128-bit-lane VPSHUFB control that gathers the
// low byte of each of two uint64 lanes to byte positions 0 and 1, zeroing
// the rest (0x80 high bit ⇒ output 0). Replicated across both 128-bit
// lanes so one VPSHUFB handles four uint64s.
DATA packlow8mask<>+0(SB)/8,  $0x8080808080800800
DATA packlow8mask<>+8(SB)/8,  $0x8080808080808080
DATA packlow8mask<>+16(SB)/8, $0x8080808080800800
DATA packlow8mask<>+24(SB)/8, $0x8080808080808080
GLOBL packlow8mask<>(SB), RODATA|NOPTR, $32

// func packBits8AVX2(out []byte, vals []uint64, n int)
//
// Packs the low byte of each of n uint64 values into n contiguous bytes
// of out (the encode inverse of unpackBits8AVX2). Equivalent to:
//
//	for i := 0; i < n; i++ {
//	    out[i] = byte(vals[i])
//	}
//
// Loads 4 uint64s (32 bytes), VPSHUFB-gathers the four low bytes to
// positions 0,1 of each 128-bit lane, then writes 2 bytes from the low
// lane and 2 from the high lane. Caller guarantees len(out) >= n,
// len(vals) >= n, n a multiple of 4, and AVX2 available.
TEXT ·packBits8AVX2(SB), NOSPLIT, $0-56
	MOVQ    out_base+0(FP), DI
	MOVQ    vals_base+24(FP), SI
	MOVQ    n+48(FP), CX
	VMOVDQU packlow8mask<>(SB), Y2

loop4_p8:
	CMPQ         CX, $4
	JL           done_p8
	VMOVDQU      (SI), Y0
	VPSHUFB      Y2, Y0, Y0
	VMOVD        X0, AX
	MOVW         AX, (DI)
	VEXTRACTI128 $1, Y0, X1
	VMOVD        X1, BX
	MOVW         BX, 2(DI)
	ADDQ         $32, SI
	ADDQ         $4, DI
	SUBQ         $4, CX
	JMP          loop4_p8

done_p8:
	VZEROUPPER
	RET

// packlow16mask gathers the low 2 bytes of each of two uint64 lanes to
// byte positions 0..3 of each 128-bit lane.
DATA packlow16mask<>+0(SB)/8,  $0x8080808009080100
DATA packlow16mask<>+8(SB)/8,  $0x8080808080808080
DATA packlow16mask<>+16(SB)/8, $0x8080808009080100
DATA packlow16mask<>+24(SB)/8, $0x8080808080808080
GLOBL packlow16mask<>(SB), RODATA|NOPTR, $32

// func packBits16AVX2(out []byte, vals []uint64, n int)
//
// Packs the low 2 bytes (LE) of each of n uint64 values into 2*n bytes of
// out. Loads 4 uint64s, VPSHUFB-gathers each lane's low 2 bytes to the
// front of its 128-bit lane, writes 4 bytes from the low lane and 4 from
// the high lane. Caller guarantees len(out) >= 2*n, len(vals) >= n, n a
// multiple of 4, AVX2 available.
TEXT ·packBits16AVX2(SB), NOSPLIT, $0-56
	MOVQ    out_base+0(FP), DI
	MOVQ    vals_base+24(FP), SI
	MOVQ    n+48(FP), CX
	VMOVDQU packlow16mask<>(SB), Y2

loop4_p16:
	CMPQ         CX, $4
	JL           done_p16
	VMOVDQU      (SI), Y0
	VPSHUFB      Y2, Y0, Y0
	VMOVD        X0, AX
	MOVL         AX, (DI)
	VEXTRACTI128 $1, Y0, X1
	VMOVD        X1, BX
	MOVL         BX, 4(DI)
	ADDQ         $32, SI
	ADDQ         $8, DI
	SUBQ         $4, CX
	JMP          loop4_p16

done_p16:
	VZEROUPPER
	RET

// packlow32mask gathers the low 4 bytes of each of two uint64 lanes to
// byte positions 0..7 of each 128-bit lane.
DATA packlow32mask<>+0(SB)/8,  $0x0b0a090803020100
DATA packlow32mask<>+8(SB)/8,  $0x8080808080808080
DATA packlow32mask<>+16(SB)/8, $0x0b0a090803020100
DATA packlow32mask<>+24(SB)/8, $0x8080808080808080
GLOBL packlow32mask<>(SB), RODATA|NOPTR, $32

// func packBits32AVX2(out []byte, vals []uint64, n int)
//
// Packs the low 4 bytes (LE) of each of n uint64 values into 4*n bytes of
// out. Loads 4 uint64s, VPSHUFB-gathers each lane's low 4 bytes to the
// front of its 128-bit lane, writes 8 bytes from the low lane and 8 from
// the high lane. Caller guarantees len(out) >= 4*n, len(vals) >= n, n a
// multiple of 4, AVX2 available.
TEXT ·packBits32AVX2(SB), NOSPLIT, $0-56
	MOVQ    out_base+0(FP), DI
	MOVQ    vals_base+24(FP), SI
	MOVQ    n+48(FP), CX
	VMOVDQU packlow32mask<>(SB), Y2

loop4_p32:
	CMPQ         CX, $4
	JL           done_p32
	VMOVDQU      (SI), Y0
	VPSHUFB      Y2, Y0, Y0
	VMOVQ        X0, AX
	MOVQ         AX, (DI)
	VEXTRACTI128 $1, Y0, X1
	VMOVQ        X1, BX
	MOVQ         BX, 8(DI)
	ADDQ         $32, SI
	ADDQ         $16, DI
	SUBQ         $4, CX
	JMP          loop4_p32

done_p32:
	VZEROUPPER
	RET

// shift12/mask12: four 12-bit values share a byte-aligned 48-bit chunk
// (6 bytes) at offsets 0,12,24,36 — all within a single 64-bit broadcast.
DATA shift12<>+0(SB)/8,  $0x0000000000000000
DATA shift12<>+8(SB)/8,  $0x000000000000000C
DATA shift12<>+16(SB)/8, $0x0000000000000018
DATA shift12<>+24(SB)/8, $0x0000000000000024
GLOBL shift12<>(SB), RODATA|NOPTR, $32
DATA mask12<>+0(SB)/8,  $0x0000000000000FFF
DATA mask12<>+8(SB)/8,  $0x0000000000000FFF
DATA mask12<>+16(SB)/8, $0x0000000000000FFF
DATA mask12<>+24(SB)/8, $0x0000000000000FFF
GLOBL mask12<>(SB), RODATA|NOPTR, $32

// func unpackBits12AVX2(out []uint64, in []byte, groups int)
//
// Decodes `groups` chunks of four 12-bit LSB-first values. Each chunk is
// byte-aligned (48 bits = 6 bytes); VPBROADCASTQ loads the 8 bytes at the
// chunk start (overreads 2) into all four lanes, VPSRLVQ shifts by
// 0/12/24/36, then AND 0xFFF isolates each value. Writes 4 uint64 per
// chunk, advances 6 input bytes. Caller guarantees len(out) >= 4*groups,
// 6*(groups-1)+8 <= len(in), AVX2 available.
TEXT ·unpackBits12AVX2(SB), NOSPLIT, $0-56
	MOVQ    out_base+0(FP), DI
	MOVQ    in_base+24(FP), SI
	MOVQ    groups+48(FP), CX
	VMOVDQU shift12<>(SB), Y1
	VMOVDQU mask12<>(SB), Y2

loop_p12:
	TESTQ        CX, CX
	JZ           done_p12
	VPBROADCASTQ (SI), Y0
	VPSRLVQ      Y1, Y0, Y0
	VPAND        Y2, Y0, Y0
	VMOVDQU      Y0, (DI)
	ADDQ         $6, SI
	ADDQ         $32, DI
	DECQ         CX
	JMP          loop_p12

done_p12:
	VZEROUPPER
	RET

// width-10: four 10-bit values share a byte-aligned 40-bit chunk (5 bytes)
// at offsets 0,10,20,30. Four lanes -> 256-bit.
DATA shift10<>+0(SB)/8,  $0x0000000000000000
DATA shift10<>+8(SB)/8,  $0x000000000000000A
DATA shift10<>+16(SB)/8, $0x0000000000000014
DATA shift10<>+24(SB)/8, $0x000000000000001E
GLOBL shift10<>(SB), RODATA|NOPTR, $32
DATA mask10<>+0(SB)/8,  $0x00000000000003FF
DATA mask10<>+8(SB)/8,  $0x00000000000003FF
DATA mask10<>+16(SB)/8, $0x00000000000003FF
DATA mask10<>+24(SB)/8, $0x00000000000003FF
GLOBL mask10<>(SB), RODATA|NOPTR, $32

// func unpackBits10AVX2(out []uint64, in []byte, groups int)
// Each group: 4 values from a 5-byte chunk. VPBROADCASTQ the 8 bytes at
// the chunk start (overreads 3) to all four lanes, VPSRLVQ by 0/10/20/30,
// AND 0x3FF. Caller guarantees len(out) >= 4*groups, 5*(groups-1)+8 <=
// len(in), AVX2 available.
TEXT ·unpackBits10AVX2(SB), NOSPLIT, $0-56
	MOVQ    out_base+0(FP), DI
	MOVQ    in_base+24(FP), SI
	MOVQ    groups+48(FP), CX
	VMOVDQU shift10<>(SB), Y1
	VMOVDQU mask10<>(SB), Y2

loop_p10:
	TESTQ        CX, CX
	JZ           done_p10
	VPBROADCASTQ (SI), Y0
	VPSRLVQ      Y1, Y0, Y0
	VPAND        Y2, Y0, Y0
	VMOVDQU      Y0, (DI)
	ADDQ         $5, SI
	ADDQ         $32, DI
	DECQ         CX
	JMP          loop_p10

done_p10:
	VZEROUPPER
	RET

// width-14: four 14-bit values share a byte-aligned 56-bit chunk (7 bytes)
// at offsets 0,14,28,42.
DATA shift14<>+0(SB)/8,  $0x0000000000000000
DATA shift14<>+8(SB)/8,  $0x000000000000000E
DATA shift14<>+16(SB)/8, $0x000000000000001C
DATA shift14<>+24(SB)/8, $0x000000000000002A
GLOBL shift14<>(SB), RODATA|NOPTR, $32
DATA mask14<>+0(SB)/8,  $0x0000000000003FFF
DATA mask14<>+8(SB)/8,  $0x0000000000003FFF
DATA mask14<>+16(SB)/8, $0x0000000000003FFF
DATA mask14<>+24(SB)/8, $0x0000000000003FFF
GLOBL mask14<>(SB), RODATA|NOPTR, $32

// func unpackBits14AVX2(out []uint64, in []byte, groups int)
// Each group: 4 values from a 7-byte chunk (overreads 1).
TEXT ·unpackBits14AVX2(SB), NOSPLIT, $0-56
	MOVQ    out_base+0(FP), DI
	MOVQ    in_base+24(FP), SI
	MOVQ    groups+48(FP), CX
	VMOVDQU shift14<>(SB), Y1
	VMOVDQU mask14<>(SB), Y2

loop_p14:
	TESTQ        CX, CX
	JZ           done_p14
	VPBROADCASTQ (SI), Y0
	VPSRLVQ      Y1, Y0, Y0
	VPAND        Y2, Y0, Y0
	VMOVDQU      Y0, (DI)
	ADDQ         $7, SI
	ADDQ         $32, DI
	DECQ         CX
	JMP          loop_p14

done_p14:
	VZEROUPPER
	RET

// width-20: two 20-bit values share a byte-aligned 40-bit chunk (5 bytes)
// at offsets 0,20. Two lanes -> 128-bit.
DATA shift20<>+0(SB)/8, $0x0000000000000000
DATA shift20<>+8(SB)/8, $0x0000000000000014
GLOBL shift20<>(SB), RODATA|NOPTR, $16
DATA mask20<>+0(SB)/8, $0x00000000000FFFFF
DATA mask20<>+8(SB)/8, $0x00000000000FFFFF
GLOBL mask20<>(SB), RODATA|NOPTR, $16

// func unpackBits20AVX2(out []uint64, in []byte, pairs int)
// Each pair: 2 values from a 5-byte chunk (overreads 3).
TEXT ·unpackBits20AVX2(SB), NOSPLIT, $0-56
	MOVQ    out_base+0(FP), DI
	MOVQ    in_base+24(FP), SI
	MOVQ    pairs+48(FP), CX
	VMOVDQU shift20<>(SB), X1
	VMOVDQU mask20<>(SB), X2

loop_p20:
	TESTQ        CX, CX
	JZ           done_p20
	VPBROADCASTQ (SI), X0
	VPSRLVQ      X1, X0, X0
	VPAND        X2, X0, X0
	VMOVDQU      X0, (DI)
	ADDQ         $5, SI
	ADDQ         $16, DI
	DECQ         CX
	JMP          loop_p20

done_p20:
	VZEROUPPER
	RET

// func unpackBitsVarAVX2(out []uint64, in []byte, groups int, fourB int, shifts *[32]uint64, mask uint64)
//
// General width-b decoder for b in [1,14]: four values per iteration. For
// width b<=14 any four consecutive values fit a single 64-bit window even
// at the worst in-byte start offset (7 + 4*14 = 63 < 64), so each group
// loads 8 bytes at the byte containing its start, VPBROADCASTQ to all
// lanes, then VPSRLVQ by per-lane shift [off, off+b, off+2b, off+3b] and
// AND mask. `shifts` is a caller-built table of those vectors indexed by
// off (0..7); `fourB` is 4*b, the per-group bit advance. The last group
// loads 8 bytes at ((groups-1)*4b)>>3, so the caller bounds groups by both
// read headroom and a byte-aligned handoff. mask = (1<<b)-1.
TEXT ·unpackBitsVarAVX2(SB), NOSPLIT, $0-80
	MOVQ         out_base+0(FP), DI
	MOVQ         in_base+24(FP), BX
	MOVQ         groups+48(FP), R10
	MOVQ         fourB+56(FP), R9
	MOVQ         shifts+64(FP), R11
	MOVQ         mask+72(FP), AX
	MOVQ         AX, X2
	VPBROADCASTQ X2, Y2
	XORQ         R8, R8

loop_var:
	TESTQ        R10, R10
	JZ           done_var
	MOVQ         R8, AX
	MOVQ         R8, DX
	SHRQ         $3, AX
	ANDQ         $7, DX
	VPBROADCASTQ (BX)(AX*1), Y0
	SHLQ         $5, DX
	VMOVDQU      (R11)(DX*1), Y1
	VPSRLVQ      Y1, Y0, Y0
	VPAND        Y2, Y0, Y0
	VMOVDQU      Y0, (DI)
	ADDQ         R9, R8
	ADDQ         $32, DI
	DECQ         R10
	JMP          loop_var

done_var:
	VZEROUPPER
	RET
