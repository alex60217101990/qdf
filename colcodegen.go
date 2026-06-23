package qdf

import (
	"math"
	"math/bits"
	"unsafe"
)

// This file exposes the no-reflect columnar primitives that qdfgen-generated
// code calls to transpose a []struct field into the tagColStruct wire frame.
// The byte layout is identical to encodeColumnar (columnar.go) so the reflect
// path and generated code interoperate. Generated code never uses reflection.

// WriteColStructHeader writes the columnar container header: tagColStruct,
// the row count n, and the column shape (field names + kind bytes). The shape
// is declared inline the first time this encoder sees it and reused by id
// afterwards, sharing encState's colStruct shape-id space with the reflect
// path. kinds[i] is the colKind byte for column i (see classifyColKind:
// int*->0, uint*->1, float64->2, bool->3, float32->6); it must match what the
// reflect encoder would emit for the same field so cross-decoding works.
//
// names and kinds back the encoder's shape-id cache by reference on the
// declaring call; the caller must NOT mutate or recycle them for a different
// shape afterward (generated code passes immutable package-level vars).
func (e *Encoder) WriteColStructHeader(n int, names []string, kinds []byte) {
	if e.state == nil {
		e.state = newEncState()
	}
	st := e.state
	e.buf = append(e.buf, tagColStruct)
	e.buf = appendUvarint(e.buf, uint64(n))
	// colKind is a uint8; reinterpret the kinds byte slice without copying.
	var ck []colKind
	if len(kinds) > 0 {
		ck = unsafe.Slice((*colKind)(unsafe.Pointer(&kinds[0])), len(kinds))
	}
	if id := st.colShapeFor(names, ck); id != 0 {
		e.buf = appendUvarint(e.buf, uint64(id))
		return
	}
	e.buf = appendUvarint(e.buf, 0)
	e.buf = appendUvarint(e.buf, uint64(len(names)))
	for i := range names {
		e.WriteString(names[i])
		e.buf = append(e.buf, kinds[i])
	}
	st.colShapeDeclare(names, ck)
}

// colState lazily initializes and returns the encoder's columnar state, which
// owns the reusable per-column transpose scratch.
func (e *Encoder) colState() *encState {
	if e.state == nil {
		e.state = newEncState()
	}
	return e.state
}

// The ScratchX getters hand generated code a reusable column buffer of length n,
// grown from the encoder's pooled columnar scratch (shared with the reflect
// path, which never runs concurrently on the same encoder). Generated code
// fills the buffer and passes it straight to the matching WriteXColumn, so a
// whole columnar []struct encodes with zero per-column allocation.

// ScratchInt returns a reusable []int64 of length n for a signed column.
func (e *Encoder) ScratchInt(n int) []int64 {
	st := e.colState()
	if cap(st.colScratchI64) < n {
		st.colScratchI64 = make([]int64, n)
	}
	st.colScratchI64 = st.colScratchI64[:n]
	return st.colScratchI64
}

// ScratchUint returns a reusable []uint64 of length n for an unsigned column.
func (e *Encoder) ScratchUint(n int) []uint64 {
	st := e.colState()
	if cap(st.colScratchU64) < n {
		st.colScratchU64 = make([]uint64, n)
	}
	st.colScratchU64 = st.colScratchU64[:n]
	return st.colScratchU64
}

// ScratchFloat64 returns a reusable []float64 of length n for a float64 column.
func (e *Encoder) ScratchFloat64(n int) []float64 {
	st := e.colState()
	if cap(st.colScratchF64) < n {
		st.colScratchF64 = make([]float64, n)
	}
	st.colScratchF64 = st.colScratchF64[:n]
	return st.colScratchF64
}

// ScratchFloat32 returns a reusable []float32 of length n for a float32 column.
func (e *Encoder) ScratchFloat32(n int) []float32 {
	st := e.colState()
	if cap(st.colScratchF32) < n {
		st.colScratchF32 = make([]float32, n)
	}
	st.colScratchF32 = st.colScratchF32[:n]
	return st.colScratchF32
}

// ScratchBool returns a reusable []bool of length n for a bool column.
func (e *Encoder) ScratchBool(n int) []bool {
	st := e.colState()
	if cap(st.colScratchBool) < n {
		st.colScratchBool = make([]bool, n)
	}
	st.colScratchBool = st.colScratchBool[:n]
	return st.colScratchBool
}

// WriteIntColumn encodes a column gathered from a signed-integer field. The
// adaptive picker chooses raw/FOR/Delta/RLE/Dict/PFOR per value range.
func (e *Encoder) WriteIntColumn(s []int64) error { return encodeSliceInt64(e, unsafe.Pointer(&s)) }

