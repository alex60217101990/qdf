package qdf

import (
	"bytes"
	"math/bits"
	"reflect"
	"time"
	"unsafe"
)

// Columnar column-level diff. For an equal-length diff of a pure-columnar
// []struct, group changes by column (sparse / arithmetic-delta / dense-whole
// per column) and emit a tagColSlicePatch only when it is strictly smaller than
// the positional patch (build-and-compare, never-larger). See
// docs/superpowers/specs/2026-06-16-columnar-column-diff-design.md.

// colDiffMode* are the per-column reconstruction modes in a tagColSlicePatch
// body.
const (
	colDiffModeSparse     byte = 0 // changed-row indices + changed cells
	colDiffModeDelta      byte = 1 // full per-row arithmetic delta (numeric/bool)
	colDiffModeDenseWhole byte = 2 // whole new column re-shipped
)

// colDiffSparseNum/Den: a column with <= n*(Num/Den) changed cells prefers
// sparse mode; otherwise delta/dense-whole. Benchstat-gated knob (Task 5).
const (
	colDiffSparseNum = 1
	colDiffSparseDen = 4
)

// diffColumnarEligible reports whether a slice's columnar plan can use the
// column-level diff: pure columnar (every field is a column, no residual) with
// fewer than 128 columns. The 128 cap keeps the nChangedCols count a single
// uvarint byte (nChanged <= len(cols) < 128 < 0x80), so the placeholder-patch
// in diffColumnar is always valid; a struct with >=128 columns (pathological)
// declines to the positional path.
func diffColumnarEligible(colPlan *columnarPlan) bool {
	return colPlan != nil && colPlan.residual == nil && len(colPlan.cols) < 128
}

// colKindColumnDiffSupported reports whether a column's kind has a column-diff
// encoder/decoder. Nullable columns are dense-whole only (pickColMode forces
// it).
//
// String/[]byte columns are supported via the wire-stateless dict/raw codecs
// (writeStringColumnStateless): those bodies ship their own table/indices and
// touch e.state only for scratch, so they can be built on the shared encoder
// state during diffColumnar's build-and-compare without polluting the intern
// state table. The stateful intern fallback (e.WriteString) and FSST are never
// used on this path.
//
// BLOCKED: nullable with a STRING base. encodeNullableColumn's string case calls
// the stateful e.writeStringColumn (which can fall through to e.WriteString),
// and nullable columns only have a dense-whole path (encodeOneColumn →
// encodeNullableColumn), so there is no stateless route for it yet. It stays
// gated out and falls back to the positional differ. All other nullable bases
// (int/uint/float/float32/bool/time) are stateless and fully supported.
func colKindColumnDiffSupported(col *colColumn) bool {
	if col.kind.isNullable() {
		// Dense-whole only (pickColMode forces it). A string base re-ships through
		// the stateful nullable string codec (encodeNullableColumn → WriteString),
		// for which there is no stateless route → gate it out.
		return col.kind.base() != colKindString
	}
	switch col.kind {
	case colKindInt, colKindUint, colKindFloat, colKindFloat32,
		colKindBool, colKindTime, colKindString:
		return true
	default:
		return false
	}
}

