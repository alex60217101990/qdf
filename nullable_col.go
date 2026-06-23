package qdf

import (
	"math"
	"math/bits"
	"reflect"
	"runtime"
	"time"
	"unsafe"
)

// Nullable (optional) columns for the columnar container. A `*T` struct
// field — where T is a scalar (int*/uint*/float*/bool) — would otherwise
// force the whole struct to the row-major fallback, losing the columnar
// codecs on every sibling column. Instead it is stored as a presence
// bitmap (1 bit per row, LSB-first) followed by a dense column of only the
// present values, encoded with T's normal slice codec. The nullable flag
// rides in the columnar shape's kind byte (colKindNullable), so there is no
// separate wire tag; the column slot is `ceil(M/8) mask bytes, <dense
// column>`.

// writeStringColumn emits a gathered string column either as a dictionary
// (when the never-worse gate accepts it) or as M per-value strings. Shared
// by the regular string column path; factored out so nullable columns could
// reuse it (nullable string columns are a follow-up).
func (e *Encoder) writeStringColumn(strs []string) {
	if e.tryWriteStringColumnDict(strs) {
		return
	}
	// FSST runs first (when enabled): on substring-sharing text (URLs, paths, log
	// lines) it beats positional packing, and it declines (never-larger) on random
	// restricted-alphabet IDs — no shared substrings — so those fall through to
	// alphabet-packing next. Alpha-packing sits before raw: on a restricted-
	// alphabet high-cardinality column (hex/base32/base64/decimal IDs) it stores
	// ceil(log2 |A|) bits/char instead of 8. Its own never-larger gate declines
	// (cheaply, after a one-pass alphabet scan that bails the moment |A| exceeds
	// 64) on everything else, so non-ID columns keep their existing form with no
	// wire/CPU regression.
	if e.fsst && e.tryWriteStringColumnFSST(strs) {
		return
	}
	if e.tryWriteStringColumnAlpha(strs) {
		return
	}
	if e.tryWriteStringColumnRaw(strs) {
		return
	}
	// Without OptDense (the codegen/Fast path) the per-value fallback below does
	// NOT intern: WriteString emits every occurrence inline, so the dict/raw
	// never-larger gates — which model the per-value cost as the Dense interned
	// form (distinct once + 1-byte state refs) — under-estimate it and decline,
	// leaving the column wire-bloated and decoding to n string allocations.
	// Mode-aware fallback: a single-distinct column collapses to one value
	// (tagColStrConst), any other column materializes in ONE slab allocation
	// (tagColStrRaw, wire-neutral + a tiny header). The reflect path (OptDense)
	// keeps the interning per-value form, which the gates model correctly — so
	// its wire is unchanged (const/raw-forced are codegen-only; the decoders
	// still read them for cross-path interop).
	//
	// An EMPTY column (len==0, e.g. an all-nil nullable string column's dense
	// part) must emit NOTHING, exactly as the plain loop below does and as the
	// reflect/Dense path does: writeStringColumnRawForced would otherwise lay
	// down a tagColStrRaw block that ReadStringColumn(0) skips (its block-tag
	// dispatch is guarded by n>0), desyncing the decode cursor.
	if !e.opts.Has(OptDense) && len(strs) > 0 {
		if e.tryWriteStringColumnConst(strs) {
			return
		}
		e.writeStringColumnRawForced(strs)
		return
	}
	for _, v := range strs {
		e.WriteString(v)
	}
}

func loadI64At(p unsafe.Pointer, width uintptr) int64 {
	switch width {
	case 1:
		return int64(*(*int8)(p))
	case 2:
		return int64(*(*int16)(p))
	case 4:
		return int64(*(*int32)(p))
	default:
		return *(*int64)(p)
	}
}

func loadU64At(p unsafe.Pointer, width uintptr) uint64 {
	switch width {
	case 1:
		return uint64(*(*uint8)(p))
	case 2:
		return uint64(*(*uint16)(p))
	case 4:
		return uint64(*(*uint32)(p))
	default:
		return *(*uint64)(p)
	}
}

