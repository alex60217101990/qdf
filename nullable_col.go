package qdf

import (
	"math/bits"
	"reflect"
	"runtime"
	"unsafe"
)

// Nullable (optional) columns for the columnar container. A `*T` struct
// field — where T is a scalar (int*/uint*/float*/bool) — would otherwise
// force the whole struct to the row-major fallback, losing the columnar
// codecs on every sibling column. Instead it is stored as a presence
// bitmap (1 bit per row, LSB-first) followed by a dense column of only the
// present values, encoded with T's normal slice codec. The nullable flag
// rides in the columnar shape's kind byte (colKindNullable), so there is no
// separate wire tag; the column slot is `ceil(M/8) mask bytes, <dense
// column>`.

// writeStringColumn emits a gathered string column either as a dictionary
// (when the never-worse gate accepts it) or as M per-value strings. Shared
// by the regular string column path; factored out so nullable columns could
// reuse it (nullable string columns are a follow-up).
func (e *Encoder) writeStringColumn(strs []string) {
	if e.tryWriteStringColumnDict(strs) {
		return
	}
	for _, v := range strs {
		e.WriteString(v)
	}
}

func loadI64At(p unsafe.Pointer, width uintptr) int64 {
	switch width {
	case 1:
		return int64(*(*int8)(p))
	case 2:
		return int64(*(*int16)(p))
	case 4:
		return int64(*(*int32)(p))
	default:
		return *(*int64)(p)
	}
}