// Benchstat (Intel i7-9750H, OptBalanced, clustered 1000-row batch, ~1/3 of one
// int column changed; -benchmem -count=8): diff ~130 us/op, 22 KB, 23 allocs;
// apply ~58 us/op, 25 KB, 3 allocs. Wire: column body 1007 B vs positional body
// 3276 B (0.31x, a 3.25x shrink). The build-and-compare picker keeps the smaller
// body, so the diff cost (positional build + column build) is paid only on an
// eligible equal-length pure-columnar batch and the wire is never larger.
// Decision: KEEP unconditionally — no pre-gate needed at this scale.
//
// diffColumnar attempts to write a tagColSlicePatch body that is smaller than
// the positional body. It returns (true, nil) when it has placed the winning
// body (column OR positional) into enc.buf, (false, nil) when an unsupported
// column kind means the caller should fall back to the positional path, or
// (false, err) on a real error. n == oldLen == newLen is guaranteed by the
// caller. Exactly one body remains in enc.buf on a true return.
func diffColumnar(enc *Encoder, elem *typeDesc, plan *columnarPlan, stride uintptr,
	oldData, newData unsafe.Pointer, n, depth int) (bool, error) {
	if enc.state == nil {
		enc.state = newEncState()
	}
	st := enc.state

	// Detect which rows changed and, per column, which columns are affected. A
	// column whose kind is not yet column-diff-supported forces a decline of the
	// WHOLE slice only if it actually changed (an unchanged unsupported column
	// emits nothing, so its presence is harmless). Decide before touching enc.buf.
	st.deltaColBitmap = newChangedBitmap(st.deltaColBitmap, n)
	markChangedRows(st.deltaColBitmap, plan, stride, oldData, newData, n, elem.pod)

	// Never-larger trial: build a positional baseline (the size-comparison
	// alternative) and the column body, keep the smaller. Suspend interning for
	// the whole trial so the DISCARDED candidate cannot leak intern ids whose
	// wire definitions are thrown away with it — a later state-ref to such an id
	// dangles (ErrUnknownStateID). This is the same trap diffKeyedSlice guards
	// against; the column string body is already wire-stateless
	// (writeStringColumnStateless), but the positional baseline interned
	// normally and, when the column body won, left orphaned ids in enc.state.
	// Under suspension the only intern substate a body mutates is lastID, which
	// is captured after the positional build and restored if positional wins.
	prevSuspended := enc.stateSuspended
	enc.stateSuspended = true
	defer func() { enc.stateSuspended = prevSuspended }()

	// preTrialLastID is the intern lastID the decoder reaches when the column
	// body wins: that body is wire-stateless (no inline strings, no state-refs),
	// so a decoder reading it never touches lastID and stays at its pre-slice
	// value. The encoder must end there too, or the two diverge and a later
	// pair/repeat state-ref desyncs (the keyed picker keeps the same invariant).
	preTrialLastID := st.lastID

	// Build the positional body into enc.buf first (the comparison baseline).
	posStart := len(enc.buf)
	if err := diffElemsPositional(enc, elem, stride, oldData, n, newData, n, depth); err != nil {
		return false, err
	}
	posLen := len(enc.buf) - posStart
	posEndLastID := st.lastID

	body := st.deltaColBuf[:0]
	body = append(body, tagColSlicePatch)
	body = appendUvarint(body, uint64(n))
	colCountPos := len(body)
	body = appendUvarint(body, 0) // nChangedCols placeholder (single byte for <128 cols)
	nChanged := 0
	// Column codecs write through enc.buf; swap it to point at our scratch body,
	// restore enc.buf afterward.
	savedBuf := enc.buf
	for ci := range plan.cols {
		col := &plan.cols[ci]
		rows := colChangedRows(st.deltaColRows, st.deltaColBitmap, col, stride, oldData, newData, n)
		st.deltaColRows = rows
		if len(rows) == 0 {
			continue
		}
		if !colKindColumnDiffSupported(col) {
			// A changed column we cannot encode → abandon the column patch; the
			// positional body already in enc.buf handles the whole slice.
			enc.buf = savedBuf
			st.deltaColBuf = body[:0]
			st.lastID = posEndLastID // positional wins; track its kept bytes
			return true, nil
		}
		nChanged++
		body = appendUvarint(body, uint64(ci))
		mode := pickColMode(col, len(rows), n)
		body = append(body, mode)
		enc.buf = body
		var err error
		switch mode {
		case colDiffModeSparse:
			err = encodeSparseColumn(enc, col, stride, oldData, newData, rows)
		case colDiffModeDelta:
			err = encodeDeltaColumn(enc, col, stride, oldData, newData, n)
		case colDiffModeDenseWhole:
			if col.kind.base() == colKindString {
				// encodeOneColumn's string path can hit the stateful intern fallback
				// (writeStringColumn → WriteString) and pollute enc.state shared with
				// the discarded positional body. Gather the whole column and emit a
				// wire-stateless dict/raw body instead. Non-string kinds are stateless
				// and use encodeOneColumn unchanged.
				s := st.colScratchStr[:0]
				for i := range n {
					s = append(s, loadStringField(newData, stride, col, i))
				}
				st.colScratchStr = s
				enc.writeStringColumnStateless(s)
			} else {
				err = enc.encodeOneColumn(plan, newData, col, n)
			}
		default:
			err = ErrInvalidPatch
		}
		if err != nil {
			enc.buf = savedBuf
			st.deltaColBuf = body[:0]
			return false, err
		}
		body = enc.buf
	}
	enc.buf = savedBuf
	// Patch the nChangedCols count (always < 128 columns → single uvarint byte).
	body[colCountPos] = byte(nChanged)
	st.deltaColBuf = body

	// Never-larger: keep the column body only if it strictly beats positional.
	if len(body) < posLen {
		enc.buf = enc.buf[:posStart]
		enc.buf = append(enc.buf, body...)
		// Column body wins. It is wire-stateless, so a decoder leaves lastID at
		// its pre-slice value — restore the encoder to match (the positional
		// build may have moved lastID, e.g. to lruInvalidID on a changed string
		// column). Without this the encoder/decoder lastID + pair predictor
		// diverge after the slice.
		st.lastID = preTrialLastID
	} else {
		// Positional wins; restore lastID to its post-positional value so it
		// tracks the bytes the decoder will actually read.
		st.lastID = posEndLastID
	}
	// Either way exactly one body remains and the slice is handled.
	return true, nil
}

