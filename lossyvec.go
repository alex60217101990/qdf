package qdf

import (
	"encoding/binary"
	"errors"
	"math"

	"github.com/alex60217101990/qdf/internal/vecquant"
)

// lossyVecMinElems is the smallest slice length that warrants the codec's
// fixed header overhead. Slices shorter than this go through the lossless path
// even when OptLossyVec is set.
const lossyVecMinElems = 32

// toF64 widens a []float32 or []float64 slice to []float64.
// Any other type returns nil.
func toF64(v any) []float64 {
	switch s := v.(type) {
	case []float64:
		return s
	case []float32:
		out := make([]float64, len(s))
		for i, x := range s {
			out[i] = float64(x)
		}
		return out
	}
	return nil
}

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

// appendLossyVec encodes vectors into dst using the lossy vector codec and
// returns the extended slice. Wire layout:
//
//	0xFD | flags(u8: bit0=elemF32) | varuint(dim) | varuint(count) |
//	u64le seed | f64le delta | varuint(len(coords)) | coords
func appendLossyVec(dst []byte, vectors [][]float64, elemF32 bool, b vecquant.Budget) []byte {
	bl := vecquant.Encode(vectors, b)
	dst = append(dst, tagColVecLossy)
	var flags byte
	if elemF32 {
		flags |= 1
	}
	dst = append(dst, flags)
	dst = binary.AppendUvarint(dst, uint64(bl.Dim))
	dst = binary.AppendUvarint(dst, uint64(bl.Count))
	dst = binary.LittleEndian.AppendUint64(dst, bl.Seed)
	dst = binary.LittleEndian.AppendUint64(dst, math.Float64bits(bl.Delta))
	dst = binary.AppendUvarint(dst, uint64(len(bl.Coords)))
	dst = append(dst, bl.Coords...)
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
	bl := vecquant.Block{
		Dim:    int(dim),
		Count:  int(count),
		Seed:   seed,
		Delta:  delta,
		Coords: src[off : off+int(clen)],
	}
	off += int(clen)
	vectors = bl.Decode()
	if vectors == nil && count != 0 {
		return nil, false, 0, errors.New("qdf: lossy-vec decode failed")
	}
	return vectors, elemF32, off, nil
}
