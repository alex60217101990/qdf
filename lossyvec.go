package qdf

import (
	"encoding/binary"
	"errors"
	"math"

	"github.com/alex60217101990/qdf/internal/vecquant"
)

// nextPow2Int returns the smallest power of two >= n (minimum 1). Mirrors
// hadamard.NextPow2 so the wire layer can recompute pdim without importing the
// internal rotation package.
func nextPow2Int(n int) int {
	if n <= 1 {
		return 1
	}
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}

// lossyVecMinElems is the smallest slice length that warrants the codec's
// fixed header overhead. Slices shorter than this go through the lossless path
// even when OptLossyVec is set.
const lossyVecMinElems = 32

// toF64Into widens a []float32 or []float64 into dst (reused when cap suffices)
// and returns the filled slice. For []float64 it returns s unchanged (no copy).
// Any other type returns nil.
func toF64Into(v any, dst []float64) []float64 {
	switch s := v.(type) {
	case []float64:
		return s // already float64; caller must copy if mutation is a concern
	case []float32:
		if cap(dst) < len(s) {
			dst = make([]float64, len(s))
		}
		dst = dst[:len(s)]
		for i, x := range s {
			dst[i] = float64(x)
		}
		return dst
	}
	return nil
}

// toF64 widens a []float32 or []float64 slice to []float64.
// For []float32 it always allocates a fresh slice (safe to pass to
// appendLossyVec, which zeroes non-finite coords in place).
// Any other type returns nil.
func toF64(v any) []float64 { return toF64Into(v, nil) }

// VectorBudget expresses the fidelity target for the lossy vector codec.
// Construct it with MaxRelError, MinCosine, or TargetSNR.
type VectorBudget struct {
	kind vecquant.BudgetKind
	val  float64
	set  bool
}

// MaxRelError bounds the per-vector relative L2 error (e.g. 1e-3).
func MaxRelError(eps float64) VectorBudget {
	return VectorBudget{kind: vecquant.KindRelError, val: eps, set: true}
}

// MinCosine bounds the minimum cosine similarity (e.g. 0.999) — for embeddings.
func MinCosine(c float64) VectorBudget {
	return VectorBudget{kind: vecquant.KindCosine, val: c, set: true}
}

// TargetSNR targets a signal-to-noise ratio in dB (e.g. 40).
func TargetSNR(db float64) VectorBudget {
	return VectorBudget{kind: vecquant.KindSNR, val: db, set: true}
}

// orDefault returns the budget itself if set, otherwise the package default.
func (b VectorBudget) orDefault() VectorBudget {
	if b.set {
		return b
	}
	return MinCosine(0.999)
}

func toBudget(b VectorBudget) vecquant.Budget {
	b = b.orDefault()
	return vecquant.Budget{Kind: b.kind, Val: b.val}
}

// vecException records a single non-finite value that must be restored after
// lossy quantization. RawBits is interpreted as float32 bits when elemF32 is
// true, or float64 bits otherwise.
type vecException struct {
	vecIdx   uint64
	coordIdx uint64
	rawBits  uint64 // float32bits or float64bits
}

// collectExceptions scans vectors for non-finite (NaN/Inf) values, zeroes them
// out in-place so the lossy pipeline sees finite data, and returns the list of
// exceptions to be restored after decode.
func collectExceptions(vectors [][]float64, elemF32 bool) []vecException {
	var exc []vecException
	for vi, v := range vectors {
		for ci, x := range v {
			if math.IsNaN(x) || math.IsInf(x, 0) {
				var bits uint64
				if elemF32 {
					bits = uint64(math.Float32bits(float32(x)))
				} else {
					bits = math.Float64bits(x)
				}
				exc = append(exc, vecException{
					vecIdx:   uint64(vi),
					coordIdx: uint64(ci),
					rawBits:  bits,
				})
				vectors[vi][ci] = 0
			}
		}
	}
	return exc
}

// appendLossyVec encodes vectors into dst using the lossy vector codec and
// returns the extended slice. Wire layout:
//
//	0xFD | flags(u8: bit0=elemF32) | varuint(dim) | varuint(count) |
//	u64le seed | f64le delta | varuint(len(coords)) | coords |
//	varuint(nExc) then nExc × (varuint vecIdx, varuint coordIdx, u32/u64 bits)
//
// Non-finite values (NaN/Inf) are stored in the exception list and zeroed for
// quantization; the decoder restores them after bl.Decode().
//
// NOTE: appendLossyVec mutates vectors in place (it zeroes non-finite coords
// before encoding). Callers must pass a slice they own; the current callers
// build it via toF64, which allocates a fresh []float64.
func appendLossyVec(vectors [][]float64, elemF32 bool, b vecquant.Budget, sc *vecquant.Scratch) []byte {
	// Collect and zero out non-finite values before encoding.
	exc := collectExceptions(vectors, elemF32)

	bl := vecquant.EncodeWith(vectors, b, sc)
	var dst []byte
	dst = append(dst, tagColVecLossy)
	var flags byte
	if elemF32 {
		flags |= 1
	}
	flags |= bl.Variant << 1 // bits1-2 carry the lattice variant
	dst = append(dst, flags)
	dst = binary.AppendUvarint(dst, uint64(bl.Dim))
	dst = binary.AppendUvarint(dst, uint64(bl.Count))
	dst = binary.LittleEndian.AppendUint64(dst, bl.Seed)
	dst = binary.LittleEndian.AppendUint64(dst, math.Float64bits(bl.Delta))
	dst = binary.AppendUvarint(dst, uint64(len(bl.Coords)))
	dst = append(dst, bl.Coords...)

	// E8 carries one coset bit per 8-D block, length-prefixed.
	if bl.Variant == vecquant.VariantE8 {
		dst = binary.AppendUvarint(dst, uint64(len(bl.Cosets)))
		dst = append(dst, bl.Cosets...)
	}

	// Append exception list: varuint(nExc) then each entry.
	dst = binary.AppendUvarint(dst, uint64(len(exc)))
	for _, e := range exc {
		dst = binary.AppendUvarint(dst, e.vecIdx)
		dst = binary.AppendUvarint(dst, e.coordIdx)
		if elemF32 {
			dst = binary.LittleEndian.AppendUint32(dst, uint32(e.rawBits))
		} else {
			dst = binary.LittleEndian.AppendUint64(dst, e.rawBits)
		}
	}
	return dst
}