// newChangedBitmap returns a []uint64 with ceil(n/64) words, reusing dst's
// backing when large enough, cleared to zero.
func newChangedBitmap(dst []uint64, n int) []uint64 {
	words := (n + 63) >> 6
	if cap(dst) >= words {
		dst = dst[:words]
		for i := range dst {
			dst[i] = 0
		}
		return dst
	}
	return make([]uint64, words)
}

// markChangedRows sets bit i for every row whose bytes differ between old and
// new. Returns true if any row changed.
//
// For a pointer-free (POD) element a row changed iff its whole stride span
// differs, so one SIMD bytes.Equal per row replaces a per-column-per-cell sweep
// — the same block-memcmp fast path the positional differ uses (equalSliceEV).
// A padding-only difference can spuriously flag a row, but then every column's
// per-cell compare in colChangedRows finds no real change → the column is
// skipped → no wrong data, only a wasted attribution pass (the documented POD
// trade). A non-POD element (string/[]byte columns) must compare per column.
func markChangedRows(bm []uint64, plan *columnarPlan, stride uintptr,
	oldData, newData unsafe.Pointer, n int, pod bool) bool {
	any := false
	if pod {
		for i := range n {
			off := uintptr(i) * stride
			ob := unsafe.Slice((*byte)(unsafe.Add(oldData, off)), stride)
			nb := unsafe.Slice((*byte)(unsafe.Add(newData, off)), stride)
			if !bytes.Equal(ob, nb) {
				bm[i>>6] |= 1 << (uint(i) & 63)
				any = true
			}
		}
		return any
	}
	for ci := range plan.cols {
		col := &plan.cols[ci]
		for i := range n {
			if bm[i>>6]&(1<<(uint(i)&63)) != 0 {
				continue // already marked by an earlier column
			}
			if !colCellEqual(col, stride, oldData, newData, i) {
				bm[i>>6] |= 1 << (uint(i) & 63)
				any = true
			}
		}
	}
	return any
}

