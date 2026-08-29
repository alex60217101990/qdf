package qdf

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"unsafe"
)

// batchFieldKind classifies how a batchField's bytes are interpreted when
// scattering a struct-of-arrays batch into per-column storage.
type batchFieldKind uint8

const (
	bfScalar batchFieldKind = iota // bool / ints / uints / floats
	bfStr                          // qdf.Str  <- wire string
	bfBytes                        // qdf.Bytes <- wire bin
	bfTime                         // qdf.Time <- wire timestamp
)

// batchField describes one wire-visible field of a batch-eligible struct:
// its wire key, byte offset within the struct, and how to interpret it.
type batchField struct {
	name       string  // wire key
	off        uintptr // byte offset within T
	width      uintptr // scalar byte width (field.Type.Size()); valid when kind == bfScalar
	kind       batchFieldKind
	scalarKind reflect.Kind // valid when kind == bfScalar
}

// batchPlan is the validated, cached layout of a pointer-free struct type
// eligible for batch (struct-of-arrays) handling. Field order follows the
// same wire field order the normal encoder uses for the equivalent struct
// (flattened embedded, tag-named) — see appendBatchFields.
type batchPlan struct {
	// mirrorSlicePtr pools *mirrorSlot values so repeated UnmarshalBatch calls
	// for the same T neither reallocate the *[]mirror box nor touch
	// reflect.Value on the hot path (the slot caches the raw slice-header
	// pointer alongside the any box Unmarshal needs).
	mirrorSlicePtr sync.Pool

	// Pointer-bearing fields lead, scalars trail (fieldalignment: trims the
	// GC-scanned prefix 128 -> 104 bytes). Constructed with named fields only.
	rt reflect.Type

	// mirror is a runtime-built struct type with the same field names/tags
	// as rt, with handle types swapped back to their decodable wire
	// counterparts: Str->string, Bytes->[]byte, Time->time.Time. Scalars are
	// unchanged. The normal reflect-driven Unmarshal decodes into
	// []mirror; the fallback copy pass (batch_decode.go) then scatters each
	// mirror row into a T row plus slab bytes. Since batchPlanOf only
	// accepts flat (non-nested) field sets, mirror's field offsets align
	// 1:1 with plan.fields via mirrorOff.
	mirror reflect.Type
	fields []batchField

	mirrorOff []uintptr // parallel to fields: byte offset within mirror

	// stride is rt.Size() — scalar tail (see the field-order note above).
	stride uintptr
}

// mirrorSlot pairs the pooled *[]mirror box (handed to Unmarshal as any) with
// its raw pointer, so the fallback path reads/writes the slice header directly
// through *sliceHeader instead of reflect.ValueOf(...).Elem()/Index/UnsafeAddr
// per decode. reflect.New runs only in the pool's New (cold, amortized); the
// hot path is reflection-free, which also keeps it neutral across the
// qdf_reflect2 build-tag split (no runtime reflect.Value in either mode).
type mirrorSlot struct {
	box any            // *[]mirror — the Unmarshal target
	ptr unsafe.Pointer // same value as a raw pointer to the slice header
}

// batchPlans caches reflect.Type -> *batchPlan (success) or error (failure),
// so a type that fails validation once doesn't re-walk reflection on every
// call.
var batchPlans sync.Map

var (
	strType   = reflect.TypeFor[Str]()
	bytesType = reflect.TypeFor[Bytes]()
	timeType2 = reflect.TypeFor[Time]()

	mirrorStringType = reflect.TypeFor[string]()
	mirrorBytesType  = reflect.TypeFor[[]byte]()
)

// buildBatchMirror builds p.mirror (a reflect.StructOf mirroring p.fields
// with handle types swapped for their decodable counterparts) and
// p.mirrorOff (each mirror field's byte offset, parallel to p.fields).
//
// Field names cannot reuse the source type's Go names: p.fields may combine
// fields flattened from anonymous-embedded structs (see appendBatchFields),
// so their original Go identifiers can collide or be unexported through the
// embedding path. Instead each mirror field gets a synthetic exported name
// (F0, F1, ...) carrying a `qdf:"<wire-name>"` tag — the same wire-key
// resolution the normal decoder already applies, so the mirror decodes the
// identical wire regardless of how T's fields were declared.
func buildBatchMirror(p *batchPlan) error {
	sf := make([]reflect.StructField, len(p.fields))
	for i, f := range p.fields {
		var ft reflect.Type
		switch f.kind {
		case bfStr:
			ft = mirrorStringType
		case bfBytes:
			ft = mirrorBytesType
		case bfTime:
			ft = timeType
		default: // bfScalar
			ft = scalarKindType(f.scalarKind)
			if ft == nil {
				return fmt.Errorf("qdf: batch type %s: field %s: unsupported scalar kind %s", p.rt, f.name, f.scalarKind)
			}
		}
		sf[i] = reflect.StructField{
			Name: fmt.Sprintf("F%d", i),
			Type: ft,
			Tag:  reflect.StructTag(fmt.Sprintf(`qdf:%q`, f.name)),
		}
	}
	mirror := reflect.StructOf(sf)
	p.mirror = mirror
	p.mirrorOff = make([]uintptr, len(p.fields))
	for i := range p.fields {
		p.mirrorOff[i] = mirror.Field(i).Offset
	}
	sliceT := reflect.SliceOf(mirror)
	p.mirrorSlicePtr.New = func() any {
		rv := reflect.New(sliceT)
		return &mirrorSlot{box: rv.Interface(), ptr: unsafe.Pointer(rv.Pointer())}
	}
	return nil
}