// loadF64At reads a nullable float64 column value. colKindFloat is float64-only
// (a *float32 field classifies as colKindFloat32, which moves raw bits via
// loadU64At at width 4), so width is always 8 here and needs no switch.
func loadF64At(p unsafe.Pointer, _ uintptr) float64 {
	return *(*float64)(p)
}

func storeI64At(p unsafe.Pointer, width uintptr, v int64) {
	switch width {
	case 1:
		*(*int8)(p) = int8(v)
	case 2:
		*(*int16)(p) = int16(v)
	case 4:
		*(*int32)(p) = int32(v)
	default:
		*(*int64)(p) = v
	}
}

func storeU64At(p unsafe.Pointer, width uintptr, v uint64) {
	switch width {
	case 1:
		*(*uint8)(p) = uint8(v)
	case 2:
		*(*uint16)(p) = uint16(v)
	case 4:
		*(*uint32)(p) = uint32(v)
	default:
		*(*uint64)(p) = v
	}
}

// storeF64At writes a nullable float64 column value. See loadF64At: colKindFloat
// is float64-only, so width is always 8.
func storeF64At(p unsafe.Pointer, _ uintptr, v float64) {
	*(*float64)(p) = v
}

// encodeNullableColumn writes the presence bitmap and the dense present-only
// column for a `*T` field.
func (e *Encoder) encodeNullableColumn(base unsafe.Pointer, plan *columnarPlan, col *colColumn, n int) error {
	st := e.state
	maskBytes := (n + 7) >> 3
	var mask []byte
	if cap(st.colMaskScratch) >= maskBytes {
		mask = st.colMaskScratch[:maskBytes]
		clear(mask)
	} else {
		mask = make([]byte, maskBytes)
	}
	st.colMaskScratch = mask
	stride, off := plan.stride, col.offset

	switch col.kind.base() {
	case colKindInt:
		s := st.colScratchI64[:0]
		for i := range n {
			pp := *(*unsafe.Pointer)(unsafe.Add(base, uintptr(i)*stride+off))
			if pp != nil {
				mask[i>>3] |= 1 << uint(i&7)
				s = append(s, loadI64At(pp, col.width))
			}
		}
		st.colScratchI64 = s
		e.buf = append(e.buf, mask...)
		return encodeSliceInt64(e, unsafe.Pointer(&s))
	case colKindUint:
		s := st.colScratchU64[:0]
		for i := range n {
			pp := *(*unsafe.Pointer)(unsafe.Add(base, uintptr(i)*stride+off))
			if pp != nil {
				mask[i>>3] |= 1 << uint(i&7)
				s = append(s, loadU64At(pp, col.width))
			}
		}
		st.colScratchU64 = s
		e.buf = append(e.buf, mask...)
		return encodeSliceUint64(e, unsafe.Pointer(&s))
	case colKindFloat32:
		// float32: loadU64At(width==4) reads *(*uint32) — the raw f32 bits — so
		// the uint codec carries them losslessly (NaN payloads survive).
		s := st.colScratchU64[:0]
		canon := e.opts.Has(OptCanonical)
		for i := range n {
			pp := *(*unsafe.Pointer)(unsafe.Add(base, uintptr(i)*stride+off))
			if pp != nil {
				mask[i>>3] |= 1 << uint(i&7)
				bits := loadU64At(pp, col.width)
				if canon {
					bits = canonicalizeFloat32Bits(bits)
				}
				s = append(s, bits)
			}
		}
		st.colScratchU64 = s
		e.buf = append(e.buf, mask...)
		return encodeSliceUint64(e, unsafe.Pointer(&s))
	case colKindFloat:
		s := st.colScratchF64[:0]
		for i := range n {
			pp := *(*unsafe.Pointer)(unsafe.Add(base, uintptr(i)*stride+off))
			if pp != nil {
				mask[i>>3] |= 1 << uint(i&7)
				s = append(s, loadF64At(pp, col.width))
			}
		}
		st.colScratchF64 = s
		e.buf = append(e.buf, mask...)
		return encodeSliceFloat64(e, unsafe.Pointer(&s))
	case colKindBool:
		s := st.colScratchBool[:0]
		for i := range n {
			pp := *(*unsafe.Pointer)(unsafe.Add(base, uintptr(i)*stride+off))
			if pp != nil {
				mask[i>>3] |= 1 << uint(i&7)
				s = append(s, *(*bool)(pp))
			}
		}
		st.colScratchBool = s
		e.buf = append(e.buf, mask...)
		return encodeSliceBool(e, unsafe.Pointer(&s))
	case colKindString:
		s := st.colScratchStr[:0]
		for i := range n {
			pp := *(*unsafe.Pointer)(unsafe.Add(base, uintptr(i)*stride+off))
			if pp != nil {
				mask[i>>3] |= 1 << uint(i&7)
				s = append(s, *(*string)(pp))
			}
		}
		st.colScratchStr = s
		e.buf = append(e.buf, mask...)
		e.writeStringColumn(s)
		return nil
	case colKindTime:
		// Gather present *time.Time values into sec+nsec dense sub-columns,
		// mirroring the non-nullable colKindTime encoder in encodeColumnar.
		sec := st.colScratchI64[:0]
		nsec := st.colScratchU64[:0]
		for i := range n {
			pp := *(*unsafe.Pointer)(unsafe.Add(base, uintptr(i)*stride+off))
			if pp != nil {
				mask[i>>3] |= 1 << uint(i&7)
				t := (*time.Time)(pp).UTC()
				sec = append(sec, t.Unix())
				nsec = append(nsec, uint64(t.Nanosecond()))
			}
		}
		st.colScratchI64 = sec
		st.colScratchU64 = nsec
		e.buf = append(e.buf, mask...)
		if err := encodeSliceInt64(e, unsafe.Pointer(&sec)); err != nil {
			return err
		}
		return encodeSliceUint64(e, unsafe.Pointer(&nsec))
	}
	return ErrBadTag
}