// colChangedRows appends, in ascending order, the indices where col differs
// between old and new, restricted to rows already flagged in bm (iterating bm
// via bits.TrailingZeros64 word-skip). dst is reused.
func colChangedRows(dst []int, bm []uint64, col *colColumn,
	stride uintptr, oldData, newData unsafe.Pointer, n int) []int {
	dst = dst[:0]
	for w := range bm {
		word := bm[w]
		base := w << 6
		for word != 0 {
			b := bits.TrailingZeros64(word)
			word &= word - 1
			i := base + b
			if i >= n {
				break
			}
			if !colCellEqual(col, stride, oldData, newData, i) {
				dst = append(dst, i)
			}
		}
	}
	return dst
}

// colCellEqual reports whether column col's cell i is byte/value-equal between
// old and new.
func colCellEqual(col *colColumn, stride uintptr, oldData, newData unsafe.Pointer, i int) bool {
	off := uintptr(i)*stride + col.offset
	op := unsafe.Add(oldData, off)
	np := unsafe.Add(newData, off)
	if col.kind.isNullable() {
		return nullableCellEqual(col, op, np)
	}
	switch col.kind {
	case colKindString:
		if col.isByte {
			return bytes.Equal(*(*[]byte)(op), *(*[]byte)(np))
		}
		return *(*string)(op) == *(*string)(np)
	case colKindTime:
		ot := (*time.Time)(op).UTC()
		nt := (*time.Time)(np).UTC()
		return ot.Unix() == nt.Unix() && ot.Nanosecond() == nt.Nanosecond()
	default:
		// pointer-free scalar: compare width bytes.
		w := col.width
		return bytes.Equal(unsafe.Slice((*byte)(op), w), unsafe.Slice((*byte)(np), w))
	}
}

// nullableCellEqual compares two *T cells: both nil, or both non-nil with equal
// pointees (string by value, otherwise width-bytes).
func nullableCellEqual(col *colColumn, op, np unsafe.Pointer) bool {
	pa := *(*unsafe.Pointer)(op)
	pb := *(*unsafe.Pointer)(np)
	if pa == nil || pb == nil {
		return pa == pb
	}
	switch col.kind.base() {
	case colKindString:
		return *(*string)(pa) == *(*string)(pb)
	default:
		return bytes.Equal(unsafe.Slice((*byte)(pa), col.width), unsafe.Slice((*byte)(pb), col.width))
	}
}

// pickColMode chooses the reconstruction mode for a column given how many of
// its n cells changed. Size heuristic only — the top-level build-and-compare is
// the never-larger guarantee.
func pickColMode(col *colColumn, nChanged, n int) byte {
	if col.kind.isNullable() {
		return colDiffModeDenseWhole // nullable: whole-column re-ship only
	}
	if nChanged*colDiffSparseDen <= n*colDiffSparseNum {
		return colDiffModeSparse
	}
	switch col.kind {
	case colKindInt, colKindUint, colKindBool:
		return colDiffModeDelta
	default: // float, float32, string, []byte, time
		return colDiffModeDenseWhole
	}
}

// writeStringColumnStateless emits a string column using only self-contained
// codecs (dict or raw), never the stateful intern fallback (e.WriteString) or
// FSST — safe to build on a shared encoder state during the column-diff
// build-and-compare. The dict/raw bodies ship their own table/indices and read
// e.state only for scratch, so the discarded loser body cannot pollute the
// intern state table and the kept body never emits a state-table ref the decoder
// did not build. Decode is automatic: decodeColumnInto/readStringColumn dispatch
// on the bulk tag we always emit.
func (e *Encoder) writeStringColumnStateless(strs []string) {
	if e.tryWriteStringColumnDict(strs) {
		return
	}
	e.writeStringColumnRawForced(strs)
}

