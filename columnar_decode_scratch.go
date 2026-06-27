package qdf

// columnar_decode_scratch.go — scratch-buffer-aware column decoders used
// exclusively by decodeColumnInto (the full-decode, non-query path).
//
// The decoder's decState carries colScratchI64/U64/F64/Bool: per-kind
// reusable buffers that survive across columns within a decode call and
// across decode calls (the Decoder is pooled). Columns are processed
// sequentially inside decodeColumnInto; each column's values are scattered
// into the output struct slice immediately after decoding, so the scratch
// buffer is safe to overwrite for the next column.
//
// These Into variants are NOT used by:
//   - decodeColumnVals (query/pushdown path) — that path retains colVals
//     across columns before scatter, so buffers must not alias each other.
//   - decodeNullableColumn — the present-count varies per column; sharing
//     a single scratch there is safe but the int64/uint64 buffers are
//     already narrower (only present values), so nullable columns get the
//     benefit too via decodeSliceInt64Into below.
//
// The helper growI64 / growU64 / growF64 / growBool grow the slice to n
// elements without allocating when the existing cap is sufficient.

import (
	"math"
	"unsafe"

	"github.com/alex60217101990/qdf/internal/bitpack"
	"github.com/alex60217101990/qdf/internal/endian"
)

// ---- grow helpers ----------------------------------------------------------

func growI64(dst *[]int64, n int) {
	if cap(*dst) >= n {
		*dst = (*dst)[:n]
	} else {
		*dst = make([]int64, n)
	}
}

func growU64(dst *[]uint64, n int) {
	if cap(*dst) >= n {
		*dst = (*dst)[:n]
	} else {
		*dst = make([]uint64, n)
	}
}

func growF64(dst *[]float64, n int) {
	if cap(*dst) >= n {
		*dst = (*dst)[:n]
	} else {
		*dst = make([]float64, n)
	}
}

func growBool(dst *[]bool, n int) {
	if cap(*dst) >= n {
		*dst = (*dst)[:n]
	} else {
		*dst = make([]bool, n)
	}
}

// ---- int64 Into variants ---------------------------------------------------

func (d *Decoder) readPackedInt64SliceInto(dst *[]int64) error {
	n, body, err := d.readPackedRawHeader(qpackKindInt64)
	if err != nil {
		return err
	}
	growI64(dst, n)
	if n == 0 {
		return nil
	}
	out := *dst
	if endian.NativeIsLittle {
		bs := unsafe.Slice((*byte)(unsafe.Pointer(&out[0])), n*8)
		copy(bs, body)
		return nil
	}
	for i := range n {
		out[i] = int64(readU64(body[i*8:]))
	}
	return nil
}

func (d *Decoder) readPackedForInt64SliceInto(dst *[]int64) error {
	bitsPer, _, mn, n, body, err := d.readPackedForHeader(qpackKindInt64)
	if err != nil {
		return err
	}
	growI64(dst, n)
	out := *dst
	if bitsPer == 0 {
		for i := range out {
			out[i] = mn
		}
		return nil
	}
	u := unsafe.Slice((*uint64)(unsafe.Pointer(unsafe.SliceData(out))), n)
	bitpack.Unpack(u, body, bitsPer)
	if mnU := uint64(mn); mnU != 0 {
		for i := range u {
			u[i] += mnU
		}
	}
	return nil
}

func (d *Decoder) readPackedDeltaForInt64SliceInto(dst *[]int64) error {
	bitsPer, _, first, minDelta, n, body, err := d.readPackedDeltaForHeader(qpackKindInt64)
	if err != nil {
		return err
	}
	growI64(dst, n)
	out := *dst
	if n == 0 {
		return nil
	}
	out[0] = first
	if n == 1 {
		return nil
	}
	if bitsPer == 0 {
		v := first
		for i := 1; i < n; i++ {
			v += minDelta
			out[i] = v
		}
		return nil
	}
	if cap(d.deltaScratch) < n-1 {
		d.deltaScratch = make([]uint64, n-1)
	}
	tmp := d.deltaScratch[:n-1]
	bitpack.Unpack(tmp, body, bitsPer)
	minU := uint64(minDelta)
	v := uint64(first)
	for i, dv := range tmp {
		v += dv + minU
		out[i+1] = int64(v)
	}
	return nil
}

