package qdf

import (
	"reflect"
	"unsafe"

	"github.com/alex60217101990/qdf/internal/reflectutil"
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

// maxColumnarElems caps the struct count a tagColStruct header may claim. The
// per-byte length sanity check used for row-major slices does not apply here:
// a constant column compresses M structs into a few bytes, so M is not bounded
// by the remaining buffer. This fixed ceiling guards the output MakeSlice
// against a hostile element count while staying well above any realistic batch
// (callers with more rows should shard or stream).
const maxColumnarElems = 1 << 24

// checkColumnarN validates a tagColStruct struct count. Unlike row-major
// slices, a columnar count is not byte-bounded (compressed columns), so it is
// checked against a fixed ceiling rather than the remaining buffer length.
func checkColumnarN(n int) error {
	if n < 0 || n > maxColumnarElems {
		return ErrInvalidLength
	}
	return nil
}

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
	sample := min(n, columnarProbeSample)
	var rowBytes, colBytes int
	for c := range plan.cols {
		col := &plan.cols[c]
		switch col.kind {
		case colKindInt, colKindUint:
			var mn, mx uint64 = ^uint64(0), 0
			distinct := map[uint64]struct{}{}
			for i := range sample {
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
			for i := range sample {
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

func (e *Encoder) encodeColumnar(plan *columnarPlan, base unsafe.Pointer, n int) error {
	st := e.state
	e.buf = append(e.buf, tagColStruct)
	e.buf = appendUvarint(e.buf, uint64(n))

	names := make([]string, len(plan.cols))
	kinds := make([]colKind, len(plan.cols))
	for i := range plan.cols {
		names[i] = plan.cols[i].name
		kinds[i] = plan.cols[i].kind
	}
	if id := st.colShapeFor(names, kinds); id != 0 {
		e.buf = appendUvarint(e.buf, uint64(id))
	} else {
		e.buf = appendUvarint(e.buf, 0)
		e.buf = appendUvarint(e.buf, uint64(len(plan.cols)))
		for i := range plan.cols {
			e.WriteString(plan.cols[i].name)
			e.buf = append(e.buf, byte(plan.cols[i].kind))
		}
		st.colShapeDeclare(names, kinds)
	}

	for c := range plan.cols {
		col := &plan.cols[c]
		switch col.kind {
		case colKindInt:
			s := st.colScratchI64[:0]
			for i := range n {
				s = append(s, int64(loadScalarU64Signed(base, plan.stride, col, i)))
			}
			st.colScratchI64 = s
			if err := encodeSliceInt64(e, unsafe.Pointer(&s)); err != nil {
				return err
			}
		case colKindUint:
			s := st.colScratchU64[:0]
			for i := range n {
				s = append(s, loadScalarU64(base, plan.stride, col, i))
			}
			st.colScratchU64 = s
			if err := encodeSliceUint64(e, unsafe.Pointer(&s)); err != nil {
				return err
			}
		case colKindFloat:
			s := st.colScratchF64[:0]
			for i := range n {
				s = append(s, loadFloat64Field(base, plan.stride, col, i))
			}
			st.colScratchF64 = s
			if err := encodeSliceFloat64(e, unsafe.Pointer(&s)); err != nil {
				return err
			}
		case colKindBool:
			s := st.colScratchBool[:0]
			for i := range n {
				p := unsafe.Add(base, uintptr(i)*plan.stride+col.offset)
				s = append(s, *(*bool)(p))
			}
			st.colScratchBool = s
			if err := encodeSliceBool(e, unsafe.Pointer(&s)); err != nil {
				return err
			}
		case colKindString:
			if col.isByte {
				for i := range n {
					p := unsafe.Add(base, uintptr(i)*plan.stride+col.offset)
					e.WriteBytes(*(*[]byte)(p))
				}
				break
			}
			s := st.colScratchStr[:0]
			for i := range n {
				s = append(s, loadStringField(base, plan.stride, col, i))
			}
			st.colScratchStr = s
			if e.tryWriteStringColumnDict(s) {
				break // dict form emitted; never-worse gate already passed
			}
			for _, v := range s {
				e.WriteString(v)
			}
		}
	}
	return nil
}

//go:nosplit
func loadScalarU64Signed(base unsafe.Pointer, stride uintptr, col *colColumn, i int) uint64 {
	p := unsafe.Add(base, uintptr(i)*stride+col.offset)
	switch col.width {
	case 1:
		return uint64(int64(*(*int8)(p)))
	case 2:
		return uint64(int64(*(*int16)(p)))
	case 4:
		return uint64(int64(*(*int32)(p)))
	default:
		return *(*uint64)(p)
	}
}

//go:nosplit
func loadFloat64Field(base unsafe.Pointer, stride uintptr, col *colColumn, i int) float64 {
	p := unsafe.Add(base, uintptr(i)*stride+col.offset)
	if col.width == 4 {
		return float64(*(*float32)(p))
	}
	return *(*float64)(p)
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

func decodeColumnar(d *Decoder, t reflect.Type, plan *columnarPlan, p unsafe.Pointer) error {
	d.i++ // consume tagColStruct
	n64, k := readUvarint(d.buf[d.i:])
	if k <= 0 {
		return ErrInvalidLength
	}
	d.i += k
	n := int(n64)
	if err := checkColumnarN(n); err != nil {
		return err
	}
	if d.state == nil {
		d.state = newDecState()
	}
	idv, k2 := readUvarint(d.buf[d.i:])
	if k2 <= 0 {
		return ErrInvalidLength
	}
	d.i += k2

	var sh *decColShape
	if idv == 0 {
		cnt64, k3 := readUvarint(d.buf[d.i:])
		if k3 <= 0 {
			return ErrInvalidLength
		}
		d.i += k3
		cnt := int(cnt64)
		if err := d.CheckLength(cnt, 1); err != nil {
			return err
		}
		names := make([]string, cnt)
		kinds := make([]colKind, cnt)
		for i := range cnt {
			s, err := d.readStringBytes()
			if err != nil {
				return err
			}
			names[i] = string(s)
			if d.i >= len(d.buf) {
				return ErrShortBuffer
			}
			kinds[i] = colKind(d.buf[d.i])
			d.i++
		}
		sh = d.state.colShapeDeclareDec(names, kinds)
	} else {
		sh = d.state.colShapeLookup(uint32(idv))
		if sh == nil {
			return ErrUnknownStateID
		}
	}

	reflectutil.MakeSlice(t, n, p)
	base := reflectutil.SliceData(t, p)

	for c := range plan.cols {
		col := &plan.cols[c]
		if c >= len(sh.kinds) || sh.kinds[c] != col.kind {
			return ErrTypeMismatch
		}
		switch col.kind {
		case colKindInt:
			var s []int64
			if err := decodeSliceInt64(d, unsafe.Pointer(&s)); err != nil {
				return err
			}
			if len(s) != n {
				return ErrTypeMismatch
			}
			for i := range n {
				storeScalarFromI64(base, plan.stride, col, i, s[i])
			}
		case colKindUint:
			var s []uint64
			if err := decodeSliceUint64(d, unsafe.Pointer(&s)); err != nil {
				return err
			}
			if len(s) != n {
				return ErrTypeMismatch
			}
			for i := range n {
				storeScalarFromU64(base, plan.stride, col, i, s[i])
			}
		case colKindFloat:
			var s []float64
			if err := decodeSliceFloat64(d, unsafe.Pointer(&s)); err != nil {
				return err
			}
			if len(s) != n {
				return ErrTypeMismatch
			}
			for i := range n {
				storeFloat64(base, plan.stride, col, i, s[i])
			}
		case colKindBool:
			var s []bool
			if err := decodeSliceBool(d, unsafe.Pointer(&s)); err != nil {
				return err
			}
			if len(s) != n {
				return ErrTypeMismatch
			}
			for i := range n {
				*(*bool)(unsafe.Add(base, uintptr(i)*plan.stride+col.offset)) = s[i]
			}
		case colKindString:
			if !col.isByte && d.i < len(d.buf) && d.buf[d.i] == tagColStrDict {
				table, idx, err := d.readStringColumnDict(n)
				if err != nil {
					return err
				}
				for i := range n {
					dp := unsafe.Add(base, uintptr(i)*plan.stride+col.offset)
					*(*string)(dp) = table[idx[i]]
				}
				break
			}
			for i := range n {
				sb, err := d.readStringBytes()
				if err != nil {
					return err
				}
				dp := unsafe.Add(base, uintptr(i)*plan.stride+col.offset)
				if col.isByte {
					*(*[]byte)(dp) = append([]byte(nil), sb...)
				} else {
					*(*string)(dp) = string(sb)
				}
			}
		}
	}
	return nil
}

//go:nosplit
func storeScalarFromI64(base unsafe.Pointer, stride uintptr, col *colColumn, i int, v int64) {
	p := unsafe.Add(base, uintptr(i)*stride+col.offset)
	switch col.width {
	case 1:
		*(*int8)(p) = int8(v)
	case 2:
		*(*int16)(p) = int16(v)
	case 4:
		*(*int32)(p) = int32(v)
	default:
		*(*int64)(p) = v
	}
}

//go:nosplit
func storeScalarFromU64(base unsafe.Pointer, stride uintptr, col *colColumn, i int, v uint64) {
	p := unsafe.Add(base, uintptr(i)*stride+col.offset)
	switch col.width {
	case 1:
		*(*uint8)(p) = uint8(v)
	case 2:
		*(*uint16)(p) = uint16(v)
	case 4:
		*(*uint32)(p) = uint32(v)
	default:
		*(*uint64)(p) = v
	}
}

//go:nosplit
func storeFloat64(base unsafe.Pointer, stride uintptr, col *colColumn, i int, v float64) {
	p := unsafe.Add(base, uintptr(i)*stride+col.offset)
	if col.width == 4 {
		*(*float32)(p) = float32(v)
	} else {
		*(*float64)(p) = v
	}
}

// decodeColumnarAny decodes a tagColStruct payload into a []any of
// map[string]any keyed by column name. Mirrors decodeColumnar's header
// and shape parse exactly; each column is decoded into a temp slice and
// the per-element value boxed into its row's map.
func decodeColumnarAny(d *Decoder) (any, error) {
	d.i++
	n64, k := readUvarint(d.buf[d.i:])
	if k <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += k
	n := int(n64)
	if err := checkColumnarN(n); err != nil {
		return nil, err
	}
	if d.state == nil {
		d.state = newDecState()
	}
	idv, k2 := readUvarint(d.buf[d.i:])
	if k2 <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += k2
	var sh *decColShape
	if idv == 0 {
		cnt64, k3 := readUvarint(d.buf[d.i:])
		if k3 <= 0 {
			return nil, ErrInvalidLength
		}
		d.i += k3
		cnt := int(cnt64)
		if err := d.CheckLength(cnt, 1); err != nil {
			return nil, err
		}
		names := make([]string, cnt)
		kinds := make([]colKind, cnt)
		for i := range cnt {
			s, err := d.readStringBytes()
			if err != nil {
				return nil, err
			}
			names[i] = string(s)
			if d.i >= len(d.buf) {
				return nil, ErrShortBuffer
			}
			kinds[i] = colKind(d.buf[d.i])
			d.i++
		}
		sh = d.state.colShapeDeclareDec(names, kinds)
	} else {
		sh = d.state.colShapeLookup(uint32(idv))
		if sh == nil {
			return nil, ErrUnknownStateID
		}
	}
	out := make([]any, n)
	for i := range out {
		out[i] = make(map[string]any, len(sh.names))
	}
	for c := range sh.kinds {
		name := sh.names[c]
		switch sh.kinds[c] {
		case colKindInt:
			var s []int64
			if err := decodeSliceInt64(d, unsafe.Pointer(&s)); err != nil {
				return nil, err
			}
			if len(s) != n {
				return nil, ErrTypeMismatch
			}
			for i := range n {
				out[i].(map[string]any)[name] = s[i]
			}
		case colKindUint:
			var s []uint64
			if err := decodeSliceUint64(d, unsafe.Pointer(&s)); err != nil {
				return nil, err
			}
			if len(s) != n {
				return nil, ErrTypeMismatch
			}
			for i := range n {
				out[i].(map[string]any)[name] = s[i]
			}
		case colKindFloat:
			var s []float64
			if err := decodeSliceFloat64(d, unsafe.Pointer(&s)); err != nil {
				return nil, err
			}
			if len(s) != n {
				return nil, ErrTypeMismatch
			}
			for i := range n {
				out[i].(map[string]any)[name] = s[i]
			}
		case colKindBool:
			var s []bool
			if err := decodeSliceBool(d, unsafe.Pointer(&s)); err != nil {
				return nil, err
			}
			if len(s) != n {
				return nil, ErrTypeMismatch
			}
			for i := range n {
				out[i].(map[string]any)[name] = s[i]
			}
		case colKindString:
			if d.i < len(d.buf) && d.buf[d.i] == tagColStrDict {
				table, idx, err := d.readStringColumnDict(n)
				if err != nil {
					return nil, err
				}
				for i := range n {
					out[i].(map[string]any)[name] = table[idx[i]]
				}
				break
			}
			for i := range n {
				sb, err := d.readStringBytes()
				if err != nil {
					return nil, err
				}
				out[i].(map[string]any)[name] = string(sb)
			}
		default:
			return nil, ErrBadTag
		}
	}
	return out, nil
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
