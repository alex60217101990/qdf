package qdf

import (
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/alex60217101990/qdf/internal/unsafestr"
)

// typeDesc is the compiled descriptor for a reflect.Type. Looked up via the
// typeCache map keyed by the Type's runtime pointer (cheap and stable).
type typeDesc struct {
	kind   reflect.Kind
	rType  reflect.Type
	fields []fieldDesc // structs only
	elem   *typeDesc   // slice/array/map-value/ptr
	key    *typeDesc   // map-key

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
			td.encode = encodeSlice(elem, t.Elem().Size())
			td.decode = decodeSlice(t, elem, t.Elem().Size())
		}
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
	return nil
}

func buildStructFields(t reflect.Type, ctx *buildCtx) ([]fieldDesc, error) {
	out := make([]fieldDesc, 0, t.NumField())
	for sf := range t.Fields() {
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
			offset:       sf.Offset,
			desc:         fd,
			preFast:      precomputeFixstrHeader(name),
			preInternStr: precomputeInternStrHeader(name),
		})
	}
	// stable order = source order (no sort) — matches encoding/json behavior
	// for fixed-layout decode.
	_ = sort.Strings
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

func encodeBytes(e *Encoder, p unsafe.Pointer) error { e.WriteBytes(*(*[]byte)(p)); return nil }
func decodeBytes(d *Decoder, p unsafe.Pointer) error {
	b, err := d.ReadBytes()
	if err != nil {
		return err
	}
	*(*[]byte)(p) = b
	return nil
}

func encodeSlice(elem *typeDesc, stride uintptr) func(*Encoder, unsafe.Pointer) error {
	return func(e *Encoder, p unsafe.Pointer) error {
		hdr := (*sliceHeader)(p)
		e.WriteArrayHeader(hdr.Len)
		base := hdr.Data
		for i := 0; i < hdr.Len; i++ {
			if err := elem.encode(e, unsafe.Add(base, uintptr(i)*stride)); err != nil {
				return err
			}
		}
		return nil
	}
}

func decodeSlice(t reflect.Type, elem *typeDesc, stride uintptr) func(*Decoder, unsafe.Pointer) error {
	return func(d *Decoder, p unsafe.Pointer) error {
		n, err := d.ReadArrayHeader()
		if err != nil {
			return err
		}
		if err := d.CheckLength(n, 1); err != nil {
			return err
		}
		// makeSliceUnsafe is swapped out under -tags qdf_reflect2 for the
		// reflect2 implementation (skips reflect.MakeSlice type checks).
		makeSliceUnsafe(t, n, p)
		base := sliceDataUnsafe(t, p)
		for i := range n {
			if err := elem.decode(d, unsafe.Add(base, uintptr(i)*stride)); err != nil {
				return err
			}
		}
		return nil
	}
}

