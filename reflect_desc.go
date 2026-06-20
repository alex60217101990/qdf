package qdf

import (
	"reflect"
	"strings"
	"sync"
	"time"
	"unsafe"
)

// typeDesc is the compiled descriptor for a reflect.Type. Looked up via the
// typeCache map keyed by the Type's runtime pointer (cheap and stable).
// Field order keeps the GC pointer-scan range tight (64 bytes vs 96 for the
// source order): the single-word pointer fields lead, the multi-word `fields`
// slice (whose len/cap words are non-pointer) follows, and the non-pointer
// scalars (schemaFP/keyOff) plus the 1-byte tails trail.
type typeDesc struct {
	rType   reflect.Type
	elem    *typeDesc     // slice/array/map-value/ptr
	colPlan *columnarPlan // non-nil on a []struct whose element is columnar-eligible

	// encode is the specialized encoder for this type. It receives a pointer
	// to a value of the type (NOT an interface) so it can dereference via
	// unsafe.Pointer without paying for reflect.Value.
	encode func(e *Encoder, p unsafe.Pointer) error
	decode func(d *Decoder, p unsafe.Pointer) error

	// keyDesc describes the key field's type for a struct element's identity key
	// (the field tagged `qdf:"...,key"`), used by keyed slice diff. nil for types
	// without a key tag. See keyOff / keyed below. Set once at build.
	keyDesc *typeDesc // descriptor of the key field's type

	fields []fieldDesc // structs only

	// schemaFP is the structural fingerprint of this descriptor's full subtree
	// (kind + field names + recursive field/element kinds). Computed once at build
	// time (descBuild) so the delta Diff/Apply path reads it with zero runtime
	// synchronization instead of a sync.Map lookup. Safe to read concurrently:
	// set under the single-threaded build before the descriptor is published to
	// typeCache via LoadOrStore (same happens-before as encode/decode).
	schemaFP uint64

	// keyOff is the byte offset of the ,key-tagged field within the struct (see
	// keyDesc above). keyed reports its presence. Set once at build.
	keyOff uintptr // byte offset of the key field within the struct

	// kind / marshalerKind / pod are 1-byte (or kind-sized) tails kept last so
	// they share one word instead of forcing padding ahead of the 8-byte fields
	// above.
	kind reflect.Kind
	// marshalerKind: 0=none, 1=Marshaler interface, 2=encoding.TextMarshaler-ish (future)
	marshalerKind uint8
	// pod reports whether this type's memory is pointer-free (noPointers). Cached
	// at build for the delta diff hot path (equalSliceEV/equalArrayEV/diffElems),
	// which would otherwise call the structural noPointers walk per element.
	pod bool
	// tightPOD reports whether this type is pointer-free AND has no internal or
	// tail padding — i.e. its entire byte span is content. The delta value
	// fingerprint may bulk-hash such a span in ONE maphash.Write (collapsing N
	// per-field/per-element hashes + reflect dispatch). It must NOT bulk-hash a
	// padded type: padding bytes are indeterminate, so two logically-equal values
	// could hash differently → a false ErrPatchBaseMismatch. Computed once at
	// descBuild (single-threaded, published with the descriptor). Shares the
	// 1-byte tail with kind/marshalerKind/pod (no new padding).
	tightPOD bool
	// keyed reports whether a ,key-tagged field is present (keyOff/keyDesc valid).
	keyed bool
}

type fieldDesc struct {
	name string
	desc *typeDesc
	// preFast holds the pre-encoded fixstr/strN header for the field name in
	// Fast mode. Cheaper than emitting per-encode.
	preFast []byte
	// for Dense mode we cannot precompute the state ref bytes because IDs
	// are per-encoder. We can however precompute the bytes that get appended
	// the first time (intern record) and rely on the encoder's state table
	// to switch to refs on subsequent encodes.
	preInternStr []byte
	offset       uintptr
	isKey        bool // this field carried the ,key tag option
}

var typeCache sync.Map // map[reflect.Type]*typeDesc — only fully-built entries

// buildCtx is a per-build cycle-breaking map. Goroutine A building T sees
// only its own partial entries here; goroutine B doing the same gets a
// separate ctx. Either both complete and one wins via LoadOrStore; the
// "loser" is still valid for its own caller's captured closures.
type buildCtx struct {
	inProgress map[reflect.Type]*typeDesc
}

