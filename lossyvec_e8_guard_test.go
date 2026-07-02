package qdf

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/alex60217101990/qdf/internal/vecquant"
)

// TestReadLossyVecE8RejectsSmallPdim guards against a hostile/corrupt E8
// lossy-vec block whose padded dimension is < 8. ReconstructE8 walks nb = n/8
// blocks and silently leaves the tail zero when n is not a multiple of 8, so a
// pdim<8 block (dim in 1..7 → pdim in {1,2,4}) would decode to partially
// zero-filled vectors instead of erroring. The honest encoder only emits E8 at
// pdim>=16 (e8Eligible), so readLossyVec must fail closed on pdim<8.
func TestReadLossyVecE8RejectsSmallPdim(t *testing.T) {
	// Build a minimal E8 block with dim=4 (pdim=4) that passes every prior
	// bound check and reaches the E8 branch.
	var b []byte
	b = append(b, tagColVecLossy)
	b = append(b, vecquant.VariantE8<<1)                           // flags: elemF32=0, variant=E8
	b = binary.AppendUvarint(b, 4)                                 // dim=4 -> pdim=4 (< 8)
	b = binary.AppendUvarint(b, 1)                                 // count=1
	b = append(b, make([]byte, 8)...)                              // seed
	b = binary.LittleEndian.AppendUint64(b, math.Float64bits(1.0)) // delta
	b = binary.AppendUvarint(b, 0)                                 // coords len = 0

	_, _, _, err := readLossyVec(b)
	if err == nil {
		t.Fatal("readLossyVec accepted an E8 block with pdim<8 (should fail closed, not zero-fill)")
	}
}