func (d *Decoder) readPackedRLEInt64SliceInto(dst *[]int64) error {
	n, err := d.readPackedRLEHeader(qpackKindInt64)
	if err != nil {
		return err
	}
	growI64(dst, n)
	out := *dst
	idx := 0
	for idx < n {
		v64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		runLen, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		if runLen == 0 || uint64(idx)+runLen > uint64(n) {
			return ErrInvalidLength
		}
		v := zigzagDecode64(v64)
		end := idx + int(runLen)
		for i := idx; i < end; i++ {
			out[i] = v
		}
		idx = end
	}
	return nil
}

func (d *Decoder) readPackedDictInt64SliceInto(dst *[]int64) error {
	count, bitsPer, err := d.readPackedDictHeader(qpackKindInt64)
	if err != nil {
		return err
	}
	var table [qpackDictMaxDistinct]int64
	for i := range count {
		v, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		table[i] = zigzagDecode64(v)
	}
	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return ErrInvalidLength
	}
	d.i += nr
	if !d.colLenOK(n64) {
		return ErrInvalidLength
	}
	n := int(n64)
	growI64(dst, n)
	out := *dst
	if bitsPer == 0 {
		v := table[0]
		for i := range out {
			out[i] = v
		}
		return nil
	}
	rem := uint64(len(d.buf) - d.i)
	if n64 > rem*8/uint64(bitsPer) {
		return ErrShortBuffer
	}
	bodyBytes := (n*bitsPer + 7) >> 3
	body := d.buf[d.i : d.i+bodyBytes]
	d.i += bodyBytes
	idx := make([]uint64, n)
	bitpack.Unpack(idx, body, bitsPer)
	for i, k := range idx {
		if k >= uint64(count) {
			return ErrBadTag
		}
		out[i] = table[k]
	}
	return nil
}

func (d *Decoder) readPackedPForInt64SliceInto(dst *[]int64) error {
	if d.i+1 > len(d.buf) || d.buf[d.i] != qpackKindInt64 {
		return ErrTypeMismatch
	}
	d.i++
	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return ErrInvalidLength
	}
	d.i += nr
	if !d.colLenOK(n64) {
		return ErrInvalidLength
	}
	if d.i >= len(d.buf) {
		return ErrShortBuffer
	}
	b := int(d.buf[d.i])
	d.i++
	if b > qpackForMaxBits {
		return ErrBadTag
	}
	mz, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return ErrInvalidLength
	}
	d.i += nr
	mnU := uint64(zigzagDecode64(mz))
	rem := uint64(len(d.buf) - d.i)
	if b > 0 && n64 > rem*8/uint64(b) {
		return ErrShortBuffer
	}
	n := int(n64)
	bodyBytes := (n*b + 7) >> 3
	if d.i+bodyBytes > len(d.buf) {
		return ErrShortBuffer
	}
	growI64(dst, n)
	out := *dst
	u := unsafe.Slice((*uint64)(unsafe.Pointer(unsafe.SliceData(out))), n)
	if b == 0 {
		// All base values equal mnU. Must fully overwrite the reused scratch
		// buffer here — bitpack.Unpack is skipped, so nothing else writes u[:n].
		for k := range u {
			u[k] = mnU
		}
	} else {
		bitpack.Unpack(u, d.buf[d.i:d.i+bodyBytes], b)
		if mnU != 0 {
			for k := range u {
				u[k] += mnU
			}
		}
	}
	d.i += bodyBytes
	excN64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return ErrInvalidLength
	}
	d.i += nr
	if excN64 > n64 {
		return ErrInvalidLength
	}
	pos := 0
	for range excN64 {
		dp, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		delta, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		pos += int(dp)
		if pos < 0 || pos >= n {
			return ErrInvalidLength
		}
		out[pos] = int64(mnU + delta)
	}
	return nil
}

