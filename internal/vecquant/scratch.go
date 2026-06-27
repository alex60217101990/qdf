package vecquant

// maxRetainedScratch caps reused scratch buffers; a one-off giant vector grows
// them, then Reset drops anything larger so memory is not pinned. Mirrors the
// encoder-side columnar scratch ceiling.
const maxRetainedScratch = 1 << 20

// Scratch holds the reusable buffers for one encode, shared across the verify-
// loop iterations and the scalar/E8 variants. It is owned by a single Encoder;
// never share a Scratch across goroutines.
type Scratch struct {
	flat    []float64 // rotated buffer, len count*pdim
	row     []float64 // streaming reconstruct row, len pdim
	qScalar []int32   // scalar quantizer codes
	qE8     []int32   // E8 quantizer codes

	zig          []byte // transient zigzag-varint staging (shared by both variants)
	ransDst      []byte // transient rANS encode staging (shared)
	coordsScalar []byte // scalar variant's final coord block (coexists with E8's)
	coordsE8     []byte // E8 variant's final coord block

	// Overflowed reports whether the chosen block's quantization saturated the
	// int32 coordinate range (set by EncodeWith). When true the caller must not
	// emit the lossy block — it would violate the fidelity budget — and should
	// keep the lossless encoding instead.
	Overflowed bool
}

// Reset prepares the scratch for the next encode, dropping any buffer that grew
// past the retention ceiling.
func (s *Scratch) Reset() {
	if cap(s.flat) > maxRetainedScratch {
		s.flat = nil
	}
	if cap(s.qScalar) > maxRetainedScratch {
		s.qScalar = nil
	}
	if cap(s.qE8) > maxRetainedScratch {
		s.qE8 = nil
	}
	if cap(s.row) > maxRetainedScratch {
		s.row = nil
	}
	if cap(s.zig) > maxRetainedScratch {
		s.zig = nil
	}
	if cap(s.ransDst) > maxRetainedScratch {
		s.ransDst = nil
	}
	if cap(s.coordsScalar) > maxRetainedScratch {
		s.coordsScalar = nil
	}
	if cap(s.coordsE8) > maxRetainedScratch {
		s.coordsE8 = nil
	}
}

func growF64(b []float64, n int) []float64 {
	if cap(b) < n {
		return make([]float64, n)
	}
	return b[:n]
}