// readNullableMask consumes the presence bitmap and returns it plus the
// present count (popcount). The caller decodes the dense column next.
func (d *Decoder) readNullableMask(n int) (mask []byte, present int, err error) {
	maskBytes := (n + 7) >> 3
	if d.i+maskBytes > len(d.buf) {
		return nil, 0, ErrShortBuffer
	}
	mask = d.buf[d.i : d.i+maskBytes]
	d.i += maskBytes
	for _, b := range mask {
		present += bits.OnesCount8(b)
	}
	// Reject a mangled bitmap. A valid encoder only sets bits for present rows
	// in [0,n), so the last byte's padding bits (positions [n%8, 8)) are always
	// zero. A hostile mask can set a padding bit while under-filling the in-range
	// bits, keeping the popcount <= n (so the present>n check alone passes) yet
	// making the scatter loop — which consumes only in-range set bits — drop the
	// trailing dense value(s) silently. Reject any set padding bit so the frame
	// errors instead. Mirrors the codegen sibling ReadColNullMask.
	if n&7 != 0 && mask[maskBytes-1]>>uint(n&7) != 0 {
		return nil, 0, ErrInvalidLength
	}
	if present > n {
		return nil, 0, ErrInvalidLength
	}
	return mask, present, nil
}

// decodeNullableColumn reads the mask + dense column and scatters the present
// values back into the `*T` field, allocating all present values in a single
// backing slice that the field pointers reference into.
//
// The int64/uint64/float64/bool scratch fields are reused here too: the dense
// column values are decoded into d.state.colScratch*, used only inside the
// inline set() call, then the scratch is eligible for reuse by the next column.
func (d *Decoder) decodeNullableColumn(base unsafe.Pointer, plan *columnarPlan, col *colColumn, n int) error {
	mask, present, err := d.readNullableMask(n)
	if err != nil {
		return err
	}
	elemSize := col.elemType.Size()
	backing := reflect.MakeSlice(reflect.SliceOf(col.elemType), present, present)
	dataPtr := backing.UnsafePointer()
	stride, off := plan.stride, col.offset
	st := d.state // always non-nil: readColShape initialises it before the loop
	k := 0
	set := func(store func(ea unsafe.Pointer, k int)) {
		for i := range n {
			fp := unsafe.Add(base, uintptr(i)*stride+off)
			if mask[i>>3]&(1<<uint(i&7)) != 0 {
				ea := unsafe.Add(dataPtr, uintptr(k)*elemSize)
				store(ea, k)
				*(*unsafe.Pointer)(fp) = ea
				k++
			} else {
				*(*unsafe.Pointer)(fp) = nil
			}
		}
	}
	switch col.kind.base() {
	case colKindInt:
		if err := decodeSliceInt64Into(d, &st.colScratchI64); err != nil {
			return err
		}
		s := st.colScratchI64
		if len(s) != present {
			return ErrTypeMismatch
		}
		set(func(ea unsafe.Pointer, k int) { storeI64At(ea, col.width, s[k]) })
	case colKindUint:
		if err := decodeSliceUint64Into(d, &st.colScratchU64); err != nil {
			return err
		}
		s := st.colScratchU64
		if len(s) != present {
			return ErrTypeMismatch
		}
		set(func(ea unsafe.Pointer, k int) { storeU64At(ea, col.width, s[k]) })
	case colKindFloat32:
		if err := decodeSliceUint64Into(d, &st.colScratchU64); err != nil {
			return err
		}
		s := st.colScratchU64
		if len(s) != present {
			return ErrTypeMismatch
		}
		set(func(ea unsafe.Pointer, k int) { storeU64At(ea, col.width, s[k]) })
	case colKindFloat:
		if err := decodeSliceFloat64Into(d, &st.colScratchF64); err != nil {
			return err
		}
		s := st.colScratchF64
		if len(s) != present {
			return ErrTypeMismatch
		}
		set(func(ea unsafe.Pointer, k int) { storeF64At(ea, col.width, s[k]) })
	case colKindBool:
		if err := decodeSliceBoolInto(d, &st.colScratchBool); err != nil {
			return err
		}
		s := st.colScratchBool
		if len(s) != present {
			return ErrTypeMismatch
		}
		set(func(ea unsafe.Pointer, k int) { *(*bool)(ea) = s[k] })
	case colKindString:
		strs, err := d.readStringColumn(present)
		if err != nil {
			return err
		}
		k := 0
		for i := range n {
			fp := unsafe.Add(base, uintptr(i)*stride+off)
			if mask[i>>3]&(1<<uint(i&7)) != 0 {
				*(*unsafe.Pointer)(fp) = unsafe.Pointer(&strs[k])
				k++
			} else {
				*(*unsafe.Pointer)(fp) = nil
			}
		}
		runtime.KeepAlive(strs)
		return nil
	case colKindTime:
		// Decode two dense sub-columns (sec []int64, nsec []uint64) for the
		// present count, reconstruct time.Time values, scatter into *time.Time
		// fields using the shared backing slice.
		if err := decodeSliceInt64Into(d, &st.colScratchI64); err != nil {
			return err
		}
		sec := st.colScratchI64
		if len(sec) != present {
			return ErrTypeMismatch
		}
		if err := decodeSliceUint64Into(d, &st.colScratchU64); err != nil {
			return err
		}
		nsec := st.colScratchU64
		if len(nsec) != present {
			return ErrTypeMismatch
		}
		if err := checkNsecColumn(nsec); err != nil {
			return err
		}
		set(func(ea unsafe.Pointer, k int) {
			*(*time.Time)(ea) = time.Unix(sec[k], int64(nsec[k])).UTC()
		})
	default:
		return ErrBadTag
	}
	runtime.KeepAlive(backing)
	return nil
}

