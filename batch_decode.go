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

	// Row-major-direct fast path: a plain row-major struct-slice wire (an array
	// header, as opposed to the tagColStruct/tagHybridColStruct container tags)
	// is decoded straight into the T rows + slab — no mirror, no per-row owned
	// string. Any other wire (hybrid, batched-vector, tagNil, …) returns
	// ok=false and drops through to the mirror fallback below.
	if n, ok, err := tryDecodeBatchRowMajor(data, plan, slab, rows); ok {
		return n, err
	}

	return unmarshalBatchMirror(data, plan, slab, rows)
}

// unmarshalBatchMirror is unmarshalBatchCore's fallback strategy, factored out
// so it can also be driven directly — bypassing both fast-path attempts above —
// by a benchmark control that measures the reflect-mirror cost of a wire the
// direct paths would otherwise handle. unmarshalBatchCore's normal call path
// reaches it only after both tryDecodeBatchColumnar and tryDecodeBatchRowMajor
// return ok=false (hybrid, batched-vector, tagNil, a nullable columnar column,
// or — for the benchmark control — a row-major wire decoded on purpose without
// the direct path).
func unmarshalBatchMirror(data []byte, plan *batchPlan, slab *batchSlab, rows func(n int) unsafe.Pointer) (int, error) {
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
				// A schema-absent time field decodes to Go's zero time.Time;
				// converting it would write a large-negative Sec, diverging from
				// the columnar fast path (which leaves the zeroed rows region as
				// Time{0,0}). Guard the zero case so both paths agree.
				if !t.IsZero() {
					*(*Time)(df) = Time{Sec: t.Unix(), Nsec: uint32(t.Nanosecond())}
				}
			default: // bfScalar
				copy(unsafe.Slice((*byte)(df), f.width), unsafe.Slice((*byte)(sf), f.width))
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
	d.selectKeys = nil
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

// tryDecodeBatchRowMajor sets up a pooled decoder exactly like
// tryDecodeBatchColumnar and peeks the top tag to DETECT a pure row-major
// struct-slice wire: a plain array header (the tagFixarr range, tagArr16, or
// tagArr32 — the same tag set ReadArrayHeader accepts) whose elements are
// struct-headers, as opposed to the tagColStruct (columnar) or
// tagHybridColStruct (hybrid) container tags. Those two container tags are
// never array-header tags, so checking "is this an array-header tag" alone
// is sufficient to rule them both out — no need to peek past the outer tag.
//
// On detection it decodes the array of per-row structs directly into the T
// rows (obtained from rows(n)) plus the slab, with NO reflect mirror and NO
// per-row owned-string allocation: each row is read via ReadStructHeader
// (shape-interned or plain map) and every wire field is matched by name
// against plan.fields and scattered through unsafe offsets, mirroring the
// generated DecodeQDF discipline. String/bytes bodies alias the input buffer
// (noCopy) and are copied once into the slab as Str/Bytes handles.
//
// Contract (same as tryDecodeBatchColumnar's real-decode branch):
//   - ok=false, nil → not this fast path (non-array-header tag, or a peek
//     error before any byte is consumed) → the caller's mirror fallback
//     re-decodes the ORIGINAL data.
//   - ok=true, nil → decoded successfully.
//   - ok=true, err → a hard decode error AFTER the header was consumed. The
//     cursor is spent, so the caller must NOT fall back to the mirror (that
//     would double-decode); the error is surfaced. Every d.Read* error is
//     propagated and a per-field wire/plan type mismatch surfaces as the
//     value reader's ErrTypeMismatch — the outer array tag proves nothing
//     about the element types, so nothing is scattered without validation.
func tryDecodeBatchRowMajor(data []byte, plan *batchPlan, slab *batchSlab, rows func(n int) unsafe.Pointer) (n int, ok bool, err error) {
	d := decPool.Get().(*Decoder)
	d.buf = data
	d.i = 0
	d.depth = 0
	d.headerRead = false
	d.mode = Fast
	d.colIndex = false
	d.colMaxLen = 0
	// noCopy mirrors tryDecodeBatchColumnar's setup: any string reads this
	// path eventually does must alias-then-copy into the slab, never
	// materialize an owned per-value string.
	d.noCopy = true
	d.arena = nil
	d.selectFields = nil
	d.selectKeys = nil
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
	if perr != nil {
		return 0, false, nil // header/peek error: let the mirror re-decode and surface it
	}

	// isRowMajor is true only for a plain array-header tag. tagColStruct and
	// tagHybridColStruct are distinct, non-array-header tag values, so this
	// single check is enough to exclude both.
	isRowMajor := tag >= tagFixarr && tag <= tagFixarr|tagFixarrMask || tag == tagArr16 || tag == tagArr32
	if !isRowMajor {
		return 0, false, nil // columnar / hybrid / tagNil / anything else → mirror fallback
	}

	// Detected a row-major struct-slice wire. From here every error is HANDLED
	// (ok=true): ReadArrayHeader consumes the peeked array tag, so the cursor is
	// spent and a mid-stream fall back to the mirror would double-decode.
	n, err = d.ReadArrayHeader()
	if err != nil {
		return 0, true, err
	}
	// Bound n by the INPUT before it drives any allocation: ReadArrayHeader's
	// tagArr32 arm already rejects a count over the remaining bytes, but its
	// tagArr16 arm does not (only 2 header bytes are checked, not the claimed
	// count) — a 3-byte hostile wire can claim up to 65535 rows with zero
	// bytes left to decode them from. Each row is a struct header of >= 1 wire
	// byte, so CheckLength(n, 1) rejects that up front, matching the row-major
	// reflect decoder's identical guard (emitDecodeSliceRowMajorBody,
	// cmd/qdfgen/gen/gen.go) and go-fuzz-oom-triage's "bound every decode
	// allocation by the input" rule.
	if err := d.CheckLength(n, 1); err != nil {
		return 0, true, err
	}
	// Bound the OUTPUT allocation like the columnar path: a wide-struct T with
	// a still-plausible (input-bounded) n could otherwise amplify a modest
	// wire into a very large rows(n) region; this caps n*stride at
	// maxColumnarBytes on top of the input-proportional guard above.
	if err = checkColumnarBytes(n, plan.stride); err != nil {
		return 0, true, err
	}
	if n == 0 {
		rows(0)
		return 0, true, nil
	}
	base := rows(n)

	// Plan fields absent from a row keep their zero value: takeRows hands back
	// a zeroed region, so a field never scattered stays Time{0,0}/0/"" — the
	// same schema-evolution semantics as the columnar and mirror paths.
	for i := range n {
		rowDst := unsafe.Add(base, uintptr(i)*plan.stride)
		names, plainN, shaped, herr := d.ReadStructHeader()
		if herr != nil {
			return 0, true, herr
		}
		if shaped {
			// Shape-interned struct: values follow positionally in names order,
			// no per-value key on the wire.
			for _, name := range names {
				nb := unsafe.Slice(unsafe.StringData(name), len(name))
				if ferr := scatterBatchRowMajorField(d, plan, slab, rowDst, nb); ferr != nil {
					return 0, true, ferr
				}
			}
			continue
		}
		// Plain map: plainN (key, value) pairs. The key bytes alias the input;
		// comparing them against a plan-field name (string(b) == name) does not
		// allocate.
		for range plainN {
			kb, kerr := d.ReadStringBytes()
			if kerr != nil {
				return 0, true, kerr
			}
			if ferr := scatterBatchRowMajorField(d, plan, slab, rowDst, kb); ferr != nil {
				return 0, true, ferr
			}
		}
	}
	return n, true, nil
}

// batchPlanFieldByName returns the plan field whose wire key equals name, or
// nil when name is not a plan field (a forward-compat wire field to skip). The
// name argument aliases the input buffer / an interned shape name; the
// string(name) == f.name comparison is compiled to a zero-alloc byte compare.
func batchPlanFieldByName(plan *batchPlan, name []byte) *batchField {
	for i := range plan.fields {
		if plan.fields[i].name == string(name) {
			return &plan.fields[i]
		}
	}
	return nil
}

// scatterBatchRowMajorField reads one wire field value for the row at rowDst,
// matched by name against plan.fields, and scatters it through the field's
// unsafe offset. An unmatched name is skipped (forward compat). Type safety is
// enforced by the value reader: ReadInt/ReadUint/ReadFloat*/ReadBool/
// ReadTimestamp/ReadString each return ErrTypeMismatch when the wire tag does
// not match the field's expected kind, so a mismatched element cannot scatter
// garbage — the outer array-header tag never implies the element types.
func scatterBatchRowMajorField(d *Decoder, plan *batchPlan, slab *batchSlab, rowDst unsafe.Pointer, name []byte) error {
	f := batchPlanFieldByName(plan, name)
	if f == nil {
		return d.Skip() // wire field not in plan → skip its single value
	}
	dst := unsafe.Add(rowDst, f.off)
	switch f.kind {
	case bfStr:
		// noCopy → s aliases the input buffer; slab.append copies it (the sole
		// owner), so the alias never escapes.
		s, err := d.ReadString()
		if err != nil {
			return err
		}
		off, ln, err := slab.append(unsafe.Slice(unsafe.StringData(s), len(s)))
		if err != nil {
			return err
		}
		*(*Str)(dst) = Str{off: off, len: ln}
	case bfBytes:
		// A nil []byte encodes as tagNil (distinct from an empty bin); the mirror
		// path (decodeBytes → decodeNilSlice) consumes it as a nil slice. Mirror
		// that: consume the tag and leave the zeroed handle (BytesOf resolves
		// Bytes{0,0} to nil), matching batchCopyRows which stores (0,0) for both
		// a nil and an empty []byte.
		t, terr := d.peekTag()
		if terr != nil {
			return terr
		}
		if t == tagNil {
			d.i++
			return nil
		}
		b, err := d.ReadBytes() // noCopy → aliases input; copied into the slab
		if err != nil {
			return err
		}
		off, ln, err := slab.append(b)
		if err != nil {
			return err
		}
		*(*Bytes)(dst) = Bytes{off: off, len: ln}
	case bfTime:
		// The wire carries sec/nsec directly (unlike the mirror path, which
		// decodes a Go time.Time and must guard its zero value): a schema-absent
		// time is never read here — it stays the zeroed Time{0,0} takeRows left,
		// matching the columnar path.
		sec, nsec, err := d.ReadTimestamp()
		if err != nil {
			return err
		}
		*(*Time)(dst) = Time{Sec: sec, Nsec: nsec}
	default: // bfScalar
		return scatterBatchRowMajorScalar(d, dst, f.scalarKind)
	}
	return nil
}

// scatterBatchRowMajorScalar reads one scalar value per the field's Go kind and
// stores it width-narrowed at dst, reusing scatterBatchScalar's width-store
// logic for a single value. The chosen reader validates the wire tag: a wire
// value whose type does not match sk yields ErrTypeMismatch instead of a
// silent reinterpretation of the bits.
func scatterBatchRowMajorScalar(d *Decoder, dst unsafe.Pointer, sk reflect.Kind) error {
	switch sk {
	case reflect.Bool:
		v, err := d.ReadBool()
		if err != nil {
			return err
		}
		*(*bool)(dst) = v
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := d.ReadInt()
		if err != nil {
			return err
		}
		switch scalarKindSize(sk) {
		case 1: // int8
			*(*int8)(dst) = int8(v)
		case 2: // int16
			*(*int16)(dst) = int16(v)
		case 4: // int32
			*(*int32)(dst) = int32(v)
		default: // 8: int, int64
			*(*int64)(dst) = v
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		v, err := d.ReadUint()
		if err != nil {
			return err
		}
		switch scalarKindSize(sk) {
		case 1: // uint8/byte
			*(*uint8)(dst) = uint8(v)
		case 2: // uint16
			*(*uint16)(dst) = uint16(v)
		case 4: // uint32
			*(*uint32)(dst) = uint32(v)
		default: // 8: uint, uint64, uintptr
			*(*uint64)(dst) = v
		}
	case reflect.Float32:
		v, err := d.ReadFloat32()
		if err != nil {
			return err
		}
		*(*float32)(dst) = v
	case reflect.Float64:
		v, err := d.ReadFloat64()
		if err != nil {
			return err
		}
		*(*float64)(dst) = v
	default:
		return ErrTypeMismatch
	}
	return nil
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
		if cap(st.colStrHandles) < n {
			st.colStrHandles = make([]Str, n)
		}
		out := st.colStrHandles[:n]
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
		// Width-specialized loops: hoisting the switch out of the per-element
		// path removes a branch+call per cell (the column's width never varies).
		switch w := scalarKindSize(sk); w {
		case 1: // int8
			for i := range n {
				*(*int8)(unsafe.Add(base, uintptr(i)*stride+off)) = int8(s[i])
			}
		case 2: // int16
			for i := range n {
				*(*int16)(unsafe.Add(base, uintptr(i)*stride+off)) = int16(s[i])
			}
		case 4: // int32
			for i := range n {
				*(*int32)(unsafe.Add(base, uintptr(i)*stride+off)) = int32(s[i])
			}
		default: // 8: int, int64
			for i := range n {
				*(*int64)(unsafe.Add(base, uintptr(i)*stride+off)) = s[i]
			}
		}
	case colKindUint:
		if err := decodeSliceUint64Into(d, &st.colScratchU64); err != nil {
			return err
		}
		s := st.colScratchU64
		if len(s) != n {
			return ErrTypeMismatch
		}
		switch w := scalarKindSize(sk); w {
		case 1: // uint8/byte
			for i := range n {
				*(*uint8)(unsafe.Add(base, uintptr(i)*stride+off)) = uint8(s[i])
			}
		case 2: // uint16
			for i := range n {
				*(*uint16)(unsafe.Add(base, uintptr(i)*stride+off)) = uint16(s[i])
			}
		case 4: // uint32
			for i := range n {
				*(*uint32)(unsafe.Add(base, uintptr(i)*stride+off)) = uint32(s[i])
			}
		default: // 8: uint, uint64, uintptr
			for i := range n {
				*(*uint64)(unsafe.Add(base, uintptr(i)*stride+off)) = s[i]
			}
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

// dictHandleScratchPool pools []Str backing arrays for the dict-table handle slice.
var (
	dictHandleScratchPool = sync.Pool{New: func() any { s := make([]Str, 0, 64); return &s }}
)

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
	case tagColStrFSST:
		// FSST decompresses straight into the slab (no temp scratch + no second
		// copy). readStringColumnFSSTInto mirrors readStringColumnFSST's header
		// parse and every bounds guard verbatim.
		return readStringColumnFSSTInto(d, n, slab, out)
	case tagColStrAlpha:
		// Alpha unpacks its characters straight onto the slab's backing (no temp
		// scratch + no second copy), the same shape as the FSST arm above.
		return d.readStringColumnAlphaInto(n, slab, out)
	default:
		// tagColStrRaw / per-row inline: reuse the general materializer, then
		// copy each body into the slab (distinct strings, so there is no dedup
		// to lose). readStringColumn's result aliases decoder scratch — copy
		// before it is recycled by the next column.
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
