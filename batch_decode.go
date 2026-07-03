package qdf

import (
	"encoding/binary"
	"errors"
	"math"
	"reflect"
	"sync"
	"time"
	"unsafe"
)

// errBatchNeedFallback is an internal sentinel: decodeBatchColumnar returns it
// when the wire carries something the v1 columnar fast path does not handle
// (a nullable column). unmarshalBatchCore catches it and re-decodes the
// ORIGINAL data through the reflect mirror fallback, which handles every wire.
var errBatchNeedFallback = errors.New("qdf: batch columnar fallback")

// unmarshalBatchCore decodes data into a T-rows region obtained from rows(n):
// the core never names T, all row writes go through unsafe offsets computed
// from plan.stride/plan.fields. rows is a closure over
// (*batchSlab).takeRows(n*plan.stride) supplied by the generic wrapper —
// the core learns n internally (from the columnar header, or from the
// mirror slice length) and calls rows(n) exactly once to get the backing
// pointer, which the wrapper then wraps into []T via unsafe.Slice. This
// keeps the pooled-rows-backing decision (takeRows) out of the core (which
// cannot name T) while letting both decode paths hand it their own
// early-known n.
//
// Fast path: a pure columnar container (tagColStruct) is decoded straight
// into the T rows + slab by decodeBatchColumnar — no mirror, no per-row
// string alloc. Anything else (row-major, hybrid, batched-vector, or a
// columnar payload with a nullable column) falls back to the reflect-driven
// mirror strategy below.
//
// Fallback strategy (correct-first, handles every wire): decode the whole
// payload through the EXISTING reflect-driven Unmarshal into a pooled
// []plan.mirror slice — a runtime struct type with the same wire field names
// as T but with handle types swapped back to string/[]byte/time.Time, so the
// normal decoder needs no new wire logic. Then one copy pass per row scatters
// each mirror row into a T row (memmove of scalar bytes) plus the slab
// (string/bytes bodies, rewritten as Str/Bytes handles) and converts qdf.Time.
// The opts parameter is reserved: arena/noCopy are deliberately inert here —
// the slab supersedes both (it owns every byte a handle points into), and no
// current QueryOption applies. Kept in the signature so the public
// UnmarshalBatch contract does not change when a batch-relevant option lands.
func unmarshalBatchCore(data []byte, plan *batchPlan, slab *batchSlab, rows func(n int) unsafe.Pointer, _ ...QueryOption) (int, error) {

	// --- Columnar fast path -------------------------------------------------
	// Attempt a pure-columnar decode on a pooled decoder. On success the T
	// rows and the slab are fully populated. On the fallback sentinel (or any
	// non-columnar wire) drop through to the mirror path, which re-decodes the
	// ORIGINAL data from scratch (so a rANS-wrapped or hybrid payload is parsed
	// cleanly regardless of how far the attempt advanced).
	if n, ok, err := tryDecodeBatchColumnar(data, plan, slab, rows); ok {
		return n, err
	}

	slot := plan.mirrorSlicePtr.Get().(*mirrorSlot)
	defer plan.mirrorSlicePtr.Put(slot)

	// Direct slice-header access instead of reflect.ValueOf/Elem/Index — the
	// slot caches the raw pointer, so the hot path is reflection-free (and
	// therefore identical under both reflect and qdf_reflect2 builds).
	hdr := (*sliceHeader)(slot.ptr)
	hdr.Len = 0

	if err := Unmarshal(data, slot.box); err != nil {
		return 0, err
	}

	n := hdr.Len
	if n == 0 {
		rows(0)
		return 0, nil
	}

	mirrorBase := hdr.Data
	mirrorStride := plan.mirror.Size()

	// Sum string/bytes body lengths up front so the slab grows exactly once
	// instead of on every append.
	var need int
	for i := range n {
		rowPtr := unsafe.Add(mirrorBase, uintptr(i)*mirrorStride)
		for fi, f := range plan.fields {
			switch f.kind {
			case bfStr:
				s := *(*string)(unsafe.Add(rowPtr, plan.mirrorOff[fi]))
				need += len(s)
			case bfBytes:
				b := *(*[]byte)(unsafe.Add(rowPtr, plan.mirrorOff[fi]))
				need += len(b)
			}
		}
	}
	slab.grow(need)

	rowsBase := rows(n)

	if err := batchCopyRows(plan, slab, mirrorBase, rowsBase, n); err != nil {
		return 0, err
	}

	return n, nil
}

