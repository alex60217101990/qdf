package qdf

import (
	"cmp"
	"fmt"
	"math"
	"reflect"
	"slices"
	"time"
	"unsafe"

	"github.com/alex60217101990/qdf/internal/reflectutil"
	"github.com/alex60217101990/qdf/internal/unsafestr"
)

// ----- per-kind encoders -----

func encodeBool(e *Encoder, p unsafe.Pointer) error {
	e.WriteBool(*(*bool)(p))
	return nil
}
func decodeBool(d *Decoder, p unsafe.Pointer) error {
	v, err := d.ReadBool()
	if err != nil {
		return err
	}
	*(*bool)(p) = v
	return nil
}

// encodeIntN/decodeIntN handle all Go signed int widths. The size parameter
// is the Go-level type size in bytes (1,2,4,8) so we can read through
// unsafe.Pointer without reinterpreting.
func encodeIntN(sz int) func(*Encoder, unsafe.Pointer) error {
	switch sz {
	case 1:
		return func(e *Encoder, p unsafe.Pointer) error { e.WriteInt(int64(*(*int8)(p))); return nil }
	case 2:
		return func(e *Encoder, p unsafe.Pointer) error { e.WriteInt(int64(*(*int16)(p))); return nil }
	case 4:
		return func(e *Encoder, p unsafe.Pointer) error { e.WriteInt(int64(*(*int32)(p))); return nil }
	default:
		return func(e *Encoder, p unsafe.Pointer) error { e.WriteInt(*(*int64)(p)); return nil }
	}
}
func decodeIntN(sz int) func(*Decoder, unsafe.Pointer) error {
	switch sz {
	case 1:
		return func(d *Decoder, p unsafe.Pointer) error {
			v, err := d.ReadInt()
			if err != nil {
				return err
			}
			*(*int8)(p) = int8(v)
			return nil
		}
	case 2:
		return func(d *Decoder, p unsafe.Pointer) error {
			v, err := d.ReadInt()
			if err != nil {
				return err
			}
			*(*int16)(p) = int16(v)
			return nil
		}
	case 4:
		return func(d *Decoder, p unsafe.Pointer) error {
			v, err := d.ReadInt()
			if err != nil {
				return err
			}
			*(*int32)(p) = int32(v)
			return nil
		}
	default:
		return func(d *Decoder, p unsafe.Pointer) error {
			v, err := d.ReadInt()
			if err != nil {
				return err
			}
			*(*int64)(p) = v
			return nil
		}
	}
}

func encodeUintN(sz int) func(*Encoder, unsafe.Pointer) error {
	switch sz {
	case 1:
		return func(e *Encoder, p unsafe.Pointer) error { e.WriteUint(uint64(*(*uint8)(p))); return nil }
	case 2:
		return func(e *Encoder, p unsafe.Pointer) error { e.WriteUint(uint64(*(*uint16)(p))); return nil }
	case 4:
		return func(e *Encoder, p unsafe.Pointer) error { e.WriteUint(uint64(*(*uint32)(p))); return nil }
	default:
		return func(e *Encoder, p unsafe.Pointer) error { e.WriteUint(*(*uint64)(p)); return nil }
	}
}
func decodeUintN(sz int) func(*Decoder, unsafe.Pointer) error {
	switch sz {
	case 1:
		return func(d *Decoder, p unsafe.Pointer) error {
			v, err := d.ReadUint()
			if err != nil {
				return err
			}
			*(*uint8)(p) = uint8(v)
			return nil
		}
	case 2:
		return func(d *Decoder, p unsafe.Pointer) error {
			v, err := d.ReadUint()
			if err != nil {
				return err
			}
			*(*uint16)(p) = uint16(v)
			return nil
		}
	case 4:
		return func(d *Decoder, p unsafe.Pointer) error {
			v, err := d.ReadUint()
			if err != nil {
				return err
			}
			*(*uint32)(p) = uint32(v)
			return nil
		}
	default:
		return func(d *Decoder, p unsafe.Pointer) error {
			v, err := d.ReadUint()
			if err != nil {
				return err
			}
			*(*uint64)(p) = v
			return nil
		}
	}
}

func encodeF32(e *Encoder, p unsafe.Pointer) error { e.WriteFloat32(*(*float32)(p)); return nil }
func decodeF32(d *Decoder, p unsafe.Pointer) error {
	v, err := d.ReadFloat32()
	if err != nil {
		return err
	}
	*(*float32)(p) = v
	return nil
}
func encodeF64(e *Encoder, p unsafe.Pointer) error { e.WriteFloat64(*(*float64)(p)); return nil }
func decodeF64(d *Decoder, p unsafe.Pointer) error {
	v, err := d.ReadFloat64()
	if err != nil {
		return err
	}
	*(*float64)(p) = v
	return nil
}

func encodeString(e *Encoder, p unsafe.Pointer) error { e.WriteString(*(*string)(p)); return nil }
func decodeString(d *Decoder, p unsafe.Pointer) error {
	s, err := d.ReadString()
	if err != nil {
		return err
	}
	*(*string)(p) = s
	return nil
}

func encodeBytes(e *Encoder, p unsafe.Pointer) error {
	if e.encodeNilSlice(p) { // nil []byte → tagNil (distinct from empty)
		return nil
	}
	e.WriteBytes(*(*[]byte)(p))
	return nil
}
func decodeBytes(d *Decoder, p unsafe.Pointer) error {
	if d.decodeNilSlice(p) {
		return nil
	}
	b, err := d.ReadBytes()
	if err != nil {
		return err
	}
	*(*[]byte)(p) = b
	return nil
}

func encodeSlice(elem *typeDesc, stride uintptr, colPlan *columnarPlan) func(*Encoder, unsafe.Pointer) error {
	return func(e *Encoder, p unsafe.Pointer) error {
		if e.encodeNilSlice(p) { // nil slice → tagNil (distinct from empty)
			return nil
		}
		// Depth guard mirroring Decoder.descend in decodeSlice: count this
		// container level so the encoder refuses a payload nested deeper than the
		// decoder will accept, instead of emitting bytes that then fail to decode
		// (slice/map/array-carried recursion was previously unbounded on encode
		// while the decoder caps it at maxDepth).
		if e.maxDepth != 0 {
			e.depth++
			if e.depth > e.maxDepth {
				e.depth--
				return ErrCycleDetected
			}
			defer func() { e.depth-- }()
		}
		hdr := (*sliceHeader)(p)
		n := hdr.Len
		// Batched lossy vector-column path: under OptLossyVec, a []struct with an
		// equal-length []float32/[]float64 field is encoded as one count=N block
		// per vector field (tagVecBatchStruct) instead of N count=1 blocks. Gated
		// to the same contexts as columnar; falls through when nothing is batchable
		// (so non-lossy and unbatchable shapes stay byte-identical).
		if e.opts.Has(OptLossyVec) && elem != nil && len(elem.vecFields) > 0 &&
			n >= columnarMinElems && e.state != nil && !e.stateSuspended &&
			e.ifaceDepth == 0 &&
			e.opts.Has(OptDense) && e.opts.Has(OptShapeIntern) {
			if done, err := e.encodeVectorBatchStruct(elem, hdr.Data, n, stride); done {
				return err
			}
		}
		// Columnar / hybrid-columnar path.
		//
		// Pure plan (every field eligible): transpose under the columnarProbe
		// gate, as before — tagColStruct.
		//
		// Hybrid plan (some residual fields): auto-fire when FSST is enabled
		// (OptFSST / OptCompression), OR — under plain Balanced — when the plan has
		// a string column and the INTERN-AWARE probe predicts a win. The plain
		// per-column probe cannot see the cross-row string deduplication that
		// row-major Dense gets from the global intern table, so a low-cardinality
		// string column looks expensive row-major and the probe mispredicts a
		// columnar win that does not exist. The intern-aware probe credits that
		// dedup (a repeated value costs a 1-byte state-ref), so a low-card string
		// column is correctly cheap row-major and only a genuine columnar win
		// (a Delta/FOR numeric column, or a restricted-alphabet ID column that
		// alpha-packs) pulls the struct in. The emit picker is never-larger per
		// column, so the worst case is a small shape-header overhead — bounded by
		// the gain threshold the probe already requires. With FSST the eligible
		// string columns are compressed by the symbol table (AD/log/RTB win at
		// OptCompression). A hybrid plan with no string column and no FSST still
		// falls through to row-major, byte-identical to today.
		if colPlan != nil && n >= columnarMinElems && e.state != nil && !e.stateSuspended &&
			e.opts.Has(OptDense) && e.opts.Has(OptShapeIntern) {
			pure := colPlan.residual == nil
			internAware := !pure && !e.fsst && colPlan.hasStringCol
			if (pure || e.fsst || internAware) && columnarProbe(colPlan, hdr.Data, n, e.fsst, e.fsstDict, internAware) {
				e.writeHeader()
				if pure {
					return e.encodeColumnar(colPlan, hdr.Data, n)
				}
				return e.encodeHybridColumnar(colPlan, hdr.Data, n)
			}
		}
		e.WriteArrayHeader(n)
		base := hdr.Data
		// Probe-and-grow for large slices: encode the first
		// sliceProbeSize records, measure the per-record buffer
		// growth, then pre-grow the output buffer for the rest in
		// one shot. Eliminates the log(n) doubling chain
		// (runtime.memmove + madvise) that dominated 50 k-record
		// encodes — at scale the buffer can balloon from 4 KiB to
		// 60 MiB through ~14 doublings, copying ~4× the final size.
		// The probe cost is negligible because the same elements
		// are emitted exactly once.
		const sliceProbeSize = 32
		if n <= sliceProbeSize {
			for i := range n {
				if err := elem.encode(e, unsafe.Add(base, uintptr(i)*stride)); err != nil {
					return err
				}
			}
			return nil
		}
		probeStart := len(e.buf)
		for i := range sliceProbeSize {
			if err := elem.encode(e, unsafe.Add(base, uintptr(i)*stride)); err != nil {
				return err
			}
		}
		probeBytes := len(e.buf) - probeStart
		if probeBytes > 0 {
			// Project total size from probe + 25 % slack so the
			// growslice short-circuit fires inside slices.Grow
			// without forcing another doubling on a slight
			// underestimate.
			remaining := n - sliceProbeSize
			// 64-bit intermediate: probeBytes*remaining can exceed int32 on a
			// 32-bit build for a large slice, wrapping projected negative and
			// panicking slices.Grow.
			projected := int(int64(probeBytes) * int64(remaining) / int64(sliceProbeSize))
			projected += projected >> 2
			e.buf = slices.Grow(e.buf, projected)
		}
		for i := sliceProbeSize; i < n; i++ {
			if err := elem.encode(e, unsafe.Add(base, uintptr(i)*stride)); err != nil {
				return err
			}
		}
		return nil
	}
}