// WriteUintColumn encodes an unsigned-integer column.
func (e *Encoder) WriteUintColumn(s []uint64) error { return encodeSliceUint64(e, unsafe.Pointer(&s)) }

// WriteFloat64Column encodes a float64 column.
func (e *Encoder) WriteFloat64Column(s []float64) error {
	return encodeSliceFloat64(e, unsafe.Pointer(&s))
}

// WriteFloat32Column encodes a float32 column as its 32-bit patterns through
// the unsigned codec, bit-exact and matching the reflect colKindFloat32 path.
// The bit-conversion temp reuses colScratchU64 (free once any uint column it
// shares the buffer with has already been encoded), so no allocation.
func (e *Encoder) WriteFloat32Column(s []float32) error {
	st := e.colState()
	if cap(st.colScratchU64) < len(s) {
		st.colScratchU64 = make([]uint64, len(s))
	}
	u := st.colScratchU64[:len(s)]
	if e.opts.Has(OptCanonical) {
		// Mirror the reflect colKindFloat32 path (columnar.go): under OptCanonical
		// normalize -0.0 -> +0.0 and every NaN -> one quiet NaN so semantically
		// equal columns are byte-identical. Without this the codegen wire diverged
		// from reflect for -0.0/NaN columns under OptCanonical.
		for i, v := range s {
			u[i] = canonicalizeFloat32Bits(uint64(math.Float32bits(v)))
		}
	} else {
		for i, v := range s {
			u[i] = uint64(math.Float32bits(v))
		}
	}
	st.colScratchU64 = u
	return encodeSliceUint64(e, unsafe.Pointer(&u))
}

// WriteBoolColumn encodes a bool column.
func (e *Encoder) WriteBoolColumn(s []bool) error { return encodeSliceBool(e, unsafe.Pointer(&s)) }

// ScratchMask returns a zeroed reusable presence bitmap covering n rows
// ((n+7)/8 bytes). Generated code sets bit i for a present (non-nil) nullable
// column row, then passes it to WriteColNullMask.
func (e *Encoder) ScratchMask(n int) []byte {
	st := e.colState()
	mb := (n + 7) >> 3
	if cap(st.colMaskScratch) < mb {
		st.colMaskScratch = make([]byte, mb)
	}
	st.colMaskScratch = st.colMaskScratch[:mb]
	clear(st.colMaskScratch)
	return st.colMaskScratch
}

// WriteColNullMask appends a nullable column's presence bitmap (raw bytes, bit i
// set ⇒ row i present), written before the dense column of present values.
// Layout matches encodeNullableColumn so the reflect path can cross-decode.
func (e *Encoder) WriteColNullMask(mask []byte) { e.buf = append(e.buf, mask...) }

// ReadColNullMask reads a nullable column's presence bitmap for n rows and
// returns it (aliased into the input buffer, read-only) plus the present
// (set-bit) count. The dense column that follows has exactly that many values.
func (d *Decoder) ReadColNullMask(n int) ([]byte, int, error) {
	mb := (n + 7) >> 3
	if d.i+mb > len(d.buf) {
		return nil, 0, ErrShortBuffer
	}
	mask := d.buf[d.i : d.i+mb]
	d.i += mb
	present := 0
	for _, b := range mask {
		present += bits.OnesCount8(b)
	}
	// Reject set padding bits (positions [n%8, 8) of the last byte): a hostile
	// mask can set one while under-filling in-range bits, keeping present<=n yet
	// silently dropping trailing dense values in the scatter loop. Mirrors the
	// reflect sibling readNullableMask.
	if n&7 != 0 && mask[mb-1]>>uint(n&7) != 0 {
		return nil, 0, ErrInvalidLength
	}
	if present > n {
		return nil, 0, ErrInvalidLength
	}
	return mask, present, nil
}

// ScratchString returns a reusable []string of length n for a string column.
func (e *Encoder) ScratchString(n int) []string {
	st := e.colState()
	if cap(st.colScratchStr) < n {
		st.colScratchStr = make([]string, n)
	}
	st.colScratchStr = st.colScratchStr[:n]
	return st.colScratchStr
}

// WriteStringColumn encodes a string column, reusing the Balanced string-column
// picker (dictionary / FSST / raw-slab / plain), which is never larger than the
// per-value row-major encoding. Same wire as the reflect colKindString path.
func (e *Encoder) WriteStringColumn(s []string) { e.writeStringColumn(s) }

