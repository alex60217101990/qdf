package qdf

import (
	"reflect"
	"unsafe"
)

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

// colShapeDeclare registers a new columnar shape (names + kinds) on the encoder
// and returns its 1-based wire ID. Always appends; call colShapeFor first to
// avoid duplicates.
func (e *encState) colShapeDeclare(names []string, kinds []colKind) uint32 {
	e.colShapeNames = append(e.colShapeNames, names)
	e.colShapeKinds = append(e.colShapeKinds, kinds)
	return uint32(len(e.colShapeNames)) // ids start at 1
}

// colShapeFor returns the 1-based wire ID for an already-declared columnar shape
// whose names and kinds match exactly, or 0 if not found.
func (e *encState) colShapeFor(names []string, kinds []colKind) uint32 {
	for i := range e.colShapeNames {
		if colShapeEq(e.colShapeNames[i], e.colShapeKinds[i], names, kinds) {
			return uint32(i + 1)
		}
	}
	return 0
}

// colShapeEq reports whether two (names, kinds) pairs are structurally identical.
func colShapeEq(an []string, ak []colKind, bn []string, bk []colKind) bool {
	if len(an) != len(bn) || len(ak) != len(bk) {
		return false
	}
	for i := range an {
		if an[i] != bn[i] || ak[i] != bk[i] {
			return false
		}
	}
	return true
}

// colShapeDeclareDec appends a new columnar shape to the decoder's table and
// returns a pointer to the stored entry (wire ID = len after append).
func (d *decState) colShapeDeclareDec(names []string, kinds []colKind) *decColShape {
	d.colShapes = append(d.colShapes, decColShape{names: names, kinds: kinds})
	return &d.colShapes[len(d.colShapes)-1]
}

// colShapeLookup returns the columnar shape with the given 1-based wire ID,
// or nil if the ID is out of range.
func (d *decState) colShapeLookup(id uint32) *decColShape {
	if id == 0 || id > uint32(len(d.colShapes)) {
		return nil
	}
	return &d.colShapes[id-1]
}

const (
	columnarProbeSample = 32
	// columnarMinGainPct is the minimum estimated wire reduction (percent of
	// the row-major estimate) required to commit to columnar.
	columnarMinGainPct = 10
)

// columnarProbe samples up to columnarProbeSample elements and estimates
// whether column-major beats row-major on those samples. Conservative: any
// uncertainty falls back to row-major (returns false).
func columnarProbe(plan *columnarPlan, base unsafe.Pointer, n int) bool {
	sample := n
	if sample > columnarProbeSample {
		sample = columnarProbeSample
	}
	var rowBytes, colBytes int
	for c := range plan.cols {
		col := &plan.cols[c]
		switch col.kind {
		case colKindInt, colKindUint:
			var mn, mx uint64 = ^uint64(0), 0
			distinct := map[uint64]struct{}{}
			for i := 0; i < sample; i++ {
				v := loadScalarU64(base, plan.stride, col, i)
				if v < mn {
					mn = v
				}
				if v > mx {
					mx = v
				}
				if len(distinct) <= 16 {
					distinct[v] = struct{}{}
				}
				rowBytes += uvarintLen(v) // row-major: one varint per value
			}
			spread := mx - mn
			bits := 0
			for spread > 0 {
				bits++
				spread >>= 1
			}
			forEst := 3 + (bits*sample+7)/8
			dictEst := 1 << 30
			if len(distinct) <= 16 {
				idxBits := 0
				for (1 << idxBits) < len(distinct) {
					idxBits++
				}
				dictEst = 3 + len(distinct)*8 + (idxBits*sample+7)/8
			}
			colBytes += min(forEst, dictEst)
		case colKindFloat:
			rowBytes += sample * 8
			colBytes += sample * 8 // raw-LE both ways; no probe win (Gorilla is opt-in)
		case colKindBool:
			rowBytes += sample
			colBytes += (sample + 7) / 8
		case colKindString:
			prev := ""
			first := true
			for i := 0; i < sample; i++ {
				s := loadStringField(base, plan.stride, col, i)
				if !first && s == prev {
					colBytes += 1 // tagStateRepeat
				} else {
					colBytes += 2 + len(s)
				}
				rowBytes += 2 + len(s)
				prev = s
				first = false
			}
		}
	}
	if rowBytes == 0 {
		return false
	}
	return colBytes*100 <= rowBytes*(100-columnarMinGainPct)
}

//go:nosplit
func loadScalarU64(base unsafe.Pointer, stride uintptr, col *colColumn, i int) uint64 {
	p := unsafe.Add(base, uintptr(i)*stride+col.offset)
	switch col.width {
	case 1:
		return uint64(*(*uint8)(p))
	case 2:
		return uint64(*(*uint16)(p))
	case 4:
		return uint64(*(*uint32)(p))
	default:
		return *(*uint64)(p)
	}
}

//go:nosplit
func loadStringField(base unsafe.Pointer, stride uintptr, col *colColumn, i int) string {
	p := unsafe.Add(base, uintptr(i)*stride+col.offset)
	if col.isByte {
		return string(*(*[]byte)(p))
	}
	return *(*string)(p)
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