func decodeSlice(t reflect.Type, elem *typeDesc, stride uintptr, colPlan *columnarPlan) func(*Decoder, unsafe.Pointer) error {
	// elemDynamic is true when the slice element is map[string]any or any, so a
	// columnar (tagColStruct) payload can be decoded dynamically into row maps
	// via decodeColumnarAny even though the static element type carries no
	// columnarPlan. This is the path UnmarshalColumns into *[]map[string]any
	// (or *[]any) takes.
	elemType := t.Elem()
	elemDynamic := elemType == reflect.TypeFor[map[string]any]() || elemType.Kind() == reflect.Interface
	elemPF := noPointers(elemType)     // gate for backing reuse (computed once per type)
	elemHasMap := typeDescHasMap(elem) // gate the map-recycle harvest (once per type)
	return func(d *Decoder, p unsafe.Pointer) error {
		if d.decodeNilSlice(p) { // tagNil → nil slice (distinct from empty)
			return nil
		}
		if err := d.descend(); err != nil {
			return err
		}
		defer d.ascend()
		// Batched lossy vector-column dispatch (tagVecBatchStruct). Works even when
		// colPlan is nil (a vector-only struct is not columnar-eligible).
		if elem != nil && len(elem.vecFields) > 0 {
			if tag, err := d.peekTag(); err == nil && tag == tagVecBatchStruct {
				if elemDynamic || d.query != nil {
					return ErrUnsupported // v1: no schemaless / query decode of a batched block
				}
				return d.decodeVectorBatchStruct(t, elem, p)
			}
		}
		// Columnar / hybrid-columnar decode dispatch.
		if colPlan != nil {
			if tag, err := d.peekTag(); err == nil {
				switch tag {
				case tagColStruct:
					if d.query != nil {
						return decodeColumnarQuery(d, t, colPlan, p)
					}
					return decodeColumnar(d, t, colPlan, p)
				case tagHybridColStruct:
					if d.query != nil {
						return ErrUnsupported // v1: query/Select over a hybrid payload
					}
					return decodeHybridColumnar(d, t, colPlan, p)
				}
			}
		}
		if elemDynamic {
			if tag, err := d.peekTag(); err == nil && (tag == tagColStruct || tag == tagHybridColStruct) {
				var rows any
				var err error
				switch {
				case tag == tagHybridColStruct:
					if d.query != nil {
						return ErrUnsupported // v1: query over a hybrid payload
					}
					rows, err = decodeHybridColumnarAny(d)
				case d.query != nil:
					rows, err = decodeColumnarQueryAny(d)
				default:
					rows, err = decodeColumnarAny(d)
				}
				if err != nil {
					return err
				}
				src := rows.([]any)
				reflectutil.MakeSlice(t, len(src), p)
				base := reflectutil.SliceData(t, p)
				for i, row := range src {
					reflect.NewAt(elemType, unsafe.Add(base, uintptr(i)*stride)).Elem().Set(reflect.ValueOf(row))
				}
				return nil
			}
		}
		// A query requires a columnar payload (tagColStruct, handled above).
		// Any non-columnar slice shape that reaches here cannot be filtered.
		if d.query != nil {
			return &QueryError{Op: "predicate pushdown", Err: ErrUnsupported}
		}
		n, err := d.ReadArrayHeader()
		if err != nil {
			return err
		}
		if err := d.CheckLength(n, 1); err != nil {
			return err
		}
		// Harvest reusable maps from the elements decode-slice-reuse is about to
		// zero, so the map decoders can recycle them instead of re-allocating.
		// Gated on elemHasMap so a map-free element type pays nothing.
		if elemHasMap {
			if old := (*sliceHeader)(p); old.Data != nil && old.Len > 0 {
				harvestMaps(d, elem, old.Data, stride, old.Len)
			}
		}
		// Reuse the caller's backing when pointer-free + cap suffices (decode
		// into a pre-sized/pooled slice), else fresh MakeSlice.
		base := reuseOrMakeSlice(t, n, p, stride, elemPF)
		for i := range n {
			if err := elem.decode(d, unsafe.Add(base, uintptr(i)*stride)); err != nil {
				return err
			}
		}
		return nil
	}
}

// encodeFixedByteArray encodes a fixed [N]byte array as ONE contiguous binary
// blob (identical wire to a []byte of length N), not N tagged elements. Real ID
// bytes are uniformly 0..255; the generic per-element path costs ~2 bytes for
// every byte >= 128, so a [16]byte trace id bloats to ~32 wire bytes — this
// writes a flat 16 plus a tiny bin header. One memcpy, no per-element calls.
//
//go:nosplit
func encodeFixedByteArray(n int) func(*Encoder, unsafe.Pointer) error {
	return func(e *Encoder, p unsafe.Pointer) error {
		e.WriteBytes(unsafe.Slice((*byte)(p), n))
		return nil
	}
}

// decodeFixedByteArray reads the blob written by encodeFixedByteArray straight
// into the struct's inline [N]byte storage — one length-checked memcpy, zero
// allocation (the array lives in the caller's struct, never on the heap).
func decodeFixedByteArray(n int) func(*Decoder, unsafe.Pointer) error {
	return func(d *Decoder, p unsafe.Pointer) error {
		b, err := d.readStringBytes()
		if err != nil {
			return err
		}
		if len(b) != n {
			return ErrTypeMismatch
		}
		copy(unsafe.Slice((*byte)(p), n), b)
		return nil
	}
}

func encodeArray(elem *typeDesc, stride uintptr, n int) func(*Encoder, unsafe.Pointer) error {
	return func(e *Encoder, p unsafe.Pointer) error {
		// Depth guard mirroring Decoder.descend in decodeArray (see encodeSlice).
		if e.maxDepth != 0 {
			e.depth++
			if e.depth > e.maxDepth {
				e.depth--
				return ErrCycleDetected
			}
			defer func() { e.depth-- }()
		}
		e.WriteArrayHeader(n)
		for i := range n {
			if err := elem.encode(e, unsafe.Add(p, uintptr(i)*stride)); err != nil {
				return err
			}
		}
		return nil
	}
}
func decodeArray(elem *typeDesc, stride uintptr, n int) func(*Decoder, unsafe.Pointer) error {
	return func(d *Decoder, p unsafe.Pointer) error {
		if err := d.descend(); err != nil {
			return err
		}
		defer d.ascend()
		m, err := d.ReadArrayHeader()
		if err != nil {
			return err
		}
		if m != n {
			return ErrTypeMismatch
		}
		for i := range n {
			if err := elem.decode(d, unsafe.Add(p, uintptr(i)*stride)); err != nil {
				return err
			}
		}
		return nil
	}
}

