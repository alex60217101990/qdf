package qdf

import (
	"reflect"
	"slices"
	"time"
	"unsafe"

	"github.com/alex60217101990/qdf/internal/reflectutil"
	"github.com/alex60217101990/qdf/internal/unsafestr"
)

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

func encodeSlice(elem *typeDesc, stride uintptr, colPlan *columnarPlan) func(*Encoder, unsafe.Pointer) error {
	return func(e *Encoder, p unsafe.Pointer) error {
		hdr := (*sliceHeader)(p)
		n := hdr.Len
		if colPlan != nil && n >= columnarMinElems && e.state != nil &&
			e.opts.Has(OptDense) && e.opts.Has(OptShapeIntern) &&
			columnarProbe(colPlan, hdr.Data, n) {
			e.writeHeader()
			return e.encodeColumnar(colPlan, hdr.Data, n)
		}
		e.WriteArrayHeader(n)
		base := hdr.Data
		// Probe-and-grow for large slices: encode the first
		// sliceProbeSize records, measure the per-record buffer
		// growth, then pre-grow the output buffer for the rest in
		// one shot. Eliminates the log(n) doubling chain
		// (runtime.memmove + madvise) that dominated 50 k-record
		// encodes — at scale the buffer can balloon from 4 KiB to
		// 60 MiB through ~14 doublings, copying ~4× the final size.
		// The probe cost is negligible because the same elements
		// are emitted exactly once.
		const sliceProbeSize = 32
		if n <= sliceProbeSize {
			for i := range n {
				if err := elem.encode(e, unsafe.Add(base, uintptr(i)*stride)); err != nil {
					return err
				}
			}
			return nil
		}
		probeStart := len(e.buf)
		for i := range sliceProbeSize {
			if err := elem.encode(e, unsafe.Add(base, uintptr(i)*stride)); err != nil {
				return err
			}
		}
		probeBytes := len(e.buf) - probeStart
		if probeBytes > 0 {
			// Project total size from probe + 25 % slack so the
			// growslice short-circuit fires inside slices.Grow
			// without forcing another doubling on a slight
			// underestimate.
			remaining := n - sliceProbeSize
			projected := probeBytes * remaining / sliceProbeSize
			projected += projected >> 2
			e.buf = slices.Grow(e.buf, projected)
		}
		for i := sliceProbeSize; i < n; i++ {
			if err := elem.encode(e, unsafe.Add(base, uintptr(i)*stride)); err != nil {
				return err
			}
		}
		return nil
	}
}