// WriteTimeColumn encodes a time.Time column as two sub-columns — sec ([]int64,
// Delta+FOR compresses monotonic series) then nsec ([]uint64) — matching the
// reflect colKindTime path. The caller gathers secs/nsec from the time fields
// (t.UTC().Unix() / t.Nanosecond()); reuse ScratchInt/ScratchUint for them.
func (e *Encoder) WriteTimeColumn(secs []int64, nsec []uint64) error {
	if err := encodeSliceInt64(e, unsafe.Pointer(&secs)); err != nil {
		return err
	}
	return encodeSliceUint64(e, unsafe.Pointer(&nsec))
}

// WriteHybridColStructHeader writes the hybrid columnar container header
// (tagHybridColStruct, row count, shape). The shape lists EVERY field in
// declaration order; residual (non-columnar) fields carry kind byte 0xFF
// (residualKind). Shares the hybrid shape-id space with the reflect path.
func (e *Encoder) WriteHybridColStructHeader(n int, names []string, kinds []byte) {
	st := e.colState()
	e.buf = append(e.buf, tagHybridColStruct)
	e.buf = appendUvarint(e.buf, uint64(n))
	var ck []colKind
	if len(kinds) > 0 {
		ck = unsafe.Slice((*colKind)(unsafe.Pointer(&kinds[0])), len(kinds))
	}
	if id := st.hybridShapeFor(names, ck); id != 0 {
		e.buf = appendUvarint(e.buf, uint64(id))
		return
	}
	e.buf = appendUvarint(e.buf, 0)
	e.buf = appendUvarint(e.buf, uint64(len(names)))
	for i := range names {
		e.WriteString(names[i])
		e.buf = append(e.buf, kinds[i])
	}
	st.hybridShapeDeclare(names, ck)
}

// PeekColStruct reports whether the next byte is a columnar container frame,
// without consuming it. Generated decode uses this to pick the columnar path
// vs the row-major fallback (a tiny slice the encoder kept row-major, or a
// reflect-produced row-major encoding of the same field).
func (d *Decoder) PeekColStruct() bool {
	return d.i < len(d.buf) && d.buf[d.i] == tagColStruct
}

// ReadColStructHeader consumes the columnar container header and returns the
// row count and column shape (names + kind bytes). It mirrors readColShape's
// non-index path. n is bounded by maxColumnarElems.
func (d *Decoder) ReadColStructHeader() (int, []string, []byte, error) {
	cs, err := d.readColShape(maxColumnarElems)
	if err != nil {
		return 0, nil, nil, err
	}
	d.colMaxLen = cs.n
	return cs.n, cs.sh.names, colKindsAsBytes(cs.sh.kinds), nil
}

// colKindsAsBytes reinterprets a cached []colKind (a uint8 alias) as []byte with
// no copy. The result is read-only and transient — valid only until the decoder
// reuses its shape cache; generated code consumes it immediately (or discards
// it), avoiding a per-decode allocation of the kinds slice.
func colKindsAsBytes(kinds []colKind) []byte {
	if len(kinds) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&kinds[0])), len(kinds))
}

// colStateDec lazily initializes and returns the decoder's columnar state,
// which owns the reusable per-column scatter scratch. ReadColStructHeader has
// already created it, but guard anyway for direct callers.
func (d *Decoder) colStateDec() *decState {
	if d.state == nil {
		d.state = newDecState()
	}
	return d.state
}

// The ReadXColumn readers decode one column of n values into the decoder's
// pooled columnar scratch and return it. Generated decode scatters the values
// into struct fields immediately, before the next column read reuses the same
// buffer — so a whole columnar []struct decodes with zero per-column transient
// allocation beyond the result slice itself.

// ReadIntColumn decodes one signed-integer column of n values.
func (d *Decoder) ReadIntColumn(n int) ([]int64, error) {
	st := d.colStateDec()
	s := st.colScratchI64[:0]
	if err := decodeSliceInt64Into(d, &s); err != nil {
		return nil, err
	}
	st.colScratchI64 = s
	if len(s) != n {
		return nil, ErrTypeMismatch
	}
	return s, nil
}

// ReadUintColumn decodes one unsigned-integer column of n values.
func (d *Decoder) ReadUintColumn(n int) ([]uint64, error) {
	st := d.colStateDec()
	s := st.colScratchU64[:0]
	if err := decodeSliceUint64Into(d, &s); err != nil {
		return nil, err
	}
	st.colScratchU64 = s
	if len(s) != n {
		return nil, ErrTypeMismatch
	}
	return s, nil
}