// descOf is the top-level entry. It returns a *fully-built* descriptor and
// is safe to call from concurrent goroutines. The fast path is a single
// sync.Map.Load; the slow path constructs a builder ctx and recurses
// through descBuild.
func descOf(t reflect.Type) (*typeDesc, error) {
	if v, ok := typeCache.Load(t); ok {
		return v.(*typeDesc), nil
	}
	ctx := &buildCtx{inProgress: make(map[reflect.Type]*typeDesc)}
	if _, err := descBuild(t, ctx); err != nil {
		return nil, err
	}
	// Publish the WHOLE graph now that every descriptor in it is fully built.
	//
	// Publishing each descriptor as its own fillDesc returned (the previous
	// design) was racy for cyclic types: building cycNode requires its *cycNode
	// pointer descriptor, whose fillDesc completes — and would publish — while
	// the enclosing cycNode struct descriptor's .encode/.decode are still nil.
	// A second goroutine that Loaded the prematurely-published *cycNode then
	// read those fields while we were writing them (data race + nil-func
	// dispatch panic). Deferring publication to here guarantees a descriptor is
	// globally visible only when its entire transitive graph is complete.
	//
	// LoadOrStore still lets a racing goroutine's graph win per type; our own
	// captured closures reference our fully-built versions regardless, and both
	// graphs encode identical wire output.
	for rt, td := range ctx.inProgress {
		typeCache.LoadOrStore(rt, td)
	}
	if v, ok := typeCache.Load(t); ok {
		return v.(*typeDesc), nil
	}
	// Unreachable in practice (t was just built into ctx.inProgress), but stay
	// defensive rather than returning a nil descriptor.
	return descBuild(t, ctx)
}

// descBuild is the recursion-safe lookup used from inside fillDesc. It MUST be
// called with a non-nil ctx so type cycles can be broken. It does NOT publish
// to the global typeCache — descOf publishes the whole graph atomically-per-
// type once every descriptor is complete. Entries stay in ctx.inProgress for
// the ctx's lifetime so sibling references reuse them instead of rebuilding.
func descBuild(t reflect.Type, ctx *buildCtx) (*typeDesc, error) {
	if v, ok := typeCache.Load(t); ok {
		return v.(*typeDesc), nil
	}
	if td, ok := ctx.inProgress[t]; ok {
		return td, nil
	}
	td := &typeDesc{rType: t, kind: t.Kind()}
	// pod is a pure structural property of t (pointer-free memory). Compute it
	// once here with the UNCACHED walk so the delta diff hot path can read the
	// field instead of calling noPointers per element. Set before fillDesc so it
	// is in place by the time descOf publishes td to typeCache.
	td.pod = noPointersWalk(t)
	// tightPOD is, like pod, a pure structural property of t (pointer-free AND no
	// internal/tail padding). Compute it here with the uncached walk so the delta
	// value-fingerprint hot path can bulk-hash a tight span in one maphash.Write.
	td.tightPOD = tightPODWalk(t)
	ctx.inProgress[t] = td
	if err := fillDesc(td, t, ctx); err != nil {
		// Leave nothing half-built reachable: the failed type stays only in
		// this ctx (never published) and is discarded with it.
		return nil, err
	}
	// schemaFP is a pure function of the now-complete descriptor subtree (its
	// children were built first, so td.fields/td.elem/td.kind are populated).
	// Compute it once here under the single-threaded build so Diff/Apply read the
	// field with no sync.Map lookup. schemaFingerprintCompute does its own
	// visited-set walk from td, independent of children's cached schemaFP, so a
	// child computed before a cycle closes is still correct.
	td.schemaFP = schemaFingerprintCompute(td)
	return td, nil
}