// batchCopyRows scatters n mirror rows (starting at mirrorPtr, each
// plan.mirror.Size() bytes apart) into rowsBase (n rows of plan.stride
// bytes each), writing scalars via memmove, strings/bytes into slab (as
// Str/Bytes handles), and qdf.Time from time.Time.
func batchCopyRows(plan *batchPlan, slab *batchSlab, mirrorPtr, rowsBase unsafe.Pointer, n int) error {
	mirrorStride := plan.mirror.Size()
	for i := range n {
		src := unsafe.Add(mirrorPtr, uintptr(i)*mirrorStride)
		dst := unsafe.Add(rowsBase, uintptr(i)*plan.stride)
		for fi, f := range plan.fields {
			sf := unsafe.Add(src, plan.mirrorOff[fi])
			df := unsafe.Add(dst, f.off)
			switch f.kind {
			case bfStr:
				s := *(*string)(sf)
				off, ln, err := slab.append(unsafe.Slice(unsafe.StringData(s), len(s)))
				if err != nil {
					return err
				}
				*(*Str)(df) = Str{off: off, len: ln}
			case bfBytes:
				b := *(*[]byte)(sf)
				off, ln, err := slab.append(b)
				if err != nil {
					return err
				}
				*(*Bytes)(df) = Bytes{off: off, len: ln}
			case bfTime:
				t := *(*time.Time)(sf)
				*(*Time)(df) = Time{Sec: t.Unix(), Nsec: uint32(t.Nanosecond())}
			default: // bfScalar
				scalarSize := scalarKindSize(f.scalarKind)
				copy(unsafe.Slice((*byte)(df), scalarSize), unsafe.Slice((*byte)(sf), scalarSize))
			}
		}
	}
	return nil
}

// scalarKindSize returns the byte width of a scalar batchField, matching
// scalarKindType's type set.
func scalarKindSize(k reflect.Kind) uintptr {
	switch k {
	case reflect.Bool, reflect.Int8, reflect.Uint8:
		return 1
	case reflect.Int16, reflect.Uint16:
		return 2
	case reflect.Int32, reflect.Uint32, reflect.Float32:
		return 4
	case reflect.Int64, reflect.Uint64, reflect.Float64:
		return 8
	case reflect.Int, reflect.Uint, reflect.Uintptr:
		return unsafe.Sizeof(uintptr(0))
	default:
		return 0
	}
}

// tryDecodeBatchColumnar sets up a pooled decoder, reads the header, and — iff
// the top tag is a plain tagColStruct — runs decodeBatchColumnar. It returns
// ok=false to signal "not the fast path, use the mirror fallback": for a
// non-columnar tag, a header/peek error, or the errBatchNeedFallback sentinel.
// A real decode error (ok=true, err!=nil) is surfaced to the caller.
func tryDecodeBatchColumnar(data []byte, plan *batchPlan, slab *batchSlab, rows func(n int) unsafe.Pointer) (n int, ok bool, err error) {
	d := decPool.Get().(*Decoder)
	d.buf = data
	d.i = 0
	d.depth = 0
	d.headerRead = false
	d.mode = Fast
	d.colIndex = false
	d.colMaxLen = 0
	// noCopy makes the string readers ALIAS the input buffer instead of
	// allocating an owned string per distinct value (materializeStr). Every
	// aliased view is immediately copied into the slab (which owns the bytes a
	// handle points into), so the classic noCopy use-after-free hazard never
	// reaches the caller: the alias lives only for the duration of the slab
	// copy. This drops the dict-table / const / raw / inline per-string allocs
	// to zero — the slab copy is the sole owner.
	d.noCopy = true
	d.arena = nil
	d.selectFields = nil
	d.query = nil
	if d.state != nil {
		d.state.reset()
	}
	defer func() {
		d.buf = nil
		d.colMaxLen = 0
		d.noCopy = false
		decPool.Put(d)
	}()

	tag, perr := d.peekTag()
	if perr != nil || tag != tagColStruct {
		return 0, false, nil // non-columnar wire → mirror fallback
	}
	n, derr := decodeBatchColumnar(d, plan, slab, rows)
	if derr != nil {
		if errors.Is(derr, errBatchNeedFallback) {
			return 0, false, nil // nullable column → mirror fallback re-decodes original
		}
		return 0, true, derr
	}
	return n, true, nil
}