// decodeNullableColumnVals decodes an optional (*T) column like
// decodeNullableColumn (presence mask via readNullableMask, then a dense
// present-only base-kind column), retaining the values expanded to length n
// with cv.present marking the non-nil rows, for later filter + compact.
func (d *Decoder) decodeNullableColumnVals(kind colKind, n int) (colVals, error) {
	var cv colVals
	cv.kind = kind
	mask, present, err := d.readNullableMask(n)
	if err != nil {
		return cv, err
	}
	pres := newBitset(n)
	for i := range n {
		if mask[i>>3]&(1<<uint(i&7)) != 0 {
			setBit(pres, i)
		}
	}
	cv.present = pres

	switch kind.base() {
	case colKindInt:
		var s []int64
		if err := decodeSliceInt64(d, unsafe.Pointer(&s)); err != nil {
			return cv, err
		}
		if len(s) != present {
			return cv, ErrTypeMismatch
		}
		full := make([]int64, n)
		k := 0
		for i := range n {
			if getBit(pres, i) {
				full[i] = s[k]
				k++
			}
		}
		cv.i64 = full
	case colKindUint:
		var s []uint64
		if err := decodeSliceUint64(d, unsafe.Pointer(&s)); err != nil {
			return cv, err
		}
		if len(s) != present {
			return cv, ErrTypeMismatch
		}
		full := make([]uint64, n)
		k := 0
		for i := range n {
			if getBit(pres, i) {
				full[i] = s[k]
				k++
			}
		}
		cv.u64 = full
	case colKindFloat32:
		var s []uint64
		if err := decodeSliceUint64(d, unsafe.Pointer(&s)); err != nil {
			return cv, err
		}
		if len(s) != present {
			return cv, ErrTypeMismatch
		}
		full := make([]uint64, n)
		k := 0
		for i := range n {
			if getBit(pres, i) {
				full[i] = s[k]
				k++
			}
		}
		cv.u64 = full
	case colKindFloat:
		var s []float64
		if err := decodeSliceFloat64(d, unsafe.Pointer(&s)); err != nil {
			return cv, err
		}
		if len(s) != present {
			return cv, ErrTypeMismatch
		}
		full := make([]float64, n)
		k := 0
		for i := range n {
			if getBit(pres, i) {
				full[i] = s[k]
				k++
			}
		}
		cv.f64 = full
	case colKindBool:
		var s []bool
		if err := decodeSliceBool(d, unsafe.Pointer(&s)); err != nil {
			return cv, err
		}
		if len(s) != present {
			return cv, ErrTypeMismatch
		}
		full := make([]bool, n)
		k := 0
		for i := range n {
			if getBit(pres, i) {
				full[i] = s[k]
				k++
			}
		}
		cv.b = full
	case colKindString:
		strs, err := d.readStringColumn(present)
		if err != nil {
			return cv, err
		}
		if len(strs) != present {
			return cv, ErrTypeMismatch
		}
		full := make([]string, n)
		k := 0
		for i := range n {
			if getBit(pres, i) {
				full[i] = strs[k]
				k++
			}
		}
		cv.s = full
	case colKindTime:
		var sec []int64
		if err := decodeSliceInt64(d, unsafe.Pointer(&sec)); err != nil {
			return cv, err
		}
		if len(sec) != present {
			return cv, ErrTypeMismatch
		}
		var nsec []uint64
		if err := decodeSliceUint64(d, unsafe.Pointer(&nsec)); err != nil {
			return cv, err
		}
		if len(nsec) != present {
			return cv, ErrTypeMismatch
		}
		if err := checkNsecColumn(nsec); err != nil {
			return cv, err
		}
		full := make([]time.Time, n)
		k := 0
		for i := range n {
			if getBit(pres, i) {
				full[i] = time.Unix(sec[k], int64(nsec[k])).UTC()
				k++
			}
		}
		cv.ts = full
	default:
		return cv, ErrBadTag
	}
	return cv, nil
}