func encodeMap(t reflect.Type, k, v *typeDesc) func(*Encoder, unsafe.Pointer) error {
	keyType := t.Key()
	valType := t.Elem()
	stringKey := keyType.Kind() == reflect.String
	return func(e *Encoder, p unsafe.Pointer) error {
		// Depth guard mirroring Decoder.descend in decodeMap (see encodeSlice).
		if e.maxDepth != 0 {
			e.depth++
			if e.depth > e.maxDepth {
				e.depth--
				return ErrCycleDetected
			}
			defer func() { e.depth-- }()
		}
		rv := reflect.NewAt(t, p).Elem()
		if rv.IsNil() {
			e.WriteNil()
			return nil
		}
		n := rv.Len()
		// Canonical emit takes precedence over the OptMapShape/OptDense shape
		// branch so exactly one deterministic, sorted emit happens regardless of
		// the other shape bits. It reuses the same key/value encode closures
		// (k.encode / v.encode) as the default path — only the iteration ORDER
		// changes; the value codec, intern, and dict are untouched.
		if n > 0 && e.opts.Has(OptCanonical) {
			return e.encodeMapCanonical(rv, keyType, valType, k, v)
		}
		if stringKey && n > 0 && e.state != nil && !e.stateSuspended && e.opts.Has(OptMapShape) && e.opts.Has(OptDense) {
			return e.encodeStringMapShaped(rv, keyType, valType, v)
		}
		e.WriteMapHeader(n)
		// MapRange beats reflect.Value.Seq2 (Go 1.26) here by ~2x on
		// throughput and ~2x on allocations: Seq2 boxes the (k, v)
		// pair into closure arguments per yield, while MapRange
		// reuses a single *MapIter and exposes Key/Value via
		// reflect.Value (struct, no per-element heap). See
		// BenchmarkMapIter_MapRangeOriginal vs BenchmarkMapIter_Seq2.
		//
		// SetIterKey/SetIterValue (Go 1.18+) write the current map
		// iter entry into a pre-allocated addressable reflect.Value.
		// Without them, reflectValueAddr would have to materialise a
		// fresh reflect.New(T).Elem() per element — 2 allocs per map
		// entry on the previous path, O(N) total.
		//
		// The two scratch holders are pooled on encState (reused across the
		// rows of a []struct — no reflect.New per map) when a state exists;
		// re-entrancy-safe via the busy flag. Fast mode (e.state == nil) has
		// no pool, so it falls back to a local pair.
		var keyHolder, valHolder reflect.Value
		var kp, vp unsafe.Pointer
		var pooled bool
		if e.state != nil {
			keyHolder, valHolder, vp, pooled = e.state.mapEnc.acquire(keyType, valType)
			kp = unsafe.Pointer(keyHolder.UnsafeAddr())
			defer e.state.mapEnc.release(pooled)
		} else {
			keyHolder = reflect.New(keyType).Elem()
			valHolder = reflect.New(valType).Elem()
			kp = unsafe.Pointer(keyHolder.UnsafeAddr())
			vp = unsafe.Pointer(valHolder.UnsafeAddr())
		}
		iter := rv.MapRange()
		for iter.Next() {
			keyHolder.SetIterKey(iter)
			valHolder.SetIterValue(iter)
			if err := k.encode(e, kp); err != nil {
				return err
			}
			if err := v.encode(e, vp); err != nil {
				return err
			}
		}
		return nil
	}
}