// decodeSliceInt64Into is the scratch-aware equivalent of decodeSliceInt64.
// On return *dst holds the decoded values; the backing array is reused
// across calls when cap is sufficient (steady-state: zero allocation).
func decodeSliceInt64Into(d *Decoder, dst *[]int64) error {
	t, err := d.peekTag()
	if err != nil {
		return err
	}
	switch t {
	case tagPackRaw:
		d.i++
		return d.readPackedInt64SliceInto(dst)
	case tagPackFor:
		d.i++
		return d.readPackedForInt64SliceInto(dst)
	case tagPackDeltaFor:
		d.i++
		return d.readPackedDeltaForInt64SliceInto(dst)
	case tagPackRLE:
		d.i++
		return d.readPackedRLEInt64SliceInto(dst)
	case tagPackDict:
		d.i++
		return d.readPackedDictInt64SliceInto(dst)
	case tagPackPFor:
		d.i++
		return d.readPackedPForInt64SliceInto(dst)
	case tagPackBlock:
		d.i++
		return d.readBlockInt64Into(dst)
	}
	// Fallback: element-by-element array decode.
	n, err := d.ReadArrayHeader()
	if err != nil {
		return err
	}
	if err := d.CheckLength(n, 1); err != nil {
		return err
	}
	growI64(dst, n)
	out := *dst
	for i := range n {
		v, err := d.ReadInt()
		if err != nil {
			return err
		}
		out[i] = v
	}
	return nil
}

// ---- uint64 Into variants --------------------------------------------------

func (d *Decoder) readPackedUint64SliceInto(dst *[]uint64) error {
	n, body, err := d.readPackedRawHeader(qpackKindUint64)
	if err != nil {
		return err
	}
	growU64(dst, n)
	if n == 0 {
		return nil
	}
	out := *dst
	if endian.NativeIsLittle {
		bs := unsafe.Slice((*byte)(unsafe.Pointer(&out[0])), n*8)
		copy(bs, body)
		return nil
	}
	for i := range n {
		out[i] = readU64(body[i*8:])
	}
	return nil
}

func (d *Decoder) readPackedForUint64SliceInto(dst *[]uint64) error {
	bitsPer, mn, _, n, body, err := d.readPackedForHeader(qpackKindUint64)
	if err != nil {
		return err
	}
	growU64(dst, n)
	out := *dst
	if bitsPer == 0 {
		for i := range out {
			out[i] = mn
		}
		return nil
	}
	bitpack.Unpack(out, body, bitsPer)
	if mn != 0 {
		for i := range out {
			out[i] += mn
		}
	}
	return nil
}

func (d *Decoder) readPackedDeltaForUint64SliceInto(dst *[]uint64) error {
	bitsPer, first, _, minDelta, n, body, err := d.readPackedDeltaForHeader(qpackKindUint64)
	if err != nil {
		return err
	}
	growU64(dst, n)
	out := *dst
	if n == 0 {
		return nil
	}
	out[0] = first
	if n == 1 {
		return nil
	}
	if bitsPer == 0 {
		step := uint64(minDelta)
		v := first
		for i := 1; i < n; i++ {
			v += step
			out[i] = v
		}
		return nil
	}
	if cap(d.deltaScratch) < n-1 {
		d.deltaScratch = make([]uint64, n-1)
	}
	tmp := d.deltaScratch[:n-1]
	bitpack.Unpack(tmp, body, bitsPer)
	minU := uint64(minDelta)
	v := first
	for i, dv := range tmp {
		v += dv + minU
		out[i+1] = v
	}
	return nil
}

func (d *Decoder) readPackedRLEUint64SliceInto(dst *[]uint64) error {
	n, err := d.readPackedRLEHeader(qpackKindUint64)
	if err != nil {
		return err
	}
	growU64(dst, n)
	out := *dst
	idx := 0
	for idx < n {
		v, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		runLen, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		if runLen == 0 || uint64(idx)+runLen > uint64(n) {
			return ErrInvalidLength
		}
		end := idx + int(runLen)
		for i := idx; i < end; i++ {
			out[i] = v
		}
		idx = end
	}
	return nil
}