// encodeSparseColumn writes nChanged, the gap-encoded ascending row indices, and
// the changed cells as a length-nChanged subcolumn via the column codec. Only
// int/uint/bool reach this via sparse mode for some kinds; float/float32/string/
// []byte/time can also be sparse (pickColMode returns sparse when changes are
// few), but never delta.
func encodeSparseColumn(enc *Encoder, col *colColumn,
	stride uintptr, oldData, newData unsafe.Pointer, rows []int) error {
	_ = oldData
	enc.buf = appendUvarint(enc.buf, uint64(len(rows)))
	prev := 0
	for _, r := range rows {
		enc.buf = appendUvarint(enc.buf, uint64(r-prev))
		prev = r
	}
	st := enc.state
	switch col.kind {
	case colKindInt:
		s := st.colScratchI64[:0]
		for _, r := range rows {
			s = append(s, loadIntCell(col, stride, newData, r))
		}
		st.colScratchI64 = s
		return encodeSliceInt64(enc, unsafe.Pointer(&s))
	case colKindUint:
		s := st.colScratchU64[:0]
		for _, r := range rows {
			s = append(s, loadUintCell(col, stride, newData, r))
		}
		st.colScratchU64 = s
		return encodeSliceUint64(enc, unsafe.Pointer(&s))
	case colKindFloat:
		s := st.colScratchF64[:0]
		for _, r := range rows {
			s = append(s, loadFloat64Field(newData, stride, col, r))
		}
		st.colScratchF64 = s
		// Lossless: a SCALAR float64 column must never become lossy under
		// OptLossyVec (which targets genuine []float64/[]float32 VECTOR fields).
		return encodeSliceFloat64Lossless(enc, s)
	case colKindFloat32:
		// Float32 cells are gathered as raw bits and emitted via the uint64 codec,
		// which never re-floats them — so the canonical -0.0/NaN normalization that
		// columnar.go applies must be repeated here under OptCanonical.
		canon := enc.opts.Has(OptCanonical)
		s := st.colScratchU64[:0]
		for _, r := range rows {
			bits := loadFloat32Bits(newData, stride, col, r)
			if canon {
				bits = canonicalizeFloat32Bits(bits)
			}
			s = append(s, bits)
		}
		st.colScratchU64 = s
		return encodeSliceUint64(enc, unsafe.Pointer(&s))
	case colKindBool:
		s := st.colScratchBool[:0]
		for _, r := range rows {
			p := unsafe.Add(newData, uintptr(r)*stride+col.offset)
			s = append(s, *(*bool)(p))
		}
		st.colScratchBool = s
		return encodeSliceBool(enc, unsafe.Pointer(&s))
	case colKindString:
		// Both string and []byte columns gather the changed cells into a []string
		// and emit a wire-stateless dict/raw body (never the intern fallback). A
		// []byte field is viewed as a string via loadStringField (col.isByte), so
		// the same codec handles both; the decoder copies back to owned bytes.
		s := st.colScratchStr[:0]
		for _, r := range rows {
			s = append(s, loadStringField(newData, stride, col, r))
		}
		st.colScratchStr = s
		enc.writeStringColumnStateless(s)
		return nil
	case colKindTime:
		sec := st.colScratchI64[:0]
		nsec := st.colScratchU64[:0]
		for _, r := range rows {
			p := unsafe.Add(newData, uintptr(r)*stride+col.offset)
			tt := (*time.Time)(p).UTC()
			sec = append(sec, tt.Unix())
			nsec = append(nsec, uint64(tt.Nanosecond()))
		}
		st.colScratchI64 = sec
		st.colScratchU64 = nsec
		if err := encodeSliceInt64(enc, unsafe.Pointer(&sec)); err != nil {
			return err
		}
		return encodeSliceUint64(enc, unsafe.Pointer(&nsec))
	}
	return ErrInvalidPatch
}