// scatterNullableRowInto writes cv's optional value at source row src into the
// *T field at compacted row dst. A present value is placed in the caller's
// backing slab at slab+dst*elemSize and the field points into it; an absent
// value leaves the pointer nil. One slab per column replaces the former per-row
// reflect.New (a wide nullable projection's 8k+ allocs collapse to ~one per
// column). The caller MakeSlices the slab and keeps it alive (runtime.KeepAlive)
// until every field pointer has been written.
func (cv *colVals) scatterNullableRowInto(base unsafe.Pointer, plan *columnarPlan, col *colColumn, src, dst int, slab unsafe.Pointer, elemSize uintptr) {
	fp := unsafe.Add(base, uintptr(dst)*plan.stride+col.offset)
	if !getBit(cv.present, src) {
		*(*unsafe.Pointer)(fp) = nil
		return
	}
	ea := unsafe.Add(slab, uintptr(dst)*elemSize) // &slab[dst]
	switch cv.kind.base() {
	case colKindInt:
		storeI64At(ea, col.width, cv.i64[src])
	case colKindUint:
		storeU64At(ea, col.width, cv.u64[src])
	case colKindFloat32:
		storeU64At(ea, col.width, cv.u64[src]) // width==4 ⇒ writes *(*uint32), the f32 bits
	case colKindFloat:
		storeF64At(ea, col.width, cv.f64[src])
	case colKindBool:
		*(*bool)(ea) = cv.b[src]
	case colKindString:
		*(*string)(ea) = cv.s[src]
	case colKindTime:
		*(*time.Time)(ea) = cv.ts[src]
	}
	*(*unsafe.Pointer)(fp) = ea
}