func encodeArray(elem *typeDesc, stride uintptr, n int) func(*Encoder, unsafe.Pointer) error {
	return func(e *Encoder, p unsafe.Pointer) error {
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
	return func(e *Encoder, p unsafe.Pointer) error {
		rv := reflect.NewAt(t, p).Elem()
		if rv.IsNil() {
			e.WriteNil()
			return nil
		}
		n := rv.Len()
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
		// entry on the previous path, O(N) total. Now the two
		// scratch Values are allocated once before the loop and reused.
		keyHolder := reflect.New(keyType).Elem()
		valHolder := reflect.New(valType).Elem()
		kp := unsafe.Pointer(keyHolder.UnsafeAddr())
		vp := unsafe.Pointer(valHolder.UnsafeAddr())
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

func decodeMap(t reflect.Type, k, v *typeDesc) func(*Decoder, unsafe.Pointer) error {
	keyType := t.Key()
	valType := t.Elem()
	return func(d *Decoder, p unsafe.Pointer) error {
		tag, err := d.peekTag()
		if err != nil {
			return err
		}
		if tag == tagNil {
			d.i++
			reflect.NewAt(t, p).Elem().Set(reflect.Zero(t))
			return nil
		}
		n, err := d.ReadMapHeader()
		if err != nil {
			return err
		}
		if err := d.CheckLength(n, 2); err != nil {
			return err
		}
		// Allocate via the swappable backend.
		makeMapUnsafe(t, n, p)
		mapVal := reflect.NewAt(t, p).Elem()
		for range n {
			kv := reflect.New(keyType).Elem()
			if err := k.decode(d, unsafe.Pointer(kv.UnsafeAddr())); err != nil {
				return err
			}
			vv := reflect.New(valType).Elem()
			if err := v.decode(d, unsafe.Pointer(vv.UnsafeAddr())); err != nil {
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
		// Emit as a map for compatibility with map-shaped consumers. The map
		// keys are the struct field names.
		e.writeHeader()
		// Map header.
		n := len(fields)
		e.WriteMapHeader(n)
		for i := range fields {
			f := &fields[i]
			if e.state != nil && len(f.name) >= e.minIntern && len(e.state.ids) < e.maxStateEntries {
				if id, ok := e.state.lookupOrAssign(f.name); ok {
					// Routed through emitStateRef so the Markov-0
					// predictor sees this state-ref emission. Without
					// this, the next WriteString / WriteBytes that
					// happens to hit the SAME intern ID as a non-
					// updated lastID would erroneously collapse to
					// tagStateRepeat and decode to the wrong value.
					e.emitStateRef(id)
				} else {
					e.buf = append(e.buf, f.preInternStr...)
					e.state.lastID = id
					e.state.lastValid = true
				}
			} else {
				e.buf = append(e.buf, f.preFast...)
				if e.state != nil {
					e.state.lastValid = false
				}
			}
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
		name string
		f    *fieldDesc
	}
	indexed := make([]idx, len(td.fields))
	for i := range td.fields {
		indexed[i] = idx{td.fields[i].name, &td.fields[i]}
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

	return func(d *Decoder, p unsafe.Pointer) error {
		tag, err := d.peekTag()
		if err != nil {
			return err
		}
		if tag == tagNil {
			d.i++
			return nil
		}
		n, err := d.ReadMapHeader()
		if err != nil {
			return err
		}
		for range n {
			kb, err := d.readStringBytes()
			if err != nil {
				return err
			}
			name := unsafestr.String(kb)
			var fd *fieldDesc
			if useMap {
				fd = byName[name]
			} else {
				for j := range indexed {
					if indexed[j].name == name {
						fd = indexed[j].f
						break
					}
				}
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
	return encodeReflect(e, iv)
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
func decodeAny(d *Decoder) (any, error) {
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
	case tagStr8, tagStr16, tagStr32, tagInternStr, tagStateRef:
		return d.ReadString()
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
		out := make(map[string]any, n)
		for range n {
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
	case tagTimestamp:
		ns, err := d.ReadTimestampNano()
		return time.Unix(0, ns), err
	}
	return nil, ErrBadTag
}

func encodeTime(e *Encoder, p unsafe.Pointer) error {
	t := *(*time.Time)(p)
	e.WriteTimestampNano(t.UnixNano())
	return nil
}
func decodeTime(d *Decoder, p unsafe.Pointer) error {
	ns, err := d.ReadTimestampNano()
	if err != nil {
		return err
	}
	*(*time.Time)(p) = time.Unix(0, ns)
	return nil
}

func encodeMarshaler(t reflect.Type) func(*Encoder, unsafe.Pointer) error {
	return func(e *Encoder, p unsafe.Pointer) error {
		m := reflect.NewAt(t, p).Interface().(Marshaler)
		e.writeHeader()
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
		u := reflect.NewAt(t, p).Interface().(Unmarshaler)
		n, err := u.UnmarshalQDF(d.buf[d.i:])
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
	// Value passed by-value: must copy to an addressable location to take
	// unsafe.Pointer of. One allocation.
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
	td, err := descOf(t)
	if err != nil {
		return err
	}
	return td.decode(d, unsafe.Pointer(rv.Pointer()))
}

// sliceHeader mirrors reflect.SliceHeader using unsafe.Pointer instead of
// uintptr so the GC can see the data pointer. Required for taking pointers
// out of a slice without losing it to the GC.
type sliceHeader struct {
	Data unsafe.Pointer
	Len  int
	Cap  int
}