// ReadFloat64Column decodes one float64 column of n values.
func (d *Decoder) ReadFloat64Column(n int) ([]float64, error) {
	st := d.colStateDec()
	s := st.colScratchF64[:0]
	if err := decodeSliceFloat64Into(d, &s); err != nil {
		return nil, err
	}
	st.colScratchF64 = s
	if len(s) != n {
		return nil, ErrTypeMismatch
	}
	return s, nil
}

// ReadFloat32Column decodes one float32 column (32-bit patterns via the
// unsigned codec, mirroring WriteFloat32Column). The bits land in colScratchU64
// and are widened into colScratchF32.
func (d *Decoder) ReadFloat32Column(n int) ([]float32, error) {
	st := d.colStateDec()
	u := st.colScratchU64[:0]
	if err := decodeSliceUint64Into(d, &u); err != nil {
		return nil, err
	}
	st.colScratchU64 = u
	if len(u) != n {
		return nil, ErrTypeMismatch
	}
	if cap(st.colScratchF32) < n {
		st.colScratchF32 = make([]float32, n)
	}
	out := st.colScratchF32[:n]
	for i, v := range u {
		out[i] = math.Float32frombits(uint32(v))
	}
	st.colScratchF32 = out
	return out, nil
}

// ReadBoolColumn decodes one bool column of n values.
func (d *Decoder) ReadBoolColumn(n int) ([]bool, error) {
	st := d.colStateDec()
	s := st.colScratchBool[:0]
	if err := decodeSliceBoolInto(d, &s); err != nil {
		return nil, err
	}
	st.colScratchBool = s
	if len(s) != n {
		return nil, ErrTypeMismatch
	}
	return s, nil
}

// ReadStringColumn decodes one string column of n values, mirroring the reflect
// colKindString decode: a dictionary / FSST / raw-slab block (tag present) goes
// through the shared block reader; otherwise the plain per-value path reads via
// ReadString so state-ref / MTF / repeat encodings written by the picker's
// plain fallback resolve correctly (and share the decode intern cache). The
// returned slice is reused scratch on the plain path; scatter before the next
// column read.
func (d *Decoder) ReadStringColumn(n int) ([]string, error) {
	if n > 0 && d.i < len(d.buf) && isStringColumnBlockTag(d.buf[d.i]) {
		s, err := d.readStringColumn(n)
		if err != nil {
			return nil, err
		}
		if len(s) != n {
			return nil, ErrTypeMismatch
		}
		return s, nil
	}
	st := d.colStateDec()
	if cap(st.colScratchStr) < n {
		st.colScratchStr = make([]string, n)
	}
	out := st.colScratchStr[:n]
	for i := range n {
		s, err := d.ReadString()
		if err != nil {
			return nil, err
		}
		out[i] = s
	}
	st.colScratchStr = out
	return out, nil
}

// ReadTimeColumn decodes a time.Time column's two sub-columns and returns the
// sec / nsec slices; the caller reconstructs each value as
// time.Unix(sec[i], int64(nsec[i])).UTC(). nsec is validated in range.
func (d *Decoder) ReadTimeColumn(n int) ([]int64, []uint64, error) {
	st := d.colStateDec()
	sec := st.colScratchI64[:0]
	if err := decodeSliceInt64Into(d, &sec); err != nil {
		return nil, nil, err
	}
	st.colScratchI64 = sec
	if len(sec) != n {
		return nil, nil, ErrTypeMismatch
	}
	nsec := st.colScratchU64[:0]
	if err := decodeSliceUint64Into(d, &nsec); err != nil {
		return nil, nil, err
	}
	st.colScratchU64 = nsec
	if len(nsec) != n {
		return nil, nil, ErrTypeMismatch
	}
	if err := checkNsecColumn(nsec); err != nil {
		return nil, nil, err
	}
	return sec, nsec, nil
}

// PeekHybridColStruct reports whether the next byte is a hybrid columnar frame
// (some fields columnar, some residual row-major), without consuming it.
func (d *Decoder) PeekHybridColStruct() bool {
	return d.i < len(d.buf) && d.buf[d.i] == tagHybridColStruct
}

// ReadHybridColStructHeader consumes the hybrid columnar header and returns the
// row count and full shape (every field in declaration order; residual fields
// carry kind byte 0xFF). n is bounded by maxColumnarElems.
func (d *Decoder) ReadHybridColStructHeader() (int, []string, []byte, error) {
	n, sh, err := d.readHybridColShape(maxColumnarElems)
	if err != nil {
		return 0, nil, nil, err
	}
	d.colMaxLen = n
	return n, sh.names, colKindsAsBytes(sh.kinds), nil
}