// decodeNullableColumnAny reads the mask + dense column and returns one boxed
// value per row (nil for absent), for the map[string]any decode path.
func (d *Decoder) decodeNullableColumnAny(kind colKind, n int) ([]any, error) {
	mask, present, err := d.readNullableMask(n)
	if err != nil {
		return nil, err
	}
	out := make([]any, n)
	k := 0
	scatter := func(box func(i, k int)) {
		for i := range n {
			if mask[i>>3]&(1<<uint(i&7)) != 0 {
				box(i, k)
				k++
			}
		}
	}
	switch kind.base() {
	case colKindInt:
		var s []int64
		if err := decodeSliceInt64(d, unsafe.Pointer(&s)); err != nil {
			return nil, err
		}
		if len(s) != present {
			return nil, ErrTypeMismatch
		}
		scatter(func(i, k int) { out[i] = s[k] })
	case colKindUint:
		var s []uint64
		if err := decodeSliceUint64(d, unsafe.Pointer(&s)); err != nil {
			return nil, err
		}
		if len(s) != present {
			return nil, ErrTypeMismatch
		}
		scatter(func(i, k int) { out[i] = s[k] })
	case colKindFloat32:
		var s []uint64
		if err := decodeSliceUint64(d, unsafe.Pointer(&s)); err != nil {
			return nil, err
		}
		if len(s) != present {
			return nil, ErrTypeMismatch
		}
		scatter(func(i, k int) { out[i] = math.Float32frombits(uint32(s[k])) })
	case colKindFloat:
		var s []float64
		if err := decodeSliceFloat64(d, unsafe.Pointer(&s)); err != nil {
			return nil, err
		}
		if len(s) != present {
			return nil, ErrTypeMismatch
		}
		scatter(func(i, k int) { out[i] = s[k] })
	case colKindBool:
		var s []bool
		if err := decodeSliceBool(d, unsafe.Pointer(&s)); err != nil {
			return nil, err
		}
		if len(s) != present {
			return nil, ErrTypeMismatch
		}
		scatter(func(i, k int) { out[i] = s[k] })
	case colKindString:
		strs, err := d.readStringColumn(present)
		if err != nil {
			return nil, err
		}
		scatter(func(i, k int) { out[i] = strs[k] })
	case colKindTime:
		var sec []int64
		if err := decodeSliceInt64(d, unsafe.Pointer(&sec)); err != nil {
			return nil, err
		}
		if len(sec) != present {
			return nil, ErrTypeMismatch
		}
		var nsec []uint64
		if err := decodeSliceUint64(d, unsafe.Pointer(&nsec)); err != nil {
			return nil, err
		}
		if len(nsec) != present {
			return nil, ErrTypeMismatch
		}
		if err := checkNsecColumn(nsec); err != nil {
			return nil, err
		}
		scatter(func(i, k int) { out[i] = time.Unix(sec[k], int64(nsec[k])).UTC() })
	default:
		return nil, ErrBadTag
	}
	return out, nil
}
