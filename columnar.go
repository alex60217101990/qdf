package qdf

import "reflect"

// colKind classifies a struct field for columnar encoding. Only these kinds
// are columnar-eligible; any other field kind makes the whole struct fall
// back to row-major.
type colKind uint8

const (
	colKindInt    colKind = iota // int, int8..int64  → []int64 column
	colKindUint                  // uint, uint8..uint64, uintptr → []uint64 column
	colKindFloat                 // float32, float64 → []float64 column
	colKindBool                  // bool → []bool column
	colKindString                // string, []byte → consecutive WriteString
)

// colColumn is one field's columnar descriptor: where it lives in each struct
// element and how to (de)serialize the column.
type colColumn struct {
	name   string
	offset uintptr
	kind   colKind
	width  uintptr // element width in bytes for the scalar load/store
	isByte bool    // true for []byte string columns (vs string)
}

// columnarPlan is cached on the slice element's typeDesc. nil means the
// element type is not columnar-eligible.
type columnarPlan struct {
	cols   []colColumn
	stride uintptr // struct size, for base + i*stride addressing
}

// columnarMinElems is the smallest slice length worth transposing; below it
// the shape-declaration + probe overhead is not amortized.
const columnarMinElems = 16

// buildColumnarPlan returns a plan if td is a struct whose every field is a
// columnar-eligible scalar/string, else nil. Called once at fillDesc time;
// the result is cached so the hot path never reflects.
func buildColumnarPlan(td *typeDesc) *columnarPlan {
	if td.kind != reflect.Struct || len(td.fields) == 0 {
		return nil
	}
	cols := make([]colColumn, 0, len(td.fields))
	for i := range td.fields {
		f := &td.fields[i]
		ck, w, isByte, ok := classifyColKind(f.desc)
		if !ok {
			return nil
		}
		cols = append(cols, colColumn{name: f.name, offset: f.offset, kind: ck, width: w, isByte: isByte})
	}
	return &columnarPlan{cols: cols, stride: td.rType.Size()}
}

func classifyColKind(fd *typeDesc) (ck colKind, width uintptr, isByte bool, ok bool) {
	if fd.marshalerKind != 0 {
		return 0, 0, false, false // custom marshaler → row-major
	}
	switch fd.kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return colKindInt, fd.rType.Size(), false, true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return colKindUint, fd.rType.Size(), false, true
	case reflect.Float32, reflect.Float64:
		return colKindFloat, fd.rType.Size(), false, true
	case reflect.Bool:
		return colKindBool, 1, false, true
	case reflect.String:
		return colKindString, 0, false, true
	case reflect.Slice:
		if fd.rType.Elem().Kind() == reflect.Uint8 { // []byte
			return colKindString, 0, true, true
		}
		return 0, 0, false, false
	default:
		return 0, 0, false, false
	}
}