func loadU64At(p unsafe.Pointer, width uintptr) uint64 {
	switch width {
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

func loadF64At(p unsafe.Pointer, width uintptr) float64 {
	if width == 4 {
		return float64(*(*float32)(p))
	}
	return *(*float64)(p)
}

func storeI64At(p unsafe.Pointer, width uintptr, v int64) {
	switch width {
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

func storeU64At(p unsafe.Pointer, width uintptr, v uint64) {
	switch width {
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

func storeF64At(p unsafe.Pointer, width uintptr, v float64) {
	if width == 4 {
		*(*float32)(p) = float32(v)
	} else {
		*(*float64)(p) = v
	}
}

// encodeNullableColumn writes the presence bitmap and the dense present-only
// column for a `*T` field.
func (e *Encoder) encodeNullableColumn(base unsafe.Pointer, plan *columnarPlan, col *colColumn, n int) error {
	st := e.state
	maskBytes := (n + 7) >> 3
	var mask []byte
	if cap(st.colMaskScratch) >= maskBytes {
		mask = st.colMaskScratch[:maskBytes]
		clear(mask)
	} else {
		mask = make([]byte, maskBytes)
	}
	st.colMaskScratch = mask
	stride, off := plan.stride, col.offset

	switch col.kind.base() {
	case colKindInt:
		s := st.colScratchI64[:0]
		for i := range n {
			pp := *(*unsafe.Pointer)(unsafe.Add(base, uintptr(i)*stride+off))
			if pp != nil {
				mask[i>>3] |= 1 << uint(i&7)
				s = append(s, loadI64At(pp, col.width))
			}
		}
		st.colScratchI64 = s
		e.buf = append(e.buf, mask...)
		return encodeSliceInt64(e, unsafe.Pointer(&s))
	case colKindUint:
		s := st.colScratchU64[:0]
		for i := range n {
			pp := *(*unsafe.Pointer)(unsafe.Add(base, uintptr(i)*stride+off))
			if pp != nil {
				mask[i>>3] |= 1 << uint(i&7)
				s = append(s, loadU64At(pp, col.width))
			}
		}
		st.colScratchU64 = s
		e.buf = append(e.buf, mask...)
		return encodeSliceUint64(e, unsafe.Pointer(&s))
	case colKindFloat:
		s := st.colScratchF64[:0]
		for i := range n {
			pp := *(*unsafe.Pointer)(unsafe.Add(base, uintptr(i)*stride+off))
			if pp != nil {
				mask[i>>3] |= 1 << uint(i&7)
				s = append(s, loadF64At(pp, col.width))
			}
		}
		st.colScratchF64 = s
		e.buf = append(e.buf, mask...)
		return encodeSliceFloat64(e, unsafe.Pointer(&s))
	case colKindBool:
		s := st.colScratchBool[:0]
		for i := range n {
			pp := *(*unsafe.Pointer)(unsafe.Add(base, uintptr(i)*stride+off))
			if pp != nil {
				mask[i>>3] |= 1 << uint(i&7)
				s = append(s, *(*bool)(pp))
			}
		}
		st.colScratchBool = s
		e.buf = append(e.buf, mask...)
		return encodeSliceBool(e, unsafe.Pointer(&s))
	case colKindString:
		s := st.colScratchStr[:0]
		for i := range n {
			pp := *(*unsafe.Pointer)(unsafe.Add(base, uintptr(i)*stride+off))
			if pp != nil {
				mask[i>>3] |= 1 << uint(i&7)
				s = append(s, *(*string)(pp))
			}
		}
		st.colScratchStr = s
		e.buf = append(e.buf, mask...)
		e.writeStringColumn(s)
		return nil
	}
	return ErrBadTag
}

// readNullableMask consumes the presence bitmap and returns it plus the
// present count (popcount). The caller decodes the dense column next.
func (d *Decoder) readNullableMask(n int) (mask []byte, present int, err error) {
	maskBytes := (n + 7) >> 3
	if d.i+maskBytes > len(d.buf) {
		return nil, 0, ErrShortBuffer
	}
	mask = d.buf[d.i : d.i+maskBytes]
	d.i += maskBytes
	for _, b := range mask {
		present += bits.OnesCount8(b)
	}
	return mask, present, nil
}

// decodeNullableColumn reads the mask + dense column and scatters the present
// values back into the `*T` field, allocating all present values in a single
// backing slice that the field pointers reference into.
func (d *Decoder) decodeNullableColumn(base unsafe.Pointer, plan *columnarPlan, col *colColumn, n int) error {
	mask, present, err := d.readNullableMask(n)
	if err != nil {
		return err
	}
	elemSize := col.elemType.Size()
	backing := reflect.MakeSlice(reflect.SliceOf(col.elemType), present, present)
	dataPtr := backing.UnsafePointer()
	stride, off := plan.stride, col.offset
	k := 0
	set := func(store func(ea unsafe.Pointer, k int)) {
		for i := range n {
			fp := unsafe.Add(base, uintptr(i)*stride+off)
			if mask[i>>3]&(1<<uint(i&7)) != 0 {
				ea := unsafe.Add(dataPtr, uintptr(k)*elemSize)
				store(ea, k)
				*(*unsafe.Pointer)(fp) = ea
				k++
			} else {
				*(*unsafe.Pointer)(fp) = nil
			}
		}
	}
	switch col.kind.base() {
	case colKindInt:
		var s []int64
		if err := decodeSliceInt64(d, unsafe.Pointer(&s)); err != nil {
			return err
		}
		if len(s) != present {
			return ErrTypeMismatch
		}
		set(func(ea unsafe.Pointer, k int) { storeI64At(ea, col.width, s[k]) })
	case colKindUint:
		var s []uint64
		if err := decodeSliceUint64(d, unsafe.Pointer(&s)); err != nil {
			return err
		}
		if len(s) != present {
			return ErrTypeMismatch
		}
		set(func(ea unsafe.Pointer, k int) { storeU64At(ea, col.width, s[k]) })
	case colKindFloat:
		var s []float64
		if err := decodeSliceFloat64(d, unsafe.Pointer(&s)); err != nil {
			return err
		}
		if len(s) != present {
			return ErrTypeMismatch
		}
		set(func(ea unsafe.Pointer, k int) { storeF64At(ea, col.width, s[k]) })
	case colKindBool:
		var s []bool
		if err := decodeSliceBool(d, unsafe.Pointer(&s)); err != nil {
			return err
		}
		if len(s) != present {
			return ErrTypeMismatch
		}
		set(func(ea unsafe.Pointer, k int) { *(*bool)(ea) = s[k] })
	case colKindString:
		strs, err := d.readStringColumn(present)
		if err != nil {
			return err
		}
		k := 0
		for i := range n {
			fp := unsafe.Add(base, uintptr(i)*stride+off)
			if mask[i>>3]&(1<<uint(i&7)) != 0 {
				*(*unsafe.Pointer)(fp) = unsafe.Pointer(&strs[k])
				k++
			} else {
				*(*unsafe.Pointer)(fp) = nil
			}
		}
		runtime.KeepAlive(strs)
		return nil
	default:
		return ErrBadTag
	}
	runtime.KeepAlive(backing)
	return nil
}

// decodeNullableColumnAny reads the mask + dense column and returns one boxed
// value per row (nil for absent), for the map[string]any decode path.
func (d *Decoder) decodeNullableColumnAny(kind colKind, n int) ([]any, error) {
	mask, present, err := d.readNullableMask(n)
	if err != nil {
		return nil, err
	}
	out := make([]any, n)
	k := 0
	scatter := func(box func(i, k int)) {
		for i := range n {
			if mask[i>>3]&(1<<uint(i&7)) != 0 {
				box(i, k)
				k++
			}
		}
	}
	switch kind.base() {
	case colKindInt:
		var s []int64
		if err := decodeSliceInt64(d, unsafe.Pointer(&s)); err != nil {
			return nil, err
		}
		if len(s) != present {
			return nil, ErrTypeMismatch
		}
		scatter(func(i, k int) { out[i] = s[k] })
	case colKindUint:
		var s []uint64
		if err := decodeSliceUint64(d, unsafe.Pointer(&s)); err != nil {
			return nil, err
		}
		if len(s) != present {
			return nil, ErrTypeMismatch
		}
		scatter(func(i, k int) { out[i] = s[k] })
	case colKindFloat:
		var s []float64
		if err := decodeSliceFloat64(d, unsafe.Pointer(&s)); err != nil {
			return nil, err
		}
		if len(s) != present {
			return nil, ErrTypeMismatch
		}
		scatter(func(i, k int) { out[i] = s[k] })
	case colKindBool:
		var s []bool
		if err := decodeSliceBool(d, unsafe.Pointer(&s)); err != nil {
			return nil, err
		}
		if len(s) != present {
			return nil, ErrTypeMismatch
		}
		scatter(func(i, k int) { out[i] = s[k] })
	case colKindString:
		strs, err := d.readStringColumn(present)
		if err != nil {
			return nil, err
		}
		scatter(func(i, k int) { out[i] = strs[k] })
	default:
		return nil, ErrBadTag
	}
	return out, nil
}