// fillDesc populates td in place. All recursive descBuild calls happen
// under the same buildCtx so cycles in t are broken via ctx.inProgress.
// Encoders/decoders captured during this pass dereference td.encode /
// td.decode at runtime — by then fillDesc has assigned them.
func fillDesc(td *typeDesc, t reflect.Type, ctx *buildCtx) error {
	// Special-case time.Time → timestamp tag.
	if t == reflect.TypeFor[time.Time]() {
		td.encode = encodeTime
		td.decode = decodeTime
		return nil
	}
	// Marshaler interface check (pointer receiver too).
	marshalerType := reflect.TypeFor[Marshaler]()
	unmarshalerType := reflect.TypeFor[Unmarshaler]()
	if reflect.PointerTo(t).Implements(marshalerType) {
		td.marshalerKind = 1
		td.encode = encodeMarshaler(t)
	}
	if reflect.PointerTo(t).Implements(unmarshalerType) {
		td.decode = decodeUnmarshaler(t)
	}
	if td.encode != nil && td.decode != nil {
		return nil
	}
	// A type may implement only ONE of Marshaler/Unmarshaler (the documented
	// asymmetric case: encode structurally, decode via UnmarshalQDF, or vice
	// versa). The structural switch below unconditionally sets BOTH td.encode
	// and td.decode, so snapshot the custom codec and restore it afterward for
	// the direction it implements — otherwise the custom method is clobbered.
	customEnc, customDec := td.encode, td.decode

	switch t.Kind() {
	case reflect.Bool:
		td.encode = encodeBool
		td.decode = decodeBool
	case reflect.Int:
		td.encode = encodeIntN(8)
		td.decode = decodeIntN(8)
	case reflect.Int8:
		td.encode = encodeIntN(1)
		td.decode = decodeIntN(1)
	case reflect.Int16:
		td.encode = encodeIntN(2)
		td.decode = decodeIntN(2)
	case reflect.Int32:
		td.encode = encodeIntN(4)
		td.decode = decodeIntN(4)
	case reflect.Int64:
		td.encode = encodeIntN(8)
		td.decode = decodeIntN(8)
	case reflect.Uint:
		td.encode = encodeUintN(8)
		td.decode = decodeUintN(8)
	case reflect.Uint8:
		td.encode = encodeUintN(1)
		td.decode = decodeUintN(1)
	case reflect.Uint16:
		td.encode = encodeUintN(2)
		td.decode = decodeUintN(2)
	case reflect.Uint32:
		td.encode = encodeUintN(4)
		td.decode = decodeUintN(4)
	case reflect.Uint64:
		td.encode = encodeUintN(8)
		td.decode = decodeUintN(8)
	case reflect.Uintptr:
		td.encode = encodeUintN(8)
		td.decode = decodeUintN(8)
	case reflect.Float32:
		td.encode = encodeF32
		td.decode = decodeF32
	case reflect.Float64:
		td.encode = encodeF64
		td.decode = decodeF64
	case reflect.String:
		td.encode = encodeString
		td.decode = decodeString
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			td.encode = encodeBytes
			td.decode = decodeBytes
		} else if enc, dec, ok := installSliceFastPath(t); ok {
			td.encode = enc
			td.decode = dec
		} else {
			elem, err := descBuild(t.Elem(), ctx)
			if err != nil {
				return err
			}
			td.elem = elem
			// Columnar transposition replays the element's structural field
			// layout and bypasses td.elem.encode / td.elem.decode. A struct that
			// implements a custom codec in EITHER direction (the documented
			// asymmetric single-direction case) keeps its structural fields
			// populated — so without this guard a []OnlyMarshaler / []OnlyUnmarshaler
			// would columnar-encode/decode and silently skip MarshalQDF/UnmarshalQDF.
			// A type implementing BOTH returns early in fillDesc with empty fields,
			// so buildColumnarPlan already yields nil there; this guard covers the
			// single-direction case the early return misses.
			et := t.Elem()
			if elem.kind == reflect.Struct &&
				!reflect.PointerTo(et).Implements(marshalerType) &&
				!reflect.PointerTo(et).Implements(unmarshalerType) {
				td.colPlan = buildColumnarPlan(elem)
			}
			td.encode = encodeSlice(elem, t.Elem().Size(), td.colPlan)
			td.decode = decodeSlice(t, elem, t.Elem().Size(), td.colPlan)
		}
		// Nil-vs-empty preservation: the []byte and reflect encoders self-handle
		// it inline (they are field-only); the typed fast paths use their nil-aware
		// *Nil variants (installSliceFastPath) which keep the shared encodeSlice*
		// funcs nil-agnostic for their internal dense-column callers.
	case reflect.Array:
		// [N]byte fast path: encode/decode as one contiguous binary blob (flat,
		// memcpy) instead of N tagged elements — far smaller wire for real byte
		// data and zero-alloc in-place decode. Covers fixed-width IDs (trace/span
		// ids, UUIDs) and any named byte type ([N]MyByte, since Kind stays Uint8).
		if t.Elem().Kind() == reflect.Uint8 {
			n := t.Len()
			td.encode = encodeFixedByteArray(n)
			td.decode = decodeFixedByteArray(n)
			break
		}
		elem, err := descBuild(t.Elem(), ctx)
		if err != nil {
			return err
		}
		td.elem = elem
		td.encode = encodeArray(elem, t.Elem().Size(), t.Len())
		td.decode = decodeArray(elem, t.Elem().Size(), t.Len())
	case reflect.Map:
		if enc, dec, ok := installMapFastPath(t); ok {
			td.encode = enc
			td.decode = dec
			break
		}
		k, err := descBuild(t.Key(), ctx)
		if err != nil {
			return err
		}
		v, err := descBuild(t.Elem(), ctx)
		if err != nil {
			return err
		}
		td.elem = v
		td.encode = encodeMap(t, k, v)
		td.decode = decodeMap(t, k, v)
	case reflect.Pointer:
		elem, err := descBuild(t.Elem(), ctx)
		if err != nil {
			return err
		}
		td.elem = elem
		td.encode = encodePtr(elem)
		td.decode = decodePtr(t, elem)
	case reflect.Interface:
		td.encode = encodeIface
		td.decode = decodeIface
	case reflect.Struct:
		fields, err := buildStructFields(t, ctx)
		if err != nil {
			return err
		}
		td.fields = fields
		// Record the identity key (the ,key-tagged field), if any. Reject more
		// than one ,key field per type.
		for i := range td.fields {
			if !td.fields[i].isKey {
				continue
			}
			if td.keyed {
				return ErrUnsupported // at most one ,key field per type
			}
			td.keyed = true
			td.keyOff = td.fields[i].offset
			td.keyDesc = td.fields[i].desc
		}
		td.encode = encodeStruct(td)
		td.decode = decodeStruct(td)
	default:
		return ErrUnsupported
	}
	// Restore a custom Marshaler/Unmarshaler for the direction it implements;
	// the structural codec above stands in only for the missing direction.
	if customEnc != nil {
		td.encode = customEnc
	}
	if customDec != nil {
		td.decode = customDec
	}
	return nil
}

