package qdf

import (
	"encoding/binary"
	"errors"
	"hash/maphash"
	"math"
	"reflect"
	"slices"
	"time"
	"unsafe"

	"github.com/alex60217101990/qdf/internal/rans"
)

// timeType is the cached descriptor key for time.Time, used by the fingerprint
// fast path (a time.Time field is common and the reflect fallback allocates).
var timeType = reflect.TypeFor[time.Time]()

// maxDeltaDepth bounds recursion in the diff/apply/fingerprint value walks,
// matching the encoder's cycle guard. A cyclic or pathologically deep value is
// rejected with ErrCycleDetected instead of overflowing the (uncatchable) stack.
const maxDeltaDepth = 10000

// schemaSeed is a process-stable seed. A fingerprint only needs to be stable
// within a single producer→consumer exchange that shares this binary; cross-
// build stability is not required for Phase 1 (both ends import the same type).
var schemaSeed = maphash.MakeSeed()

// schemaFingerprintCompute hashes a type descriptor's shape (kind + field names +
// recursive field/element kinds) so Apply can reject a patch built for a
// different type. Cycles are broken by a visited set. It is called exactly once
// per descriptor at build time (descBuild) and the result is stored on
// td.schemaFP; Diff/Apply read that field directly with no runtime synchronization.
func schemaFingerprintCompute(td *typeDesc) uint64 {
	var h maphash.Hash
	h.SetSeed(schemaSeed)
	visited := map[*typeDesc]bool{}
	hashDesc(&h, td, visited)
	return h.Sum64()
}

func hashDesc(h *maphash.Hash, td *typeDesc, visited map[*typeDesc]bool) {
	if td == nil {
		_ = h.WriteByte(0xFF)
		return
	}
	if visited[td] {
		_ = h.WriteByte(0xFE) // cycle marker
		return
	}
	visited[td] = true
	_ = h.WriteByte(byte(td.kind))
	_ = h.WriteByte(td.marshalerKind)
	// td.elem is only the value/element descriptor; fold in the parts of the
	// shape it omits so the cross-type guard does not collide: a map's KEY type
	// (else map[int]V and map[string]V share a fingerprint) and an array's LENGTH
	// (else [3]T and [4]T collide).
	switch td.kind {
	case reflect.Map:
		_, _ = h.WriteString(td.rType.Key().String())
	case reflect.Array:
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], uint64(td.rType.Len()))
		_, _ = h.Write(b[:])
	}
	for i := range td.fields {
		f := &td.fields[i]
		_, _ = h.WriteString(f.name)
		hashDesc(h, f.desc, visited)
	}
	if td.elem != nil {
		hashDesc(h, td.elem, visited)
	}
}

var (
	// ErrInvalidPatch is returned when a patch blob is truncated, has a bad
	// magic/version, or is otherwise malformed.
	ErrInvalidPatch = errors.New("qdf: invalid or truncated patch")
	// ErrPatchSchemaMismatch is returned by Apply when the patch was built for
	// a different type than the supplied base.
	ErrPatchSchemaMismatch = errors.New("qdf: patch schema fingerprint mismatch")
	// ErrPatchBaseMismatch is returned by Apply when the patch carries a base
	// fingerprint that does not match the supplied base value.
	ErrPatchBaseMismatch = errors.New("qdf: patch base fingerprint mismatch")
)

// Diff computes a patch carrying only the structural difference (new − old).
func Diff[T any](old, new T, opts Options) ([]byte, error) {
	return AppendDiff(nil, old, new, opts)
}

// AppendDiff appends the patch to dst and returns the extended slice.
func AppendDiff[T any](dst []byte, old, new T, opts Options) ([]byte, error) {
	td, err := descOf(reflect.TypeFor[T]())
	if err != nil {
		return dst, err
	}
	enc := encPool.Get().(*Encoder)
	enc.Reset()
	enc.applyOpts(opts)

	flags := byte(0)
	if enc.mode == Dense {
		flags |= flagPatchDense
	}
	var baseFP uint64
	if !opts.Has(OptDeltaNoBaseFingerprint) {
		flags |= flagPatchBaseFP // baseFP defaults ON
		baseFP = valueFingerprint(td, unsafe.Pointer(&old))
	}
	schemaFP := td.schemaFP

	start := len(dst)
	enc.buf = writePatchHeader(dst, flags, schemaFP, baseFP)
	enc.MarkHeaderWritten() // QDP header, not QDF: suppress value-codec QDF header

	if err := diffValue(enc, td, unsafe.Pointer(&old), unsafe.Pointer(&new), 0); err != nil {
		enc.buf = nil
		putEnc(enc, &encPool)
		return dst, err
	}
	maybeApplyPatchRANS(enc, start)
	out := slices.Clone(enc.buf)
	enc.buf = nil
	putEnc(enc, &encPool)
	return out, nil
}