// decodeBatchColumnar decodes a pure columnar (tagColStruct) payload straight
// into the T rows (obtained from rows(n), see unmarshalBatchCore) plus the
// slab, with no reflect mirror and no per-row string allocation. It mirrors
// decodeColumnar's reading discipline (readColShape bounds, colMaxLen,
// checkColumnarBytes, per-kind dispatch) but scatters into batch fields via
// unsafe offsets and materializes string bodies into the slab as Str/Bytes
// handles.
//
// A nullable wire column is out of scope in v1 (a pointer-free T cannot map
// one anyway): the shape is validated up front and, if any column is nullable,
// errBatchNeedFallback is returned BEFORE any column body is consumed so the
// caller can cleanly re-decode via the mirror path.
func decodeBatchColumnar(d *Decoder, plan *batchPlan, slab *batchSlab, rows func(n int) unsafe.Pointer) (int, error) {
	n, nCols, colLens, err := batchReadColShape(d, plan, slab)
	if err != nil {
		return 0, err
	}

	// Validate the shape up front: any nullable column bails to the mirror
	// fallback before we consume a single column body (partial consumption is
	// harmless — the caller re-decodes the original data from scratch).
	kinds := slab.shapeKinds[:nCols]
	fidx := slab.shapeFidx[:nCols]
	for c := range kinds {
		if kinds[c].isNullable() {
			return 0, errBatchNeedFallback
		}
	}

	d.colMaxLen = n
	defer func() { d.colMaxLen = 0 }()

	if err := checkColumnarBytes(n, plan.stride); err != nil {
		return 0, err
	}

	// Obtain the T rows backing via the wrapper's takeRows closure. T is
	// guaranteed pointer-free (batchPlanOf rejects any pointer/handle-carrying
	// field), so a plain noscan []byte region is a valid backing for []T: the
	// GC never scans it. Steady-state (slab reused across decodes) this is a
	// cap-reuse + clear, not an allocation — the columnar fast path needs to
	// stay inside its handful-of-allocs budget.
	if n == 0 {
		rows(0)
		return 0, nil
	}
	base := rows(n)

	for c := range kinds {
		if fidx[c] < 0 {
			// Column present on the wire but not in the plan → skip its body
			// (forward compat). With no colIndex we must fully decode it to
			// keep the cursor in sync; with an index we can seek past it.
			if d.colIndex {
				if d.i+int(colLens[c]) > len(d.buf) {
					return 0, ErrShortBuffer
				}
				d.i += int(colLens[c])
				continue
			}
			if err := d.skipColumnValue(kinds[c], n); err != nil {
				return 0, err
			}
			continue
		}
		f := &plan.fields[fidx[c]]
		if err := scatterBatchColumn(d, plan, slab, base, f, kinds[c], n); err != nil {
			return 0, err
		}
	}
	// Plan fields absent from the wire keep their zero value (the make([]byte)
	// backing is zeroed) — the same schema-evolution semantics as normal decode.
	return n, nil
}

