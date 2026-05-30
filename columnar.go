package qdf

import (
	"encoding/binary"
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

	// colKindNullable is OR'd onto a base kind in the columnar shape's kind
	// byte to mark an optional (pointer-to-scalar/string) column: a `*T`
	// field. The column is stored as a presence bitmap (1 bit per row) plus
	// a dense column of only the present values, encoded with the base
	// kind's normal codec. Nullability is a static property of the field
	// type, so it travels in the shape declaration — no separate wire tag.
	colKindNullable colKind = 0x80
)

// base returns the kind without the nullable flag.
func (k colKind) base() colKind { return k &^ colKindNullable }

// isNullable reports whether the column is an optional (pointer) column.
func (k colKind) isNullable() bool { return k&colKindNullable != 0 }

// colColumn is one field's columnar descriptor: where it lives in each struct
// element and how to (de)serialize the column.
type colColumn struct {
	name   string
	offset uintptr
	kind   colKind // base kind, OR'd with colKindNullable for *T columns
	width  uintptr // element width in bytes for the scalar load/store
	isByte bool    // true for []byte string columns (vs string)
	// elemType is the pointed-to type for a nullable (*T) column, used to
	// allocate the present values on decode. nil for non-nullable columns.
	elemType reflect.Type
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

// maxColumnarAnyElems caps the row count for the reflective map[string]any
// decode, which allocates one map per row up front (much heavier than the
// typed struct path's single backing slice). Kept well above any realistic
// ad-hoc decode; callers with more rows should decode into a typed struct.
const maxColumnarAnyElems = 1 << 16

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
		fd := f.desc
		// An optional (*T) field becomes a nullable column: classify the
		// pointed-to type and remember its reflect.Type for decode allocation.
		var elemType reflect.Type
		nullable := false
		if fd.kind == reflect.Pointer {
			if fd.elem == nil {
				return nil
			}
			nullable = true
			elemType = fd.rType.Elem()
			fd = fd.elem
		}
		ck, w, isByte, ok := classifyColKind(fd)
		if !ok {
			return nil
		}
		if nullable {
			// Nullable scalar/bool/string pointers are columnar; nullable
			// []byte still falls back to row-major.
			if isByte { // nullable []byte still unsupported; nullable string now allowed
				return nil
			}
			ck |= colKindNullable
		}
		cols = append(cols, colColumn{name: f.name, offset: f.offset, kind: ck, width: w, isByte: isByte, elemType: elemType})
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
			// Distinct values bounded to 17 (cardinality > 16 disables the dict
			// estimate) tracked in a stack array with a linear scan — no map
			// allocation per probed column.
			var seen [17]uint64
			ndistinct := 0
			for i := range sample {
				v := loadScalarU64(base, plan.stride, col, i)
				if v < mn {
					mn = v
				}
				if v > mx {
					mx = v
				}
				if ndistinct <= 16 {
					found := false
					for j := 0; j < ndistinct; j++ {
						if seen[j] == v {
							found = true
							break
						}
					}
					if !found {
						seen[ndistinct] = v
						ndistinct++
					}
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
			if ndistinct <= 16 {
				idxBits := 0
				for (1 << idxBits) < ndistinct {
					idxBits++
				}
				dictEst = 3 + ndistinct*8 + (idxBits*sample+7)/8
			}
			colBytes += min(forEst, dictEst)
		case colKindFloat:
			rowBytes += sample * 8
			colBytes += sample * 8 // raw-LE both ways; no probe win (Gorilla is opt-in)
		case colKindBool:
			rowBytes += sample
			colBytes += (sample + 7) / 8
		case colKindString:
			// Estimate the column two ways over the sample and keep the
			// cheaper, mirroring the encoder: per-value (with consecutive-repeat
			// collapse) OR a string dictionary (distinct table + ceil(log2
			// distinct) bits/row). Without the dictionary term the probe would
			// decline columnar for a low-cardinality string column that never
			// repeats consecutively, even though the dictionary crushes it.
			//
			// Distinct values are tracked in a stack array with a linear scan,
			// not a map: the sample is <= columnarProbeSample, so the set is
			// tiny and a map would heap-allocate buckets on every probed column.
			var seen [columnarProbeSample]string
			nseen := 0
			var tableBytes, perValue int
			prev := ""
			first := true
			for i := range sample {
				s := loadStringField(base, plan.stride, col, i)
				fresh := true
				for j := 0; j < nseen; j++ {
					if seen[j] == s {
						fresh = false
						break
					}
				}
				if fresh && nseen < len(seen) {
					seen[nseen] = s
					nseen++
					tableBytes += 2 + len(s)
				}
				if !first && s == prev {
					perValue += 1 // tagStateRepeat
				} else {
					perValue += 2 + len(s)
				}
				rowBytes += 2 + len(s)
				prev = s
				first = false
			}
			dictBytes := tableBytes + (sample*bitsForDistinct(nseen)+7)/8
			colBytes += min(perValue, dictBytes)
		default:
			if !col.kind.isNullable() {
				continue // unknown kind contributes nothing
			}
			// Nullable column. Row-major spends ~1 tag byte per row (tagNil for
			// absent, a value tag for present) plus the present value bytes;
			// columnar spends a presence bitmap plus a dense column no larger
			// than those value bytes (FOR-packing only shrinks it further), so
			// the mask-vs-byte-per-row difference is the conservative win.
			valBytes := 0
			for i := range sample {
				pp := *(*unsafe.Pointer)(unsafe.Add(base, uintptr(i)*plan.stride+col.offset))
				if pp == nil {
					continue
				}
				switch col.kind.base() {
				case colKindInt:
					valBytes += uvarintLen(zigzagEncode64(loadI64At(pp, col.width)))
				case colKindUint:
					valBytes += uvarintLen(loadU64At(pp, col.width))
				case colKindFloat:
					valBytes += 8
				case colKindBool:
					valBytes++
				}
			}
			rowBytes += sample + valBytes
			colBytes += (sample+7)/8 + valBytes
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

	idxAt := -1
	if e.colIndex {
		idxAt = len(e.buf)
		e.buf = append(e.buf, make([]byte, 4*len(plan.cols))...)
	}
	colStart := len(e.buf)
	for c := range plan.cols {
		col := &plan.cols[c]
		if col.kind.isNullable() {
			if err := e.encodeNullableColumn(base, plan, col, n); err != nil {
				return err
			}
			if e.colIndex {
				end := len(e.buf)
				binary.LittleEndian.PutUint32(e.buf[idxAt+4*c:], uint32(end-colStart))
				colStart = end
			}
			continue
		}
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
			e.writeStringColumn(s)
		}
		if e.colIndex {
			end := len(e.buf)
			binary.LittleEndian.PutUint32(e.buf[idxAt+4*c:], uint32(end-colStart))
			colStart = end
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
	// Every column holds exactly n elements; bound each column codec's
	// claimed length so a constant/zero-width codec cannot allocate past n.
	d.colMaxLen = n
	defer func() { d.colMaxLen = 0 }()
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

	var colLens []uint32
	if d.colIndex {
		k := len(sh.kinds)
		if d.i+4*k > len(d.buf) {
			return ErrShortBuffer
		}
		if cap(d.state.colLenScratch) >= k {
			colLens = d.state.colLenScratch[:k]
		} else {
			colLens = make([]uint32, k)
		}
		d.state.colLenScratch = colLens
		var sum uint64
		for c := 0; c < k; c++ {
			colLens[c] = binary.LittleEndian.Uint32(d.buf[d.i+4*c:])
			sum += uint64(colLens[c])
		}
		d.i += 4 * k
		if sum > uint64(len(d.buf)-d.i) {
			return ErrShortBuffer
		}
	}

	reflectutil.MakeSlice(t, n, p)
	base := reflectutil.SliceData(t, p)

	want := wantedColumns(plan, sh.names)
	for c := range sh.kinds {
		col := want[c]
		if col == nil {
			// unwanted column → skip its body via the index
			if d.colIndex {
				if d.i+int(colLens[c]) > len(d.buf) {
					return ErrShortBuffer
				}
				d.i += int(colLens[c])
				continue
			}
			// no index: decode-and-discard fallback (implemented in a later
			// task). For now return ErrBadTag — the test always uses the index.
			return ErrBadTag
		}
		if sh.kinds[c] != col.kind {
			return ErrTypeMismatch
		}
		if err := d.decodeColumnInto(base, plan, col, n); err != nil {
			return err
		}
	}
	return nil
}

// decodeColumnInto decodes a single column body from the wire into the matched
// target plan column. Behavior is identical to the per-column body of the
// former positional decode loop.
func (d *Decoder) decodeColumnInto(base unsafe.Pointer, plan *columnarPlan, col *colColumn, n int) error {
	if col.kind.isNullable() {
		return d.decodeNullableColumn(base, plan, col, n)
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
	// The map-per-row reflection decode allocates n maps up front, far heavier
	// than the struct path's single backing slice, so it gets a tighter
	// element ceiling. A constant column can claim a huge n from a tiny body;
	// without this a small hostile input would drive a multi-gigabyte map
	// allocation. Callers decoding millions of rows should use a typed struct.
	if n > maxColumnarAnyElems {
		return nil, ErrInvalidLength
	}
	d.colMaxLen = n
	defer func() { d.colMaxLen = 0 }()
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
	var colLens []uint32
	if d.colIndex {
		k := len(sh.kinds)
		if d.i+4*k > len(d.buf) {
			return nil, ErrShortBuffer
		}
		if cap(d.state.colLenScratch) >= k {
			colLens = d.state.colLenScratch[:k]
		} else {
			colLens = make([]uint32, k)
		}
		d.state.colLenScratch = colLens
		var sum uint64
		for c := 0; c < k; c++ {
			colLens[c] = binary.LittleEndian.Uint32(d.buf[d.i+4*c:])
			sum += uint64(colLens[c])
		}
		d.i += 4 * k
		if sum > uint64(len(d.buf)-d.i) {
			return nil, ErrShortBuffer
		}
	}

	out := make([]any, n)
	for i := range out {
		out[i] = make(map[string]any, len(sh.names))
	}
	for c := range sh.kinds {
		name := sh.names[c]
		// When a field filter is active, skip unrequested columns. With the
		// column-length index present we advance past the whole column body
		// without decoding (the perf path); without it we still must decode to
		// stay in sync, but the value is simply not stored below.
		if !d.wantField(name) && colLens != nil {
			if d.i+int(colLens[c]) > len(d.buf) {
				return nil, ErrShortBuffer
			}
			d.i += int(colLens[c])
			continue
		}
		// store is false only on the no-index skip path: the column body still
		// has to be decoded to keep the cursor in sync, but its values are
		// dropped rather than boxed into the row maps.
		store := d.wantField(name)
		if sh.kinds[c].isNullable() {
			vals, err := d.decodeNullableColumnAny(sh.kinds[c], n)
			if err != nil {
				return nil, err
			}
			if store {
				for i := range n {
					out[i].(map[string]any)[name] = vals[i]
				}
			}
			continue
		}
		switch sh.kinds[c] {
		case colKindInt:
			var s []int64
			if err := decodeSliceInt64(d, unsafe.Pointer(&s)); err != nil {
				return nil, err
			}
			if len(s) != n {
				return nil, ErrTypeMismatch
			}
			if store {
				for i := range n {
					out[i].(map[string]any)[name] = s[i]
				}
			}
		case colKindUint:
			var s []uint64
			if err := decodeSliceUint64(d, unsafe.Pointer(&s)); err != nil {
				return nil, err
			}
			if len(s) != n {
				return nil, ErrTypeMismatch
			}
			if store {
				for i := range n {
					out[i].(map[string]any)[name] = s[i]
				}
			}
		case colKindFloat:
			var s []float64
			if err := decodeSliceFloat64(d, unsafe.Pointer(&s)); err != nil {
				return nil, err
			}
			if len(s) != n {
				return nil, ErrTypeMismatch
			}
			if store {
				for i := range n {
					out[i].(map[string]any)[name] = s[i]
				}
			}
		case colKindBool:
			var s []bool
			if err := decodeSliceBool(d, unsafe.Pointer(&s)); err != nil {
				return nil, err
			}
			if len(s) != n {
				return nil, ErrTypeMismatch
			}
			if store {
				for i := range n {
					out[i].(map[string]any)[name] = s[i]
				}
			}
		case colKindString:
			if d.i < len(d.buf) && d.buf[d.i] == tagColStrDict {
				table, idx, err := d.readStringColumnDict(n)
				if err != nil {
					return nil, err
				}
				if store {
					for i := range n {
						out[i].(map[string]any)[name] = table[idx[i]]
					}
				}
				break
			}
			for i := range n {
				sb, err := d.readStringBytes()
				if err != nil {
					return nil, err
				}
				if store {
					out[i].(map[string]any)[name] = string(sb)
				}
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
