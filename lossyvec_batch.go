package qdf

import (
	"encoding/binary"
	"errors"
	"math"
	"reflect"
	"unsafe"

	"github.com/alex60217101990/qdf/internal/reflectutil"
	"github.com/alex60217101990/qdf/internal/vecquant"
)

// Batched lossy vector-column codec (tagVecBatchStruct, 0xFE).
//
// A []struct with a []float32/[]float64 vector field, encoded the naive way,
// emits one count=1 lossy block PER ROW — so the fixed block header AND the rANS
// frequency framing are paid per vector and never amortized (~+70% vs batched on
// a typical embedding corpus). This codec gathers each equal-length vector field
// across all N rows into ONE count=N block, encoded once, and decodes by
// scattering the N reconstructed vectors back into the rows. Non-batched fields
// (scalars, strings, varying-length / short vectors) stay row-major.

// encodeVectorBatchStruct batches the equal-length vector fields of a []struct
// (element descriptor td, n rows starting at base, stride bytes apart). It
// returns done=true after writing the full tagVecBatchStruct payload when at
// least one field was batched; done=false (nothing written) when no field is
// batchable, so the caller falls back to the normal []struct encoding.
func (e *Encoder) encodeVectorBatchStruct(td *typeDesc, base unsafe.Pointer, n int, stride uintptr) (done bool, err error) {
	nv := len(td.vecFields)
	if nv == 0 || nv > 8 { // batchedMask is one byte
		return false, nil
	}
	blocks := make([][]byte, nv)
	var mask, polarMask byte
	budget := toBudget(e.vecBudget)
	for vi := range td.vecFields {
		blk, polar, ok := e.buildVecColumnBlock(&td.vecFields[vi], base, n, stride, budget)
		if ok {
			blocks[vi] = blk
			mask |= 1 << uint(vi)
			if polar {
				polarMask |= 1 << uint(vi)
			}
		}
	}
	if mask == 0 {
		return false, nil // nothing batchable — let the caller emit row-major
	}

	e.writeHeader()
	e.buf = append(e.buf, tagVecBatchStruct)
	e.buf = appendUvarint(e.buf, uint64(n))
	e.buf = append(e.buf, byte(nv), mask, polarMask)
	// Total struct field count: lets Skip() (schema evolution) walk the per-row
	// non-batched fields without a typeDesc — it knows nf-popcount(mask) fields
	// follow each row and replays their intern/shape state via d.Skip().
	e.buf = appendUvarint(e.buf, uint64(len(td.fields)))
	for vi := range blocks {
		if mask&(1<<uint(vi)) != 0 {
			e.buf = append(e.buf, blocks[vi]...)
		}
	}

	// Per-row: every NON-batched field in declaration order, via its own codec.
	skip := batchedFieldSet(td, mask)
	for i := range n {
		rowPtr := unsafe.Add(base, uintptr(i)*stride)
		for fi := range td.fields {
			if skip[fi] {
				continue
			}
			if err := td.fields[fi].desc.encode(e, unsafe.Add(rowPtr, td.fields[fi].offset)); err != nil {
				return true, err
			}
		}
	}
	return true, nil
}

// polarMinCV is the smallest per-vector norm coefficient of variation
// (std/mean) at which the polar-split variant is worth attempting. Below it the
// norms are near-constant (unit-norm embeddings), so storing them separately
// only adds overhead — the never-worse compare would reject polar anyway, so we
// skip the second encode. Above it (varying-norm data: weights, raw vectors) a
// shared step over the raw vectors is driven by the largest norm and wastes bits
// on the small ones; normalizing first lets one tight step serve every direction.
const polarMinCV = 0.08