// batchReadColShape is the batch fast path's zero-alloc replacement for
// readColShape's inline-declaration branch. The generic reader materializes
// []string names + []colKind and registers the shape on the decoder state —
// per-message allocations a batch decode pays on every independent payload.
// Batch already knows every field name from the plan, so each wire column
// name is matched as a raw []byte view (string compare against a []byte does
// not allocate) and recorded as a plan-field index in slab scratch:
// slab.shapeFidx[c] (or -1 when the column is not in the plan) and
// slab.shapeKinds[c]. Reading discipline (bounds, MaxInt clauses, colIndex
// lens) mirrors readColShape exactly.
//
// A shape REFERENCE (id != 0) cannot resolve here: the pooled decoder's state
// is reset per batch decode and a single top-level columnar payload declares
// its shape inline, so a reference is either hostile or a multi-block wire
// this path does not handle — bail to the mirror fallback, which replays the
// original bytes through the reference decoder.
func batchReadColShape(d *Decoder, plan *batchPlan, slab *batchSlab) (n, nCols int, colLens []uint32, err error) {
	// Init decoder state unconditionally, exactly like readColShape: the
	// downstream scatter helpers dereference d.state scratch. Today a
	// tagColStruct wire implies FlagDense (whose header path inits state), but
	// that is an incidental invariant — keep the explicit init so a future
	// non-Dense columnar wire cannot nil-panic here.
	if d.state == nil {
		d.state = newDecState()
	}
	d.i++ // consume tagColStruct (caller peeked it)
	n64, k := readUvarint(d.buf[d.i:])
	if k <= 0 {
		return 0, 0, nil, ErrInvalidLength
	}
	d.i += k
	n = int(n64)
	if err := checkColumnarN(n); err != nil {
		return 0, 0, nil, err
	}
	idv, k2 := readUvarint(d.buf[d.i:])
	if k2 <= 0 {
		return 0, 0, nil, ErrInvalidLength
	}
	if idv != 0 {
		return 0, 0, nil, errBatchNeedFallback
	}
	d.i += k2
	cnt64, k3 := readUvarint(d.buf[d.i:])
	if k3 <= 0 {
		return 0, 0, nil, ErrInvalidLength
	}
	d.i += k3
	if cnt64 > uint64(math.MaxInt) { // 32-bit: int() would wrap negative
		return 0, 0, nil, ErrInvalidLength
	}
	nCols = int(cnt64)
	if err := d.CheckLength(nCols, 1); err != nil {
		return 0, 0, nil, err
	}
	if cap(slab.shapeFidx) < nCols {
		slab.shapeFidx = make([]int16, nCols)
		slab.shapeKinds = make([]colKind, nCols)
	}
	fidx := slab.shapeFidx[:nCols]
	kinds := slab.shapeKinds[:nCols]
	for c := range nCols {
		s, err := d.readStringBytes() // view into d.buf; matched, never kept
		if err != nil {
			return 0, 0, nil, err
		}
		if d.i >= len(d.buf) {
			return 0, 0, nil, ErrShortBuffer
		}
		kinds[c] = colKind(d.buf[d.i])
		d.i++
		fidx[c] = -1
		for i := range plan.fields {
			if plan.fields[i].name == string(s) { // no alloc: compare only
				fidx[c] = int16(i)
				break
			}
		}
	}
	// colIndex lens: same pooled read as readColShape.
	if d.colIndex {
		if d.i+4*nCols > len(d.buf) {
			return 0, 0, nil, ErrShortBuffer
		}
		if d.state == nil {
			d.state = newDecState()
		}
		if cap(d.state.colLenScratch) >= nCols {
			colLens = d.state.colLenScratch[:nCols]
		} else {
			colLens = make([]uint32, nCols)
		}
		d.state.colLenScratch = colLens
		var sum uint64
		for c := range nCols {
			colLens[c] = binary.LittleEndian.Uint32(d.buf[d.i+4*c:])
			sum += uint64(colLens[c])
		}
		d.i += 4 * nCols
		if sum > uint64(len(d.buf)-d.i) {
			return 0, 0, nil, ErrShortBuffer
		}
	}
	return n, nCols, colLens, nil
}

// scatterBatchColumn decodes one wire column of the given kind and scatters it
// into the matched batch field across all n rows. Scalars/bool reuse the
// decoder's *Into scratch then a width-switched store; time uses the sec+nsec
// sub-columns; string/bytes materialize into the slab as handles.
func scatterBatchColumn(d *Decoder, plan *batchPlan, slab *batchSlab, base unsafe.Pointer, f *batchField, kind colKind, n int) error {
	stride := plan.stride
	off := f.off
	st := d.state // non-nil: readColShape initialised it

	switch f.kind {
	case bfStr, bfBytes:
		if kind != colKindString {
			return ErrTypeMismatch
		}
		out := getStrHandleScratch(n)
		defer putStrHandleScratch(out)
		if err := readStringColumnHandles(d, n, slab, out); err != nil {
			return err
		}
		// Str and Bytes have identical layout (off,len uint32); one store path.
		for i := range n {
			dp := unsafe.Add(base, uintptr(i)*stride+off)
			*(*Str)(dp) = out[i]
		}
		return nil
	case bfTime:
		if kind != colKindTime {
			return ErrTypeMismatch
		}
		if err := decodeSliceInt64Into(d, &st.colScratchI64); err != nil {
			return err
		}
		sec := st.colScratchI64
		if len(sec) != n {
			return ErrTypeMismatch
		}
		if err := decodeSliceUint64Into(d, &st.colScratchU64); err != nil {
			return err
		}
		nsec := st.colScratchU64
		if len(nsec) != n {
			return ErrTypeMismatch
		}
		if err := checkNsecColumn(nsec); err != nil {
			return err
		}
		for i := range n {
			dp := unsafe.Add(base, uintptr(i)*stride+off)
			*(*Time)(dp) = Time{Sec: sec[i], Nsec: uint32(nsec[i])}
		}
		return nil
	default: // bfScalar
		return scatterBatchScalar(d, base, stride, off, f.scalarKind, kind, n)
	}
}

