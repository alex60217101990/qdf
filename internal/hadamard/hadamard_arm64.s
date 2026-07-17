#include "textflag.h"

// func fwhtNEON(a []float64)
// Go ABI: a_base+0(FP)=ptr, a_len+8(FP)=len, a_cap+16(FP)=cap (unused)
//
// Register allocation:
//   R0  = &a[0]    (base pointer, preserved)
//   R1  = n        (len, preserved)
//   R3  = h        (stage stride, element count)
//   R4  = i        (outer loop element index)
//   R9  = i*8      (byte offset of a[i])
//   R10 = h*8      (byte stride to a[i+h])
//   R11 = pA       (&a[i+j], walks forward in inner loop)
//   R12 = pB       (&a[i+h+j], walks forward in inner loop)
//   R13 = j_count  (remaining butterflies in this block)
//   R14 = h<<1     (temp: distance between blocks)
//   F0–F15         = butterfly operands (4-pair unrolled body)
TEXT ·fwhtNEON(SB), NOSPLIT, $0-24
	MOVD  a_base+0(FP), R0
	MOVD  a_len+8(FP), R1

	// Nothing to do for n < 2.
	CMP   $2, R1
	BLT   done

	// ---- Stage h=1: scalar butterfly on adjacent pairs ----
	// For h=1, one j-step per 2-element block: (a[i],a[i+1]) pairs.
	// Use pointer walking to avoid per-iteration address computation.
	MOVD  R0, R11              // pA = &a[0]
	ADD   $8, R11, R12         // pB = &a[1]
	LSL   $3, R1, R13          // R13 = n*8 (total byte count)
h1_inner:
	CMP   $16, R13             // 16 bytes = one adjacent pair
	BLT   after_stage1
	FMOVD (R11), F0            // F0 = a[i]
	FMOVD (R12), F1            // F1 = a[i+1]
	FADDD F0, F1, F2           // F2 = a[i]+a[i+1]
	FSUBD F1, F0, F3           // F3 = a[i]-a[i+1]  (FSUBD Fm,Fn,Fd → Fd=Fn-Fm)
	FMOVD F2, (R11)
	FMOVD F3, (R12)
	ADD   $16, R11, R11        // pA += 16 (step over pair)
	ADD   $16, R12, R12        // pB += 16
	SUB   $16, R13, R13
	B     h1_inner
after_stage1:

	// ---- Stages h=2,4,8,…: unrolled scalar butterfly ----
	// Inner loop uses a 4-pair tier (F0–F15) when j_count≥4, and a 2-pair
	// tier (F0–F7) for the remainder. All pairs in each tier are independent,
	// giving the OoO engine on Apple M-series 4 concurrent FP chains.
	MOVD  $2, R3               // h = 2
hloop:
	CMP   R1, R3               // flags ← R3 - R1 (h - n)
	BGE   done                 // exit when h >= n
	MOVD  $0, R4               // i = 0
outer:
	CMP   R1, R4               // flags ← R4 - R1 (i - n)
	BGE   next_h               // next stage when i >= n
	LSL   $3, R4, R9           // R9  = i*8
	LSL   $3, R3, R10          // R10 = h*8
	ADD   R0, R9, R11          // R11 = &a[i]
	ADD   R11, R10, R12        // R12 = &a[i+h]
	MOVD  R3, R13              // j_count = h
inner4:
	CMP   $4, R13              // 4-pair tier: needs j_count >= 4
	BLT   inner2
	// ---- 4 independent butterfly pairs ----
	// Pair 0
	FMOVD (R11), F0
	FMOVD (R12), F1
	// Pair 1
	FMOVD 8(R11), F2
	FMOVD 8(R12), F3
	// Pair 2
	FMOVD 16(R11), F8
	FMOVD 16(R12), F9
	// Pair 3
	FMOVD 24(R11), F10
	FMOVD 24(R12), F11
	// Butterfly (all 4 chains fully independent — CPU can issue simultaneously)
	FADDD F0, F1, F4           // s0 = x0+y0
	FSUBD F1, F0, F5           // d0 = x0-y0
	FADDD F2, F3, F6           // s1 = x1+y1
	FSUBD F3, F2, F7           // d1 = x1-y1
	FADDD F8, F9, F12          // s2 = x2+y2
	FSUBD F9, F8, F13          // d2 = x2-y2
	FADDD F10, F11, F14        // s3 = x3+y3
	FSUBD F11, F10, F15        // d3 = x3-y3
	// Store
	FMOVD F4, (R11)
	FMOVD F6, 8(R11)
	FMOVD F12, 16(R11)
	FMOVD F14, 24(R11)
	FMOVD F5, (R12)
	FMOVD F7, 8(R12)
	FMOVD F13, 16(R12)
	FMOVD F15, 24(R12)
	ADD   $32, R11, R11        // pA += 4×float64
	ADD   $32, R12, R12        // pB += 4×float64
	SUB   $4, R13, R13
	B     inner4
inner2:
	CMP   $2, R13              // 2-pair tier: needs j_count >= 2
	BLT   inner_tail
	// ---- 2 independent butterfly pairs ----
	FMOVD (R11), F0
	FMOVD (R12), F1
	FMOVD 8(R11), F2
	FMOVD 8(R12), F3
	FADDD F0, F1, F4
	FSUBD F1, F0, F5
	FADDD F2, F3, F6
	FSUBD F3, F2, F7
	FMOVD F4, (R11)
	FMOVD F5, (R12)
	FMOVD F6, 8(R11)
	FMOVD F7, 8(R12)
	ADD   $16, R11, R11
	ADD   $16, R12, R12
	SUB   $2, R13, R13
inner_tail:
	// Scalar tail: 0 or 1 remaining element.
	// (h is always a power-of-2 ≥ 2, so j_count is always even; this
	//  branch is dead for valid inputs but retained for correctness.)
	CBZ   R13, inner_done
	FMOVD (R11), F0
	FMOVD (R12), F1
	FADDD F0, F1, F2
	FSUBD F1, F0, F3
	FMOVD F2, (R11)
	FMOVD F3, (R12)
inner_done:
	LSL   $1, R3, R14          // R14 = h<<1
	ADD   R14, R4, R4          // i += h<<1
	B     outer
next_h:
	LSL   $1, R3, R3           // h <<= 1
	B     hloop
done:
	RET
