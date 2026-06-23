package qdf

import (
	"encoding/binary"
	"math"
	"reflect"
	"runtime"
	"slices"
	"time"
	"unsafe"

	"github.com/alex60217101990/qdf/internal/fsst"
)

// colKind classifies a struct field for columnar encoding. Only these kinds
// are columnar-eligible; any other field kind makes the whole struct fall
// back to row-major.
type colKind uint8

const (
	colKindInt    colKind = iota // int, int8..int64  → []int64 column
	colKindUint                  // uint, uint8..uint64, uintptr → []uint64 column
	colKindFloat                 // float64 → []float64 column (float32 → colKindFloat32)
	colKindBool                  // bool → []bool column
	colKindString                // string, []byte → consecutive WriteString
	colKindTime                  // time.Time → sec []int64 sub-column + nsec []uint64 sub-column
	// colKindFloat32 → []uint32-bits column (math.Float32bits), encoded with the
	// uint codec. Kept SEPARATE from colKindFloat (which is float64-only) because
	// a float32→float64→float32 round trip is not bit-preserving for NaN payloads
	// (signaling NaNs are quieted). Carrying the raw 32 bits is both lossless and
	// narrower on the wire (4 B vs the 8 B a float64 column used). Appended last
	// so the existing kinds keep their wire values.
	colKindFloat32

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

// String returns a human-readable name for the kind, used in error messages.
func (k colKind) String() string {
	n := ""
	switch k.base() {
	case colKindInt:
		n = "int"
	case colKindUint:
		n = "uint"
	case colKindFloat:
		n = "float"
	case colKindFloat32:
		n = "float32"
	case colKindBool:
		n = "bool"
	case colKindString:
		n = "string"
	case colKindTime:
		n = "time"
	default:
		n = "unknown"
	}
	if k.isNullable() {
		n = "*" + n
	}
	return n
}

// colColumn is one field's columnar descriptor: where it lives in each struct
// element and how to (de)serialize the column.
// Fields are ordered to pack without padding AND to keep the GC pointer-scan
// range tight: the interface (elemType, two pointer words) leads, then the
// string, then the non-pointer scalars/flags trail; one colColumn per field is
// built once per columnar struct type and scanned per batch in encodeColumnar.
type colColumn struct {
	// elemType is the pointed-to type for a nullable (*T) column, used to
	// allocate the present values on decode. nil for non-nullable columns.
	elemType reflect.Type
	name     string
	offset   uintptr
	width    uintptr // element width in bytes for the scalar load/store
	kind     colKind // base kind, OR'd with colKindNullable for *T columns
	isByte   bool    // true for []byte string columns (vs string)
}

// residualField describes one struct field that is NOT columnar-eligible
// (map, non-[]byte slice, nested struct, interface, nullable []byte). In a
// hybrid columnar payload these fields are kept row-major — encoded/decoded
// per row via the field's existing typeDesc codecs — instead of disqualifying
// the whole struct from columnar transposition.
type residualField struct {
	desc   *typeDesc // the field's encode/decode closures (row-major path)
	name   string
	offset uintptr
}

// residualKind is a shape-byte sentinel marking a residual field in a hybrid
// columnar shape. It is outside the valid colKind range (base kinds 0x00-0x06,
// nullable-OR'd 0x80-0x86), so it can never be mistaken for a real column kind.
const residualKind colKind = 0xFF

// columnarPlan is cached on the slice element's typeDesc. nil means the
// element type is not columnar-eligible.
//
// Three shapes:
//   - residual == nil          : pure columnar (every field eligible) — tagColStruct
//   - residual != nil          : hybrid (some fields eligible, some residual) — tagHybridColStruct
//   - (buildColumnarPlan nil)  : no eligible field at all — full row-major
type columnarPlan struct {
	cols []colColumn
	// colNames / colKinds mirror cols[].name / cols[].kind as standalone slices,
	// precomputed once at build so encodeColumnar hands them to the shape
	// lookup/declare without rebuilding a transient pair every batch. The plan is
	// immutable after build, so the slice the shape table retains stays valid.
	colNames []string
	colKinds []colKind

	// residual holds the non-columnar fields (in struct declaration order) for a
	// hybrid plan; nil for a pure-columnar plan. hybridNames / hybridKinds are the
	// full field list (ALL fields, declaration order) for the hybrid shape, with
	// residualKind marking the residual entries — built once so the hybrid encode
	// path hands them to the shape lookup/declare without rebuilding per batch.
	residual    []residualField
	hybridNames []string
	hybridKinds []colKind

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

// maxColumnarBytes caps the OUTPUT allocation of a columnar struct decode. The
// element-count ceiling (maxColumnarElems) does NOT bound the n*elemSize output
// slice: a constant / RLE / zero-width column compresses many rows into a few
// wire bytes, so a hostile header can claim maxColumnarElems rows of a wide
// struct and amplify a ~1 KB input into a multi-GB allocation. This byte ceiling
// caps that amplification before the slice is made; callers with larger batches
// shard or stream.
const maxColumnarBytes = 256 << 20

// checkColumnarBytes rejects a columnar output whose total size (n elements of
// elemSize bytes) exceeds maxColumnarBytes — the byte-bounded companion to
// checkColumnarN's element-count ceiling. n must already be >= 0 (checkColumnarN).
func checkColumnarBytes(n int, elemSize uintptr) error {
	if uint64(n)*uint64(elemSize) > maxColumnarBytes {
		return ErrInvalidLength
	}
	return nil
}

// CheckColumnarBytes is the exported guard cmd/qdfgen-generated columnar
// decoders call after ReadColStructHeader / ReadHybridColStructHeader, before
// allocating the row slice, to reject the same memory-amplification a hostile
// row count would cause on the reflect path. elemSize is unsafe.Sizeof(row).
func CheckColumnarBytes(n int, elemSize uintptr) error { return checkColumnarBytes(n, elemSize) }

// buildColumnarPlan classifies a struct's fields into columnar-eligible columns
// and (hybrid) residual fields. Called once at fillDesc time; the result is
// cached so the hot path never reflects.
//
// Returns:
//   - nil                      if td is not a struct, has no fields, or has NO
//     columnar-eligible field (nothing to transpose → full row-major).
//   - plan with residual == nil if EVERY field is eligible (pure columnar).
//   - plan with residual != nil if some fields are eligible and some are not
//     (hybrid: transpose the eligible columns, keep the rest row-major).
//
// A field that classifyColKind rejects (map, non-[]byte slice, nested struct,
// interface) — or a nullable []byte — becomes residual instead of
// disqualifying the whole struct, which is what unlocks columnar for the common
// "mostly-scalar struct with one map/slice field" shape (AD/log/RTB records).
func buildColumnarPlan(td *typeDesc) *columnarPlan {
	if td.kind != reflect.Struct || len(td.fields) == 0 {
		return nil
	}
	cols := make([]colColumn, 0, len(td.fields))
	var residual []residualField
	// hybridNames/hybridKinds record EVERY field in declaration order (with
	// residualKind for the non-columnar ones) so a hybrid shape can reconstruct
	// the struct on decode. Only retained on the plan if it turns out hybrid.
	hybridNames := make([]string, 0, len(td.fields))
	hybridKinds := make([]colKind, 0, len(td.fields))

	addResidual := func(f *fieldDesc) {
		residual = append(residual, residualField{name: f.name, offset: f.offset, desc: f.desc})
		hybridNames = append(hybridNames, f.name)
		hybridKinds = append(hybridKinds, residualKind)
	}

	for i := range td.fields {
		f := &td.fields[i]
		fd := f.desc
		// An optional (*T) field becomes a nullable column: classify the
		// pointed-to type and remember its reflect.Type for decode allocation.
		var elemType reflect.Type
		nullable := false
		if fd.kind == reflect.Pointer {
			if fd.elem == nil {
				addResidual(f) // pointer to an undescribable type — keep row-major
				continue
			}
			nullable = true
			elemType = fd.rType.Elem()
			fd = fd.elem
		}
		ck, w, isByte, ok := classifyColKind(fd)
		// Not columnar-eligible (or a nullable []byte, still unsupported as a
		// column) → residual rather than disqualifying the whole struct.
		if !ok || (nullable && isByte) {
			addResidual(f)
			continue
		}
		if nullable {
			ck |= colKindNullable
		}
		cols = append(cols, colColumn{name: f.name, offset: f.offset, kind: ck, width: w, isByte: isByte, elemType: elemType})
		hybridNames = append(hybridNames, f.name)
		hybridKinds = append(hybridKinds, ck)
	}

	if len(cols) == 0 {
		return nil // no eligible column — nothing to transpose, full row-major
	}
	names := make([]string, len(cols))
	kinds := make([]colKind, len(cols))
	for i := range cols {
		names[i] = cols[i].name
		kinds[i] = cols[i].kind
	}
	plan := &columnarPlan{cols: cols, stride: td.rType.Size(), colNames: names, colKinds: kinds}
	if len(residual) > 0 {
		// Hybrid: keep the full ordered field list for the hybrid shape.
		plan.residual = residual
		plan.hybridNames = hybridNames
		plan.hybridKinds = hybridKinds
	}
	return plan
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

// hybridShapeDeclare / hybridShapeFor / hybridShapeDeclareDec / hybridShapeLookup
// mirror the colShape* helpers but operate on the SEPARATE hybrid shape table
// (hybridShapeNames/hybridShapeKinds on the encoder, hybridShapes on the
// decoder). Keeping a distinct ID space means a stream can interleave
// tagColStruct and tagHybridColStruct payloads without ID aliasing: hybrid
// shape ID 3 and columnar shape ID 3 are independent. The kinds slice carries
// residualKind (0xFF) for residual fields and a real colKind for eligible ones.
func (e *encState) hybridShapeDeclare(names []string, kinds []colKind) uint32 {
	e.hybridShapeNames = append(e.hybridShapeNames, names)
	e.hybridShapeKinds = append(e.hybridShapeKinds, kinds)
	return uint32(len(e.hybridShapeNames)) // ids start at 1
}

func (e *encState) hybridShapeFor(names []string, kinds []colKind) uint32 {
	for i := range e.hybridShapeNames {
		if colShapeEq(e.hybridShapeNames[i], e.hybridShapeKinds[i], names, kinds) {
			return uint32(i + 1)
		}
	}
	return 0
}

func (d *decState) hybridShapeDeclareDec(names []string, kinds []colKind) *decColShape {
	d.hybridShapes = append(d.hybridShapes, decColShape{names: names, kinds: kinds})
	return &d.hybridShapes[len(d.hybridShapes)-1]
}

func (d *decState) hybridShapeLookup(id uint32) *decColShape {
	if id == 0 || id > uint32(len(d.hybridShapes)) {
		return nil
	}
	return &d.hybridShapes[id-1]
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
func columnarProbe(plan *columnarPlan, base unsafe.Pointer, n int, fsstEnabled bool, fsstDict *fsst.SymbolTable) bool {
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
		case colKindFloat32:
			// float32 column carries raw 32-bit patterns via the uint codec:
			// 4 B in the dense column vs 5 B row-major (tag + 4). Without this
			// case a float32-only struct probed to 0 bytes and stayed row-major.
			rowBytes += sample * 5
			colBytes += sample * 4
		case colKindBool:
			rowBytes += sample
			colBytes += (sample + 7) / 8
		case colKindTime:
			// A time.Time column encodes as two sub-columns: sec ([]int64) and
			// nsec ([]uint64). Estimate both as two FOR-packed integer columns.
			// Monotonic timestamps compress extremely well with Delta+FOR on sec;
			// nsec is often 0 or small. Conservative estimate: treat like two
			// int columns over the sample.
			var mnSec, mxSec uint64 = ^uint64(0), 0
			for i := range sample {
				p := unsafe.Add(base, uintptr(i)*plan.stride+col.offset)
				t := (*time.Time)(p).UTC()
				v := uint64(t.Unix() + (1 << 62)) // shift to unsigned for range calc
				if v < mnSec {
					mnSec = v
				}
				if v > mxSec {
					mxSec = v
				}
				rowBytes += 8 + 4 // row-major: tagTimestamp + 8-byte sec + 4-byte nsec
			}
			spreadSec := mxSec - mnSec
			bitsSec := 0
			for spreadSec > 0 {
				bitsSec++
				spreadSec >>= 1
			}
			// sec sub-column (Delta+FOR compresses monotonic runs to near zero):
			colBytes += 3 + (bitsSec*sample+7)/8
			// nsec sub-column (small integers, usually 0): a few bytes overhead
			colBytes += 3 + (30*sample+7)/8 // conservative: up to 30 bits for nsec
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
			best := min(perValue, dictBytes)
			// NOTE: the probe deliberately does NOT model alpha-packing. Modelling
			// it shifts the columnar-vs-row-major boundary and flips borderline
			// structs (pure single-string columns, prefix-shared dict columns) into
			// columnar that row-major / front-coding encode more cheaply. Alpha only
			// needs the struct to reach columnar for OTHER reasons (a FOR-packable
			// numeric or dict-able enum column does this on the trace/log payloads it
			// targets); it then fires on the restricted-alphabet string columns via
			// the never-larger emit picker. A struct whose only compressible signal
			// is a restricted-alphabet ID stays row-major — a missed case traded for
			// zero columnar-decision regression. Intern-aware Balanced hybrid (which
			// would capture it) is the deferred follow-up noted in encodeSlice.
			// FSST competes only when enabled (OptFSST). High-cardinality,
			// substring-sharing columns (URLs, log lines) where dict and
			// per-value both stay near raw are exactly where FSST wins — without
			// this term the probe would route them to row-major and FSST, which
			// only runs inside the columnar string-column picker, would never
			// fire. The symbol table is a one-time cost over the whole column, so
			// it is amortized to the sample window (× sample/n) rather than
			// charged in full against the 32-row probe.
			if fsstEnabled {
				best = min(best, estimateFSSTColumnBytes(base, plan.stride, col, sample, n, fsstDict))
			}
			colBytes += best
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
				case colKindFloat32:
					valBytes += 4
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

	if id := st.colShapeFor(plan.colNames, plan.colKinds); id != 0 {
		e.buf = appendUvarint(e.buf, uint64(id))
	} else {
		e.buf = appendUvarint(e.buf, 0)
		e.buf = appendUvarint(e.buf, uint64(len(plan.cols)))
		for i := range plan.cols {
			e.WriteString(plan.cols[i].name)
			e.buf = append(e.buf, byte(plan.cols[i].kind))
		}
		st.colShapeDeclare(plan.colNames, plan.colKinds)
	}

	idxAt := -1
	if e.colIndex {
		// Backpatch the header flag now that an index is actually emitted, so
		// OptColumnIndex on a non-columnar payload stays a no-op.
		e.buf[e.headerFlagAt] |= FlagColIndex
		idxAt = len(e.buf)
		e.buf = append(e.buf, make([]byte, 4*len(plan.cols))...)
	}
	colStart := len(e.buf)
	for c := range plan.cols {
		col := &plan.cols[c]
		if err := e.encodeOneColumn(plan, base, col, n); err != nil {
			return err
		}
		if e.colIndex {
			end := len(e.buf)
			binary.LittleEndian.PutUint32(e.buf[idxAt+4*c:], uint32(end-colStart))
			colStart = end
		}
	}
	return nil
}

// encodeOneColumn emits a single eligible column's body using the pooled
// transpose scratch. It does NO colIndex bookkeeping — the caller backpatches
// the per-column length. Shared by encodeColumnar (tagColStruct) and
// encodeHybridColumnar (tagHybridColStruct) so both get the identical
// per-column encoding (constant/FOR/dict/FSST/Gorilla/bitpack).
func (e *Encoder) encodeOneColumn(plan *columnarPlan, base unsafe.Pointer, col *colColumn, n int) error {
	st := e.state
	if col.kind.isNullable() {
		return e.encodeNullableColumn(base, plan, col, n)
	}
	switch col.kind {
	case colKindInt:
		s := gatherColI64(st.colScratchI64, base, plan.stride, col, n)
		st.colScratchI64 = s
		return encodeSliceInt64(e, unsafe.Pointer(&s))
	case colKindUint:
		s := gatherColU64(st.colScratchU64, base, plan.stride, col, n)
		st.colScratchU64 = s
		return encodeSliceUint64(e, unsafe.Pointer(&s))
	case colKindFloat:
		s := st.colScratchF64[:0]
		for i := range n {
			s = append(s, loadFloat64Field(base, plan.stride, col, i))
		}
		st.colScratchF64 = s
		return encodeSliceFloat64(e, unsafe.Pointer(&s))
	case colKindFloat32:
		// float32 column: store raw 32-bit patterns via the uint codec (4 B,
		// bit-exact). Reuses the u64 scratch — the high 32 bits are always zero.
		s := st.colScratchU64[:0]
		if e.opts.Has(OptCanonical) {
			for i := range n {
				s = append(s, canonicalizeFloat32Bits(loadFloat32Bits(base, plan.stride, col, i)))
			}
		} else {
			for i := range n {
				s = append(s, loadFloat32Bits(base, plan.stride, col, i))
			}
		}
		st.colScratchU64 = s
		return encodeSliceUint64(e, unsafe.Pointer(&s))
	case colKindBool:
		s := st.colScratchBool[:0]
		for i := range n {
			p := unsafe.Add(base, uintptr(i)*plan.stride+col.offset)
			s = append(s, *(*bool)(p))
		}
		st.colScratchBool = s
		return encodeSliceBool(e, unsafe.Pointer(&s))
	case colKindString:
		if col.isByte {
			for i := range n {
				p := unsafe.Add(base, uintptr(i)*plan.stride+col.offset)
				e.WriteBytes(*(*[]byte)(p))
			}
			return nil
		}
		s := st.colScratchStr[:0]
		for i := range n {
			s = append(s, loadStringField(base, plan.stride, col, i))
		}
		st.colScratchStr = s
		e.writeStringColumn(s)
		return nil
	case colKindTime:
		// Two sub-columns: sec ([]int64) + nsec ([]uint64). Delta+FOR on sec
		// compresses monotonic timestamp series efficiently.
		sec := st.colScratchI64[:0]
		nsec := st.colScratchU64[:0]
		for i := range n {
			p := unsafe.Add(base, uintptr(i)*plan.stride+col.offset)
			t := (*time.Time)(p).UTC()
			sec = append(sec, t.Unix())
			nsec = append(nsec, uint64(t.Nanosecond()))
		}
		st.colScratchI64 = sec
		st.colScratchU64 = nsec
		if err := encodeSliceInt64(e, unsafe.Pointer(&sec)); err != nil {
			return err
		}
		return encodeSliceUint64(e, unsafe.Pointer(&nsec))
	}
	return nil
}

// encodeHybridColumnar emits a slice of mixed structs as tagHybridColStruct:
// the eligible columns transposed (identical per-column encoding to
// tagColStruct, via encodeOneColumn) followed by a per-row residual block where
// the non-columnar fields are encoded row-major using their own codecs. The
// never-larger decision is made by the caller's columnarProbe (it measures the
// eligible columns; the residual block is byte-identical to what row-major
// would emit, so it does not affect the columnar-vs-row-major comparison).
// No FlagColIndex is emitted for hybrid in v1.
func (e *Encoder) encodeHybridColumnar(plan *columnarPlan, base unsafe.Pointer, n int) error {
	st := e.state
	e.buf = append(e.buf, tagHybridColStruct)
	e.buf = appendUvarint(e.buf, uint64(n))

	// Shape: ALL fields in declaration order; residualKind marks residual ones.
	if id := st.hybridShapeFor(plan.hybridNames, plan.hybridKinds); id != 0 {
		e.buf = appendUvarint(e.buf, uint64(id))
	} else {
		e.buf = appendUvarint(e.buf, 0)
		e.buf = appendUvarint(e.buf, uint64(len(plan.hybridNames)))
		for i := range plan.hybridNames {
			e.WriteString(plan.hybridNames[i])
			e.buf = append(e.buf, byte(plan.hybridKinds[i]))
		}
		st.hybridShapeDeclare(plan.hybridNames, plan.hybridKinds)
	}

	// Eligible columns, transposed.
	for c := range plan.cols {
		if err := e.encodeOneColumn(plan, base, &plan.cols[c], n); err != nil {
			return err
		}
	}

	// Residual block: for each row, its non-columnar fields in declaration
	// order, via the field's own (row-major) encoder.
	for i := range n {
		rowPtr := unsafe.Add(base, uintptr(i)*plan.stride)
		for r := range plan.residual {
			rf := &plan.residual[r]
			if err := rf.desc.encode(e, unsafe.Add(rowPtr, rf.offset)); err != nil {
				return err
			}
		}
	}
	return nil
}

//go:nosplit
func loadFloat64Field(base unsafe.Pointer, stride uintptr, col *colColumn, i int) float64 {
	// colKindFloat is float64-only (width 8); float32 uses colKindFloat32 +
	// loadFloat32Bits, so there is no width==4 case here.
	p := unsafe.Add(base, uintptr(i)*stride+col.offset)
	return *(*float64)(p)
}

// loadFloat32Bits reads a float32 field's raw IEEE-754 bits (zero-extended to
// uint64) for a colKindFloat32 column. Bit-exact: never goes through a numeric
// float64 conversion, so NaN payloads/signaling bits survive.
//
//go:nosplit
func loadFloat32Bits(base unsafe.Pointer, stride uintptr, col *colColumn, i int) uint64 {
	p := unsafe.Add(base, uintptr(i)*stride+col.offset)
	return uint64(*(*uint32)(p))
}

// storeFloat32Bits writes raw float32 bits (low 32 of v) into a float32 field
// for a colKindFloat32 column. Inverse of loadFloat32Bits.
//
//go:nosplit
func storeFloat32Bits(base unsafe.Pointer, stride uintptr, col *colColumn, i int, v uint64) {
	p := unsafe.Add(base, uintptr(i)*stride+col.offset)
	*(*uint32)(p) = uint32(v)
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

// loadStringFieldBytes returns a zero-copy []byte view of the i-th string field
// (no allocation). The view aliases the caller's live struct, which is stable
// for the duration of the encode; it is read-only (passed to FSST train/probe).
func loadStringFieldBytes(base unsafe.Pointer, stride uintptr, col *colColumn, i int) []byte {
	p := unsafe.Add(base, uintptr(i)*stride+col.offset)
	if col.isByte {
		return *(*[]byte)(p)
	}
	s := *(*string)(p)
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// estimateFSSTColumnBytes estimates the FSST-coded byte cost of a string column
// over the probe sample, amortizing the one-time symbol table across the whole
// column (× sample/n). Zero string copies; reuses one compress scratch buffer.
// Called only when FSST is enabled (OptFSST), so its training cost stays off the
// Speed/Balanced hot path.
func estimateFSSTColumnBytes(base unsafe.Pointer, stride uintptr, col *colColumn, sample, n int, dict *fsst.SymbolTable) int {
	var strs [columnarProbeSample][]byte
	for i := range sample {
		strs[i] = loadStringFieldBytes(base, stride, col, i)
	}
	tbl := dict
	var bld *fsst.Builder
	if tbl == nil {
		bld = fsstBuilderPool.Get().(*fsst.Builder)
		tbl = bld.BuildRounds(strs[:sample], fsstProbeRounds) // coarse estimate
	}
	body := 0
	for i := range sample {
		clen := tbl.CompressedLen(strs[i]) // length only; no throwaway buffer
		body += clen + uvarintLen(uint64(clen))
	}
	sz := body + tbl.SerializedSize()*sample/n
	if bld != nil {
		fsstBuilderPool.Put(bld)
	}
	return sz
}

// colShapeRead is the parsed columnar header: the shape (names+kinds), the row
// count n, and — when the column-length index is present (d.colIndex) — the
// per-column byte lengths. It is the common prefix of every columnar decode
// path (typed struct, dynamic map, and predicate query).
type colShapeRead struct {
	sh      *decColShape
	colLens []uint32 // nil when d.colIndex is false
	n       int
}

// readColShape consumes the tagColStruct tag, the row count, the shape
// (declared inline or by id), and the optional column-length index, validating
// bounds exactly as decodeColumnar did. On return d.i points at the first
// column body. maxN bounds the row count (pass 0 for the struct path, which
// uses only checkColumnarN; pass maxColumnarAnyElems for the map path).
func (d *Decoder) readColShape(maxN int) (colShapeRead, error) {
	var out colShapeRead
	d.i++ // consume tagColStruct
	n64, k := readUvarint(d.buf[d.i:])
	if k <= 0 {
		return out, ErrInvalidLength
	}
	d.i += k
	n := int(n64)
	if err := checkColumnarN(n); err != nil {
		return out, err
	}
	if maxN > 0 && n > maxN {
		return out, ErrInvalidLength
	}
	out.n = n
	if d.state == nil {
		d.state = newDecState()
	}
	idv, k2 := readUvarint(d.buf[d.i:])
	if k2 <= 0 {
		return out, ErrInvalidLength
	}
	if idv > uint64(^uint32(0)) {
		return out, ErrUnknownStateID // would truncate on the uint32 cast below
	}
	d.i += k2
	if idv == 0 {
		cnt64, k3 := readUvarint(d.buf[d.i:])
		if k3 <= 0 {
			return out, ErrInvalidLength
		}
		d.i += k3
		cnt := int(cnt64)
		if err := d.CheckLength(cnt, 1); err != nil {
			return out, err
		}
		names := make([]string, cnt)
		kinds := make([]colKind, cnt)
		for i := range cnt {
			s, err := d.readStringBytes()
			if err != nil {
				return out, err
			}
			names[i] = string(s)
			if d.i >= len(d.buf) {
				return out, ErrShortBuffer
			}
			kinds[i] = colKind(d.buf[d.i])
			d.i++
		}
		out.sh = d.state.colShapeDeclareDec(names, kinds)
	} else {
		out.sh = d.state.colShapeLookup(uint32(idv))
		if out.sh == nil {
			return out, ErrUnknownStateID
		}
	}
	if d.colIndex {
		kk := len(out.sh.kinds)
		if d.i+4*kk > len(d.buf) {
			return out, ErrShortBuffer
		}
		var colLens []uint32
		if cap(d.state.colLenScratch) >= kk {
			colLens = d.state.colLenScratch[:kk]
		} else {
			colLens = make([]uint32, kk)
		}
		d.state.colLenScratch = colLens
		var sum uint64
		for c := range kk {
			colLens[c] = binary.LittleEndian.Uint32(d.buf[d.i+4*c:])
			sum += uint64(colLens[c])
		}
		d.i += 4 * kk
		if sum > uint64(len(d.buf)-d.i) {
			return out, ErrShortBuffer
		}
		out.colLens = colLens
	}
	return out, nil
}

// colVals holds one column decoded into its native scratch, retained until the
// row mask is known so matched rows can be compacted into the output. Exactly
// one typed slice is populated, matching kind.base(). For a nullable column,
// present marks which rows carry a value (dense expanded to length n); a row
// whose present bit is 0 is a nil/zero row.
type colVals struct {
	i64     []int64
	u64     []uint64
	f64     []float64
	b       []bool
	s       []string
	bs      [][]byte
	ts      []time.Time // colKindTime only
	present []uint64    // nullable only; nil otherwise
	kind    colKind     // 1-byte tail: kept last so it adds no padding before the slices
}

// decodeColumnVals decodes a column body (length n) of the given kind into a
// fresh colVals, retaining the decoded slice instead of scattering. isByte
// selects the []byte representation for a string-kind column.
func (d *Decoder) decodeColumnVals(kind colKind, n int, isByte bool) (colVals, error) {
	var cv colVals
	cv.kind = kind
	if kind.isNullable() {
		return d.decodeNullableColumnVals(kind, n)
	}
	switch kind {
	case colKindInt:
		var s []int64
		if err := decodeSliceInt64(d, unsafe.Pointer(&s)); err != nil {
			return cv, err
		}
		if len(s) != n {
			return cv, ErrTypeMismatch
		}
		cv.i64 = s
	case colKindUint:
		var s []uint64
		if err := decodeSliceUint64(d, unsafe.Pointer(&s)); err != nil {
			return cv, err
		}
		if len(s) != n {
			return cv, ErrTypeMismatch
		}
		cv.u64 = s
	case colKindFloat:
		var s []float64
		if err := decodeSliceFloat64(d, unsafe.Pointer(&s)); err != nil {
			return cv, err
		}
		if len(s) != n {
			return cv, ErrTypeMismatch
		}
		cv.f64 = s
	case colKindFloat32:
		// float32 bits live in u64 (high 32 zero); cv.kind tags the f32 meaning.
		var s []uint64
		if err := decodeSliceUint64(d, unsafe.Pointer(&s)); err != nil {
			return cv, err
		}
		if len(s) != n {
			return cv, ErrTypeMismatch
		}
		cv.u64 = s
	case colKindBool:
		var s []bool
		if err := decodeSliceBool(d, unsafe.Pointer(&s)); err != nil {
			return cv, err
		}
		if len(s) != n {
			return cv, ErrTypeMismatch
		}
		cv.b = s
	case colKindString:
		if d.i < len(d.buf) && isStringColumnBlockTag(d.buf[d.i]) {
			s, err := d.readStringColumn(n)
			if err != nil {
				return cv, err
			}
			if isByte {
				// A string column decodes into a []byte target field too (the
				// row-major path supports this interchange). The block readers
				// return strings, so copy each into an owned []byte.
				bs := make([][]byte, n)
				for i := range n {
					bs[i] = append([]byte(nil), s[i]...)
				}
				cv.bs = bs
			} else {
				cv.s = s
			}
			break
		}
		if isByte {
			bs := make([][]byte, n)
			for i := range n {
				sb, err := d.readStringBytes()
				if err != nil {
					return cv, err
				}
				bs[i] = append([]byte(nil), sb...)
			}
			cv.bs = bs
			break
		}
		s := make([]string, n)
		for i := range n {
			// ReadString shares repeated values via the decode-side intern
			// cache (and aliases under noCopy), so a low-cardinality column that
			// fell below the dict gate decodes to ~distinct allocs, not n.
			str, err := d.ReadString()
			if err != nil {
				return cv, err
			}
			s[i] = str
		}
		cv.s = s
	case colKindTime:
		var sec []int64
		if err := decodeSliceInt64(d, unsafe.Pointer(&sec)); err != nil {
			return cv, err
		}
		if len(sec) != n {
			return cv, ErrTypeMismatch
		}
		var nsec []uint64
		if err := decodeSliceUint64(d, unsafe.Pointer(&nsec)); err != nil {
			return cv, err
		}
		if len(nsec) != n {
			return cv, ErrTypeMismatch
		}
		if err := checkNsecColumn(nsec); err != nil {
			return cv, err
		}
		ts := make([]time.Time, n)
		for i := range n {
			ts[i] = time.Unix(sec[i], int64(nsec[i])).UTC()
		}
		cv.ts = ts
	default:
		return cv, ErrBadTag
	}
	return cv, nil
}

// eval runs term against cv, setting mask bits for matching rows. A nullable
// row whose present bit is 0 never matches. The common non-nullable column
// takes a tight loop with no per-row presence test (split out so the predictable
// presence branch is hoisted out of the hot path entirely).
func (cv *colVals) eval(term *predTerm, n int, mask []uint64) {
	if cv.present == nil {
		cv.evalDense(term, n, mask)
		return
	}
	cv.evalNullable(term, n, mask)
}

// evalDense is eval for a non-nullable column: no presence test per row.
func (cv *colVals) evalDense(term *predTerm, n int, mask []uint64) {
	switch term.want {
	case colKindInt:
		for i := range n {
			if term.pI64(cv.i64[i]) {
				setBit(mask, i)
			}
		}
	case colKindUint:
		for i := range n {
			if term.pU64(cv.u64[i]) {
				setBit(mask, i)
			}
		}
	case colKindFloat:
		for i := range n {
			if term.pF64(cv.f64[i]) {
				setBit(mask, i)
			}
		}
	case colKindFloat32:
		for i := range n {
			if term.pF64(float64(math.Float32frombits(uint32(cv.u64[i])))) {
				setBit(mask, i)
			}
		}
	case colKindBool:
		for i := range n {
			if term.pBool(cv.b[i]) {
				setBit(mask, i)
			}
		}
	case colKindString:
		for i := range n {
			if term.pStr(cv.strAt(i)) {
				setBit(mask, i)
			}
		}
	}
}

// strAt returns row i's string for predicate evaluation, sourcing it from the
// []byte materialization (cv.bs) when the column was projected into a []byte
// target field (cv.s left nil). The string is transient — handed to the
// predicate only — so aliasing the owned cv.bs[i] copy is safe and alloc-free.
func (cv *colVals) strAt(i int) string {
	if cv.s != nil {
		return cv.s[i]
	}
	b := cv.bs[i]
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// evalNullable is eval for a nullable column: an absent row never matches.
func (cv *colVals) evalNullable(term *predTerm, n int, mask []uint64) {
	switch term.want {
	case colKindInt:
		for i := range n {
			if getBit(cv.present, i) && term.pI64(cv.i64[i]) {
				setBit(mask, i)
			}
		}
	case colKindUint:
		for i := range n {
			if getBit(cv.present, i) && term.pU64(cv.u64[i]) {
				setBit(mask, i)
			}
		}
	case colKindFloat:
		for i := range n {
			if getBit(cv.present, i) && term.pF64(cv.f64[i]) {
				setBit(mask, i)
			}
		}
	case colKindFloat32:
		for i := range n {
			if getBit(cv.present, i) && term.pF64(float64(math.Float32frombits(uint32(cv.u64[i])))) {
				setBit(mask, i)
			}
		}
	case colKindBool:
		for i := range n {
			if getBit(cv.present, i) && term.pBool(cv.b[i]) {
				setBit(mask, i)
			}
		}
	case colKindString:
		for i := range n {
			if getBit(cv.present, i) && term.pStr(cv.strAt(i)) {
				setBit(mask, i)
			}
		}
	}
}

// evalMasks evaluates term against cv into a fresh T mask (rows that are TRUE).
// For a nullable column it also returns an explicit F mask (rows that are
// FALSE); nil F means the column has no nil rows, so F is the complement of T.
// Built on eval, which already gates on the presence bitmap, so nil rows are
// set in neither T nor F (SQL UNKNOWN).
func (cv *colVals) evalMasks(term *predTerm, n int) (t, f []uint64) {
	t = newBitset(n)
	cv.eval(term, n, t)
	if cv.present == nil {
		return t, nil
	}
	f = slices.Clone(cv.present)
	bitsetAndNot(f, t) // f = present &^ t  (present rows that failed the predicate)
	return t, f
}

// scatterRow writes a non-nullable column's value at source row src into the
// output struct slice at compacted row dst. Mirrors decodeColumnInto's store
// half. Nullable columns are scattered via scatterNullableRowInto (one shared
// backing slab instead of a per-row alloc).
func (cv *colVals) scatterRow(base unsafe.Pointer, plan *columnarPlan, col *colColumn, src, dst int) {
	switch cv.kind {
	case colKindInt:
		storeScalarFromI64(base, plan.stride, col, dst, cv.i64[src])
	case colKindUint:
		storeScalarFromU64(base, plan.stride, col, dst, cv.u64[src])
	case colKindFloat:
		storeFloat64(base, plan.stride, col, dst, cv.f64[src])
	case colKindFloat32:
		storeFloat32Bits(base, plan.stride, col, dst, cv.u64[src])
	case colKindBool:
		*(*bool)(unsafe.Add(base, uintptr(dst)*plan.stride+col.offset)) = cv.b[src]
	case colKindString:
		dp := unsafe.Add(base, uintptr(dst)*plan.stride+col.offset)
		if col.isByte {
			*(*[]byte)(dp) = append([]byte(nil), cv.bs[src]...)
		} else {
			*(*string)(dp) = cv.s[src]
		}
	case colKindTime:
		dp := unsafe.Add(base, uintptr(dst)*plan.stride+col.offset)
		*(*time.Time)(dp) = cv.ts[src]
	}
}

// anyAt returns cv's value at row i as an any, mirroring decodeColumnarAny's
// per-kind boxing (int64/uint64/float64/bool, string for string-kind). A
// nullable nil row returns a nil any.
func (cv *colVals) anyAt(i int) any {
	if cv.present != nil && !getBit(cv.present, i) {
		return nil
	}
	switch cv.kind.base() {
	case colKindInt:
		return cv.i64[i]
	case colKindUint:
		return cv.u64[i]
	case colKindFloat:
		return cv.f64[i]
	case colKindFloat32:
		return math.Float32frombits(uint32(cv.u64[i]))
	case colKindBool:
		return cv.b[i]
	case colKindString:
		if cv.bs != nil {
			return string(cv.bs[i])
		}
		return cv.s[i]
	case colKindTime:
		return cv.ts[i]
	}
	return nil
}

// runQueryColumns resolves d.query's predicate tree against the shape, then
// makes a single forward pass over the wire columns: predicate and projected
// columns are decoded into retained colVals (predicates evaluated via the tree),
// and the rest are skipped (via the column-length index when present, else
// decode-and-discard). It returns the retained column values (indexed by wire
// column, nil where not retained) and the surviving row indices.
//
// isProj reports whether wire column c should be retained for the caller to
// materialise. isByte is forwarded to decodeColumnVals for column c (the typed
// path passes the target field's isByte; the map path passes false).
func (d *Decoder) runQueryColumns(
	sh *decColShape, colLens []uint32, n int,
	isProj func(c int) bool, isByte func(c int) bool,
) (retained []*colVals, matched []int, err error) {
	// Flatten the predicate tree into a local, int-indexed slice (no per-node
	// maps; never mutates the shared QueryOption tree).
	flat := flattenCond(d.query.root)

	// Resolve each leaf to a wire column and validate its kind.
	referenced := make([]bool, len(sh.kinds))
	for i := range flat {
		if flat[i].op != condLeaf {
			continue
		}
		field := flat[i].term.field
		wi := -1
		for c, name := range sh.names {
			if name == field {
				wi = c
				break
			}
		}
		if wi < 0 {
			return nil, nil, &QueryError{Op: "predicate pushdown", Field: field, Err: ErrFieldNotFound}
		}
		if sh.kinds[wi].base() != flat[i].term.want {
			return nil, nil, &QueryError{Op: "predicate pushdown", Field: field, Want: flat[i].term.want, Got: sh.kinds[wi], Err: ErrTypeMismatch}
		}
		flat[i].col = wi
		referenced[wi] = true
	}

	// Single forward pass: decode projected or referenced columns once, skip rest.
	retained = make([]*colVals, len(sh.kinds))
	colCV := make([]*colVals, len(sh.kinds))
	for c := range sh.kinds {
		proj := isProj(c)
		if !proj && !referenced[c] {
			if colLens != nil {
				if d.i+int(colLens[c]) > len(d.buf) {
					return nil, nil, ErrShortBuffer
				}
				d.i += int(colLens[c])
			} else if e := d.skipColumnValue(sh.kinds[c], n); e != nil {
				return nil, nil, e
			}
			continue
		}
		cv, e := d.decodeColumnVals(sh.kinds[c], n, isByte(c))
		if e != nil {
			return nil, nil, e
		}
		if proj {
			retained[c] = &cv
		}
		if referenced[c] {
			colCV[c] = &cv
		}
	}

	// Bind each leaf to its decoded column values.
	for i := range flat {
		if flat[i].op == condLeaf {
			flat[i].cv = colCV[flat[i].col]
		}
	}

	var combined []uint64
	if len(flat) == 0 {
		combined = fullBitset(n) // no filter: every row matches
	} else {
		markUnknown(flat)
		combined, _ = evalCond(flat, 0, n)
	}
	matched = matchedIndices(combined, n, nil)
	return retained, matched, nil
}

// decodeColumnarQuery decodes a columnar struct slice applying d.query: it runs
// the AND of the plan's predicates to select rows, then materialises only the
// matched rows of the projected columns into out (*[]Struct). Filter columns
// need not be projected. Wire order is preserved.
func decodeColumnarQuery(d *Decoder, t reflect.Type, plan *columnarPlan, p unsafe.Pointer) error {
	cs, err := d.readColShape(0)
	if err != nil {
		return err
	}
	n, sh, colLens := cs.n, cs.sh, cs.colLens
	// Bound by bytes before runQueryColumns materialises n-element column
	// scratch (memory amplification from a compressed column count).
	if err := checkColumnarBytes(n, t.Elem().Size()); err != nil {
		return err
	}
	d.colMaxLen = n
	defer func() { d.colMaxLen = 0 }()

	// Projected output columns: wire index -> target plan column (or nil).
	want := wantedColumns(plan, sh.names)
	if d.query.selectFields != nil {
		for c, name := range sh.names {
			if !slices.Contains(d.query.selectFields, name) {
				want[c] = nil
			}
		}
	}

	retained, matched, err := d.runQueryColumns(sh, colLens, n,
		func(c int) bool { return want[c] != nil },
		func(c int) bool { return want[c] != nil && want[c].isByte },
	)
	if err != nil {
		return err
	}

	// Allocate the compacted output slice and scatter matched rows — reuse the
	// caller backing when pointer-free + cap suffices, else fresh.
	base := reuseOrMakeSlice(t, len(matched), p, t.Elem().Size(), noPointers(t.Elem()))
	for c := range sh.kinds {
		cv := retained[c]
		if cv == nil || want[c] == nil {
			continue
		}
		col := want[c]
		// Full-kind compare (incl. the nullable flag), matching the full-decode
		// path. The scatter below branches on cv.present (the WIRE nullability),
		// so a wire/plan nullability mismatch must be rejected here: otherwise a
		// nullable wire column scattered into a non-nullable plan column hits
		// reflect.SliceOf(nil) (col.elemType==nil) → panic, and the reverse
		// scatters a raw value into a *T field slot → corruption.
		if sh.kinds[c] != col.kind {
			return ErrTypeMismatch
		}
		if cv.present != nil {
			// Nullable column: one backing slab holds all present values; each
			// *T field points into it. Replaces a per-row reflect.New.
			slab := reflect.MakeSlice(reflect.SliceOf(col.elemType), len(matched), len(matched))
			slabBase := slab.UnsafePointer()
			elemSize := col.elemType.Size()
			for dst, src := range matched {
				cv.scatterNullableRowInto(base, plan, col, src, dst, slabBase, elemSize)
			}
			runtime.KeepAlive(slab)
			continue
		}
		for dst, src := range matched {
			cv.scatterRow(base, plan, col, src, dst)
		}
	}
	return nil
}

// decodeColumnarQueryAny is the *[]map[string]any (or *[]any) form of predicate
// pushdown. It runs the AND of the plan's predicates to select rows, then
// returns one map[string]any per matched row containing only the projected
// columns (Select fields, or all columns when no Select was given). Filter
// columns need not be projected. The result is a []any of map[string]any so the
// dynamic slice routing can box it like decodeColumnarAny's. Boxing into any is
// intrinsic to the map form; predicate evaluation stays unboxed.
func decodeColumnarQueryAny(d *Decoder) (any, error) {
	cs, err := d.readColShape(maxColumnarAnyElems)
	if err != nil {
		return nil, err
	}
	n, sh, colLens := cs.n, cs.sh, cs.colLens
	d.colMaxLen = n
	defer func() { d.colMaxLen = 0 }()

	// Projection: selectFields, or all columns when none given.
	projected := make([]bool, len(sh.kinds))
	for c, name := range sh.names {
		projected[c] = d.query.selectFields == nil || slices.Contains(d.query.selectFields, name)
	}

	retained, matched, err := d.runQueryColumns(sh, colLens, n,
		func(c int) bool { return projected[c] },
		func(c int) bool { return false },
	)
	if err != nil {
		return nil, err
	}

	out := make([]any, len(matched))
	for dst, src := range matched {
		row := make(map[string]any, len(sh.kinds))
		for c := range sh.kinds {
			cv := retained[c]
			if cv == nil {
				continue
			}
			row[sh.names[c]] = cv.anyAt(src)
		}
		out[dst] = row
	}
	return out, nil
}

func decodeColumnar(d *Decoder, t reflect.Type, plan *columnarPlan, p unsafe.Pointer) error {
	cs, err := d.readColShape(0)
	if err != nil {
		return err
	}
	n := cs.n
	sh := cs.sh
	colLens := cs.colLens
	// Every column holds exactly n elements; bound each column codec's
	// claimed length so a constant/zero-width codec cannot allocate past n.
	d.colMaxLen = n
	defer func() { d.colMaxLen = 0 }()

	// Bound the output by bytes, not just element count: a compressed column
	// can claim maxColumnarElems rows from a tiny input (memory amplification).
	if err := checkColumnarBytes(n, t.Elem().Size()); err != nil {
		return err
	}

	// Reuse the caller's backing when the row struct is pointer-free and the
	// pre-sized slice has cap >= n (decode into a pooled slice), else fresh.
	base := reuseOrMakeSlice(t, n, p, t.Elem().Size(), noPointers(t.Elem()))

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
			// No index: there is no column-length to seek past, so we must fully
			// decode the column body to keep the cursor in sync — its values are
			// simply discarded rather than scattered into the struct slice.
			if err := d.skipColumnValue(sh.kinds[c], n); err != nil {
				return err
			}
			continue
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

// readHybridColShape consumes tagHybridColStruct + N + the hybrid shape
// (declare-inline or reuse-by-ID against the SEPARATE hybrid shape table). No
// colIndex (hybrid v1 does not emit one). Mirrors readColShape otherwise.
func (d *Decoder) readHybridColShape(maxN int) (int, *decColShape, error) {
	d.i++ // consume tagHybridColStruct
	n64, k := readUvarint(d.buf[d.i:])
	if k <= 0 {
		return 0, nil, ErrInvalidLength
	}
	d.i += k
	n := int(n64)
	if err := checkColumnarN(n); err != nil {
		return 0, nil, err
	}
	if maxN > 0 && n > maxN {
		return 0, nil, ErrInvalidLength
	}
	if d.state == nil {
		d.state = newDecState()
	}
	idv, k2 := readUvarint(d.buf[d.i:])
	if k2 <= 0 {
		return 0, nil, ErrInvalidLength
	}
	if idv > uint64(^uint32(0)) {
		return 0, nil, ErrUnknownStateID
	}
	d.i += k2
	if idv == 0 {
		cnt64, k3 := readUvarint(d.buf[d.i:])
		if k3 <= 0 {
			return 0, nil, ErrInvalidLength
		}
		d.i += k3
		cnt := int(cnt64)
		if err := d.CheckLength(cnt, 1); err != nil {
			return 0, nil, err
		}
		names := make([]string, cnt)
		kinds := make([]colKind, cnt)
		for i := range cnt {
			s, err := d.readStringBytes()
			if err != nil {
				return 0, nil, err
			}
			names[i] = string(s)
			if d.i >= len(d.buf) {
				return 0, nil, ErrShortBuffer
			}
			kinds[i] = colKind(d.buf[d.i])
			d.i++
		}
		return n, d.state.hybridShapeDeclareDec(names, kinds), nil
	}
	sh := d.state.hybridShapeLookup(uint32(idv))
	if sh == nil {
		return 0, nil, ErrUnknownStateID
	}
	return n, sh, nil
}

func findCol(plan *columnarPlan, name string) *colColumn {
	for c := range plan.cols {
		if plan.cols[c].name == name {
			return &plan.cols[c]
		}
	}
	return nil
}

func findResidual(plan *columnarPlan, name string) *residualField {
	for r := range plan.residual {
		if plan.residual[r].name == name {
			return &plan.residual[r]
		}
	}
	return nil
}

// decodeHybridColumnar decodes a tagHybridColStruct payload: the eligible
// columns (transposed) scatter into the result structs exactly as
// decodeColumnar does, then the per-row residual block decodes each
// non-columnar field row-major via its own codec. Schema evolution is handled
// like decodeColumnar: an eligible wire column the target struct lacks is
// skipped; a residual wire field with no target is consumed via d.Skip.
func decodeHybridColumnar(d *Decoder, t reflect.Type, plan *columnarPlan, p unsafe.Pointer) error {
	n, sh, err := d.readHybridColShape(0)
	if err != nil {
		return err
	}
	// Bound every eligible column codec's claimed length to n.
	d.colMaxLen = n
	defer func() { d.colMaxLen = 0 }()

	// Bound the output by bytes, not just element count (memory amplification).
	if err := checkColumnarBytes(n, t.Elem().Size()); err != nil {
		return err
	}

	base := reuseOrMakeSlice(t, n, p, t.Elem().Size(), noPointers(t.Elem()))

	// Eligible columns: shape entries with a real colKind, in wire order
	// (which equals struct declaration order on the encode side). Matched by
	// name to the target plan column.
	for c := range sh.kinds {
		if sh.kinds[c] == residualKind {
			continue
		}
		col := findCol(plan, sh.names[c])
		if col == nil {
			// Wire has an eligible column the target struct lacks → skip its body.
			if err := d.skipColumnValue(sh.kinds[c], n); err != nil {
				return err
			}
			continue
		}
		if sh.kinds[c] != col.kind {
			return ErrTypeMismatch
		}
		if err := d.decodeColumnInto(base, plan, col, n); err != nil {
			return err
		}
	}

	// Residual block. Precompute the wire-residual → target mapping once
	// (nil target = a residual field the struct lacks, consumed via Skip).
	var targets []*residualField
	for c := range sh.kinds {
		if sh.kinds[c] == residualKind {
			targets = append(targets, findResidual(plan, sh.names[c]))
		}
	}
	// Residual fields are row-major, not columnar columns: their slice/map/
	// nested values may legitimately hold any number of elements, unrelated to
	// the row count n. Clear the per-column length bound (set to n above for the
	// eligible columns) before decoding them, or a residual collection longer
	// than n is wrongly rejected by colLenOK as ErrInvalidLength. (A nested
	// columnar residual sets and restores its own colMaxLen.)
	d.colMaxLen = 0
	for i := range n {
		rowPtr := unsafe.Add(base, uintptr(i)*plan.stride)
		for _, rf := range targets {
			if rf == nil {
				if err := d.Skip(); err != nil {
					return err
				}
				continue
			}
			if err := rf.desc.decode(d, unsafe.Add(rowPtr, rf.offset)); err != nil {
				return err
			}
		}
	}
	return nil
}

// decodeColumnInto decodes a single column body from the wire into the matched
// target plan column. Behavior is identical to the per-column body of the
// former positional decode loop.
//
// The int64/uint64/float64/bool cases reuse decoder-held scratch slices
// (d.state.colScratchI64 etc.) so that the transient column buffer is not
// allocated fresh on every call — it is grown once and reused across columns
// and across decode calls (the Decoder is pooled). The scratch slice is only
// valid until the scatter loop below, which copies each element into the
// output struct; it is never aliased by the output.
func (d *Decoder) decodeColumnInto(base unsafe.Pointer, plan *columnarPlan, col *colColumn, n int) error {
	if col.kind.isNullable() {
		return d.decodeNullableColumn(base, plan, col, n)
	}
	st := d.state // always non-nil: readColShape initialises it before the loop
	switch col.kind {
	case colKindInt:
		if err := decodeSliceInt64Into(d, &st.colScratchI64); err != nil {
			return err
		}
		s := st.colScratchI64
		if len(s) != n {
			return ErrTypeMismatch
		}
		scatterColI64(base, plan.stride, col, s)
	case colKindUint:
		if err := decodeSliceUint64Into(d, &st.colScratchU64); err != nil {
			return err
		}
		s := st.colScratchU64
		if len(s) != n {
			return ErrTypeMismatch
		}
		scatterColU64(base, plan.stride, col, s)
	case colKindFloat:
		if err := decodeSliceFloat64Into(d, &st.colScratchF64); err != nil {
			return err
		}
		s := st.colScratchF64
		if len(s) != n {
			return ErrTypeMismatch
		}
		for i := range n {
			storeFloat64(base, plan.stride, col, i, s[i])
		}
	case colKindFloat32:
		if err := decodeSliceUint64Into(d, &st.colScratchU64); err != nil {
			return err
		}
		s := st.colScratchU64
		if len(s) != n {
			return ErrTypeMismatch
		}
		for i := range n {
			storeFloat32Bits(base, plan.stride, col, i, s[i])
		}
	case colKindBool:
		if err := decodeSliceBoolInto(d, &st.colScratchBool); err != nil {
			return err
		}
		s := st.colScratchBool
		if len(s) != n {
			return ErrTypeMismatch
		}
		for i := range n {
			*(*bool)(unsafe.Add(base, uintptr(i)*plan.stride+col.offset)) = s[i]
		}
	case colKindString:
		if d.i < len(d.buf) && isStringColumnBlockTag(d.buf[d.i]) {
			strs, err := d.readStringColumn(n)
			if err != nil {
				return err
			}
			for i := range n {
				dp := unsafe.Add(base, uintptr(i)*plan.stride+col.offset)
				if col.isByte {
					// A string column scatters into a []byte target field too
					// (row-major supports this interchange); copy to owned bytes.
					*(*[]byte)(dp) = append([]byte(nil), strs[i]...)
				} else {
					*(*string)(dp) = strs[i]
				}
			}
			break
		}
		for i := range n {
			dp := unsafe.Add(base, uintptr(i)*plan.stride+col.offset)
			if col.isByte {
				sb, err := d.readStringBytes()
				if err != nil {
					return err
				}
				*(*[]byte)(dp) = append([]byte(nil), sb...)
				continue
			}
			// ReadString shares repeated values via the decode-side intern
			// cache, so a low-cardinality string column that fell below the dict
			// gate scatters to ~distinct allocs, not one per row.
			str, err := d.ReadString()
			if err != nil {
				return err
			}
			*(*string)(dp) = str
		}
	case colKindTime:
		// Decode sec sub-column then nsec sub-column, reusing scratch buffers.
		if err := decodeSliceInt64Into(d, &st.colScratchI64); err != nil {
			return err
		}
		sec := st.colScratchI64
		if len(sec) != n {
			return ErrTypeMismatch
		}
		if err := decodeSliceUint64Into(d, &st.colScratchU64); err != nil {
			return err
		}
		nsec := st.colScratchU64
		if len(nsec) != n {
			return ErrTypeMismatch
		}
		if err := checkNsecColumn(nsec); err != nil {
			return err
		}
		for i := range n {
			dp := unsafe.Add(base, uintptr(i)*plan.stride+col.offset)
			*(*time.Time)(dp) = time.Unix(sec[i], int64(nsec[i])).UTC()
		}
	}
	return nil
}

// skipColumnValue fully decodes a column body of the given kind but discards
// the result. It is the slow correctness fallback for selective decode when the
// producer did not emit a column-length index: with no length to seek past, the
// cursor can only be advanced by actually decoding the body. It mirrors the
// cursor-advancing half of decodeColumnInto without the store/scatter step.
func (d *Decoder) skipColumnValue(kind colKind, n int) error {
	if kind.isNullable() {
		_, err := d.decodeNullableColumnAny(kind, n)
		return err
	}
	switch kind {
	case colKindInt:
		var s []int64
		return decodeSliceInt64(d, unsafe.Pointer(&s))
	case colKindUint:
		var s []uint64
		return decodeSliceUint64(d, unsafe.Pointer(&s))
	case colKindFloat:
		var s []float64
		return decodeSliceFloat64(d, unsafe.Pointer(&s))
	case colKindFloat32:
		var s []uint64
		return decodeSliceUint64(d, unsafe.Pointer(&s))
	case colKindBool:
		var s []bool
		return decodeSliceBool(d, unsafe.Pointer(&s))
	case colKindString:
		if d.i < len(d.buf) && isStringColumnBlockTag(d.buf[d.i]) {
			_, err := d.readStringColumn(n)
			return err
		}
		for range n {
			if _, err := d.readStringBytes(); err != nil {
				return err
			}
		}
		return nil
	case colKindTime:
		// Skip sec sub-column then nsec sub-column.
		var sec []int64
		if err := decodeSliceInt64(d, unsafe.Pointer(&sec)); err != nil {
			return err
		}
		var nsec []uint64
		return decodeSliceUint64(d, unsafe.Pointer(&nsec))
	default:
		return ErrBadTag
	}
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

// gatherColI64 / gatherColU64 / scatterColI64 / scatterColU64 are the bulk
// columnar gather/scatter loops. They hoist the loop-invariant col.width switch
// OUT of the per-element loop (one branch per column instead of per element) and
// presize the destination once. Measured ~13% faster encode / ~16% faster decode
// on narrow-int (int8/16/32, uint8/16/32) columnar structs vs the per-element
// loadScalarU64*/storeScalarFrom* calls; byte-identical wire. The per-element
// helpers above are kept for the probe-sample and query-scatter paths that store
// one value at a time.

func gatherColI64(dst []int64, base unsafe.Pointer, stride uintptr, col *colColumn, n int) []int64 {
	dst = slices.Grow(dst[:0], n)[:n]
	off := col.offset
	switch col.width {
	case 1:
		for i := range n {
			dst[i] = int64(*(*int8)(unsafe.Add(base, uintptr(i)*stride+off)))
		}
	case 2:
		for i := range n {
			dst[i] = int64(*(*int16)(unsafe.Add(base, uintptr(i)*stride+off)))
		}
	case 4:
		for i := range n {
			dst[i] = int64(*(*int32)(unsafe.Add(base, uintptr(i)*stride+off)))
		}
	default:
		for i := range n {
			dst[i] = *(*int64)(unsafe.Add(base, uintptr(i)*stride+off))
		}
	}
	return dst
}

func gatherColU64(dst []uint64, base unsafe.Pointer, stride uintptr, col *colColumn, n int) []uint64 {
	dst = slices.Grow(dst[:0], n)[:n]
	off := col.offset
	switch col.width {
	case 1:
		for i := range n {
			dst[i] = uint64(*(*uint8)(unsafe.Add(base, uintptr(i)*stride+off)))
		}
	case 2:
		for i := range n {
			dst[i] = uint64(*(*uint16)(unsafe.Add(base, uintptr(i)*stride+off)))
		}
	case 4:
		for i := range n {
			dst[i] = uint64(*(*uint32)(unsafe.Add(base, uintptr(i)*stride+off)))
		}
	default:
		for i := range n {
			dst[i] = *(*uint64)(unsafe.Add(base, uintptr(i)*stride+off))
		}
	}
	return dst
}

func scatterColI64(base unsafe.Pointer, stride uintptr, col *colColumn, s []int64) {
	off := col.offset
	switch col.width {
	case 1:
		for i := range s {
			*(*int8)(unsafe.Add(base, uintptr(i)*stride+off)) = int8(s[i])
		}
	case 2:
		for i := range s {
			*(*int16)(unsafe.Add(base, uintptr(i)*stride+off)) = int16(s[i])
		}
	case 4:
		for i := range s {
			*(*int32)(unsafe.Add(base, uintptr(i)*stride+off)) = int32(s[i])
		}
	default:
		for i := range s {
			*(*int64)(unsafe.Add(base, uintptr(i)*stride+off)) = s[i]
		}
	}
}

func scatterColU64(base unsafe.Pointer, stride uintptr, col *colColumn, s []uint64) {
	off := col.offset
	switch col.width {
	case 1:
		for i := range s {
			*(*uint8)(unsafe.Add(base, uintptr(i)*stride+off)) = uint8(s[i])
		}
	case 2:
		for i := range s {
			*(*uint16)(unsafe.Add(base, uintptr(i)*stride+off)) = uint16(s[i])
		}
	case 4:
		for i := range s {
			*(*uint32)(unsafe.Add(base, uintptr(i)*stride+off)) = uint32(s[i])
		}
	default:
		for i := range s {
			*(*uint64)(unsafe.Add(base, uintptr(i)*stride+off)) = s[i]
		}
	}
}

// checkNsecColumn rejects a time column whose nanosecond sub-column carries a
// value outside [0, 999_999_999], matching Decoder.ReadTimestamp. Without it the
// columnar/nullable time paths would silently normalize a hostile out-of-range
// nsec into a different valid instant instead of erroring.
func checkNsecColumn(nsec []uint64) error {
	for _, v := range nsec {
		if v > 999_999_999 {
			return ErrInvalidLength
		}
	}
	return nil
}

//go:nosplit
func storeFloat64(base unsafe.Pointer, stride uintptr, col *colColumn, i int, v float64) {
	// colKindFloat is float64-only (width 8); float32 uses colKindFloat32 +
	// storeFloat32Bits, so there is no width==4 case here.
	p := unsafe.Add(base, uintptr(i)*stride+col.offset)
	*(*float64)(p) = v
}

// decodeColumnarAny decodes a tagColStruct payload into a []any of
// map[string]any keyed by column name. Mirrors decodeColumnar's header
// and shape parse exactly; each column is decoded into a temp slice and
// the per-element value boxed into its row's map.
func decodeColumnarAny(d *Decoder) (any, error) {
	// The map-per-row reflection decode allocates n maps up front, far heavier
	// than the struct path's single backing slice, so it gets a tighter
	// element ceiling. A constant column can claim a huge n from a tiny body;
	// without this a small hostile input would drive a multi-gigabyte map
	// allocation. Callers decoding millions of rows should use a typed struct.
	cs, err := d.readColShape(maxColumnarAnyElems)
	if err != nil {
		return nil, err
	}
	n := cs.n
	sh := cs.sh
	colLens := cs.colLens
	d.colMaxLen = n
	defer func() { d.colMaxLen = 0 }()

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
		want := d.wantField(name)
		if !want && colLens != nil {
			if d.i+int(colLens[c]) > len(d.buf) {
				return nil, ErrShortBuffer
			}
			d.i += int(colLens[c])
			continue
		}
		// store is false only on the no-index skip path: the column body still
		// has to be decoded to keep the cursor in sync, but its values are
		// dropped rather than boxed into the row maps.
		store := want
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
		case colKindFloat32:
			var s []uint64
			if err := decodeSliceUint64(d, unsafe.Pointer(&s)); err != nil {
				return nil, err
			}
			if len(s) != n {
				return nil, ErrTypeMismatch
			}
			if store {
				for i := range n {
					out[i].(map[string]any)[name] = math.Float32frombits(uint32(s[i]))
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
			if d.i < len(d.buf) && isStringColumnBlockTag(d.buf[d.i]) {
				strs, err := d.readStringColumn(n)
				if err != nil {
					return nil, err
				}
				if store {
					for i := range n {
						out[i].(map[string]any)[name] = strs[i]
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
		case colKindTime:
			var sec []int64
			if err := decodeSliceInt64(d, unsafe.Pointer(&sec)); err != nil {
				return nil, err
			}
			if len(sec) != n {
				return nil, ErrTypeMismatch
			}
			var nsec []uint64
			if err := decodeSliceUint64(d, unsafe.Pointer(&nsec)); err != nil {
				return nil, err
			}
			if len(nsec) != n {
				return nil, ErrTypeMismatch
			}
			if err := checkNsecColumn(nsec); err != nil {
				return nil, err
			}
			if store {
				for i := range n {
					out[i].(map[string]any)[name] = time.Unix(sec[i], int64(nsec[i])).UTC()
				}
			}
		default:
			return nil, ErrBadTag
		}
	}
	return out, nil
}

// decodeColumnBodyAny decodes one eligible column body into the per-row maps
// (or discards it when store is false). Mirrors the per-kind switch in
// decodeColumnarAny; used by the hybrid any/skip path (decodeHybridColumnarAny).
func (d *Decoder) decodeColumnBodyAny(kind colKind, name string, n int, store bool, out []any) error {
	if kind.isNullable() {
		vals, err := d.decodeNullableColumnAny(kind, n)
		if err != nil {
			return err
		}
		if store {
			for i := range n {
				out[i].(map[string]any)[name] = vals[i]
			}
		}
		return nil
	}
	switch kind {
	case colKindInt:
		var s []int64
		if err := decodeSliceInt64(d, unsafe.Pointer(&s)); err != nil {
			return err
		}
		if len(s) != n {
			return ErrTypeMismatch
		}
		if store {
			for i := range n {
				out[i].(map[string]any)[name] = s[i]
			}
		}
	case colKindUint:
		var s []uint64
		if err := decodeSliceUint64(d, unsafe.Pointer(&s)); err != nil {
			return err
		}
		if len(s) != n {
			return ErrTypeMismatch
		}
		if store {
			for i := range n {
				out[i].(map[string]any)[name] = s[i]
			}
		}
	case colKindFloat:
		var s []float64
		if err := decodeSliceFloat64(d, unsafe.Pointer(&s)); err != nil {
			return err
		}
		if len(s) != n {
			return ErrTypeMismatch
		}
		if store {
			for i := range n {
				out[i].(map[string]any)[name] = s[i]
			}
		}
	case colKindFloat32:
		var s []uint64
		if err := decodeSliceUint64(d, unsafe.Pointer(&s)); err != nil {
			return err
		}
		if len(s) != n {
			return ErrTypeMismatch
		}
		if store {
			for i := range n {
				out[i].(map[string]any)[name] = math.Float32frombits(uint32(s[i]))
			}
		}
	case colKindBool:
		var s []bool
		if err := decodeSliceBool(d, unsafe.Pointer(&s)); err != nil {
			return err
		}
		if len(s) != n {
			return ErrTypeMismatch
		}
		if store {
			for i := range n {
				out[i].(map[string]any)[name] = s[i]
			}
		}
	case colKindString:
		if d.i < len(d.buf) && isStringColumnBlockTag(d.buf[d.i]) {
			strs, err := d.readStringColumn(n)
			if err != nil {
				return err
			}
			if store {
				for i := range n {
					out[i].(map[string]any)[name] = strs[i]
				}
			}
			return nil
		}
		for i := range n {
			sb, err := d.readStringBytes()
			if err != nil {
				return err
			}
			if store {
				out[i].(map[string]any)[name] = string(sb)
			}
		}
	case colKindTime:
		var sec []int64
		if err := decodeSliceInt64(d, unsafe.Pointer(&sec)); err != nil {
			return err
		}
		if len(sec) != n {
			return ErrTypeMismatch
		}
		var nsec []uint64
		if err := decodeSliceUint64(d, unsafe.Pointer(&nsec)); err != nil {
			return err
		}
		if len(nsec) != n {
			return ErrTypeMismatch
		}
		if err := checkNsecColumn(nsec); err != nil {
			return err
		}
		if store {
			for i := range n {
				out[i].(map[string]any)[name] = time.Unix(sec[i], int64(nsec[i])).UTC()
			}
		}
	default:
		return ErrBadTag
	}
	return nil
}

// decodeHybridColumnarAny decodes a tagHybridColStruct payload into a []any of
// map[string]any rows — the dynamic / any / Skip path (parallel to
// decodeColumnarAny). It fully decodes (not a byte-skip) so the intern + shape
// tables stay in sync for later state-refs. Eligible columns scatter into the
// row maps; each residual field decodes per row via the generic any decoder.
func decodeHybridColumnarAny(d *Decoder) (any, error) {
	n, sh, err := d.readHybridColShape(maxColumnarAnyElems)
	if err != nil {
		return nil, err
	}
	d.colMaxLen = n
	defer func() { d.colMaxLen = 0 }()

	out := make([]any, n)
	for i := range out {
		out[i] = make(map[string]any, len(sh.names))
	}
	// Eligible columns (residualKind entries have no column body — handled below).
	for c := range sh.kinds {
		if sh.kinds[c] == residualKind {
			continue
		}
		if err := d.decodeColumnBodyAny(sh.kinds[c], sh.names[c], n, true, out); err != nil {
			return nil, err
		}
	}
	// Residual block: per row, each residual field in shape order via decodeAny.
	// Residual fields are row-major; clear the columnar length bound so a
	// residual collection longer than n is not rejected by colLenOK (see
	// decodeHybridColumnar for the detailed rationale).
	d.colMaxLen = 0
	for i := range n {
		row := out[i].(map[string]any)
		for c := range sh.kinds {
			if sh.kinds[c] != residualKind {
				continue
			}
			v, err := decodeAny(d)
			if err != nil {
				return nil, err
			}
			row[sh.names[c]] = v
		}
	}
	return out, nil
}

func classifyColKind(fd *typeDesc) (ck colKind, width uintptr, isByte bool, ok bool) {
	if fd.marshalerKind != 0 {
		return 0, 0, false, false // custom marshaler → row-major
	}
	// time.Time is a struct that has its own scalar codec (encodeTime/decodeTime).
	// It must be detected before the generic struct fall-through so it gets
	// colKindTime instead of being rejected as ineligible.
	if fd.rType == reflect.TypeFor[time.Time]() {
		return colKindTime, unsafe.Sizeof(time.Time{}), false, true
	}
	switch fd.kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return colKindInt, fd.rType.Size(), false, true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return colKindUint, fd.rType.Size(), false, true
	case reflect.Float32:
		return colKindFloat32, fd.rType.Size(), false, true
	case reflect.Float64:
		return colKindFloat, fd.rType.Size(), false, true
	case reflect.Bool:
		return colKindBool, 1, false, true
	case reflect.String:
		return colKindString, 0, false, true
	case reflect.Slice:
		if fd.rType.Elem().Kind() == reflect.Uint8 { // []byte
			// Known limitation: a []byte column is a plain string column (no
			// per-row presence bit), kept that way so it stays wire-compatible
			// with a string column for the string<->[]byte schema interchange
			// (TestColumnarStringIntoByteField). The consequence is that the
			// reflect columnar path (n >= columnarMinElems) cannot distinguish a
			// nil []byte from an empty []byte{} — both decode to nil. The
			// row-major path (n < columnarMinElems) preserves the distinction.
			// Same inherent columnar-lossiness class as int->int64 widening and
			// time monotonic-clock stripping. (The qdfgen codegen path, which has
			// no string<->[]byte interchange to preserve, routes []byte through a
			// nullable column and keeps nil vs empty distinct.)
			return colKindString, 0, true, true
		}
		return 0, 0, false, false
	default:
		return 0, 0, false, false
	}
}
