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
type typeDesc struct {
	kind    reflect.Kind
	rType   reflect.Type
	fields  []fieldDesc   // structs only
	elem    *typeDesc     // slice/array/map-value/ptr
	key     *typeDesc     // map-key
	colPlan *columnarPlan // non-nil on a []struct whose element is columnar-eligible

	// encode is the specialized encoder for this type. It receives a pointer
	// to a value of the type (NOT an interface) so it can dereference via
	// unsafe.Pointer without paying for reflect.Value.
	encode func(e *Encoder, p unsafe.Pointer) error
	decode func(d *Decoder, p unsafe.Pointer) error

	// marshalerKind: 0=none, 1=Marshaler interface, 2=encoding.TextMarshaler-ish (future)
	marshalerKind uint8
}

type fieldDesc struct {
	name   string
	offset uintptr
	desc   *typeDesc
	// preFast holds the pre-encoded fixstr/strN header for the field name in
	// Fast mode. Cheaper than emitting per-encode.
	preFast []byte
	// for Dense mode we cannot precompute the state ref bytes because IDs
	// are per-encoder. We can however precompute the bytes that get appended
	// the first time (intern record) and rely on the encoder's state table
	// to switch to refs on subsequent encodes.
	preInternStr []byte
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
	return descBuild(t, &buildCtx{inProgress: make(map[reflect.Type]*typeDesc)})
}

// descBuild is the recursion-safe lookup used from inside fillDesc. It
// MUST be called with a non-nil ctx so type cycles can be broken.
func descBuild(t reflect.Type, ctx *buildCtx) (*typeDesc, error) {
	if v, ok := typeCache.Load(t); ok {
		return v.(*typeDesc), nil
	}
	if td, ok := ctx.inProgress[t]; ok {
		return td, nil
	}
	td := &typeDesc{rType: t, kind: t.Kind()}
	ctx.inProgress[t] = td
	err := fillDesc(td, t, ctx)
	delete(ctx.inProgress, t)
	if err != nil {
		return nil, err
	}
	// Race-safe publish. If another goroutine already published the same
	// type while we were building, prefer theirs; our td stays alive only
	// as long as our caller's closures keep it referenced. Either way,
	// both descriptors encode identical wire output.
	actual, _ := typeCache.LoadOrStore(t, td)
	return actual.(*typeDesc), nil
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
			if elem.kind == reflect.Struct {
				td.colPlan = buildColumnarPlan(elem)
			}
			td.encode = encodeSlice(elem, t.Elem().Size(), td.colPlan)
			td.decode = decodeSlice(t, elem, t.Elem().Size(), td.colPlan)
		}
		// Preserve the nil-vs-empty distinction for every slice-typed FIELD
		// (bytes / typed-fast-path / reflect), consistent with maps and pointers.
		// Wrapping at the field descriptor keeps the shared encodeSlice* funcs
		// nil-agnostic for their internal direct callers (dense columns).
		td.encode = preserveNilSliceEncode(td.encode)
		td.decode = preserveNilSliceDecode(td.decode)
	case reflect.Array:
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
		td.key = k
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
		if tag, ok := sf.Tag.Lookup("qdf"); ok {
			if tag == "-" {
				continue
			}
			parts := strings.Split(tag, ",")
			if parts[0] != "" {
				name = parts[0]
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