func (d *Decoder) readPackedDictUint64SliceInto(dst *[]uint64) error {
	count, bitsPer, err := d.readPackedDictHeader(qpackKindUint64)
	if err != nil {
		return err
	}
	var table [qpackDictMaxDistinct]uint64
	for i := range count {
		v, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		table[i] = v
	}
	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return ErrInvalidLength
	}
	d.i += nr
	if !d.colLenOK(n64) {
		return ErrInvalidLength
	}
	n := int(n64)
	growU64(dst, n)
	out := *dst
	if bitsPer == 0 {
		v := table[0]
		for i := range out {
			out[i] = v
		}
		return nil
	}
	rem := uint64(len(d.buf) - d.i)
	if n64 > rem*8/uint64(bitsPer) {
		return ErrShortBuffer
	}
	bodyBytes := (n*bitsPer + 7) >> 3
	body := d.buf[d.i : d.i+bodyBytes]
	d.i += bodyBytes
	idx := make([]uint64, n)
	bitpack.Unpack(idx, body, bitsPer)
	for i, k := range idx {
		if k >= uint64(count) {
			return ErrBadTag
		}
		out[i] = table[k]
	}
	return nil
}

func (d *Decoder) readPackedPForUint64SliceInto(dst *[]uint64) error {
	if d.i+1 > len(d.buf) {
		return ErrShortBuffer
	}
	if d.buf[d.i] != qpackKindUint64 {
		return ErrTypeMismatch
	}
	d.i++
	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return ErrInvalidLength
	}
	d.i += nr
	if !d.colLenOK(n64) {
		return ErrInvalidLength
	}
	if d.i >= len(d.buf) {
		return ErrShortBuffer
	}
	b := int(d.buf[d.i])
	d.i++
	if b > qpackForMaxBits {
		return ErrBadTag
	}
	mn, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return ErrInvalidLength
	}
	d.i += nr
	rem := uint64(len(d.buf) - d.i)
	if b > 0 && n64 > rem*8/uint64(b) {
		return ErrShortBuffer
	}
	n := int(n64)
	bodyBytes := (n*b + 7) >> 3
	if d.i+bodyBytes > len(d.buf) {
		return ErrShortBuffer
	}
	body := d.buf[d.i : d.i+bodyBytes]
	d.i += bodyBytes
	growU64(dst, n)
	out := *dst
	if b == 0 {
		// All base values equal mn. Must fully overwrite the reused scratch
		// buffer here — bitpack.Unpack is skipped, so nothing else writes out[:n].
		for k := range out {
			out[k] = mn
		}
	} else {
		bitpack.Unpack(out, body, b)
		if mn != 0 {
			for k := range out {
				out[k] += mn
			}
		}
	}
	excN64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return ErrInvalidLength
	}
	d.i += nr
	if excN64 > n64 {
		return ErrInvalidLength
	}
	pos := 0
	for range excN64 {
		dp, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		delta, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		pos += int(dp)
		if pos < 0 || pos >= n {
			return ErrInvalidLength
		}
		out[pos] = mn + delta
	}
	return nil
}

// decodeSliceUint64Into is the scratch-aware equivalent of decodeSliceUint64.
func decodeSliceUint64Into(d *Decoder, dst *[]uint64) error {
	t, err := d.peekTag()
	if err != nil {
		return err
	}
	switch t {
	case tagPackRaw:
		d.i++
		return d.readPackedUint64SliceInto(dst)
	case tagPackFor:
		d.i++
		return d.readPackedForUint64SliceInto(dst)
	case tagPackDeltaFor:
		d.i++
		return d.readPackedDeltaForUint64SliceInto(dst)
	case tagPackRLE:
		d.i++
		return d.readPackedRLEUint64SliceInto(dst)
	case tagPackDict:
		d.i++
		return d.readPackedDictUint64SliceInto(dst)
	case tagPackPFor:
		d.i++
		return d.readPackedPForUint64SliceInto(dst)
	case tagPackBlock:
		d.i++
		return d.readBlockUint64Into(dst)
	}
	// Fallback: element-by-element array decode.
	n, err := d.ReadArrayHeader()
	if err != nil {
		return err
	}
	if err := d.CheckLength(n, 1); err != nil {
		return err
	}
	growU64(dst, n)
	out := *dst
	for i := range n {
		v, err := d.ReadUint()
		if err != nil {
			return err
		}
		out[i] = v
	}
	return nil
}

// ---- float64 Into variant --------------------------------------------------