// readLossyVec decodes a lossy-vec block from src and returns the vectors,
// the elemF32 flag, the number of bytes consumed, and any error.
func readLossyVec(src []byte) (vectors [][]float64, elemF32 bool, used int, err error) {
	if len(src) < 2 || src[0] != tagColVecLossy {
		return nil, false, 0, errors.New("qdf: not a lossy-vec block")
	}
	off := 1
	flags := src[off]
	off++
	elemF32 = flags&1 != 0
	variant := (flags >> 1) & 0x3
	if variant != vecquant.VariantScalar && variant != vecquant.VariantE8 {
		return nil, false, 0, errors.New("qdf: bad lossy-vec variant")
	}
	dim, k := binary.Uvarint(src[off:])
	if k <= 0 {
		return nil, false, 0, errors.New("qdf: bad dim")
	}
	off += k
	count, k := binary.Uvarint(src[off:])
	if k <= 0 {
		return nil, false, 0, errors.New("qdf: bad count")
	}
	off += k
	if off+16 > len(src) {
		return nil, false, 0, errors.New("qdf: short header")
	}
	seed := binary.LittleEndian.Uint64(src[off:])
	off += 8
	delta := math.Float64frombits(binary.LittleEndian.Uint64(src[off:]))
	off += 8
	clen, k := binary.Uvarint(src[off:])
	if k <= 0 {
		return nil, false, 0, errors.New("qdf: bad coords len")
	}
	off += k
	if uint64(len(src)-off) < clen {
		return nil, false, 0, errors.New("qdf: short coords")
	}
	// Bound the output allocation: dim*count float64s must fit within
	// maxColumnarBytes (×8 accounts for float64 element size). dim and count
	// are uint64; guard each factor before the product so the multiply itself
	// cannot wrap for adversarial inputs near 2^32.
	const maxVecFactor = maxColumnarBytes / 8
	if dim > maxVecFactor || count > maxVecFactor || dim*count*8 > maxColumnarBytes {
		return nil, false, 0, errors.New("qdf: lossy-vec output exceeds bound")
	}
	coords := src[off : off+int(clen)]
	off += int(clen)

	// E8 carries a coset stream (one bit per 8-D block) after the coords.
	var cosets []byte
	if variant == vecquant.VariantE8 {
		pdim := nextPow2Int(int(dim))
		blocks := int(count) * pdim / 8
		wantCoset := (blocks + 7) / 8
		cs, used2, cerr := vecquant.ReadCosets(src[off:], wantCoset)
		if cerr != nil {
			return nil, false, 0, cerr
		}
		cosets = cs
		off += used2
	}

	bl := vecquant.Block{
		Dim:     int(dim),
		Count:   int(count),
		Seed:    seed,
		Delta:   delta,
		Coords:  coords,
		Variant: variant,
		Cosets:  cosets,
	}
	vectors = bl.Decode()
	if vectors == nil && count != 0 {
		return nil, false, 0, errors.New("qdf: lossy-vec decode failed")
	}

	// Read exception list: varuint(nExc) then each (vecIdx, coordIdx, bits).
	nExc, k := binary.Uvarint(src[off:])
	if k <= 0 {
		return nil, false, 0, errors.New("qdf: bad exception count")
	}
	off += k
	// Bound to prevent hostile blocks from causing OOM.
	if nExc > dim*count {
		return nil, false, 0, errors.New("qdf: exception count exceeds dim*count")
	}
	for range nExc {
		vi, k := binary.Uvarint(src[off:])
		if k <= 0 {
			return nil, false, 0, errors.New("qdf: bad exception vecIdx")
		}
		off += k
		ci, k := binary.Uvarint(src[off:])
		if k <= 0 {
			return nil, false, 0, errors.New("qdf: bad exception coordIdx")
		}
		off += k
		if vi >= count || ci >= dim {
			return nil, false, 0, errors.New("qdf: exception index out of range")
		}
		if elemF32 {
			if off+4 > len(src) {
				return nil, false, 0, errors.New("qdf: short exception bits (f32)")
			}
			bits := binary.LittleEndian.Uint32(src[off:])
			off += 4
			vectors[vi][ci] = float64(math.Float32frombits(bits))
		} else {
			if off+8 > len(src) {
				return nil, false, 0, errors.New("qdf: short exception bits (f64)")
			}
			bits := binary.LittleEndian.Uint64(src[off:])
			off += 8
			vectors[vi][ci] = math.Float64frombits(bits)
		}
	}
	return vectors, elemF32, off, nil
}