// scalarKindType maps a reflect.Kind (as recorded in batchField.scalarKind)
// back to its reflect.Type, for building the mirror struct's scalar fields.
func scalarKindType(k reflect.Kind) reflect.Type {
	switch k {
	case reflect.Bool:
		return reflect.TypeFor[bool]()
	case reflect.Int:
		return reflect.TypeFor[int]()
	case reflect.Int8:
		return reflect.TypeFor[int8]()
	case reflect.Int16:
		return reflect.TypeFor[int16]()
	case reflect.Int32:
		return reflect.TypeFor[int32]()
	case reflect.Int64:
		return reflect.TypeFor[int64]()
	case reflect.Uint:
		return reflect.TypeFor[uint]()
	case reflect.Uint8:
		return reflect.TypeFor[uint8]()
	case reflect.Uint16:
		return reflect.TypeFor[uint16]()
	case reflect.Uint32:
		return reflect.TypeFor[uint32]()
	case reflect.Uint64:
		return reflect.TypeFor[uint64]()
	case reflect.Uintptr:
		return reflect.TypeFor[uintptr]()
	case reflect.Float32:
		return reflect.TypeFor[float32]()
	case reflect.Float64:
		return reflect.TypeFor[float64]()
	default:
		return nil
	}
}

// batchPlanOf returns the cached batchPlan for t, validating and building it
// on first use. t must be a struct type that is entirely pointer-free:
// scalars (bool/int*/uint*/float*), qdf.Str, qdf.Bytes, qdf.Time, or an
// exported flattened-embedded struct composed of the same. See the v1 SCOPE
// DECISION in appendBatchFields for what is rejected.
func batchPlanOf(t reflect.Type) (*batchPlan, error) {
	if v, ok := batchPlans.Load(t); ok {
		if p, ok := v.(*batchPlan); ok {
			return p, nil
		}
		if err, ok := v.(error); ok {
			return nil, err
		}
		return nil, fmt.Errorf("qdf: batch plan cache holds %T for %s", v, t)
	}
	p := &batchPlan{rt: t, stride: t.Size()}
	if t.Kind() != reflect.Struct {
		err := fmt.Errorf("qdf: batch type %s: not a struct", t)
		batchPlans.Store(t, err)
		return nil, err
	}
	if err := appendBatchFields(p, t, 0, ""); err != nil {
		batchPlans.Store(t, err)
		return nil, err
	}
	if err := buildBatchMirror(p); err != nil {
		batchPlans.Store(t, err)
		return nil, err
	}
	actual, _ := batchPlans.LoadOrStore(t, p)
	if pp, ok := actual.(*batchPlan); ok {
		return pp, nil
	}
	return p, nil
}

// wireFieldKey extracts the wire key for a struct field using the same
// qdf/json tag rules as appendStructFields (reflect_desc.go): qdf:"-" or
// json:"-" (first comma segment) skips the field; qdf:"name" or json:"name"
// names it; otherwise the Go field name is used.
func wireFieldKey(sf reflect.StructField) (string, bool) {
	if tag, ok := sf.Tag.Lookup("qdf"); ok {
		if tag == "-" {
			return "", true
		}
		parts := strings.Split(tag, ",")
		if parts[0] != "" {
			return parts[0], false
		}
		return sf.Name, false
	}
	if tag, ok := sf.Tag.Lookup("json"); ok {
		parts := strings.Split(tag, ",")
		if parts[0] == "-" {
			return "", true
		}
		if parts[0] != "" {
			return parts[0], false
		}
		return sf.Name, false
	}
	return sf.Name, false
}

