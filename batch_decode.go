package qdf

import (
	"reflect"
	"time"
	"unsafe"

	"github.com/alex60217101990/qdf/internal/reflectutil"
)

// unmarshalBatchCore decodes data into rowsOut (a *[]T, viewed generically:
// the core never names T, all row writes go through unsafe offsets computed
// from plan.stride/plan.fields).
//
// v1 fallback strategy (correct-first; a columnar fast path replaces this
// for columnar wire in a later task): decode the whole payload through the
// EXISTING reflect-driven Unmarshal into a pooled []plan.mirror slice — a
// runtime struct type with the same wire field names as T but with handle
// types swapped back to string/[]byte/time.Time, so the normal decoder needs
// no new wire logic. Then one copy pass per row scatters each mirror row
// into a T row (memmove of scalar bytes) plus the slab (string/bytes bodies,
// rewritten as Str/Bytes handles) and converts qdf.Time.
func unmarshalBatchCore(data []byte, plan *batchPlan, slab *batchSlab, rowsOut unsafe.Pointer, opts ...QueryOption) (int, error) {
	// opts: arena/noCopy are accepted but ignored — the slab supersedes both
	// (it owns every byte a handle points into; there is nothing for an
	// arena or a no-copy alias to usefully do here). No opts are forwarded
	// to Unmarshal.
	_ = opts

	mirrorPtr := plan.mirrorSlicePtr.Get()
	defer plan.mirrorSlicePtr.Put(mirrorPtr)

	mv := reflect.ValueOf(mirrorPtr).Elem()
	mv.SetLen(0)

	if err := Unmarshal(data, mirrorPtr); err != nil {
		return 0, err
	}

	n := mv.Len()
	if n == 0 {
		*(*sliceHeader)(rowsOut) = sliceHeader{}
		return 0, nil
	}

	mirrorBase := unsafe.Pointer(mv.Index(0).UnsafeAddr())
	mirrorStride := plan.mirror.Size()

	// Sum string/bytes body lengths up front so the slab grows exactly once
	// instead of on every append.
	var need int
	for i := range n {
		rowPtr := unsafe.Add(mirrorBase, uintptr(i)*mirrorStride)
		for fi, f := range plan.fields {
			switch f.kind {
			case bfStr:
				s := *(*string)(unsafe.Add(rowPtr, plan.mirrorOff[fi]))
				need += len(s)
			case bfBytes:
				b := *(*[]byte)(unsafe.Add(rowPtr, plan.mirrorOff[fi]))
				need += len(b)
			}
		}
	}
	slab.grow(need)

	reflectutil.MakeSlice(reflect.SliceOf(plan.rt), n, rowsOut)
	rows := reflectutil.SliceData(reflect.SliceOf(plan.rt), rowsOut)

	batchCopyRows(plan, slab, mirrorBase, rows, n)

	return n, nil
}

// batchCopyRows scatters n mirror rows (starting at mirrorPtr, each
// plan.mirror.Size() bytes apart) into rowsBase (n rows of plan.stride
// bytes each), writing scalars via memmove, strings/bytes into slab (as
// Str/Bytes handles), and qdf.Time from time.Time.
func batchCopyRows(plan *batchPlan, slab *batchSlab, mirrorPtr, rowsBase unsafe.Pointer, n int) {
	mirrorStride := plan.mirror.Size()
	for i := range n {
		src := unsafe.Add(mirrorPtr, uintptr(i)*mirrorStride)
		dst := unsafe.Add(rowsBase, uintptr(i)*plan.stride)
		for fi, f := range plan.fields {
			sf := unsafe.Add(src, plan.mirrorOff[fi])
			df := unsafe.Add(dst, f.off)
			switch f.kind {
			case bfStr:
				s := *(*string)(sf)
				off, ln := slab.append(unsafe.Slice(unsafe.StringData(s), len(s)))
				*(*Str)(df) = Str{off: off, len: ln}
			case bfBytes:
				b := *(*[]byte)(sf)
				off, ln := slab.append(b)
				*(*Bytes)(df) = Bytes{off: off, len: ln}
			case bfTime:
				t := *(*time.Time)(sf)
				*(*Time)(df) = Time{Sec: t.Unix(), Nsec: uint32(t.Nanosecond())}
			default: // bfScalar
				scalarSize := scalarKindSize(f.scalarKind)
				copy(unsafe.Slice((*byte)(df), scalarSize), unsafe.Slice((*byte)(sf), scalarSize))
			}
		}
	}
}

// scalarKindSize returns the byte width of a scalar batchField, matching
// scalarKindType's type set.
func scalarKindSize(k reflect.Kind) uintptr {
	switch k {
	case reflect.Bool, reflect.Int8, reflect.Uint8:
		return 1
	case reflect.Int16, reflect.Uint16:
		return 2
	case reflect.Int32, reflect.Uint32, reflect.Float32:
		return 4
	case reflect.Int64, reflect.Uint64, reflect.Float64:
		return 8
	case reflect.Int, reflect.Uint, reflect.Uintptr:
		return unsafe.Sizeof(uintptr(0))
	default:
		return 0
	}
}