// ClearColMaxLen resets the per-column length bound. Generated hybrid decode
// calls it between the eligible columns (bounded to n) and the residual block
// (whose nested slices/maps may legitimately exceed n), mirroring
// decodeHybridColumnar.
func (d *Decoder) ClearColMaxLen() { d.colMaxLen = 0 }

// stringColumnsBeneficial samples the given string columns and reports whether a
// columnar string column would beat plain per-value row-major encoding by the
// min-gain threshold. Reflection-free; it mirrors the reflect columnarProbe's
// colKindString branch byte-for-byte so the generated code makes the SAME
// columnar-vs-row-major decision as reflect.
//
// internAware selects the model the reflect probe uses for the element kind:
//   - false (a PURE string-only element): per-value / dict estimate, gain 10 —
//     identical to columnarProbe with internAware=false.
//   - true (a HYBRID string-only element, i.e. residual map/slice fields with no
//     numeric column): additionally credits Dense intern dedup in the row-major
//     baseline (a repeat costs 1 byte) AND alpha-packing on a restricted-alphabet
//     high-cardinality column, with the wider gain 30 — identical to
//     columnarProbe with internAware=true. Without this a hybrid hex-ID element
//     would stay row-major in codegen while reflect goes columnar + alpha.
func stringColumnsBeneficial(internAware bool, cols ...[]string) bool {
	if len(cols) == 0 || len(cols[0]) == 0 {
		return false
	}
	sample := min(len(cols[0]), columnarProbeSample)
	var colBytes, rowBytes int
	for _, strs := range cols {
		var seen [columnarProbeSample]string
		nseen := 0
		var tableBytes, perValue, sampleChars int
		prev := ""
		first := true
		for i := range sample {
			s := strs[i]
			fresh := true
			for j := 0; j < nseen; j++ {
				if seen[j] == s {
					fresh = false
					break
				}
			}
			if fresh && nseen < len(seen) {
				seen[nseen] = s
				nseen++
				tableBytes += 2 + len(s)
			}
			if !first && s == prev {
				perValue++
			} else {
				perValue += 2 + len(s)
			}
			if internAware && !fresh {
				rowBytes += 1
			} else {
				rowBytes += 2 + len(s)
			}
			sampleChars += len(s)
			prev = s
			first = false
		}
		dictBytes := tableBytes + (sample*bitsForDistinct(nseen)+7)/8
		best := min(perValue, dictBytes)
		// Defer the O(chars) per-byte alphabet scan behind the cheap alpha
		// preconditions (cardinality + average length), so low-card / short string
		// columns skip it entirely — byte-identical to the columnar decision (the
		// scan's alphaOK/alphaCount feed only the alphaEst below). Mirrors
		// columnarProbe.
		if internAware &&
			nseen*100 >= sample*alphaMinDistinctPct &&
			sampleChars >= sample*alphaProbeMinAvgLen {
			var alphaSeen [256]bool
			alphaCount := 0
			alphaOK := true
			for i := range sample {
				s := strs[i]
				for k := 0; k < len(s); k++ {
					if !alphaSeen[s[k]] {
						if alphaCount >= qpackStrAlphaMaxAlphabet {
							alphaOK = false
							break
						}
						alphaSeen[s[k]] = true
						alphaCount++
					}
				}
				if !alphaOK {
					break
				}
			}
			if alphaOK && alphaCount >= 2 {
				alphaEst := alphaCount + sample + (sampleChars*bitsForDistinct(alphaCount)+7)/8
				if alphaEst < best {
					best = alphaEst
				}
			}
		}
		colBytes += best
	}
	if rowBytes == 0 {
		return false
	}
	gain := columnarMinGainPct
	if internAware {
		gain = columnarMinGainPctInternAware
	}
	return colBytes*100 <= rowBytes*(100-gain)
}

// StringColumnsBeneficial gates a PURE string-only element's columnar encode in
// generated code (mirrors columnarProbe with internAware=false).
func StringColumnsBeneficial(cols ...[]string) bool { return stringColumnsBeneficial(false, cols...) }

// StringColumnsBeneficialHybrid gates a HYBRID (residual-bearing) string-only
// element's columnar encode in generated code (mirrors columnarProbe with
// internAware=true: intern-credited baseline + alpha-packing estimate + the
// wider gain), so codegen flips such an element into the columnar form exactly
// when reflect does.
func StringColumnsBeneficialHybrid(cols ...[]string) bool {
	return stringColumnsBeneficial(true, cols...)
}