// buildVecColumnBlock gathers field vf across all n rows and returns its batched
// payload plus polar (true when the payload is the polar-split form: a per-vector
// norm stream followed by the lossy block of unit directions). ok=false when the
// field is not batchable (varying length, shorter than lossyVecMinElems, or no
// form beats raw).
func (e *Encoder) buildVecColumnBlock(vf *vecBatchField, base unsafe.Pointer, n int, stride uintptr, b vecquant.Budget) (payload []byte, polar bool, ok bool) {
	dim := -1
	for i := range n {
		fp := unsafe.Add(base, uintptr(i)*stride+vf.offset)
		var l int
		if vf.elemF32 {
			l = len(*(*[]float32)(fp))
		} else {
			l = len(*(*[]float64)(fp))
		}
		if i == 0 {
			dim = l
		} else if l != dim {
			return nil, false, false // varying length across rows: cannot batch
		}
	}
	if dim < lossyVecMinElems {
		return nil, false, false
	}
	// Bound the gather before n*dim is computed as int (cannot overflow on 32-bit;
	// also keeps the batch within the columnar byte budget). Over the cap, the
	// field stays row-major.
	if int64(n)*int64(dim) > int64(maxColumnarBytes/8) {
		return nil, false, false
	}

	// Gather n vectors into reused scratch (appendLossyVec zeroes NaN/Inf in
	// place, so it must never see the caller's backing arrays).
	if cap(e.vecBatchFlat) < n*dim {
		e.vecBatchFlat = make([]float64, n*dim)
	}
	flat := e.vecBatchFlat[:n*dim]
	if cap(e.vecBatchRows) < n {
		e.vecBatchRows = make([][]float64, n)
	}
	rows := e.vecBatchRows[:n]
	// Gather, and in the same pass compute each vector's L2 norm and whether any
	// coordinate is non-finite — BEFORE the plain encode, which zeroes NaN/Inf in
	// flat (so the norms must be read here, while flat still holds the originals).
	if cap(e.vecBatchNorms) < n {
		e.vecBatchNorms = make([]float64, n)
	}
	norms := e.vecBatchNorms[:n]
	finite := true
	var mean float64
	for i := range n {
		fp := unsafe.Add(base, uintptr(i)*stride+vf.offset)
		dst := flat[i*dim : (i+1)*dim]
		if vf.elemF32 {
			for j, x := range *(*[]float32)(fp) {
				dst[j] = float64(x)
			}
		} else {
			copy(dst, *(*[]float64)(fp))
		}
		rows[i] = dst
		var s float64
		for _, x := range dst {
			s += x * x
		}
		nrm := math.Sqrt(s)
		norms[i] = nrm
		if nrm <= 0 || math.IsNaN(nrm) || math.IsInf(nrm, 0) {
			finite = false
		}
		mean += nrm
	}
	plain, plainOK := appendLossyVec(rows, vf.elemF32, b, &e.vecScratch)
	if !plainOK {
		// Quantization saturated int32 (over-tight budget): keep the field
		// row-major rather than ship a budget-violating lossy block.
		return nil, false, false
	}

	elemSize := 8
	if vf.elemF32 {
		elemSize = 4
	}
	rawBytes := n * dim * elemSize
	best, bestPolar := plain, false

	// Polar candidate: only when every vector is finite and positive-norm (so the
	// plain encode did not zero any coordinate in flat) and the norm spread (CV)
	// is large enough to plausibly beat the per-vector norm overhead.
	if finite {
		mean /= float64(n)
		var varsum float64
		for _, nrm := range norms {
			d := nrm - mean
			varsum += d * d
		}
		cv := math.Sqrt(varsum/float64(n)) / (mean + 1e-30)
		if cv > polarMinCV {
			// Normalize flat in place (no NaN/Inf in this branch, so plain's encode
			// left it intact) → unit directions; encode those.
			for i := range n {
				inv := 1.0 / norms[i]
				seg := flat[i*dim : (i+1)*dim]
				for j := range seg {
					seg[j] *= inv
				}
			}
			dirsBlk, dirsOK := appendLossyVec(rows, vf.elemF32, b, &e.vecScratch)
			if dirsOK {
				polarPayload := appendNormStream(nil, norms)
				polarPayload = append(polarPayload, dirsBlk...)
				if len(polarPayload) < len(best) {
					best, bestPolar = polarPayload, true
				}
			}
		}
	}

	// Never-worse vs raw: if even the smaller form is not below raw, keep the
	// field row-major (its own per-vector never-worse picker applies there).
	if len(best) >= rawBytes {
		return nil, false, false
	}
	return best, bestPolar, true
}