// encodeMapCanonical emits a map under OptCanonical: keys in sorted order so
// logically-equal maps serialize byte-identically. It writes the same plain map
// header and reuses the SAME key/value encode closures as the default path
// (k.encode / v.encode) — only the order changes. A single reused addressable
// key holder fetches each value via rv.MapIndex(kh), with no per-lookup boxing
// allocation. The common integer/string/bool key kinds use a type-specialized,
// pooled, monomorphized sort; float and exotic comparable key kinds (struct /
// array / interface) take a slow reflect-comparator fallback (rare).
func (e *Encoder) encodeMapCanonical(rv reflect.Value, keyType, valType reflect.Type, k, v *typeDesc) error {
	n := rv.Len()
	e.WriteMapHeader(n)

	// Reused holders: a key holder set per sorted key for MapIndex, and a value
	// holder the value codec reads through (addressable, so v.encode can take its
	// pointer). Pooled on the state when available (no reflect.New per map row).
	var keyHolder, valHolder reflect.Value
	var vp unsafe.Pointer
	var pooled bool
	if e.state != nil {
		keyHolder, valHolder, vp, pooled = e.state.mapEnc.acquire(keyType, valType)
		defer e.state.mapEnc.release(pooled)
	} else {
		keyHolder = reflect.New(keyType).Elem()
		valHolder = reflect.New(valType).Elem()
		vp = unsafe.Pointer(valHolder.UnsafeAddr())
	}
	kp := unsafe.Pointer(keyHolder.UnsafeAddr())

	// emit fetches the value for the (already-set) key holder, copies it into the
	// value holder, and runs both encode closures. MapIndex returns a fresh
	// non-addressable Value; valHolder.Set copies it into the addressable holder
	// the codec reads through vp.
	emit := func() error {
		valHolder.Set(rv.MapIndex(keyHolder))
		if err := k.encode(e, kp); err != nil {
			return err
		}
		return v.encode(e, vp)
	}

	switch keyType.Kind() {
	case reflect.String:
		keys, pooled := e.gatherStringKeys(rv)
		defer e.canonKeysRelease(pooled)
		for _, key := range keys {
			keyHolder.SetString(key)
			if err := emit(); err != nil {
				return err
			}
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		keys, pooled := e.gatherIntKeys(rv)
		defer e.canonKeysRelease(pooled)
		for _, key := range keys {
			keyHolder.SetInt(key)
			if err := emit(); err != nil {
				return err
			}
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		keys, pooled := e.gatherUintKeys(rv)
		defer e.canonKeysRelease(pooled)
		for _, key := range keys {
			keyHolder.SetUint(key)
			if err := emit(); err != nil {
				return err
			}
		}
	case reflect.Bool:
		// At most two keys; emit false then true if present (no sort needed).
		for _, b := range [...]bool{false, true} {
			keyHolder.SetBool(b)
			mv := rv.MapIndex(keyHolder)
			if !mv.IsValid() {
				continue
			}
			valHolder.Set(mv)
			if err := k.encode(e, kp); err != nil {
				return err
			}
			if err := v.encode(e, vp); err != nil {
				return err
			}
		}
	default:
		// Float and exotic comparable key kinds (struct / array / interface):
		// gather (key, value) PAIRS via MapRange and sort by key. Slow but rare;
		// correctness over speed. We must NOT re-fetch the value via MapIndex here:
		// a NaN float key (NaN != NaN) is unfindable by MapIndex, which would return
		// an invalid Value and panic on Set. MapRange yields every entry, NaN keys
		// included, so carrying the value alongside the key avoids the lookup.
		type kvPair struct{ k, v reflect.Value }
		pairs := make([]kvPair, 0, rv.Len())
		for it := rv.MapRange(); it.Next(); {
			pairs = append(pairs, kvPair{it.Key(), it.Value()})
		}
		slices.SortFunc(pairs, func(x, y kvPair) int { return canonReflectKeyCompare(x.k, y.k) })
		for i := range pairs {
			keyHolder.Set(pairs[i].k)
			valHolder.Set(pairs[i].v)
			if err := k.encode(e, kp); err != nil {
				return err
			}
			if err := v.encode(e, vp); err != nil {
				return err
			}
		}
	}
	return nil
}

// gatherStringKeys collects the map's string keys, sorted. The returned slice is
// valid until the caller calls canonKeysRelease(pooled). pooled is true when the
// shared canonKeysStr scratch was used (the zero-alloc flat-map case); it is
// false for a nested re-entrant gather (a map whose values contain maps), which
// allocates a fresh local slice so it cannot clobber the outer map's keys still
// being iterated. The caller MUST release after it finishes iterating.
func (e *Encoder) gatherStringKeys(rv reflect.Value) (keys []string, pooled bool) {
	var buf []string
	if e.state != nil && !e.state.canonKeysBusy {
		buf = e.state.canonKeysStr[:0]
		pooled = true
	}
	it := rv.MapRange()
	for it.Next() {
		buf = append(buf, it.Key().String())
	}
	slices.Sort(buf)
	if pooled {
		e.state.canonKeysStr = buf
		e.state.canonKeysBusy = true
	}
	return buf, pooled
}

func (e *Encoder) gatherIntKeys(rv reflect.Value) (keys []int64, pooled bool) {
	var buf []int64
	if e.state != nil && !e.state.canonKeysBusy {
		buf = e.state.canonKeysI64[:0]
		pooled = true
	}
	it := rv.MapRange()
	for it.Next() {
		buf = append(buf, it.Key().Int())
	}
	slices.Sort(buf)
	if pooled {
		e.state.canonKeysI64 = buf
		e.state.canonKeysBusy = true
	}
	return buf, pooled
}

func (e *Encoder) gatherUintKeys(rv reflect.Value) (keys []uint64, pooled bool) {
	var buf []uint64
	if e.state != nil && !e.state.canonKeysBusy {
		buf = e.state.canonKeysU64[:0]
		pooled = true
	}
	it := rv.MapRange()
	for it.Next() {
		buf = append(buf, it.Key().Uint())
	}
	slices.Sort(buf)
	if pooled {
		e.state.canonKeysU64 = buf
		e.state.canonKeysBusy = true
	}
	return buf, pooled
}

// canonKeysRelease clears the re-entrancy guard set by a pooled gather. A no-op
// when the gather allocated a fresh local slice (pooled false).
func (e *Encoder) canonKeysRelease(pooled bool) {
	if pooled && e.state != nil {
		e.state.canonKeysBusy = false
	}
}

// canonReflectKeyCompare is a stable total order over comparable reflect key
// values for the slow canonical fallback (float / struct / array / interface
// keys). Floats compare by canonicalized value then raw bits (so -0.0 and +0.0
// tie and distinct NaNs collapse). Structs/arrays compare field/element-wise.
// Other kinds fall back to a string rendering, which is stable within a type.
func canonReflectKeyCompare(a, b reflect.Value) int {
	// Kind-mismatch guard: interface keys (map[any]K) unwrap to differing dynamic
	// kinds, and an invalid (nil-interface) Value has Kind Invalid. Order by kind
	// ordinal so the kind-specific accessors below are never called on the wrong
	// kind (e.g. b.Int() on a float64) — that would panic.
	if a.Kind() != b.Kind() {
		return cmp.Compare(int(a.Kind()), int(b.Kind()))
	}
	switch a.Kind() {
	case reflect.Float32, reflect.Float64:
		// Order by raw bits — a strict total order even across NaN payloads (so
		// distinct NaN keys sort deterministically, not tie). -0.0 and +0.0 are the
		// SAME Go map key so never coexist. The EMITTED key bytes are normalized
		// separately by the float choke points (WriteFloat*); ordering only needs
		// to be deterministic, not numeric.
		return cmp.Compare(math.Float64bits(a.Float()), math.Float64bits(b.Float()))
	case reflect.String:
		return cmp.Compare(a.String(), b.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return cmp.Compare(a.Int(), b.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return cmp.Compare(a.Uint(), b.Uint())
	case reflect.Bool:
		if a.Bool() == b.Bool() {
			return 0
		}
		if !a.Bool() {
			return -1
		}
		return 1
	case reflect.Struct:
		for i := range a.NumField() {
			if c := canonReflectKeyCompare(a.Field(i), b.Field(i)); c != 0 {
				return c
			}
		}
		return 0
	case reflect.Array:
		for i := range a.Len() {
			if c := canonReflectKeyCompare(a.Index(i), b.Index(i)); c != 0 {
				return c
			}
		}
		return 0
	case reflect.Interface, reflect.Pointer:
		return canonReflectKeyCompare(a.Elem(), b.Elem())
	default:
		// Stable rendering for any residual comparable kind (complex, chan, etc.).
		return cmp.Compare(reflectRender(a), reflectRender(b))
	}
}

func reflectRender(v reflect.Value) string {
	if !v.IsValid() {
		return ""
	}
	return v.Type().String() + ":" + valueRenderString(v)
}

func valueRenderString(v reflect.Value) string {
	switch v.Kind() {
	case reflect.String:
		return v.String()
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

// encodeStringMapShaped emits a string-keyed map via tagMapShape (OptMapShape).
// A recurring key-set declares once (keys interned); reuses emit only the shape
// ID + values in canonical (sorted) key order. Collision-safe: the set-hash
// only finds a candidate; the keys are verified present (count + membership)
// before reuse, falling back to a fresh declaration on any mismatch. This is
// the general reflect path; the common map[string]string/etc. types take the
// concrete fast path in maps_fast_generated.go.
//
// A reusable keyHolder/valHolder avoids per-key reflect.ValueOf boxing; the hot
// reuse path allocates nothing (no keys slice — values are fetched in the bound
// canonical order). The keys slice is built only on a fresh declaration (rare:
// once per distinct key-set).
func (e *Encoder) encodeStringMapShaped(rv reflect.Value, keyType, valType reflect.Type, v *typeDesc) error {
	// Ensure the stream header precedes the first tag (top-level map case; the
	// plain path emits it via WriteMapHeader). Idempotent.
	e.writeHeader()
	n := rv.Len()
	st := e.state
	// Pooled, re-entrancy-safe key/value holders — avoids reflect.New per map
	// (a 1000-row batch pays 2 reflect.New total, not 2 per row).
	keyHolder, valHolder, vp, pooled := st.mapEnc.acquire(keyType, valType)
	defer st.mapEnc.release(pooled)

	emitValues := func(order []string) error {
		for _, name := range order {
			keyHolder.SetString(name)
			valHolder.Set(rv.MapIndex(keyHolder))
			if err := v.encode(e, vp); err != nil {
				return err
			}
		}
		return nil
	}
	// hasAll reports whether the map's key-set equals order (caller checks
	// len==n). Verifies via the map's own keys under MapRange — string keys do
	// not allocate — instead of MapIndex per key, which would heap-copy a
	// struct value type. The linear scan is O(n²) but n (a key-set) is tiny.
	hasAll := func(order []string) bool {
		it := rv.MapRange()
		for it.Next() {
			keyHolder.SetIterKey(it) // SetIterKey reuses the holder; it.Key() would alloc
			k := keyHolder.String()
			found := slices.Contains(order, k)
			if !found {
				return false
			}
		}
		return true
	}

	// Fast path: same key-set as the previous map — verify directly, no hash.
	if st.lastMapShapeID != 0 && len(st.lastMapShapeKeys) == n && hasAll(st.lastMapShapeKeys) {
		e.buf = append(e.buf, tagMapShape)
		e.buf = appendUvarint(e.buf, uint64(st.lastMapShapeID))
		return emitValues(st.lastMapShapeKeys)
	}

	// Key-set changed: order-independent set-hash to find an earlier shape.
	var setHash uint64
	iter := rv.MapRange()
	for iter.Next() {
		keyHolder.SetIterKey(iter) // reuse holder; iter.Key() would alloc
		setHash += internKeyHash(keyHolder.String())
	}
	// Scan EVERY shape with this (setHash, n) and verify by keys — setHash is not
	// collision-proof. Returning only the first (setHash, n) row and declaring on
	// a key mismatch would re-declare a colliding key-set on every encode: under
	// two alternating sets that collide on setHash, the already-registered second
	// set is never found again, so mapShapes grows without bound. The s.n == n
	// filter guarantees len(s.keys) == n, so hasAll ⇒ set equality.
	for i := range st.mapShapes {
		s := &st.mapShapes[i]
		if s.setHash == setHash && s.n == n && hasAll(s.keys) {
			st.lastMapShapeID, st.lastMapShapeKeys = s.id, s.keys
			e.buf = append(e.buf, tagMapShape)
			e.buf = appendUvarint(e.buf, uint64(s.id))
			return emitValues(s.keys)
		}
	}

	// Declare path (first sight of this key-set).
	keys := make([]string, 0, n)
	it2 := rv.MapRange()
	for it2.Next() {
		keyHolder.SetIterKey(it2) // reuse holder; it2.Key() would alloc
		keys = append(keys, keyHolder.String())
	}
	slices.Sort(keys)
	id := st.shapeDeclareEnc()
	st.mapShapeRegister(setHash, n, keys, id)
	st.lastMapShapeID, st.lastMapShapeKeys = id, keys
	e.buf = append(e.buf, tagMapShape)
	e.buf = appendUvarint(e.buf, 0)
	e.buf = appendUvarint(e.buf, uint64(n))
	for _, name := range keys {
		e.WriteString(name)
	}
	return emitValues(keys)
}

func decodeMap(t reflect.Type, k, v *typeDesc) func(*Decoder, unsafe.Pointer) error {
	keyType := t.Key()
	valType := t.Elem()
	return func(d *Decoder, p unsafe.Pointer) error {
		if err := d.descend(); err != nil {
			return err
		}
		defer d.ascend()
		tag, err := d.peekTag()
		if err != nil {
			return err
		}
		if tag == tagNil {
			d.i++
			reflect.NewAt(t, p).Elem().Set(reflect.Zero(t))
			return nil
		}
		// tagMapShape: string-keyed map encoded via the key-set shape codec
		// (OptMapShape). Mirrors decodeStruct's shape branch; the decoder's
		// shape table is destination-agnostic (ordered names + N values).
		if tag == tagMapShape && keyType.Kind() == reflect.String {
			names, err := decodeMapStringShapeHeader(d)
			if err != nil {
				return err
			}
			reuseOrMakeMapReflect(d, t, len(names), p)
			mapVal := reflect.NewAt(t, p).Elem()
			// Pooled, re-entrancy-safe holders hoisted out of the loop:
			// SetMapIndex copies key/value into the map, so one pair is reused
			// for every entry (and across rows) — no reflect.New per entry.
			kh, vh, vp, pooled := d.state.mapDec.acquire(keyType, valType)
			defer d.state.mapDec.release(pooled)
			for _, name := range names {
				kh.SetString(name)
				// Reset the value holder each entry: a slice-backed value type
				// would otherwise have its backing array reused across entries
				// (reuseOrMakeSlice keeps a cap>=n backing), so every map value
				// would alias the last one decoded. Zeroing forces a fresh
				// MakeSlice per entry. Keys can't contain slices (comparable).
				vh.SetZero()
				if err := v.decode(d, vp); err != nil {
					return err
				}
				mapVal.SetMapIndex(kh, vh)
			}
			return nil
		}
		n, err := d.ReadMapHeader()
		if err != nil {
			return err
		}
		if err := d.CheckLength(n, 2); err != nil {
			return err
		}
		// Allocate via the swappable reflectutil backend.
		reuseOrMakeMapReflect(d, t, n, p)
		mapVal := reflect.NewAt(t, p).Elem()
		// Hoist the key/value holders out of the loop: SetMapIndex copies them
		// into the map, so one addressable pair is reused for every entry.
		// Previously this did reflect.New twice PER ENTRY (2 allocs × n);
		// now it is 2 per map. The locals stay re-entrancy-safe — a nested map
		// value decodes through its own decodeMap call with its own holders.
		kv := reflect.New(keyType).Elem()
		vv := reflect.New(valType).Elem()
		kp := unsafe.Pointer(kv.UnsafeAddr())
		vp := unsafe.Pointer(vv.UnsafeAddr())
		for range n {
			if err := k.decode(d, kp); err != nil {
				return err
			}
			// Reset the value holder each entry so a slice-backed value type does
			// not reuse the previous entry's backing array (reuseOrMakeSlice keeps
			// a cap>=n backing) and alias every map value onto the last one. Keys
			// can't contain slices (map keys are comparable), so kv needs no reset.
			vv.SetZero()
			if err := v.decode(d, vp); err != nil {
				return err
			}
			mapVal.SetMapIndex(kv, vv)
		}
		return nil
	}
}

func encodePtr(elem *typeDesc) func(*Encoder, unsafe.Pointer) error {
	return func(e *Encoder, p unsafe.Pointer) error {
		raw := *(*unsafe.Pointer)(p)
		if raw == nil {
			e.WriteNil()
			return nil
		}
		// Depth-based cycle guard. Cheaper than a per-pointer set and
		// catches both genuine *T->*T cycles and pathologically deep
		// payloads. maxDepth==0 disables the check for callers that
		// know their input is acyclic.
		if e.maxDepth != 0 {
			e.depth++
			if e.depth > e.maxDepth {
				e.depth--
				return ErrCycleDetected
			}
			defer func() { e.depth-- }()
		}
		return elem.encode(e, raw)
	}
}
func decodePtr(t reflect.Type, elem *typeDesc) func(*Decoder, unsafe.Pointer) error {
	return func(d *Decoder, p unsafe.Pointer) error {
		if err := d.descend(); err != nil {
			return err
		}
		defer d.ascend()
		tag, err := d.peekTag()
		if err != nil {
			return err
		}
		if tag == tagNil {
			d.i++
			*(*unsafe.Pointer)(p) = nil
			return nil
		}
		// Allocate via reflect for GC-safety.
		nv := reflect.New(t.Elem())
		if err := elem.decode(d, unsafe.Pointer(nv.Elem().UnsafeAddr())); err != nil {
			return err
		}
		reflect.NewAt(t, p).Elem().Set(nv)
		return nil
	}
}

func encodeStruct(td *typeDesc) func(*Encoder, unsafe.Pointer) error {
	fields := td.fields
	return func(e *Encoder, p unsafe.Pointer) error {
		e.writeHeader()
		n := len(fields)
		// Dense mode: route through the tagMapShape path when
		// OptShapeIntern is set. On the first emit of a struct type the
		// encoder declares the shape (writing field names through the
		// normal intern path); on every subsequent emit it writes only
		// 0xEC + shapeID + values. With OptShapeIntern off, Dense
		// falls back to the tagMap8/16/32 encoding so the rest of the
		// state stack (intern + Markov / MTF / Pair) still applies
		// per-field.
		if e.opts.Has(OptDense) && e.state != nil && !e.stateSuspended && e.opts.Has(OptShapeIntern) {
			if id := e.state.shapeForType(td); id != 0 {
				e.buf = append(e.buf, tagMapShape)
				e.buf = appendUvarint(e.buf, uint64(id))
				for i := range fields {
					f := &fields[i]
					if err := f.desc.encode(e, unsafe.Add(p, f.offset)); err != nil {
						return err
					}
				}
				return nil
			}
			// First time: declare and emit keys via the standard intern path.
			shapeID := e.state.shapeDeclareEnc()
			e.state.shapeBindType(td, shapeID)
			st := e.state
			e.buf = append(e.buf, tagMapShape)
			e.buf = appendUvarint(e.buf, 0) // 0 ⇒ declaration follows
			e.buf = appendUvarint(e.buf, uint64(n))
			pairOn := e.pairPred
			for i := range fields {
				f := &fields[i]
				if len(f.name) >= e.minIntern && int(st.internLoad) < e.maxStateEntries {
					if id, ok := st.lookupOrAssign(f.name); ok {
						if st.lastID == id {
							e.buf = append(e.buf, tagStateRepeat)
							if pairOn {
								st.pairRecord(id, id)
							}
						} else {
							e.emitStateRef(id)
						}
					} else {
						e.buf = append(e.buf, f.preInternStr...)
						if st.lastID != lruInvalidID && pairOn {
							st.pairRecord(st.lastID, id)
						}
						st.lastID = id
					}
				} else {
					e.buf = append(e.buf, f.preFast...)
					st.lastID = lruInvalidID
				}
			}
			for i := range fields {
				f := &fields[i]
				if err := f.desc.encode(e, unsafe.Add(p, f.offset)); err != nil {
					return err
				}
			}
			return nil
		}
		// Dense without OptShapeIntern: tagMap8/16/32 header + per-field
		// intern via WriteString. Keys still go through the intern
		// path so state-ref / MTF / Pair codecs cover them when their
		// options are on.
		if e.opts.Has(OptDense) && e.state != nil {
			e.WriteMapHeader(n)
			st := e.state
			pairOn := e.pairPred
			for i := range fields {
				f := &fields[i]
				if len(f.name) >= e.minIntern && !e.stateSuspended && int(st.internLoad) < e.maxStateEntries {
					if id, ok := st.lookupOrAssign(f.name); ok {
						if st.lastID == id {
							e.buf = append(e.buf, tagStateRepeat)
							if pairOn {
								st.pairRecord(id, id)
							}
						} else {
							e.emitStateRef(id)
						}
					} else {
						e.buf = append(e.buf, f.preInternStr...)
						if st.lastID != lruInvalidID && pairOn {
							st.pairRecord(st.lastID, id)
						}
						st.lastID = id
					}
				} else {
					e.buf = append(e.buf, f.preFast...)
					st.lastID = lruInvalidID
				}
				if err := f.desc.encode(e, unsafe.Add(p, f.offset)); err != nil {
					return err
				}
			}
			return nil
		}
		// Fast mode (no Dense): plain tagMap8/16/32 encoding — no
		// intern, no shape, fixstr field-name headers from preFast.
		e.WriteMapHeader(n)
		for i := range fields {
			f := &fields[i]
			e.buf = append(e.buf, f.preFast...)
			if err := f.desc.encode(e, unsafe.Add(p, f.offset)); err != nil {
				return err
			}
		}
		return nil
	}
}

func decodeStruct(td *typeDesc) func(*Decoder, unsafe.Pointer) error {
	// Build a name → field-index lookup so decode order doesn't have to
	// match encode order. Sorted lookup keeps it cache-friendly.
	type idx struct {
		f    *fieldDesc
		name string
	}
	indexed := make([]idx, len(td.fields))
	for i := range td.fields {
		indexed[i] = idx{f: &td.fields[i], name: td.fields[i].name}
	}
	// Linear scan is fine for ≤16 fields; for wide structs (rare), we use a
	// small map.
	useMap := len(indexed) > 16
	var byName map[string]*fieldDesc
	if useMap {
		byName = make(map[string]*fieldDesc, len(indexed))
		for i := range indexed {
			byName[indexed[i].name] = indexed[i].f
		}
	}

	resolveField := func(name string) *fieldDesc {
		if useMap {
			return byName[name]
		}
		for j := range indexed {
			if indexed[j].name == name {
				return indexed[j].f
			}
		}
		return nil
	}

	return func(d *Decoder, p unsafe.Pointer) error {
		tag, err := d.peekTag()
		if err != nil {
			return err
		}
		if tag == tagNil {
			d.i++
			return nil
		}
		// tagMapShape path: structs encoded via the Dense shape codec.
		if tag == tagMapShape {
			d.i++
			shapeID, n := readUvarint(d.buf[d.i:])
			if n <= 0 {
				return ErrInvalidLength
			}
			if shapeID > uint64(math.MaxUint32) {
				return ErrUnknownStateID // would truncate on the uint32 cast below
			}
			d.i += n
			if d.state == nil {
				d.state = newDecState()
			}
			var fieldNames []string
			if shapeID == 0 {
				// Declaration: read count, then N keys, then N values.
				cnt64, n := readUvarint(d.buf[d.i:])
				if n <= 0 {
					return ErrInvalidLength
				}
				d.i += n
				if cnt64 > uint64(math.MaxInt) { // 32-bit: int(cnt64) would wrap before CheckLength
					return ErrInvalidLength
				}
				cnt := int(cnt64)
				if err := d.CheckLength(cnt, 1); err != nil {
					return err
				}
				sh := d.state.shapeDeclare()
				keys := make([]string, 0, cnt)
				for range cnt {
					kb, err := d.readStringBytes()
					if err != nil {
						return err
					}
					keys = append(keys, d.keyCache.Make(kb))
				}
				sh.names = keys
				fieldNames = keys
			} else {
				sh := d.state.shapeLookup(uint32(shapeID))
				if sh == nil {
					return ErrUnknownStateID
				}
				fieldNames = sh.names
			}
			cur := 0
			for _, name := range fieldNames {
				var fd *fieldDesc
				if cur < len(indexed) && indexed[cur].name == name {
					fd = indexed[cur].f
					cur++
				} else {
					fd = resolveField(name)
				}
				if fd == nil {
					if err := d.Skip(); err != nil {
						return err
					}
					continue
				}
				if err := fd.desc.decode(d, unsafe.Add(p, fd.offset)); err != nil {
					return err
				}
			}
			return nil
		}
		// tagMap8/16/32 path — used by Fast mode, by Dense without
		// OptShapeIntern, and by any external encoder that does not
		// emit shape headers.
		n, err := d.ReadMapHeader()
		if err != nil {
			return err
		}
		// Fields arrive in struct-declaration order in the common case (qdf
		// encodes them that way), so try the expected next field before the
		// map/linear lookup. An in-order hit is one string compare — no hash,
		// no map access. cur is left unchanged on a miss so a skipped unknown
		// field doesn't desync the cursor for the following in-order fields.
		cur := 0
		for range n {
			kb, err := d.readStringBytes()
			if err != nil {
				return err
			}
			name := unsafestr.String(kb)
			var fd *fieldDesc
			if cur < len(indexed) && indexed[cur].name == name {
				fd = indexed[cur].f
				cur++
			} else {
				fd = resolveField(name)
			}
			if fd == nil {
				if err := d.Skip(); err != nil {
					return err
				}
				continue
			}
			if err := fd.desc.decode(d, unsafe.Add(p, fd.offset)); err != nil {
				return err
			}
		}
		return nil
	}
}

// encodeIface dispatches on the dynamic type of an interface{}. Slow path.
func encodeIface(e *Encoder, p unsafe.Pointer) error {
	iv := *(*any)(p)
	if iv == nil {
		e.WriteNil()
		return nil
	}
	// Bound recursion through the dynamic (interface) dispatch. encodePtr guards
	// the static *T path, but a cycle routed through an any-typed field re-enters
	// the reflect machinery here without touching that counter; without this a
	// self-referential graph (a.Next = a, Next any) recurses unbounded into a
	// fatal stack overflow. This is the interface chokepoint — every []any /
	// map[K]any / any-field recursion funnels through it.
	if e.maxDepth != 0 {
		e.depth++
		if e.depth > e.maxDepth {
			e.depth--
			return ErrCycleDetected
		}
		defer func() { e.depth-- }()
	}
	// Mark the schemaless context: everything under a dynamic dispatch decodes via
	// decodeAny, which cannot read a batched vector-column block. Balanced inc/dec
	// so nested ifaces stay positive and the counter returns to 0 at the top.
	e.ifaceDepth++
	defer func() { e.ifaceDepth-- }()
	return encodeReflect(e, iv)
}

// encodeSliceAny encodes a []any without the per-value reflect.New that the
// generic encodeReflect path would take for the slice header. It is byte- and
// behavior-identical to the descOf([]any).encode closure (encodeSlice): a []any
// element carries no columnar plan, so that path reduces to nil-guard + depth
// guard + WriteArrayHeader + per-element encodeIface — exactly what this does,
// including the probe-and-grow buffer pre-sizing that avoids the log(n) realloc
// chain on large arrays. The pointer is not retained, so the caller's &tv stays
// on the stack.
func encodeSliceAny(e *Encoder, p unsafe.Pointer) error {
	if e.encodeNilSlice(p) { // nil slice → tagNil (distinct from empty)
		return nil
	}
	// Depth guard mirroring encodeSlice (and Decoder.descend in decodeSlice).
	if e.maxDepth != 0 {
		e.depth++
		if e.depth > e.maxDepth {
			e.depth--
			return ErrCycleDetected
		}
		defer func() { e.depth-- }()
	}
	s := *(*[]any)(p)
	n := len(s)
	e.WriteArrayHeader(n)
	// encodeIface applies its own per-element depth guard then dispatches via
	// encodeReflect — identical to elem.encode for an interface element.
	const probe = 32
	if n <= probe {
		for i := range s {
			if err := encodeIface(e, unsafe.Pointer(&s[i])); err != nil {
				return err
			}
		}
		return nil
	}
	probeStart := len(e.buf)
	for i := range probe {
		if err := encodeIface(e, unsafe.Pointer(&s[i])); err != nil {
			return err
		}
	}
	// Project the remaining size from the probe (+25% slack) and grow once,
	// killing the doubling chain on large dynamic arrays.
	if probeBytes := len(e.buf) - probeStart; probeBytes > 0 {
		// 64-bit intermediate: the product can exceed int32 on a 32-bit build for
		// a large slice, wrapping projected negative and panicking slices.Grow.
		projected := int(int64(probeBytes) * int64(n-probe) / int64(probe))
		projected += projected >> 2
		e.buf = slices.Grow(e.buf, projected)
	}
	for i := probe; i < n; i++ {
		if err := encodeIface(e, unsafe.Pointer(&s[i])); err != nil {
			return err
		}
	}
	return nil
}

func decodeIface(d *Decoder, p unsafe.Pointer) error {
	v, err := decodeAny(d)
	if err != nil {
		return err
	}
	*(*any)(p) = v
	return nil
}

// decodeAny reads the next value as a generic any, mirroring encoding/json.
// isStringKeyTag reports whether tag begins a string value — i.e. a map key
// that d.readStringBytes can consume. Mirrors decodeAny's string-producing
// cases. Used to tell a string-keyed map (→ map[string]any) from a non-string-
// keyed one (→ map[any]any) when decoding a map schemalessly.
func isStringKeyTag(tag byte) bool {
	if tag >= tagFixstr && tag <= tagFixstr|tagFixstrMask {
		return true
	}
	switch tag {
	case tagStr8, tagStr16, tagStr32, tagInternStr,
		tagStateRef, tagStateRepeat, tagStateMTF, tagStatePair:
		return true
	}
	return false
}

// readStringRefAny reads a string that arrived as a back-REFERENCE to an
// already-interned value (tagStateRef / MTF / pair / repeat — the caller
// dispatches on the tag) and returns it boxed into an `any`, reusing ONE shared
// box per value: the box is cached in decState.boxValues keyed by the same
// state id, so every later reference returns it with zero allocation. This is
// the dominant win on repetitive dynamic data (map[string]any). It is called
// ONLY for reference tags — inline / first-occurrence strings box directly in
// decodeAny and never reach here, so high-cardinality data pays no overhead
// (its values never become references). Gated on !noCopy so the cached box
// matches the shared stringValues string (under noCopy ReadString returns an
// input-aliased view, boxed directly as before).
func (d *Decoder) readStringRefAny() (any, error) {
	if d.noCopy || d.state == nil {
		return d.ReadString()
	}
	s, err := d.ReadString()
	if err != nil {
		return nil, err
	}
	if box, ok := d.state.getBoxStr(d.state.lastID, d.arena); ok {
		return box, nil
	}
	return s, nil
}

func decodeAny(d *Decoder) (any, error) {
	if err := d.descend(); err != nil {
		return nil, err
	}
	defer d.ascend()
	tag, err := d.peekTag()
	if err != nil {
		return nil, err
	}
	switch {
	case tag <= tagFixintMax:
		v, err := d.ReadUint()
		return v, err
	case tag >= tagFixstr && tag <= tagFixstr|tagFixstrMask:
		return d.ReadString()
	case tag >= tagFixarr && tag <= tagFixarr|tagFixarrMask:
		n, err := d.ReadArrayHeader()
		if err != nil {
			return nil, err
		}
		if err := d.CheckLength(n, 1); err != nil {
			return nil, err
		}
		out := make([]any, n)
		for i := range n {
			v, err := decodeAny(d)
			if err != nil {
				return nil, err
			}
			out[i] = v
		}
		return out, nil
	case tag >= tagNegfixint && tag <= tagNegfixint|tagNegfixintMask:
		return d.ReadInt()
	}
	switch tag {
	case tagNil:
		d.i++
		return nil, nil
	case tagTrue, tagFalse:
		return d.ReadBool()
	case tagUint8, tagUint16, tagUint32, tagUint64:
		return d.ReadUint()
	case tagInt8, tagInt16, tagInt32, tagInt64:
		return d.ReadInt()
	case tagFloat32:
		return d.ReadFloat32()
	case tagFloat64:
		return d.ReadFloat64()
	case tagStr8, tagStr16, tagStr32, tagInternStr:
		// Inline / first-occurrence string: box directly, zero cache overhead.
		return d.ReadString()
	case tagStateRef, tagStateRepeat, tagStateMTF, tagStatePair:
		// Back-reference to an interned value → return the shared cached box.
		return d.readStringRefAny()
	case tagBin8, tagBin16, tagBin32, tagInternBin:
		return d.ReadBytes()
	case tagArr16, tagArr32:
		n, err := d.ReadArrayHeader()
		if err != nil {
			return nil, err
		}
		if err := d.CheckLength(n, 1); err != nil {
			return nil, err
		}
		out := make([]any, n)
		for i := range n {
			v, err := decodeAny(d)
			if err != nil {
				return nil, err
			}
			out[i] = v
		}
		return out, nil
	case tagMap8, tagMap16, tagMap32:
		n, err := d.ReadMapHeader()
		if err != nil {
			return nil, err
		}
		if err := d.CheckLength(n, 2); err != nil {
			return nil, err
		}
		if n == 0 {
			return map[string]any{}, nil
		}
		// Peek the first key's tag. A string-keyed map (the common case)
		// decodes to map[string]any with interned keys. A non-string-keyed map
		// (e.g. map[int]V boxed in an any/interface, whose keys were written via
		// WriteInt/WriteUint) must NOT be read as string keys — doing so
		// returned ErrTypeMismatch and silently lost data that round-trips fine
		// into a typed destination. The wire cannot distinguish int from uint
		// for small (fixint) keys, so the only lossless schemaless form is
		// map[any]any, each key decoded via decodeAny (int64/uint64/etc.).
		// Peek EACH key's tag, not just the first: a map[any]any can carry mixed
		// key types in any order. Stay on the fast string-keyed path (→
		// map[string]any) as long as keys are strings; the moment a non-string
		// key appears, migrate the string keys decoded so far into map[any]any
		// and finish via decodeAny (which handles string and non-string keys
		// alike). map[any]any is the only lossless schemaless form — the wire
		// can't tell int from uint for small (fixint) keys.
		out := popOrMakeMap[string, any](d, n)
		for idx := 0; idx < n; idx++ {
			ktag, err := d.peekTag()
			if err != nil {
				return nil, err
			}
			if !isStringKeyTag(ktag) {
				anyOut := popOrMakeMap[any, any](d, n)
				for k, v := range out {
					anyOut[k] = v
				}
				for ; idx < n; idx++ {
					k, err := decodeAny(d)
					if err != nil {
						return nil, err
					}
					// A valid map key is always comparable; reject a hostile wire
					// whose "key" decodes to a slice/map so anyOut[k] can't panic.
					if k != nil && !reflect.TypeOf(k).Comparable() {
						return nil, ErrBadTag
					}
					v, err := decodeAny(d)
					if err != nil {
						return nil, err
					}
					anyOut[k] = v
				}
				return anyOut, nil
			}
			kb, err := d.readStringBytes()
			if err != nil {
				return nil, err
			}
			v, err := decodeAny(d)
			if err != nil {
				return nil, err
			}
			out[d.keyCache.Make(kb)] = v
		}
		return out, nil
	case tagMapShape:
		d.i++
		shapeID, n := readUvarint(d.buf[d.i:])
		if n <= 0 {
			return nil, ErrInvalidLength
		}
		if shapeID > uint64(math.MaxUint32) {
			return nil, ErrUnknownStateID // would truncate on the uint32 cast below
		}
		d.i += n
		if d.state == nil {
			d.state = newDecState()
		}
		var names []string
		if shapeID == 0 {
			cnt64, n := readUvarint(d.buf[d.i:])
			if n <= 0 {
				return nil, ErrInvalidLength
			}
			d.i += n
			if cnt64 > uint64(math.MaxInt) { // 32-bit: int(cnt64) would wrap before CheckLength
				return nil, ErrInvalidLength
			}
			cnt := int(cnt64)
			if err := d.CheckLength(cnt, 1); err != nil {
				return nil, err
			}
			sh := d.state.shapeDeclare()
			sh.names = make([]string, 0, cnt)
			for range cnt {
				kb, err := d.readStringBytes()
				if err != nil {
					return nil, err
				}
				sh.names = append(sh.names, d.keyCache.Make(kb))
			}
			names = sh.names
		} else {
			sh := d.state.shapeLookup(uint32(shapeID))
			if sh == nil {
				return nil, ErrUnknownStateID
			}
			names = sh.names
		}
		out := popOrMakeMap[string, any](d, len(names))
		for _, name := range names {
			v, err := decodeAny(d)
			if err != nil {
				return nil, err
			}
			out[name] = v
		}
		return out, nil
	case tagColStruct:
		return decodeColumnarAny(d)
	case tagHybridColStruct:
		return decodeHybridColumnarAny(d)
	case tagTimestamp:
		sec, nsec, err := d.ReadTimestamp()
		if err != nil {
			return nil, err
		}
		t := time.Unix(sec, int64(nsec)).UTC()
		// The zero time.Time (unset time fields) repeats heavily in real data
		// and boxes to a 24-byte heap value; share one immutable box for it.
		// A single IsZero() branch — no table — so non-zero timestamps (the
		// high-cardinality case) pay nothing.
		if t.IsZero() && d.state != nil {
			if d.state.zeroTimeBox == nil {
				d.state.zeroTimeBox = t
			}
			return d.state.zeroTimeBox, nil
		}
		return t, nil
	case tagPackBool:
		// A bool slice encoded under OptQPack. Decode into a typed []bool.
		var s []bool
		if err := decodeSliceBool(d, unsafe.Pointer(&s)); err != nil {
			return nil, err
		}
		return s, nil
	case tagPackRaw, tagPackFor, tagPackDeltaFor, tagPackRLE,
		tagPackDict, tagPackPFor, tagPackGorilla, tagPackALP:
		// A numeric slice encoded under OptQPack/OptBalanced/OptCompression.
		// Without these cases decodeAny fell through to ErrBadTag, so any
		// interface{}/map[string]any value holding such a slice failed to
		// decode. Materialise into the matching typed slice (the values
		// round-trip; the int codecs widen to 64-bit on the wire).
		return decodeAnyPackedSlice(d)
	case tagPackBlock:
		// A long int/uint slice encoded with the per-block adaptive codec.
		// Emitted by writeQPackInt64/Uint64 for any []int/[]int64/[]uint/[]uint64
		// (no ifaceDepth gate), so it can reach decodeAny through an any /
		// map[K]any value / any-typed field; without this case that failed with
		// ErrBadTag. The block-kind byte after the tag selects int64 vs uint64
		// (a namespace disjoint from decodeAnyPackedSlice's qpackKind byte, so
		// it cannot share that path). decodeSliceInt64/Uint64 re-peek the tag.
		if d.i+1 >= len(d.buf) {
			return nil, ErrShortBuffer
		}
		switch d.buf[d.i+1] {
		case blockKindInt:
			var s []int64
			if err := decodeSliceInt64(d, unsafe.Pointer(&s)); err != nil {
				return nil, err
			}
			return s, nil
		case blockKindUint:
			var s []uint64
			if err := decodeSliceUint64(d, unsafe.Pointer(&s)); err != nil {
				return nil, err
			}
			return s, nil
		default:
			return nil, ErrBadTag
		}
	}
	return nil, ErrBadTag
}

// decodeAnyPackedSlice materialises a QPack-encoded numeric slice into a typed
// slice for the generic any decode path. The tag (still at d.i) is followed by a
// one-byte kind that selects the element family/width: integer codecs widen to
// 64-bit (Int64/Uint64), floats stay Float32/Float64 — the four kinds the
// encoder emits for these tags. The typed decodeSlice* helper re-peeks the tag
// and handles every pack variant for that element type.
func decodeAnyPackedSlice(d *Decoder) (any, error) {
	if d.i+1 >= len(d.buf) {
		return nil, ErrShortBuffer
	}
	switch d.buf[d.i+1] {
	case qpackKindInt64:
		var s []int64
		err := decodeSliceInt64(d, unsafe.Pointer(&s))
		return s, err
	case qpackKindInt32:
		// raw-LE preserves the native width (the bit-packing codecs widen to
		// Int64); int32 slices that don't compress land here.
		var s []int32
		err := decodeSliceInt32(d, unsafe.Pointer(&s))
		return s, err
	case qpackKindUint64:
		var s []uint64
		err := decodeSliceUint64(d, unsafe.Pointer(&s))
		return s, err
	case qpackKindUint32:
		var s []uint32
		err := decodeSliceUint32(d, unsafe.Pointer(&s))
		return s, err
	case qpackKindFloat64:
		var s []float64
		err := decodeSliceFloat64(d, unsafe.Pointer(&s))
		return s, err
	case qpackKindFloat32:
		var s []float32
		err := decodeSliceFloat32(d, unsafe.Pointer(&s))
		return s, err
	default:
		// Narrower kinds (Int8/Int16/Uint8/Uint16) never reach a pack tag — they
		// encode as a plain array (or []byte for uint8), handled by decodeAny's
		// array/bin cases — so any other kind is malformed input.
		return nil, ErrBadTag
	}
}

func encodeTime(e *Encoder, p unsafe.Pointer) error {
	t := (*time.Time)(p).UTC()
	e.WriteTimestamp(t.Unix(), uint32(t.Nanosecond()))
	return nil
}
func decodeTime(d *Decoder, p unsafe.Pointer) error {
	sec, nsec, err := d.ReadTimestamp()
	if err != nil {
		return err
	}
	*(*time.Time)(p) = time.Unix(sec, int64(nsec)).UTC()
	return nil
}

func encodeMarshaler(t reflect.Type) func(*Encoder, unsafe.Pointer) error {
	return func(e *Encoder, p unsafe.Pointer) error {
		// A Marshaler emits its own Fast-format body and ignores the
		// encoder's Options. When it is the top-level value, frame the
		// stream honestly as Fast (flag 0) instead of stamping the
		// encoder's Dense/QPack mode onto a Fast body — otherwise the
		// header lies, UnmarshalDirect takes a needless reflect fallback,
		// and the decoder allocates dense state it never uses. Nested
		// Marshaler fields (header already emitted) are unaffected: their
		// custom body is written inline as before.
		if !e.headerOut {
			savedMode, savedQPack := e.mode, e.qpack
			e.mode, e.qpack = Fast, false
			e.writeHeader()
			e.mode, e.qpack = savedMode, savedQPack
			// The top-level Marshaler owns the framing (Fast). Mark it so the
			// post-encode rANS pass leaves the body opts-invariant.
			e.customFramed = true
		}
		m := reflect.NewAt(t, p).Interface().(Marshaler)
		// Thread the shared encoder when the type supports it (generated code):
		// no fresh encoder (and its state) per element, and shape/intern state is
		// shared across a slice so it can be interned. The top-level framing above
		// has already run; EncodeQDF writes the body into e directly.
		if em, ok := m.(EncoderMarshaler); ok {
			return em.EncodeQDF(e)
		}
		out, err := m.MarshalQDF(e.buf)
		if err != nil {
			return err
		}
		e.buf = out
		return nil
	}
}
func decodeUnmarshaler(t reflect.Type) func(*Decoder, unsafe.Pointer) error {
	return func(d *Decoder, p unsafe.Pointer) error {
		// Consume the 5-byte stream header when this is the top-level
		// decoder (no-op once headerRead is set, e.g. a nested Unmarshaler
		// field whose outer decoder already read it). Without this, a
		// top-level Unmarshal into an Unmarshaler type hands the user's
		// UnmarshalQDF the magic+flags bytes instead of the body —
		// mirroring UnmarshalDirect, which slices data[5:].
		if err := d.readHeader(); err != nil {
			return err
		}
		u := reflect.NewAt(t, p).Interface().(Unmarshaler)
		// Thread the shared decoder when the type supports it (generated code):
		// no fresh decoder per element, and it inherits d's noCopy / arena. This
		// is what drops the per-element decoder on a []GeneratedStruct decode.
		if du, ok := u.(DecoderUnmarshaler); ok {
			return du.DecodeQDF(d)
		}
		var n int
		var err error
		if d.arena != nil {
			n, err = UnmarshalNestedArena(u, d.buf[d.i:], d.noCopy, d.arena)
		} else {
			n, err = UnmarshalNested(u, d.buf[d.i:], d.noCopy)
		}
		if err != nil {
			return err
		}
		d.i += n
		return nil
	}
}

// encodeReflect is the entry point from Marshal. v can be any.
func encodeReflect(e *Encoder, v any) error {
	if v == nil {
		e.WriteNil()
		return nil
	}
	// Fast path for common primitive top-level encodings: skip the
	// descriptor lookup and the reflect.New copy.
	switch tv := v.(type) {
	case bool:
		e.WriteBool(tv)
		return nil
	case int:
		e.WriteInt(int64(tv))
		return nil
	case int64:
		e.WriteInt(tv)
		return nil
	case uint64:
		e.WriteUint(tv)
		return nil
	case float64:
		e.WriteFloat64(tv)
		return nil
	case float32:
		e.WriteFloat32(tv)
		return nil
	case string:
		e.WriteString(tv)
		return nil
	case []byte:
		e.WriteBytes(tv)
		return nil
	case map[string]any:
		// Fast path for the dominant dynamic shapes (json.Unmarshal output):
		// take the address of the local typed copy and hand it to the concrete
		// generated encoder. This is byte-identical to the general path below
		// (descOf(map[string]any).encode IS encodeMapStringAny) but skips the
		// reflect.New + Set copy — and because encodeMapStringAny does not retain
		// the pointer, &tv stays on the stack, so it is allocation-free. Without
		// this, every nested map[string]any in an any-tree cost one reflect.New.
		return encodeMapStringAny(e, unsafe.Pointer(&tv))
	case []any:
		return encodeSliceAny(e, unsafe.Pointer(&tv))
	}
	rv := reflect.ValueOf(v)
	t := rv.Type()
	// Unwrap pointer once to match encoding/json behavior for *T. When the
	// caller passes a pointer we can skip the reflect.New copy because
	// rv.Elem() is already addressable.
	if t.Kind() == reflect.Pointer {
		if rv.IsNil() {
			e.WriteNil()
			return nil
		}
		elemT := t.Elem()
		td, err := descOf(elemT)
		if err != nil {
			return err
		}
		return td.encode(e, unsafe.Pointer(rv.Pointer()))
	}
	td, err := descOf(t)
	if err != nil {
		return err
	}
	// Value passed by-value: need an addressable location for unsafe.Pointer.
	// For struct and array types larger than a pointer, the interface boxing at
	// the call site already allocated a heap copy and eface.data IS that copy.
	// We extract it directly to skip reflect.New. This removes one alloc per
	// Marshal(structVal) for all structs that are not "direct interface" types.
	//
	// Direct-interface structs (size == ptrSize with exactly one pointer field,
	// e.g. struct{M map[K]V}) store the pointer value itself in eface.data, not
	// a pointer to the struct — so the trick is unsafe for them. Size > ptrSize
	// is a sufficient condition to rule them out on all platforms.
	//
	// Safety: the data pointer is valid for the lifetime of v (the any parameter
	// stays alive until encodeReflect returns), and td.encode does not retain p.
	if k := t.Kind(); (k == reflect.Struct || k == reflect.Array) &&
		uintptr(t.Size()) > unsafe.Sizeof(unsafe.Pointer(nil)) {
		type eface struct {
			_ unsafe.Pointer
			p unsafe.Pointer
		}
		return td.encode(e, (*eface)(unsafe.Pointer(&v)).p)
	}
	// Fallback: named scalar types, small structs, and direct-interface structs.
	ptr := reflect.New(t)
	ptr.Elem().Set(rv)
	return td.encode(e, unsafe.Pointer(ptr.Pointer()))
}

func decodeReflect(d *Decoder, out any) error {
	rv := reflect.ValueOf(out)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return ErrTypeMismatch
	}
	t := rv.Type().Elem()
	// A query (predicate pushdown / projection) is only meaningful for a
	// columnar slice payload. A non-slice target (single struct, scalar, map)
	// can never be columnar, so reject it before decoding. Non-columnar slice
	// shapes are caught inside the slice decoder (it sees the wire tag).
	if d.query != nil && t.Kind() != reflect.Slice {
		return &QueryError{Op: "predicate pushdown", Err: ErrUnsupported}
	}
	td, err := descOf(t)
	if err != nil {
		return err
	}
	return td.decode(d, unsafe.Pointer(rv.Pointer()))
}

// encodeNilSlice emits tagNil and returns true when the slice at p is nil
// (Data==nil), distinct from an empty (non-nil, len-0) slice — the nil-vs-empty
// distinction maps and pointers already keep, and encoding/json keeps as null vs
// []. Called as the first line of the nil-aware slice FIELD encoders so a nil
// slice round-trips as nil. A small direct (inlinable) call, NOT a wrapping
// closure, so the hot slice paths pay no extra indirection; the shared
// encodeSlice* funcs stay nil-agnostic for their internal direct callers
// (nullable/columnar dense columns), which must still emit a real header.
func (e *Encoder) encodeNilSlice(p unsafe.Pointer) bool {
	if (*sliceHeader)(p).Data == nil {
		e.WriteNil()
		return true
	}
	return false
}

// decodeNilSlice consumes a tagNil nil-slice marker, zeroes the destination, and
// returns true. The first line of the nil-aware slice decoders. The header is
// read first on a top-level decode (headerRead==false); the common struct-field/
// element path is a bounds test + tag compare.
func (d *Decoder) decodeNilSlice(p unsafe.Pointer) bool {
	if !d.headerRead {
		if err := d.readHeaderSlow(); err != nil {
			return false // the caller's normal decode re-surfaces the error
		}
	}
	if d.i < len(d.buf) && d.buf[d.i] == tagNil {
		d.i++
		*(*sliceHeader)(p) = sliceHeader{}
		return true
	}
	return false
}