func decodeSlice(t reflect.Type, elem *typeDesc, stride uintptr, colPlan *columnarPlan) func(*Decoder, unsafe.Pointer) error {
	// elemDynamic is true when the slice element is map[string]any or any, so a
	// columnar (tagColStruct) payload can be decoded dynamically into row maps
	// via decodeColumnarAny even though the static element type carries no
	// columnarPlan. This is the path UnmarshalColumns into *[]map[string]any
	// (or *[]any) takes.
	elemType := t.Elem()
	elemDynamic := elemType == reflect.TypeFor[map[string]any]() || elemType.Kind() == reflect.Interface
	return func(d *Decoder, p unsafe.Pointer) error {
		if colPlan != nil {
			if tag, err := d.peekTag(); err == nil && tag == tagColStruct {
				return decodeColumnar(d, t, colPlan, p)
			}
		}
		if elemDynamic {
			if tag, err := d.peekTag(); err == nil && tag == tagColStruct {
				rows, err := decodeColumnarAny(d)
				if err != nil {
					return err
				}
				src := rows.([]any)
				reflectutil.MakeSlice(t, len(src), p)
				base := reflectutil.SliceData(t, p)
				for i, row := range src {
					reflect.NewAt(elemType, unsafe.Add(base, uintptr(i)*stride)).Elem().Set(reflect.ValueOf(row))
				}
				return nil
			}
		}
		n, err := d.ReadArrayHeader()
		if err != nil {
			return err
		}
		if err := d.CheckLength(n, 1); err != nil {
			return err
		}
		// reflectutil.MakeSlice is swapped out under -tags qdf_reflect2
		// for the reflect2 implementation (skips reflect.MakeSlice
		// type checks).
		reflectutil.MakeSlice(t, n, p)
		base := reflectutil.SliceData(t, p)
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
		// Allocate via the swappable reflectutil backend.
		reflectutil.MakeMap(t, n, p)
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
		e.writeHeader()
		n := len(fields)
		// Dense mode: route through the tagMapShape path when
		// OptShapeIntern is set. On the first emit of a struct type the
		// encoder declares the shape (writing field names through the
		// normal intern path); on every subsequent emit it writes only
		// 0xEC + shapeID + values. With OptShapeIntern off, Dense
		// falls back to the tagMap8/16/32 encoding so the rest of the
		// state stack (intern + Markov / MTF / Pair) still applies
		// per-field.
		if e.opts.Has(OptDense) && e.state != nil && e.opts.Has(OptShapeIntern) {
			if id := e.state.shapeForType(td); id != 0 {
				e.buf = append(e.buf, tagMapShape)
				e.buf = appendUvarint(e.buf, uint64(id))
				for i := range fields {
					f := &fields[i]
					if err := f.desc.encode(e, unsafe.Add(p, f.offset)); err != nil {
						return err
					}
				}
				return nil
			}
			// First time: declare and emit keys via the standard intern path.
			shapeID := e.state.shapeDeclareEnc()
			e.state.shapeBindType(td, shapeID)
			st := e.state
			e.buf = append(e.buf, tagMapShape)
			e.buf = appendUvarint(e.buf, 0) // 0 ⇒ declaration follows
			e.buf = appendUvarint(e.buf, uint64(n))
			pairOn := e.opts.Has(OptPairPred)
			for i := range fields {
				f := &fields[i]
				if len(f.name) >= e.minIntern && int(st.internLoad) < e.maxStateEntries {
					if id, ok := st.lookupOrAssign(f.name); ok {
						if st.lastID == id {
							e.buf = append(e.buf, tagStateRepeat)
							if pairOn {
								st.pairRecord(id, id)
							}
						} else {
							e.emitStateRef(id)
						}
					} else {
						e.buf = append(e.buf, f.preInternStr...)
						if st.lastID != lruInvalidID && pairOn {
							st.pairRecord(st.lastID, id)
						}
						st.lastID = id
					}
				} else {
					e.buf = append(e.buf, f.preFast...)
					st.lastID = lruInvalidID
				}
			}
			for i := range fields {
				f := &fields[i]
				if err := f.desc.encode(e, unsafe.Add(p, f.offset)); err != nil {
					return err
				}
			}
			return nil
		}
		// Dense without OptShapeIntern: tagMap8/16/32 header + per-field
		// intern via WriteString. Keys still go through the intern
		// path so state-ref / MTF / Pair codecs cover them when their
		// options are on.
		if e.opts.Has(OptDense) && e.state != nil {
			e.WriteMapHeader(n)
			st := e.state
			pairOn := e.opts.Has(OptPairPred)
			for i := range fields {
				f := &fields[i]
				if len(f.name) >= e.minIntern && int(st.internLoad) < e.maxStateEntries {
					if id, ok := st.lookupOrAssign(f.name); ok {
						if st.lastID == id {
							e.buf = append(e.buf, tagStateRepeat)
							if pairOn {
								st.pairRecord(id, id)
							}
						} else {
							e.emitStateRef(id)
						}
					} else {
						e.buf = append(e.buf, f.preInternStr...)
						if st.lastID != lruInvalidID && pairOn {
							st.pairRecord(st.lastID, id)
						}
						st.lastID = id
					}
				} else {
					e.buf = append(e.buf, f.preFast...)
					st.lastID = lruInvalidID
				}
				if err := f.desc.encode(e, unsafe.Add(p, f.offset)); err != nil {
					return err
				}
			}
			return nil
		}
		// Fast mode (no Dense): plain tagMap8/16/32 encoding — no
		// intern, no shape, fixstr field-name headers from preFast.
		e.WriteMapHeader(n)
		for i := range fields {
			f := &fields[i]
			e.buf = append(e.buf, f.preFast...)
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

	resolveField := func(name string) *fieldDesc {
		if useMap {
			return byName[name]
		}
		for j := range indexed {
			if indexed[j].name == name {
				return indexed[j].f
			}
		}
		return nil
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
		// tagMapShape path: structs encoded via the Dense shape codec.
		if tag == tagMapShape {
			d.i++
			shapeID, n := readUvarint(d.buf[d.i:])
			if n <= 0 {
				return ErrInvalidLength
			}
			d.i += n
			if d.state == nil {
				d.state = newDecState()
			}
			var fieldNames []string
			if shapeID == 0 {
				// Declaration: read count, then N keys, then N values.
				cnt64, n := readUvarint(d.buf[d.i:])
				if n <= 0 {
					return ErrInvalidLength
				}
				d.i += n
				cnt := int(cnt64)
				if err := d.CheckLength(cnt, 1); err != nil {
					return err
				}
				sh := d.state.shapeDeclare()
				sh.keyIDs = make([]uint32, 0, cnt)
				keys := make([]string, 0, cnt)
				for range cnt {
					kb, err := d.readStringBytes()
					if err != nil {
						return err
					}
					// Cache the resolved name. The decoder state's
					// lastID is updated by readStringBytes for state-ref
					// variants, so we capture it as the key's intern ID
					// when available (purely informational; we look up
					// by name string anyway).
					keys = append(keys, d.keyCache.Make(kb))
					if d.state.lastID != lruInvalidID {
						sh.keyIDs = append(sh.keyIDs, d.state.lastID)
					} else {
						sh.keyIDs = append(sh.keyIDs, 0)
					}
				}
				// Attach the field name slice to the shape entry by
				// stashing it after keyIDs: we reuse a small parallel
				// slice (the names) keyed by index.
				sh.names = keys
				fieldNames = keys
			} else {
				sh := d.state.shapeLookup(uint32(shapeID))
				if sh == nil {
					return ErrUnknownStateID
				}
				fieldNames = sh.names
			}
			for _, name := range fieldNames {
				fd := resolveField(name)
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
		// tagMap8/16/32 path — used by Fast mode, by Dense without
		// OptShapeIntern, and by any external encoder that does not
		// emit shape headers.
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
			fd := resolveField(name)
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
	case tagStr8, tagStr16, tagStr32, tagInternStr, tagStateRef,
		tagStateRepeat, tagStateMTF, tagStatePair:
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
	case tagMapShape:
		d.i++
		shapeID, n := readUvarint(d.buf[d.i:])
		if n <= 0 {
			return nil, ErrInvalidLength
		}
		d.i += n
		if d.state == nil {
			d.state = newDecState()
		}
		var names []string
		if shapeID == 0 {
			cnt64, n := readUvarint(d.buf[d.i:])
			if n <= 0 {
				return nil, ErrInvalidLength
			}
			d.i += n
			cnt := int(cnt64)
			if err := d.CheckLength(cnt, 1); err != nil {
				return nil, err
			}
			sh := d.state.shapeDeclare()
			sh.keyIDs = make([]uint32, 0, cnt)
			sh.names = make([]string, 0, cnt)
			for range cnt {
				kb, err := d.readStringBytes()
				if err != nil {
					return nil, err
				}
				sh.names = append(sh.names, d.keyCache.Make(kb))
				if d.state.lastID != lruInvalidID {
					sh.keyIDs = append(sh.keyIDs, d.state.lastID)
				} else {
					sh.keyIDs = append(sh.keyIDs, 0)
				}
			}
			names = sh.names
		} else {
			sh := d.state.shapeLookup(uint32(shapeID))
			if sh == nil {
				return nil, ErrUnknownStateID
			}
			names = sh.names
		}
		out := make(map[string]any, len(names))
		for _, name := range names {
			v, err := decodeAny(d)
			if err != nil {
				return nil, err
			}
			out[name] = v
		}
		return out, nil
	case tagColStruct:
		return decodeColumnarAny(d)
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