// appendNormStream writes the polar-split norm stream: f64 log-min, f64 log-max,
// then one uint16 per vector quantizing log(norm) over [log-min, log-max]. The
// log domain gives uniform RELATIVE precision, so a wide norm range (1000x)
// still round-trips to ~1e-4. norms must all be finite and positive.
func appendNormStream(dst []byte, norms []float64) []byte {
	if len(norms) == 0 {
		return dst // never reached in practice (n >= columnarMinElems); guards ±Inf header
	}
	logMin, logMax := math.Inf(1), math.Inf(-1)
	for _, nrm := range norms {
		l := math.Log(nrm)
		logMin, logMax = math.Min(logMin, l), math.Max(logMax, l)
	}
	dst = binary.LittleEndian.AppendUint64(dst, math.Float64bits(logMin))
	dst = binary.LittleEndian.AppendUint64(dst, math.Float64bits(logMax))
	span := logMax - logMin
	if span <= 0 {
		span = 1 // all norms equal: every quantum maps to log-min
	}
	for _, nrm := range norms {
		q := math.RoundToEven((math.Log(nrm) - logMin) / span * 65535)
		dst = binary.LittleEndian.AppendUint16(dst, uint16(q))
	}
	return dst
}

// readNormStream reads n quantized norms written by appendNormStream and returns
// them plus the number of bytes consumed.
func readNormStream(src []byte, n int) (norms []float64, used int, err error) {
	if uint64(len(src)) < 16+2*uint64(n) { // uint64 intermediate: 2*n cannot wrap
		return nil, 0, ErrShortBuffer
	}
	logMin := math.Float64frombits(binary.LittleEndian.Uint64(src[0:]))
	logMax := math.Float64frombits(binary.LittleEndian.Uint64(src[8:]))
	if math.IsNaN(logMin) || math.IsNaN(logMax) || math.IsInf(logMin, 0) || math.IsInf(logMax, 0) {
		// Hostile header: NaN/Inf would make span NaN (span<=0 is false for NaN)
		// and yield math.Exp(NaN)=NaN for every decoded norm. The encoder only
		// writes finite logMin/logMax (appendNormStream requires finite norms).
		return nil, 0, ErrInvalidLength
	}
	span := logMax - logMin
	if span <= 0 {
		span = 1
	}
	// Finite header values can still overflow math.Exp: the largest exponent
	// produced below is logMin+span (q=65535), and the honest encoder never
	// writes a log-norm above log(MaxFloat64) (~709.78) because it logs finite
	// norms. Reject anything past that so a hostile header cannot decode to
	// +Inf norms (which would silently corrupt every denormalized vector).
	if logMin+span > math.Log(math.MaxFloat64) {
		return nil, 0, ErrInvalidLength
	}
	norms = make([]float64, n)
	off := 16
	for i := range n {
		q := binary.LittleEndian.Uint16(src[off:])
		off += 2
		norms[i] = math.Exp(logMin + float64(q)/65535*span)
	}
	return norms, off, nil
}

// batchedFieldSet returns a per-field bool: true when that struct field is a
// vector field batched under mask.
func batchedFieldSet(td *typeDesc, mask byte) []bool {
	skip := make([]bool, len(td.fields))
	for vi := range td.vecFields {
		if mask&(1<<uint(vi)) != 0 {
			skip[td.vecFields[vi].fieldIdx] = true
		}
	}
	return skip
}