// encodeDeltaColumn writes the full-length per-row arithmetic delta column
// (new[i] - old[i]). Unchanged cells are 0 → delta/FOR/RLE crush them.
func encodeDeltaColumn(enc *Encoder, col *colColumn,
	stride uintptr, oldData, newData unsafe.Pointer, n int) error {
	st := enc.state
	switch col.kind {
	case colKindInt:
		// Gather new and old contiguously (each gather hoists the width switch
		// once), then a vectorizable subtract — no per-element width branch.
		cur := gatherColI64(st.colScratchI64, newData, stride, col, n)
		prev := gatherColI64(st.deltaColAuxI64, oldData, stride, col, n)
		st.colScratchI64, st.deltaColAuxI64 = cur, prev
		for i := range cur {
			cur[i] -= prev[i]
		}
		return encodeSliceInt64(enc, unsafe.Pointer(&cur))
	case colKindUint:
		cur := gatherColU64(st.colScratchU64, newData, stride, col, n)
		prev := gatherColU64(st.deltaColAuxU64, oldData, stride, col, n)
		st.colScratchU64, st.deltaColAuxU64 = cur, prev
		for i := range cur {
			cur[i] -= prev[i]
		}
		return encodeSliceUint64(enc, unsafe.Pointer(&cur))
	case colKindBool:
		// changed-flag column: true where the cell flipped. Apply XORs base where set.
		s := st.colScratchBool[:0]
		for i := range n {
			po := unsafe.Add(oldData, uintptr(i)*stride+col.offset)
			pn := unsafe.Add(newData, uintptr(i)*stride+col.offset)
			s = append(s, *(*bool)(po) != *(*bool)(pn))
		}
		st.colScratchBool = s
		return encodeSliceBool(enc, unsafe.Pointer(&s))
	}
	return ErrInvalidPatch
}

// loadIntCell / loadUintCell / storeIntCell / storeUintCell read or write a
// width-normalized scalar at column cell i. They reuse the loadI64At/loadU64At/
// storeI64At/storeU64At width helpers (nullable_col.go) so the width switch
// lives in one place.
func loadIntCell(col *colColumn, stride uintptr, data unsafe.Pointer, i int) int64 {
	return loadI64At(unsafe.Add(data, uintptr(i)*stride+col.offset), col.width)
}

func loadUintCell(col *colColumn, stride uintptr, data unsafe.Pointer, i int) uint64 {
	return loadU64At(unsafe.Add(data, uintptr(i)*stride+col.offset), col.width)
}

func storeIntCell(col *colColumn, stride uintptr, data unsafe.Pointer, i int, v int64) {
	storeI64At(unsafe.Add(data, uintptr(i)*stride+col.offset), col.width, v)
}

func storeUintCell(col *colColumn, stride uintptr, data unsafe.Pointer, i int, v uint64) {
	storeU64At(unsafe.Add(data, uintptr(i)*stride+col.offset), col.width, v)
}

