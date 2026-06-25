package qdf

import (
	"encoding/binary"
	"errors"
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
	var mask byte
	budget := toBudget(e.vecBudget)
	for vi := range td.vecFields {
		blk, ok := e.buildVecColumnBlock(&td.vecFields[vi], base, n, stride, budget)
		if ok {
			blocks[vi] = blk
			mask |= 1 << uint(vi)
		}
	}
	if mask == 0 {
		return false, nil // nothing batchable — let the caller emit row-major
	}

	e.writeHeader()
	e.buf = append(e.buf, tagVecBatchStruct)
	e.buf = appendUvarint(e.buf, uint64(n))
	e.buf = append(e.buf, byte(nv), mask)
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

// buildVecColumnBlock gathers field vf across all n rows and returns its batched
// lossy block, or ok=false when the field is not batchable (varying length,
// shorter than lossyVecMinElems, or the lossy block is not smaller than raw).
func (e *Encoder) buildVecColumnBlock(vf *vecBatchField, base unsafe.Pointer, n int, stride uintptr, b vecquant.Budget) ([]byte, bool) {
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
			return nil, false // varying length across rows: cannot batch
		}
	}
	if dim < lossyVecMinElems {
		return nil, false
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
	}
	blk := appendLossyVec(rows, vf.elemF32, b, &e.vecScratch)

	// Never-worse vs raw element bytes (the lossless floor's upper bound): if the
	// batched lossy block is not smaller than raw, skip batching this field so it
	// stays row-major (where its own never-worse picker applies per vector).
	elemSize := 8
	if vf.elemF32 {
		elemSize = 4
	}
	if len(blk) >= n*dim*elemSize {
		return nil, false
	}
	// blk aliases e.buf-independent scratch inside appendLossyVec's return; it is
	// a freshly appended slice, safe to retain until written.
	return blk, true
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
	n := int(n64)
	if err := d.CheckLength(n, 1); err != nil {
		return err
	}
	if d.i+2 > len(d.buf) {
		return ErrShortBuffer
	}
	nv := int(d.buf[d.i])
	mask := d.buf[d.i+1]
	d.i += 2
	if nv != len(td.vecFields) {
		return errors.New("qdf: vec-batch shape mismatch")
	}

	// Read each batched column's block (count=n vectors).
	batched := make([][][]float64, nv)
	for vi := range nv {
		if mask&(1<<uint(vi)) == 0 {
			continue
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

	for i := range n {
		rowPtr := unsafe.Add(base, uintptr(i)*stride)
		for fi := range td.fields {
			fp := unsafe.Add(rowPtr, td.fields[fi].offset)
			if skip[fi] {
				vec := batched[fieldVi[fi]][i]
				if td.vecFields[fieldVi[fi]].elemF32 {
					out := make([]float32, len(vec))
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