// batchScalarColKind maps a scalar field's Go kind to the columnar kind the
// wire must carry for it. Used to reject a schema-evolution mismatch (e.g. a
// wire int column landing on a float field) BEFORE decoding, mirroring
// decodeColumnar's exact sh.kinds[c] == col.kind guard — without it the width
// dispatch below would reinterpret the bits and silently corrupt the value.
func batchScalarColKind(k reflect.Kind) (colKind, bool) {
	switch k {
	case reflect.Bool:
		return colKindBool, true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return colKindInt, true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return colKindUint, true
	case reflect.Float32:
		return colKindFloat32, true
	case reflect.Float64:
		return colKindFloat, true
	default:
		return 0, false
	}
}

// scatterBatchScalar decodes a scalar/bool column into scratch then stores each
// value at base+i*stride+off using the field's Go kind width. The wire colKind
// must equal the field's expected columnar kind (per batchScalarColKind).
func scatterBatchScalar(d *Decoder, base unsafe.Pointer, stride, off uintptr, sk reflect.Kind, kind colKind, n int) error {
	if want, ok := batchScalarColKind(sk); !ok || want != kind {
		return ErrTypeMismatch
	}
	st := d.state
	switch kind {
	case colKindInt:
		if err := decodeSliceInt64Into(d, &st.colScratchI64); err != nil {
			return err
		}
		s := st.colScratchI64
		if len(s) != n {
			return ErrTypeMismatch
		}
		w := scalarKindSize(sk)
		for i := range n {
			storeIntWidth(unsafe.Add(base, uintptr(i)*stride+off), w, s[i])
		}
	case colKindUint:
		if err := decodeSliceUint64Into(d, &st.colScratchU64); err != nil {
			return err
		}
		s := st.colScratchU64
		if len(s) != n {
			return ErrTypeMismatch
		}
		w := scalarKindSize(sk)
		for i := range n {
			storeUintWidth(unsafe.Add(base, uintptr(i)*stride+off), w, s[i])
		}
	case colKindFloat:
		if err := decodeSliceFloat64Into(d, &st.colScratchF64); err != nil {
			return err
		}
		s := st.colScratchF64
		if len(s) != n {
			return ErrTypeMismatch
		}
		for i := range n {
			*(*float64)(unsafe.Add(base, uintptr(i)*stride+off)) = s[i]
		}
	case colKindFloat32:
		if err := decodeSliceUint64Into(d, &st.colScratchU64); err != nil {
			return err
		}
		s := st.colScratchU64
		if len(s) != n {
			return ErrTypeMismatch
		}
		for i := range n {
			*(*uint32)(unsafe.Add(base, uintptr(i)*stride+off)) = uint32(s[i])
		}
	case colKindBool:
		if err := decodeSliceBoolInto(d, &st.colScratchBool); err != nil {
			return err
		}
		s := st.colScratchBool
		if len(s) != n {
			return ErrTypeMismatch
		}
		for i := range n {
			*(*bool)(unsafe.Add(base, uintptr(i)*stride+off)) = s[i]
		}
	default:
		return ErrTypeMismatch
	}
	return nil
}

// storeIntWidth / storeUintWidth write a decoded 64-bit column value into a
// struct field of the given byte width w (= field.Type.Size(), the same
// dispatch scatterColI64/scatterColU64 in columnar.go use). The switch is on
// the field's size in BYTES, not a wire tag.
//
//go:nosplit
func storeIntWidth(p unsafe.Pointer, w uintptr, v int64) {
	switch w {
	case 1: // int8
		*(*int8)(p) = int8(v)
	case 2: // int16
		*(*int16)(p) = int16(v)
	case 4: // int32
		*(*int32)(p) = int32(v)
	default: // 8: int, int64
		*(*int64)(p) = v
	}
}

//go:nosplit
func storeUintWidth(p unsafe.Pointer, w uintptr, v uint64) {
	switch w {
	case 1: // uint8/byte
		*(*uint8)(p) = uint8(v)
	case 2: // uint16
		*(*uint16)(p) = uint16(v)
	case 4: // uint32 (also float32 bit patterns)
		*(*uint32)(p) = uint32(v)
	default: // 8: uint, uint64, uintptr, float64 bits
		*(*uint64)(p) = v
	}
}

