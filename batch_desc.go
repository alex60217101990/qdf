package qdf

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
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
	name       string // wire key
	off        uintptr
	kind       batchFieldKind
	scalarKind reflect.Kind // valid when kind == bfScalar
}

// batchPlan is the validated, cached layout of a pointer-free struct type
// eligible for batch (struct-of-arrays) handling. Field order follows the
// same wire field order the normal encoder uses for the equivalent struct
// (flattened embedded, tag-named) — see appendBatchFields.
type batchPlan struct {
	rt     reflect.Type
	stride uintptr
	fields []batchField
}

// batchPlans caches reflect.Type -> *batchPlan (success) or error (failure),
// so a type that fails validation once doesn't re-walk reflection on every
// call.
var batchPlans sync.Map

var (
	strType    = reflect.TypeFor[Str]()
	bytesType  = reflect.TypeFor[Bytes]()
	timeType2  = reflect.TypeFor[Time]()
	timeTimeRT = reflect.TypeFor[time.Time]()
)

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
		return nil, v.(error)
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
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		fieldPath := sf.Name
		if path != "" {
			fieldPath = path + "." + sf.Name
		}
		// Mirror the encoder's embedded flattening (appendStructFields):
		// anonymous value-structs flatten unless they carry their own codec.
		if sf.Anonymous && sf.Type.Kind() == reflect.Struct &&
			sf.Type != strType && sf.Type != bytesType && sf.Type != timeType2 {
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
		case timeTimeRT:
			return fmt.Errorf("qdf: batch type: field %s is time.Time — use qdf.Time", fieldPath)
		default:
			switch k := sf.Type.Kind(); k {
			case reflect.Bool,
				reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
				reflect.Float32, reflect.Float64:
				bf.kind, bf.scalarKind = bfScalar, k
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
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
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
	case timeTimeRT:
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
