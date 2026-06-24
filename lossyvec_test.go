package qdf

import (
	"encoding/binary"
	"math"
	"math/rand"
	"testing"
)

func TestLossyVecWireRoundTrip(t *testing.T) {
	vecs := make([][]float64, 64)
	for i := range vecs {
		v := make([]float64, 256)
		for j := range v {
			v[j] = math.Sin(float64(i*256+j) * 0.01)
		}
		vecs[i] = v
	}
	enc := appendLossyVec(vecs, false, toBudget(MinCosine(0.999)))
	got, isF32, used, err := readLossyVec(enc)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if used != len(enc) {
		t.Fatalf("used %d != %d", used, len(enc))
	}
	if isF32 {
		t.Fatalf("elem type flipped")
	}
	if len(got) != len(vecs) || len(got[0]) != 256 {
		t.Fatalf("shape lost")
	}
	for i := range vecs {
		var dot, na, nb float64
		for j := range vecs[i] {
			dot += vecs[i][j] * got[i][j]
			na += vecs[i][j] * vecs[i][j]
			nb += got[i][j] * got[i][j]
		}
		if dot/(math.Sqrt(na)*math.Sqrt(nb)) < 0.999*0.999 {
			t.Fatalf("i=%d below cosine target", i)
		}
	}
}

func TestLossyVecSmallerThanRaw(t *testing.T) {
	vecs := make([][]float64, 128)
	for i := range vecs {
		v := make([]float64, 256)
		for j := range v {
			v[j] = float64((i + j) % 5) // low entropy
		}
		vecs[i] = v
	}
	enc := appendLossyVec(vecs, true, toBudget(MaxRelError(0.02)))
	raw := 128 * 256 * 4 // f32 raw bytes
	if len(enc) >= raw {
		t.Fatalf("lossy %d not smaller than raw %d", len(enc), raw)
	}
}

func TestLossyVecE8WireRoundTrip(t *testing.T) {
	// E8 wins over scalar in the high-fidelity regime (tight budget), where its
	// packing gain outweighs the coset-bit overhead. Use Gaussian vectors at a
	// tight MaxRelError so the codec selects the E8 variant, exercising the
	// coset-stream wire path.
	r := rand.New(rand.NewSource(7))
	const dim = 256
	vecs := make([][]float64, 128)
	for i := range vecs {
		v := make([]float64, dim)
		for j := range v {
			v[j] = r.NormFloat64()
		}
		vecs[i] = v
	}
	enc := appendLossyVec(vecs, false, toBudget(MaxRelError(0.02)))
	// Assert the E8 variant was actually selected so this test covers the E8
	// wire encode/decode, not the scalar fallback. flags is at index 1, with
	// the variant in bits 1-2.
	if v := (enc[1] >> 1) & 0x3; v != 1 {
		t.Fatalf("expected E8 variant (1) on the wire, got %d — corpus no longer selects E8", v)
	}
	got, isF32, used, err := readLossyVec(enc)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if used != len(enc) {
		t.Fatalf("used %d != %d", used, len(enc))
	}
	if isF32 {
		t.Fatalf("elem type flipped")
	}
	if len(got) != len(vecs) || len(got[0]) != dim {
		t.Fatalf("shape lost")
	}
	for i := range vecs {
		var dot, na, nb float64
		for j := range vecs[i] {
			dot += vecs[i][j] * got[i][j]
			na += vecs[i][j] * vecs[i][j]
			nb += got[i][j] * got[i][j]
		}
		if dot/(math.Sqrt(na)*math.Sqrt(nb)) < 0.999*0.999 {
			t.Fatalf("i=%d cosine below target", i)
		}
	}
}

// TestReadLossyVecRejectsHugeShape verifies that readLossyVec returns an error
// immediately (without attempting the allocation) when dim*count*8 exceeds
// maxColumnarBytes.  A hostile block with dim=70000, count=70000 would attempt
// a ~37 GB allocation; the bound check must reject it before bl.Decode runs.
func TestReadLossyVecRejectsHugeShape(t *testing.T) {
	const hugeDim = 70000
	const hugeCount = 70000

	// Build a minimal but syntactically valid 0xFD block:
	//   0xFD | flags=0 | varuint(dim) | varuint(count) |
	//   u64le seed | f64le delta | varuint(clen=1) | 1 coord byte
	var buf []byte
	buf = append(buf, tagColVecLossy) // 0xFD
	buf = append(buf, 0x00)           // flags = 0
	buf = binary.AppendUvarint(buf, hugeDim)
	buf = binary.AppendUvarint(buf, hugeCount)
	buf = binary.LittleEndian.AppendUint64(buf, 0)                     // seed
	buf = binary.LittleEndian.AppendUint64(buf, math.Float64bits(1.0)) // delta
	buf = binary.AppendUvarint(buf, 1)                                 // clen = 1
	buf = append(buf, 0x00)                                            // one coord byte

	_, _, _, err := readLossyVec(buf)
	if err == nil {
		t.Fatal("expected error for huge dim*count shape, got nil")
	}
}