// applyColSlice applies a tagColSlicePatch onto the base slice at baseP. It does
// not recurse (every column is a flat reconstruction), so it takes no depth.
func applyColSlice(dec *Decoder, td *typeDesc, baseP unsafe.Pointer) error {
	if dec.i >= len(dec.buf) || dec.buf[dec.i] != tagColSlicePatch {
		return ErrInvalidPatch
	}
	dec.i++
	// The columnar plan lives on the slice descriptor (td.colPlan), built by
	// descBuild; it is not mirrored onto the element desc.
	plan := td.colPlan
	if plan == nil {
		return ErrInvalidPatch
	}
	n64, k := readUvarint(dec.buf[dec.i:])
	if k <= 0 {
		return ErrInvalidPatch
	}
	dec.i += k
	bv := reflect.NewAt(td.rType, baseP).Elem()
	if int64(n64) != int64(bv.Len()) {
		return ErrInvalidPatch // equal-length invariant
	}
	n := int(n64)
	base := (*sliceHeader)(baseP).Data
	if dec.state == nil {
		dec.state = newDecState()
	}
	// Bound every column codec's element claim to n (matches decodeColumnInto and
	// the other columnar decode sites); applySparseColumn tightens it to nc for
	// the subcolumn. Defends against an allocation-amplification patch.
	savedColMax := dec.colMaxLen
	dec.colMaxLen = n
	defer func() { dec.colMaxLen = savedColMax }()
	nCols64, k2 := readUvarint(dec.buf[dec.i:])
	if k2 <= 0 || nCols64 > uint64(len(plan.cols)) {
		return ErrInvalidPatch
	}
	dec.i += k2
	prevCol := -1
	for range int(nCols64) {
		colIdx64, k3 := readUvarint(dec.buf[dec.i:])
		if k3 <= 0 || colIdx64 >= uint64(len(plan.cols)) {
			return ErrInvalidPatch
		}
		dec.i += k3
		if int(colIdx64) <= prevCol {
			return ErrInvalidPatch // columns must be ascending, no duplicates
		}
		prevCol = int(colIdx64)
		col := &plan.cols[colIdx64]
		if dec.i >= len(dec.buf) {
			return ErrInvalidPatch
		}
		mode := dec.buf[dec.i]
		dec.i++
		var err error
		switch mode {
		case colDiffModeSparse:
			err = applySparseColumn(dec, plan, col, base, n)
		case colDiffModeDelta:
			err = applyDeltaColumn(dec, plan, col, base, n)
		case colDiffModeDenseWhole:
			err = dec.decodeColumnInto(base, plan, col, n)
		default:
			err = ErrInvalidPatch
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// applySparseColumn reads nChanged, gap-decodes ascending row indices, decodes
// the length-nChanged subcolumn, and scatters into base at those rows.
//
// Gap encoding: gap[0] = idx0, gap[j] = idx[j] - idx[j-1] (>= 1 for j>0).
func applySparseColumn(dec *Decoder, plan *columnarPlan, col *colColumn, base unsafe.Pointer, n int) error {
	nc64, k := readUvarint(dec.buf[dec.i:])
	if k <= 0 || nc64 > uint64(n) {
		return ErrInvalidPatch
	}
	dec.i += k
	nc := int(nc64)
	st := dec.state
	rows := st.deltaColRows[:0]
	prev := -1
	for j := range nc {
		gap, kg := readUvarint(dec.buf[dec.i:])
		if kg <= 0 {
			return ErrInvalidPatch
		}
		dec.i += kg
		var idx int
		if j == 0 {
			idx = int(gap)
		} else {
			if gap == 0 {
				return ErrInvalidPatch // non-ascending
			}
			idx = prev + int(gap)
		}
		if idx < 0 || idx >= n {
			return ErrInvalidPatch
		}
		rows = append(rows, idx)
		prev = idx
	}
	st.deltaColRows = rows
	// The subcolumn has exactly nc elements; tighten the per-column bound so a
	// codec cannot claim more than the changed-cell count before the len check.
	savedColMax := dec.colMaxLen
	dec.colMaxLen = nc
	defer func() { dec.colMaxLen = savedColMax }()
	switch col.kind {
	case colKindInt:
		if err := decodeSliceInt64Into(dec, &st.colScratchI64); err != nil {
			return err
		}
		s := st.colScratchI64
		if len(s) != nc {
			return ErrTypeMismatch
		}
		for j, r := range rows {
			storeIntCell(col, plan.stride, base, r, s[j])
		}
	case colKindUint:
		if err := decodeSliceUint64Into(dec, &st.colScratchU64); err != nil {
			return err
		}
		s := st.colScratchU64
		if len(s) != nc {
			return ErrTypeMismatch
		}
		for j, r := range rows {
			storeUintCell(col, plan.stride, base, r, s[j])
		}
	case colKindFloat:
		if err := decodeSliceFloat64Into(dec, &st.colScratchF64); err != nil {
			return err
		}
		s := st.colScratchF64
		if len(s) != nc {
			return ErrTypeMismatch
		}
		for j, r := range rows {
			storeFloat64(base, plan.stride, col, r, s[j])
		}
	case colKindFloat32:
		if err := decodeSliceUint64Into(dec, &st.colScratchU64); err != nil {
			return err
		}
		s := st.colScratchU64
		if len(s) != nc {
			return ErrTypeMismatch
		}
		for j, r := range rows {
			storeFloat32Bits(base, plan.stride, col, r, s[j])
		}
	case colKindBool:
		if err := decodeSliceBoolInto(dec, &st.colScratchBool); err != nil {
			return err
		}
		s := st.colScratchBool
		if len(s) != nc {
			return ErrTypeMismatch
		}
		for j, r := range rows {
			*(*bool)(unsafe.Add(base, uintptr(r)*plan.stride+col.offset)) = s[j]
		}
	case colKindString:
		// Producer emitted a wire-stateless dict/raw body (writeStringColumnStateless)
		// for both string and []byte columns; readStringColumn dispatches on the
		// bulk tag. dec.colMaxLen is already tightened to nc above.
		strs, err := dec.readStringColumn(nc)
		if err != nil {
			return err
		}
		if len(strs) != nc {
			return ErrTypeMismatch
		}
		if col.isByte {
			for j, r := range rows {
				dp := unsafe.Add(base, uintptr(r)*plan.stride+col.offset)
				*(*[]byte)(dp) = append([]byte(nil), strs[j]...)
			}
		} else {
			for j, r := range rows {
				*(*string)(unsafe.Add(base, uintptr(r)*plan.stride+col.offset)) = strs[j]
			}
		}
	case colKindTime:
		if err := decodeSliceInt64Into(dec, &st.colScratchI64); err != nil {
			return err
		}
		sec := st.colScratchI64 // distinct scratch from nsec (U64) → no copy needed
		if len(sec) != nc {
			return ErrTypeMismatch
		}
		if err := decodeSliceUint64Into(dec, &st.colScratchU64); err != nil {
			return err
		}
		nsec := st.colScratchU64
		if len(nsec) != nc {
			return ErrTypeMismatch
		}
		if err := checkNsecColumn(nsec); err != nil {
			return err
		}
		for j, r := range rows {
			dp := unsafe.Add(base, uintptr(r)*plan.stride+col.offset)
			*(*time.Time)(dp) = time.Unix(sec[j], int64(nsec[j])).UTC()
		}
	default:
		return ErrInvalidPatch
	}
	return nil
}

// applyDeltaColumn reads the full-length delta column and adds it onto base.
func applyDeltaColumn(dec *Decoder, plan *columnarPlan, col *colColumn, base unsafe.Pointer, n int) error {
	st := dec.state
	switch col.kind {
	case colKindInt:
		if err := decodeSliceInt64Into(dec, &st.colScratchI64); err != nil {
			return err
		}
		s := st.colScratchI64
		if len(s) != n {
			return ErrTypeMismatch
		}
		for i := range n {
			storeIntCell(col, plan.stride, base, i, loadIntCell(col, plan.stride, base, i)+s[i])
		}
	case colKindUint:
		if err := decodeSliceUint64Into(dec, &st.colScratchU64); err != nil {
			return err
		}
		s := st.colScratchU64
		if len(s) != n {
			return ErrTypeMismatch
		}
		for i := range n {
			storeUintCell(col, plan.stride, base, i, loadUintCell(col, plan.stride, base, i)+s[i])
		}
	case colKindBool:
		// changed-flag column: flip base where the flag is set.
		if err := decodeSliceBoolInto(dec, &st.colScratchBool); err != nil {
			return err
		}
		s := st.colScratchBool
		if len(s) != n {
			return ErrTypeMismatch
		}
		for i := range n {
			if s[i] {
				p := unsafe.Add(base, uintptr(i)*plan.stride+col.offset)
				*(*bool)(p) = !*(*bool)(p)
			}
		}
	default:
		return ErrInvalidPatch
	}
	return nil
}