// appendBatchFields walks t's fields (mirroring appendStructFields's
// embedded-flattening and tag rules so wire order matches the normal
// encoder) and appends validated batchFields to p.
//
// v1 SCOPE DECISION: nested named (non-anonymous) structs and [N]arrays
// inside T are validated pointer-free but rejected with a clear error in
// v1 — they complicate the columnar scatter. Flatten them (embed instead of
// name) or drop them; this may be revisited in a later phase.
func appendBatchFields(p *batchPlan, t reflect.Type, base uintptr, path string) error {
	for sf := range t.Fields() {
		fieldPath := sf.Name
		if path != "" {
			fieldPath = path + "." + sf.Name
		}
		// Mirror the encoder's embedded flattening (appendStructFields,
		// reflect_desc.go): an anonymous value-struct flattens unless the type
		// carries its own value codec — time.Time or a Marshaler/Unmarshaler
		// implementor (the SAME interface rule the encoder applies; Str/Bytes/
		// Time fall out of it via the handle-type checks below when they reach
		// the regular field path). A `qdf:"-"`/`json:"-"` tag on the embedded
		// field itself opts the whole nested block out, exactly like the
		// encoder — without this the plan's field set diverges from the wire.
		if sf.Anonymous && sf.Type.Kind() == reflect.Struct &&
			sf.Type != strType && sf.Type != bytesType && sf.Type != timeType2 &&
			sf.Type != timeType &&
			!reflect.PointerTo(sf.Type).Implements(reflect.TypeFor[Marshaler]()) &&
			!reflect.PointerTo(sf.Type).Implements(reflect.TypeFor[Unmarshaler]()) {
			if _, skip := wireFieldKey(sf); skip {
				continue
			}
			if err := appendBatchFields(p, sf.Type, base+sf.Offset, path); err != nil {
				return err
			}
			continue
		}
		if !sf.IsExported() {
			continue
		}
		key, skip := wireFieldKey(sf)
		if skip {
			continue
		}
		bf := batchField{name: key, off: base + sf.Offset}
		switch sf.Type {
		case strType:
			bf.kind = bfStr
		case bytesType:
			bf.kind = bfBytes
		case timeType2:
			bf.kind = bfTime
		case timeType:
			return fmt.Errorf("qdf: batch type: field %s is time.Time — use qdf.Time", fieldPath)
		default:
			switch k := sf.Type.Kind(); k {
			case reflect.Bool,
				reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
				reflect.Float32, reflect.Float64:
				bf.kind, bf.scalarKind, bf.width = bfScalar, k, sf.Type.Size()
			case reflect.Struct:
				// nested pointer-free struct: flatten is NOT applied to named
				// (non-anonymous) fields on the wire; they are nested values.
				// v1: recurse for VALIDATION ONLY, decode via fallback path.
				if err := validateBatchStruct(sf.Type, fieldPath); err != nil {
					return err
				}
				return fmt.Errorf("qdf: batch type: field %s: nested struct fields decode via the row-major fallback in v1 — flatten it or use scalar/handle fields", fieldPath)
			case reflect.Array:
				if err := validateBatchElem(sf.Type.Elem(), fieldPath); err != nil {
					return err
				}
				return fmt.Errorf("qdf: batch type: field %s: array fields are v1-fallback only", fieldPath)
			case reflect.String:
				return fmt.Errorf("qdf: batch type: field %s is string — use qdf.Str", fieldPath)
			case reflect.Slice:
				return fmt.Errorf("qdf: batch type: field %s is a slice — use qdf.Bytes for []byte, or drop the field", fieldPath)
			case reflect.Map, reflect.Pointer, reflect.Interface, reflect.Chan, reflect.Func:
				return fmt.Errorf("qdf: batch type: field %s (%s) is not pointer-free", fieldPath, k)
			default:
				return fmt.Errorf("qdf: batch type: field %s: unsupported kind %s", fieldPath, k)
			}
		}
		p.fields = append(p.fields, bf)
	}
	return nil
}

// validateBatchStruct recursively checks that a nested struct type (one that
// v1 rejects for columnar scatter regardless) is at least pointer-free, so
// the caller's error names the actual offending nested field
// (e.g. "In.M") rather than the outer struct field.
func validateBatchStruct(t reflect.Type, path string) error {
	for sf := range t.Fields() {
		fieldPath := path + "." + sf.Name
		if !sf.IsExported() {
			continue
		}
		if err := validateBatchElem(sf.Type, fieldPath); err != nil {
			return err
		}
	}
	return nil
}

// validateBatchElem checks that a type (a struct field's type, or an array
// element type) is pointer-free: a scalar, one of the handle types, a
// pointer-free nested struct, or a pointer-free array. Anything else
// (string, slice, map, pointer, interface, chan, func) is rejected.
func validateBatchElem(t reflect.Type, path string) error {
	switch t {
	case strType, bytesType, timeType2:
		return nil
	case timeType:
		return fmt.Errorf("qdf: batch type: field %s is time.Time — use qdf.Time", path)
	}
	switch k := t.Kind(); k {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		return nil
	case reflect.Struct:
		return validateBatchStruct(t, path)
	case reflect.Array:
		return validateBatchElem(t.Elem(), path)
	case reflect.String:
		return fmt.Errorf("qdf: batch type: field %s is string — use qdf.Str", path)
	case reflect.Slice:
		return fmt.Errorf("qdf: batch type: field %s is a slice — use qdf.Bytes for []byte, or drop the field", path)
	case reflect.Map, reflect.Pointer, reflect.Interface, reflect.Chan, reflect.Func:
		return fmt.Errorf("qdf: batch type: field %s (%s) is not pointer-free", path, k)
	default:
		return fmt.Errorf("qdf: batch type: field %s: unsupported kind %s", path, k)
	}
}