// Apply merges patch onto base in place, reconstructing new. Unchanged fields
// are untouched.
func Apply[T any](base *T, patch []byte) error {
	return applyImpl(base, patch, nil)
}

// ApplyArena is Apply with a caller-provided decode arena: every string (and
// []byte-as-string) value the patch REPLACES is copied into arena's contiguous
// bump blocks instead of one heap allocation per string. It is otherwise
// byte-for-byte identical to Apply — same result, same errors, same wire — and
// only differs in where replaced string bodies live.
//
// This helps only when a patch replaces MANY strings (a large changed []string
// or []struct{...string} field, a map with many changed string values). A
// typical small patch changes few strings and the arena has little to batch; in
// that case Apply is the simpler choice.
//
// Lifetime / safety (identical to the decode arena, see Arena): strings written
// into base by ApplyArena ALIAS arena's memory and stay valid only as long as
// arena (or any string from it) is reachable. The safe pattern is one Arena per
// epoch, then drop it; Reset reuses it but invalidates every string from the
// prior epoch. Unchanged string fields already present in base are NOT touched
// and do not alias arena.
func ApplyArena[T any](base *T, patch []byte, arena *Arena) error {
	return applyImpl(base, patch, arena)
}

func applyImpl[T any](base *T, patch []byte, arena *Arena) error {
	if base == nil {
		return ErrTypeMismatch
	}
	td, err := descOf(reflect.TypeFor[T]())
	if err != nil {
		return err
	}
	h, n, err := readPatchHeader(patch)
	if err != nil {
		return err
	}
	if h.schemaFP != td.schemaFP {
		return ErrPatchSchemaMismatch
	}
	if h.flags&flagPatchBaseFP != 0 {
		if valueFingerprint(td, unsafe.Pointer(base)) != h.baseFP {
			return ErrPatchBaseMismatch
		}
	}
	body := patch[n:]
	if h.flags&flagPatchRANS != 0 {
		body, err = decompressPatchBody(body)
		if err != nil {
			return err
		}
	}
	if len(body) == 0 {
		// Empty body: the root value was unchanged (diffValue wrote no op).
		// Base already equals new — nothing to apply.
		return nil
	}
	dec := decPool.Get().(*Decoder)
	resetPatchDecoder(dec, body, h.flags&flagPatchDense != 0)
	dec.arena = arena // nil for Apply; set for ApplyArena so replaced strings bump-allocate
	err = applyValue(dec, td, unsafe.Pointer(base), 0)
	dec.buf = nil
	dec.arena = nil // never pin the caller's arena across pooled reuse
	if cap(dec.deltaScratch) > maxRetainedDeltaScratch {
		dec.deltaScratch = nil
	}
	if len(dec.keyIdx) > 1<<16 {
		dec.keyIdx = nil
	}
	decPool.Put(dec)
	return err
}

// resetPatchDecoder prepares a pooled decoder to read patch body bytes. headerRead
// is forced true so value codecs invoked for opReplace skip the QDF header.
func resetPatchDecoder(dec *Decoder, body []byte, dense bool) {
	dec.SetInput(body)
	// SetInput resets buf/i/depth/colIndex/selectFields/query/mapFreeList/state
	// but leaves noCopy and arena sticky from a prior decode. Inheriting noCopy
	// would alias the caller's patch buffer in an opReplace string → corruption
	// after the caller mutates/reuses it; clear both (mirrors UnmarshalT).
	dec.noCopy = false
	dec.arena = nil
	dec.headerRead = true
	if dense {
		dec.mode = Dense
		if dec.state == nil {
			dec.state = newDecState()
		}
	}
}

// maybeApplyPatchRANS optionally rANS-compresses the patch body in place after
// the QDP header (offset start). Mirrors maybeApplyRANS.
func maybeApplyPatchRANS(enc *Encoder, start int) {
	if !enc.rans {
		return
	}
	hdr := 13
	if enc.buf[start+4]&flagPatchBaseFP != 0 {
		hdr = 21
	}
	if len(enc.buf)-start < hdr+ransMinBytes {
		return
	}
	body := enc.buf[start+hdr:]
	cand := appendUvarint(make([]byte, 0, len(body)/2+512), uint64(len(body)))
	cand = rans.Encode(cand, body)
	if len(cand) >= len(body) {
		return
	}
	if uint64(len(body)) > uint64(hdr+len(cand))*64+(1<<20) {
		return
	}
	enc.buf = append(enc.buf[:start+hdr], cand...)
	enc.buf[start+4] |= flagPatchRANS
}