// decodeVectorBatchStruct decodes a tagVecBatchStruct payload into a *[]T (t is
// the slice type, td its struct element descriptor). The 0xFE tag has been
// peeked but not consumed.
func (d *Decoder) decodeVectorBatchStruct(t reflect.Type, td *typeDesc, p unsafe.Pointer) error {
	if d.i >= len(d.buf) || d.buf[d.i] != tagVecBatchStruct {
		return ErrBadTag
	}
	d.i++
	n64, k := binary.Uvarint(d.buf[d.i:])
	if k <= 0 {
		return errors.New("qdf: bad vec-batch count")
	}
	d.i += k
	if n64 > uint64(maxColumnarElems) { // uint64->int maxInt clause (32-bit) + ceiling
		return ErrInvalidLength
	}
	n := int(n64)
	// Bound the result allocation by the input: MakeSlice(t, n) below allocates
	// n*stride, so CheckLength(n, 1) is not enough — a hostile count amplifies by
	// the struct stride. Mirror the columnar decoders' count + byte caps.
	if err := checkColumnarN(n); err != nil {
		return err
	}
	if err := checkColumnarBytes(n, t.Elem().Size()); err != nil {
		return err
	}
	if d.i+3 > len(d.buf) {
		return ErrShortBuffer
	}
	nv := int(d.buf[d.i])
	mask := d.buf[d.i+1]
	polarMask := d.buf[d.i+2]
	d.i += 3
	nf64, kf := binary.Uvarint(d.buf[d.i:])
	if kf <= 0 {
		return errors.New("qdf: bad vec-batch field count")
	}
	d.i += kf
	if nv != len(td.vecFields) {
		return errors.New("qdf: vec-batch shape mismatch")
	}
	if int(nf64) != len(td.fields) {
		return errors.New("qdf: vec-batch field count mismatch")
	}
	if polarMask&^mask != 0 { // polar bit on a non-batched field would desync d.i
		return errors.New("qdf: vec-batch polarMask not a subset of mask")
	}

	// Read each batched column's block (count=n vectors). A polar field carries a
	// norm stream before the directions block; rescale to recover the vectors.
	//
	// Each readLossyVec independently caps its own reconstruction at
	// maxColumnarBytes, but up to 8 batched columns would then allow ~8×256MB
	// from a tiny rANS-compressed input. Bound the AGGREGATE across all columns
	// so the batched path amplifies no more than a single columnar decode.
	var totalVecBytes uint64
	batched := make([][][]float64, nv)
	for vi := range nv {
		if mask&(1<<uint(vi)) == 0 {
			continue
		}
		var norms []float64
		if polarMask&(1<<uint(vi)) != 0 {
			ns, used, err := readNormStream(d.buf[d.i:], n)
			if err != nil {
				return err
			}
			d.i += used
			norms = ns
		}
		vecs, elemF32, used, err := readLossyVec(d.buf[d.i:])
		if err != nil {
			return err
		}
		d.i += used
		if len(vecs) != n {
			return errors.New("qdf: vec-batch block row count mismatch")
		}
		if elemF32 != td.vecFields[vi].elemF32 {
			return ErrTypeMismatch
		}
		if len(vecs) > 0 {
			totalVecBytes += uint64(len(vecs)) * uint64(len(vecs[0])) * 8
			if totalVecBytes > maxColumnarBytes {
				return ErrInvalidLength
			}
		}
		if norms != nil {
			for i := range vecs {
				scale := norms[i]
				for j := range vecs[i] {
					vecs[i][j] *= scale
				}
			}
		}
		batched[vi] = vecs
	}

	reflectutil.MakeSlice(t, n, p)
	base := reflectutil.SliceData(t, p)
	stride := t.Elem().Size()

	skip := make([]bool, len(td.fields))
	fieldVi := make([]int, len(td.fields))
	for i := range fieldVi {
		fieldVi[i] = -1
	}
	for vi := range nv {
		fieldVi[td.vecFields[vi].fieldIdx] = vi
		if mask&(1<<uint(vi)) != 0 {
			skip[td.vecFields[vi].fieldIdx] = true
		}
	}

	// Pre-alloc one flat []float32 slab per elemF32 vector field so the
	// scatter loop below slices into it instead of calling make([]float32,dim)
	// once per row. For n=1000, dim=512 this eliminates 1000 heap allocs of
	// 2 KB each, replacing them with a single 2 MB alloc per field.
	f32Flat := make([][]float32, nv)
	for vi := range nv {
		if batched[vi] == nil || !td.vecFields[vi].elemF32 {
			continue
		}
		if n > 0 {
			if dim := len(batched[vi][0]); dim > 0 {
				f32Flat[vi] = make([]float32, n*dim)
			}
		}
	}

	for i := range n {
		rowPtr := unsafe.Add(base, uintptr(i)*stride)
		for fi := range td.fields {
			fp := unsafe.Add(rowPtr, td.fields[fi].offset)
			if skip[fi] {
				vi := fieldVi[fi]
				vec := batched[vi][i]
				if td.vecFields[vi].elemF32 {
					dim := len(vec)
					var out []float32
					if flat := f32Flat[vi]; flat != nil {
						out = flat[i*dim : (i+1)*dim : (i+1)*dim]
					} else {
						out = make([]float32, dim)
					}
					for j, x := range vec {
						out[j] = float32(x)
					}
					*(*[]float32)(fp) = out
				} else {
					*(*[]float64)(fp) = vec
				}
				continue
			}
			if err := td.fields[fi].desc.decode(d, fp); err != nil {
				return err
			}
		}
	}
	return nil
}