func buildStructFields(t reflect.Type, ctx *buildCtx) ([]fieldDesc, error) {
	return appendStructFields(nil, t, 0, ctx)
}

// appendStructFields walks t's fields and appends them to out. base
// is the byte offset of t within the enclosing struct (0 for the
// top-level call). Anonymous embedded struct fields are flattened
// recursively into the parent's wire layout — their inner fields
// appear at the parent level just like encoding/json. A field with
// an explicit qdf / json tag of "-" is skipped at any depth.
func appendStructFields(out []fieldDesc, t reflect.Type, base uintptr, ctx *buildCtx) ([]fieldDesc, error) {
	if out == nil {
		out = make([]fieldDesc, 0, t.NumField())
	}
	for sf := range t.Fields() {
		// Anonymous embedded struct → flatten. This covers the
		// common encoding/json idiom where an unexported lower-case
		// type with exported fields is embedded; silently dropping
		// such a field loses data on round-trip. Pointer-typed
		// embedded fields fall through to the regular field path so
		// they encode as a pointer-to-struct value.
		if sf.Anonymous && sf.Type.Kind() == reflect.Struct {
			// Tag "-" on the embedded field itself opts the whole
			// nested layout out — matches encoding/json.
			if tag, ok := sf.Tag.Lookup("qdf"); ok && tag == "-" {
				continue
			}
			if tag, ok := sf.Tag.Lookup("json"); ok &&
				strings.Split(tag, ",")[0] == "-" {
				continue
			}
			nested, err := appendStructFields(out, sf.Type, base+sf.Offset, ctx)
			if err != nil {
				return nil, err
			}
			out = nested
			continue
		}
		if !sf.IsExported() {
			continue
		}
		name := sf.Name
		isKey := false
		if tag, ok := sf.Tag.Lookup("qdf"); ok {
			if tag == "-" {
				continue
			}
			parts := strings.Split(tag, ",")
			if parts[0] != "" {
				name = parts[0]
			}
			for _, opt := range parts[1:] {
				if opt == "key" {
					if !sf.Type.Comparable() {
						return nil, ErrUnsupported // a key field must be comparable
					}
					isKey = true
				}
			}
		} else if tag, ok := sf.Tag.Lookup("json"); ok {
			parts := strings.Split(tag, ",")
			if parts[0] == "-" {
				continue
			}
			if parts[0] != "" {
				name = parts[0]
			}
		}
		fd, err := descBuild(sf.Type, ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, fieldDesc{
			name:         name,
			offset:       base + sf.Offset,
			desc:         fd,
			preFast:      precomputeFixstrHeader(name),
			preInternStr: precomputeInternStrHeader(name),
			isKey:        isKey,
		})
	}
	// stable order = source order (no sort) — matches encoding/json behavior
	// for fixed-layout decode.
	return out, nil
}

func precomputeFixstrHeader(name string) []byte {
	n := len(name)
	switch {
	case n <= int(tagFixstrMask):
		out := make([]byte, 1+n)
		out[0] = tagFixstr | byte(n)
		copy(out[1:], name)
		return out
	case n <= 0xFF:
		out := make([]byte, 2+n)
		out[0] = tagStr8
		out[1] = byte(n)
		copy(out[2:], name)
		return out
	case n <= 0xFFFF:
		out := make([]byte, 3+n)
		out[0] = tagStr16
		out[1] = byte(n)
		out[2] = byte(n >> 8)
		copy(out[3:], name)
		return out
	default:
		out := make([]byte, 5+n)
		out[0] = tagStr32
		out[1] = byte(n)
		out[2] = byte(n >> 8)
		out[3] = byte(n >> 16)
		out[4] = byte(n >> 24)
		copy(out[5:], name)
		return out
	}
}

func precomputeInternStrHeader(name string) []byte {
	out := []byte{tagInternStr}
	out = appendUvarint(out, uint64(len(name)))
	out = append(out, name...)
	return out
}