// decompressPatchBody reverses maybeApplyPatchRANS: varuint(origLen) + rANS stream.
func decompressPatchBody(body []byte) ([]byte, error) {
	origLen, k := readUvarint(body)
	if k <= 0 {
		return nil, ErrInvalidPatch
	}
	if origLen == 0 {
		// The encoder only emits a rANS body for bodies >= ransMinBytes; a
		// decoded origLen of 0 is malformed.
		return nil, ErrInvalidPatch
	}
	if origLen > uint64(len(body))*64+(1<<20) {
		return nil, ErrInvalidPatch
	}
	out, err := rans.Decode(body[k:], int(origLen))
	if err != nil {
		return nil, ErrInvalidPatch
	}
	return out, nil
}

// valueFingerprint produces an order-independent hash of a value so map-bearing
// types fingerprint deterministically regardless of map iteration order. It is
// computed once over old (Diff) and once over base (Apply); a mismatch means
// the caller's base is not the old the patch was built against.
func valueFingerprint(td *typeDesc, p unsafe.Pointer) uint64 {
	var h maphash.Hash
	h.SetSeed(schemaSeed)
	fpHash(&h, td, p, 0)
	return h.Sum64()
}

// fpHash hashes the value of type td at p into h. For a pointer-free, padding-
// free ("tight POD") type it issues a single maphash.Write over the whole
// contiguous byte span — collapsing N per-field/per-element writes and all
// reflect dispatch into one vectorized hash. Non-tight types (those carrying
// pointers, strings, padding, or maps) recurse structurally, hashing per field /
// per element so padding bytes are never read. nil-vs-empty markers for
// slice/map/pointer are preserved exactly as the prior reflect walk emitted them.
func fpHash(h *maphash.Hash, td *typeDesc, p unsafe.Pointer, depth int) {
	if depth > maxDeltaDepth {
		// A truncated fingerprint is fine here: the diff walk applies its own cap
		// and surfaces the cycle as ErrCycleDetected.
		return
	}
	if td.tightPOD {
		// One write over the whole value: scalar, tight array, or tightly-packed
		// pointer-free struct. The span is all content (no padding) so the hash is
		// determined purely by the logical value.
		_, _ = h.Write(unsafe.Slice((*byte)(p), td.rType.Size()))
		return
	}
	switch td.kind {
	case reflect.String:
		_, _ = h.WriteString(*(*string)(p))
	case reflect.Bool:
		if *(*bool)(p) {
			_ = h.WriteByte(1)
		} else {
			_ = h.WriteByte(0)
		}
	case reflect.Slice:
		sh := (*sliceHeader)(p)
		if sh.Data == nil {
			_ = h.WriteByte(0) // nil-slice marker (distinct from empty-non-nil)
			return
		}
		_ = h.WriteByte(1)
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], uint64(sh.Len))
		_, _ = h.Write(b[:])
		if sh.Len == 0 {
			return
		}
		if td.rType.Elem().Kind() == reflect.Uint8 {
			_, _ = h.Write(unsafe.Slice((*byte)(sh.Data), sh.Len))
			return
		}
		elem := td.elem
		if elem == nil {
			elem, _ = descOf(td.rType.Elem())
		}
		stride := td.rType.Elem().Size()
		if elem != nil && elem.tightPOD {
			// Whole backing array is content: one write over len*stride bytes.
			_, _ = h.Write(unsafe.Slice((*byte)(sh.Data), uintptr(sh.Len)*stride))
			return
		}
		for i := range sh.Len {
			fpHash(h, elem, unsafe.Add(sh.Data, uintptr(i)*stride), depth+1)
		}
	case reflect.Array:
		// A tight array was handled by the tightPOD branch; a non-tight array has a
		// non-tight element (strings/pointers/padding), so hash element by element.
		n := td.rType.Len()
		stride := td.rType.Elem().Size()
		elem := td.elem
		if elem == nil {
			elem, _ = descOf(td.rType.Elem())
		}
		for i := range n {
			fpHash(h, elem, unsafe.Add(p, uintptr(i)*stride), depth+1)
		}
	case reflect.Struct:
		// A tight struct was handled above; this one has strings/pointers/padding.
		// A fields-less struct (time.Time, a custom Marshaler) carries no td.fields
		// to walk — hash its real fields via reflect so the fingerprint is not blind
		// to them (consistent with equalValue's DeepEqual for the same shape).
		if len(td.fields) == 0 {
			if td.rType == timeType {
				// time.Time is the common fields-less struct. Hash the instant the
				// codec actually round-trips (UTC sec + nsec) — alloc-free and no
				// reflect, vs fpHashReflect which walks *Location and allocates.
				t := (*time.Time)(p).UTC()
				var b [12]byte
				binary.LittleEndian.PutUint64(b[:8], uint64(t.Unix()))
				binary.LittleEndian.PutUint32(b[8:], uint32(t.Nanosecond()))
				_, _ = h.Write(b[:])
				return
			}
			fpHashReflect(h, reflect.NewAt(td.rType, p).Elem(), depth)
			return
		}
		// Hash per field (skipping any padding between fields).
		for i := range td.fields {
			f := &td.fields[i]
			fpHash(h, f.desc, unsafe.Add(p, f.offset), depth+1)
		}
	case reflect.Pointer:
		ep := *(*unsafe.Pointer)(p)
		if ep == nil {
			_ = h.WriteByte(0)
			return
		}
		_ = h.WriteByte(1)
		elem := td.elem
		if elem == nil {
			elem, _ = descOf(td.rType.Elem())
		}
		fpHash(h, elem, ep, depth+1)
	case reflect.Map:
		mp := *(*unsafe.Pointer)(p)
		if mp == nil {
			_ = h.WriteByte(0) // nil-map marker
			return
		}
		_ = h.WriteByte(1)
		v := reflect.NewAt(td.rType, p).Elem()
		// Maps need reflect to iterate; hash key+value with the reflect fallback
		// (no per-entry addressable scratch — that allocated two reflect.New values
		// per entry and regressed map-heavy values). Commutative XOR fold keeps the
		// result order-independent.
		var acc uint64
		for it := v.MapRange(); it.Next(); {
			var e maphash.Hash
			e.SetSeed(schemaSeed)
			fpHashReflect(&e, it.Key(), depth+1)
			fpHashReflect(&e, it.Value(), depth+1)
			acc ^= e.Sum64()
		}
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], acc)
		_, _ = h.Write(b[:])
	case reflect.Interface:
		v := reflect.NewAt(td.rType, p).Elem()
		if v.IsNil() {
			_ = h.WriteByte(0)
			return
		}
		_ = h.WriteByte(1)
		el := v.Elem()
		_, _ = h.WriteString(el.Type().String())
		fpHashReflect(h, el, depth+1)
	default:
		// Exotic kinds (chan/func/complex via interface, etc.): hash the type name.
		// Rare and off the hot path.
		_, _ = h.WriteString(td.rType.String())
	}
}

