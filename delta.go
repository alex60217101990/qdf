package qdf

import (
	"encoding/binary"
	"errors"
	"hash/maphash"
	"math"
	"reflect"
	"slices"
	"unsafe"

	"github.com/alex60217101990/qdf/internal/rans"
)

// schemaSeed is a process-stable seed. A fingerprint only needs to be stable
// within a single producer→consumer exchange that shares this binary; cross-
// build stability is not required for Phase 1 (both ends import the same type).
var schemaSeed = maphash.MakeSeed()

// schemaFingerprint hashes a type descriptor's shape (kind + field names +
// recursive field/element kinds) so Apply can reject a patch built for a
// different type. Cycles are broken by a visited set.
func schemaFingerprint(td *typeDesc) uint64 {
	var h maphash.Hash
	h.SetSeed(schemaSeed)
	visited := map[*typeDesc]bool{}
	hashDesc(&h, td, visited)
	return h.Sum64()
}

func hashDesc(h *maphash.Hash, td *typeDesc, visited map[*typeDesc]bool) {
	if td == nil {
		h.WriteByte(0xFF)
		return
	}
	if visited[td] {
		h.WriteByte(0xFE) // cycle marker
		return
	}
	visited[td] = true
	h.WriteByte(byte(td.kind))
	h.WriteByte(td.marshalerKind)
	for i := range td.fields {
		f := &td.fields[i]
		h.WriteString(f.name)
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
	flags |= flagPatchBaseFP // baseFP defaults ON
	baseFP := valueFingerprint(td, unsafe.Pointer(&old))
	schemaFP := schemaFingerprint(td)

	start := len(dst)
	enc.buf = writePatchHeader(dst, flags, schemaFP, baseFP)
	enc.MarkHeaderWritten() // QDP header, not QDF: suppress value-codec QDF header

	if _, err := diffValue(enc, td, unsafe.Pointer(&old), unsafe.Pointer(&new)); err != nil {
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
	if h.schemaFP != schemaFingerprint(td) {
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
	err = applyValue(dec, td, unsafe.Pointer(base))
	dec.buf = nil
	if cap(dec.deltaScratch) > maxRetainedDeltaScratch {
		dec.deltaScratch = nil
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
	fpWalk(&h, reflect.NewAt(td.rType, p).Elem())
	return h.Sum64()
}

func fpWalk(h *maphash.Hash, v reflect.Value) {
	switch v.Kind() {
	case reflect.Map:
		var acc uint64
		iter := v.MapRange()
		for iter.Next() {
			var e maphash.Hash
			e.SetSeed(schemaSeed)
			fpWalk(&e, iter.Key())
			fpWalk(&e, iter.Value())
			acc ^= e.Sum64() // commutative: order-independent
		}
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], acc)
		h.Write(b[:])
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			fpWalk(h, v.Field(i))
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			fpWalk(h, v.Index(i))
		}
	case reflect.Pointer:
		if v.IsNil() {
			h.WriteByte(0)
		} else {
			h.WriteByte(1)
			fpWalk(h, v.Elem())
		}
	case reflect.String:
		h.WriteString(v.String())
	case reflect.Bool:
		if v.Bool() {
			h.WriteByte(1)
		} else {
			h.WriteByte(0)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], uint64(v.Int()))
		h.Write(b[:])
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], v.Uint())
		h.Write(b[:])
	case reflect.Float32, reflect.Float64:
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], math.Float64bits(v.Float()))
		h.Write(b[:])
	default:
		h.WriteString(v.Type().String())
	}
}