// strHandleScratchPool pools []Str backing arrays so per-column handle scatter
// does not allocate on repeat decodes. dictHandleScratchPool is a SEPARATE pool
// for the dict-table handle slice: the dict arm holds an out-scratch drawn from
// the first pool while it also needs a table-handle scratch, and drawing both
// from one pool would force a fresh alloc for the second (each pool caches one
// per P).
var (
	strHandleScratchPool  = sync.Pool{New: func() any { s := make([]Str, 0, 64); return &s }}
	dictHandleScratchPool = sync.Pool{New: func() any { s := make([]Str, 0, 64); return &s }}
)

func getStrHandleScratch(n int) []Str  { return getPooledStrScratch(&strHandleScratchPool, n) }
func putStrHandleScratch(s []Str)      { putPooledStrScratch(&strHandleScratchPool, s) }
func getDictHandleScratch(n int) []Str { return getPooledStrScratch(&dictHandleScratchPool, n) }
func putDictHandleScratch(s []Str)     { putPooledStrScratch(&dictHandleScratchPool, s) }

func getPooledStrScratch(pool *sync.Pool, n int) []Str {
	sp := pool.Get().(*[]Str)
	s := *sp
	if cap(s) < n {
		s = make([]Str, n)
	} else {
		s = s[:n]
	}
	return s
}

func putPooledStrScratch(pool *sync.Pool, s []Str) {
	s = s[:0]
	pool.Put(&s)
}

// readStringColumnHandles decodes a string column body (length n) and writes a
// per-row slab handle into out[0:n], materializing each distinct string body
// into the slab exactly once where the codec makes dedup free:
//
//   - tagColStrConst: one slab.append, the same handle for every row.
//   - dict family (tagColStrDict/DictQ/DictFC): each dict ENTRY is materialized
//     once; a row's handle is its entry's handle (free dedup).
//   - everything else (tagColStrRaw / FSST / alpha / per-row inline strings):
//     the existing readStringColumn machinery decodes each string; we copy the
//     bytes out into the slab before that machinery's scratch is recycled.
//
// bfBytes columns share the string-column wire, so this covers them too.
func readStringColumnHandles(d *Decoder, n int, slab *batchSlab, out []Str) error {
	if n == 0 {
		return nil
	}
	if d.i >= len(d.buf) {
		// n > 0 with no bytes left: truncated/hostile input. Without this the
		// bare tag index below panics — the reference decodeColumnInto guards
		// every entry the same way (public API must error, never panic).
		return ErrShortBuffer
	}
	tag := d.buf[d.i]
	switch tag {
	case tagColStrConst:
		strs, err := d.readStringColumnConst(n)
		if err != nil {
			return err
		}
		v := strs[0]
		off, ln, err := slab.append(unsafe.Slice(unsafe.StringData(v), len(v)))
		if err != nil {
			return err
		}
		h := Str{off: off, len: ln}
		for i := range n {
			out[i] = h
		}
		return nil
	case tagColStrDict, tagColStrDictFC, tagColStrDictQ:
		var (
			table []string
			idx   []uint32
			err   error
		)
		switch tag {
		case tagColStrDictFC:
			table, idx, err = d.readStringColumnDictFC(n)
		case tagColStrDictQ:
			table, idx, err = d.readStringColumnDictQ(n)
		default:
			table, idx, err = d.readStringColumnDict(n)
		}
		if err != nil {
			return err
		}
		// Materialize each distinct table entry into the slab once, then map
		// per-row indices to their entry handles.
		th := getDictHandleScratch(len(table))
		defer putDictHandleScratch(th)
		for k := range table {
			v := table[k]
			off, ln, err := slab.append(unsafe.Slice(unsafe.StringData(v), len(v)))
			if err != nil {
				return err
			}
			th[k] = Str{off: off, len: ln}
		}
		for i := range n {
			j := idx[i]
			if int(j) >= len(table) {
				return ErrInvalidLength
			}
			out[i] = th[j]
		}
		return nil
	default:
		// tagColStrRaw / FSST / alpha / per-row inline: reuse the general
		// materializer, then copy each body into the slab (distinct strings,
		// so there is no dedup to lose). readStringColumn's result aliases
		// decoder scratch — copy before it is recycled by the next column.
		strs, err := d.readStringColumn(n)
		if err != nil {
			return err
		}
		for i := range n {
			v := strs[i]
			off, ln, err := slab.append(unsafe.Slice(unsafe.StringData(v), len(v)))
			if err != nil {
				return err
			}
			out[i] = Str{off: off, len: ln}
		}
		return nil
	}
}