// fpHashReflect is the reflect fallback used only for the boxed dynamic value
// inside an interface, where a stable typed pointer is awkward to obtain. It
// mirrors the prior reflect walk for that one rare branch; perf does not matter.
func fpHashReflect(h *maphash.Hash, v reflect.Value, depth int) {
	if depth > maxDeltaDepth {
		return
	}
	switch v.Kind() {
	case reflect.Map:
		var acc uint64
		iter := v.MapRange()
		for iter.Next() {
			var e maphash.Hash
			e.SetSeed(schemaSeed)
			fpHashReflect(&e, iter.Key(), depth+1)
			fpHashReflect(&e, iter.Value(), depth+1)
			acc ^= e.Sum64()
		}
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], acc)
		_, _ = h.Write(b[:])
	case reflect.Struct:
		for _, field := range v.Fields() {
			fpHashReflect(h, field, depth+1)
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			fpHashReflect(h, v.Index(i), depth+1)
		}
	case reflect.Pointer:
		if v.IsNil() {
			_ = h.WriteByte(0)
		} else {
			_ = h.WriteByte(1)
			fpHashReflect(h, v.Elem(), depth+1)
		}
	case reflect.String:
		_, _ = h.WriteString(v.String())
	case reflect.Bool:
		if v.Bool() {
			_ = h.WriteByte(1)
		} else {
			_ = h.WriteByte(0)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], uint64(v.Int()))
		_, _ = h.Write(b[:])
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], v.Uint())
		_, _ = h.Write(b[:])
	case reflect.Float32, reflect.Float64:
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], math.Float64bits(v.Float()))
		_, _ = h.Write(b[:])
	case reflect.Interface:
		if v.IsNil() {
			_ = h.WriteByte(0)
			return
		}
		_ = h.WriteByte(1)
		el := v.Elem()
		_, _ = h.WriteString(el.Type().String())
		fpHashReflect(h, el, depth+1)
	default:
		_, _ = h.WriteString(v.Type().String())
	}
}