// decodeSliceFloat64Into is the scratch-aware equivalent of decodeSliceFloat64.
// Gorilla and ALP are streaming decoders without an easy in-place form;
// they write into a freshly grown slice (still benefits if cap is sufficient).
func decodeSliceFloat64Into(d *Decoder, dst *[]float64) error {
	t, err := d.peekTag()
	if err != nil {
		return err
	}
	if t == tagColVecLossy {
		vecs, elemF32, used, err2 := readLossyVec(d.buf[d.i:])
		if err2 != nil {
			return err2
		}
		// A scalar float64 column is one vector (the whole column); reject a
		// multi-vector block rather than silently dropping vecs[1:].
		if len(vecs) != 1 {
			return ErrInvalidLength
		}
		if elemF32 {
			return ErrTypeMismatch
		}
		d.i += used
		*dst = vecs[0]
		return nil
	}
	if t == tagPackRaw {
		d.i++
		n, body, err2 := d.readPackedRawHeader(qpackKindFloat64)
		if err2 != nil {
			return err2
		}
		growF64(dst, n)
		if n == 0 {
			return nil
		}
		out := *dst
		if endian.NativeIsLittle {
			bs := unsafe.Slice((*byte)(unsafe.Pointer(&out[0])), n*8)
			copy(bs, body)
			return nil
		}
		for i := range n {
			out[i] = math.Float64frombits(readU64(body[i*8:]))
		}
		return nil
	}
	if t == tagPackGorilla {
		d.i++
		v, err2 := d.readPackedGorillaFloat64Slice()
		if err2 != nil {
			return err2
		}
		*dst = v
		return nil
	}
	if t == tagPackALP {
		d.i++
		v, err2 := d.readPackedALPFloat64Slice()
		if err2 != nil {
			return err2
		}
		*dst = v
		return nil
	}
	// Fallback: element-by-element array decode.
	n, err := d.ReadArrayHeader()
	if err != nil {
		return err
	}
	if err := d.CheckLength(n, 1); err != nil {
		return err
	}
	growF64(dst, n)
	out := *dst
	for i := range n {
		v, err := d.ReadFloat64()
		if err != nil {
			return err
		}
		out[i] = v
	}
	return nil
}

// ---- bool Into variant -----------------------------------------------------

// decodeSliceBoolInto is the scratch-aware equivalent of decodeSliceBool.
func decodeSliceBoolInto(d *Decoder, dst *[]bool) error {
	t, err := d.peekTag()
	if err != nil {
		return err
	}
	if t == tagPackBool {
		d.i++
		n64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		// Bound the claimed element count by the columnar row count (colMaxLen)
		// before allocating, mirroring every other columnar codec reader. The
		// body-byte bound below alone would still admit up to 8× the remaining
		// buffer in bool scratch (8 bits/byte); colLenOK ties it to the actual
		// row count so a bool column claiming n >> rows is rejected, not allocated.
		if !d.colLenOK(n64) {
			return ErrInvalidLength
		}
		if n64 > uint64(len(d.buf)-d.i)*8 {
			return ErrShortBuffer
		}
		n := int(n64)
		nBytes := (n + 7) >> 3
		if d.i+nBytes > len(d.buf) {
			return ErrShortBuffer
		}
		growBool(dst, n)
		out := *dst
		base := d.i
		i := 0
		for ; i+8 <= n; i += 8 {
			b := d.buf[base+(i>>3)]
			out[i] = b&(1<<0) != 0
			out[i+1] = b&(1<<1) != 0
			out[i+2] = b&(1<<2) != 0
			out[i+3] = b&(1<<3) != 0
			out[i+4] = b&(1<<4) != 0
			out[i+5] = b&(1<<5) != 0
			out[i+6] = b&(1<<6) != 0
			out[i+7] = b&(1<<7) != 0
		}
		for ; i < n; i++ {
			out[i] = d.buf[base+(i>>3)]&(1<<uint(i&7)) != 0
		}
		d.i += nBytes
		return nil
	}
	// Fallback: element-by-element array decode.
	n, err := d.ReadArrayHeader()
	if err != nil {
		return err
	}
	if err := d.CheckLength(n, 1); err != nil {
		return err
	}
	growBool(dst, n)
	out := *dst
	for i := range n {
		v, err := d.ReadBool()
		if err != nil {
			return err
		}
		out[i] = v
	}
	return nil
}
