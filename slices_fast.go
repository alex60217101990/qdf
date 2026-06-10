package qdf

import (
	"reflect"
	"unsafe"
)

// Specialized fast paths for slices of common primitive element types.
// Selected in fillDesc when the element kind matches.

var (
	sliceStringType  = reflect.TypeFor[[]string]()
	sliceIntType     = reflect.TypeFor[[]int]()
	sliceInt32Type   = reflect.TypeFor[[]int32]()
	sliceInt64Type   = reflect.TypeFor[[]int64]()
	sliceUint32Type  = reflect.TypeFor[[]uint32]()
	sliceUint64Type  = reflect.TypeFor[[]uint64]()
	sliceFloat32Type = reflect.TypeFor[[]float32]()
	sliceFloat64Type = reflect.TypeFor[[]float64]()
	sliceBoolType    = reflect.TypeFor[[]bool]()
)

// installSliceFastPath returns (encode, decode, true) if t is a recognized
// primitive slice. Returns (_, _, false) for the generic case.
func installSliceFastPath(t reflect.Type) (
	enc func(*Encoder, unsafe.Pointer) error,
	dec func(*Decoder, unsafe.Pointer) error,
	ok bool,
) {
	// The *Nil variants preserve the nil-vs-empty distinction at the FIELD level
	// (a nil slice → tagNil) while the bare encodeSlice*/decodeSlice* stay
	// nil-agnostic for their internal direct callers (dense columns). Each *Nil
	// adds an inline nil check then a DIRECT call to the bare codec — no wrapping
	// closure, so a slice field encode/decode costs no extra indirect call.
	switch t {
	case sliceStringType:
		return encodeSliceStringNil, decodeSliceStringNil, true
	case sliceIntType:
		return encodeSliceIntNil, decodeSliceIntNil, true
	case sliceInt32Type:
		return encodeSliceInt32Nil, decodeSliceInt32Nil, true
	case sliceInt64Type:
		return encodeSliceInt64Nil, decodeSliceInt64Nil, true
	case sliceUint32Type:
		return encodeSliceUint32Nil, decodeSliceUint32Nil, true
	case sliceUint64Type:
		return encodeSliceUint64Nil, decodeSliceUint64Nil, true
	case sliceFloat32Type:
		return encodeSliceFloat32Nil, decodeSliceFloat32Nil, true
	case sliceFloat64Type:
		return encodeSliceFloat64Nil, decodeSliceFloat64Nil, true
	case sliceBoolType:
		return encodeSliceBoolNil, decodeSliceBoolNil, true
	}
	return nil, nil, false
}

// Nil-aware field variants: inline nil check + DIRECT call to the bare codec.
func encodeSliceStringNil(e *Encoder, p unsafe.Pointer) error {
	if e.encodeNilSlice(p) {
		return nil
	}
	return encodeSliceString(e, p)
}
func decodeSliceStringNil(d *Decoder, p unsafe.Pointer) error {
	if d.decodeNilSlice(p) {
		return nil
	}
	return decodeSliceString(d, p)
}
func encodeSliceIntNil(e *Encoder, p unsafe.Pointer) error {
	if e.encodeNilSlice(p) {
		return nil
	}
	return encodeSliceInt(e, p)
}
func decodeSliceIntNil(d *Decoder, p unsafe.Pointer) error {
	if d.decodeNilSlice(p) {
		return nil
	}
	return decodeSliceInt(d, p)
}
func encodeSliceInt32Nil(e *Encoder, p unsafe.Pointer) error {
	if e.encodeNilSlice(p) {
		return nil
	}
	return encodeSliceInt32(e, p)
}
func decodeSliceInt32Nil(d *Decoder, p unsafe.Pointer) error {
	if d.decodeNilSlice(p) {
		return nil
	}
	return decodeSliceInt32(d, p)
}
func encodeSliceInt64Nil(e *Encoder, p unsafe.Pointer) error {
	if e.encodeNilSlice(p) {
		return nil
	}
	return encodeSliceInt64(e, p)
}
func decodeSliceInt64Nil(d *Decoder, p unsafe.Pointer) error {
	if d.decodeNilSlice(p) {
		return nil
	}
	return decodeSliceInt64(d, p)
}
func encodeSliceUint32Nil(e *Encoder, p unsafe.Pointer) error {
	if e.encodeNilSlice(p) {
		return nil
	}
	return encodeSliceUint32(e, p)
}
func decodeSliceUint32Nil(d *Decoder, p unsafe.Pointer) error {
	if d.decodeNilSlice(p) {
		return nil
	}
	return decodeSliceUint32(d, p)
}
func encodeSliceUint64Nil(e *Encoder, p unsafe.Pointer) error {
	if e.encodeNilSlice(p) {
		return nil
	}
	return encodeSliceUint64(e, p)
}
func decodeSliceUint64Nil(d *Decoder, p unsafe.Pointer) error {
	if d.decodeNilSlice(p) {
		return nil
	}
	return decodeSliceUint64(d, p)
}
func encodeSliceFloat32Nil(e *Encoder, p unsafe.Pointer) error {
	if e.encodeNilSlice(p) {
		return nil
	}
	return encodeSliceFloat32(e, p)
}
func decodeSliceFloat32Nil(d *Decoder, p unsafe.Pointer) error {
	if d.decodeNilSlice(p) {
		return nil
	}
	return decodeSliceFloat32(d, p)
}
func encodeSliceFloat64Nil(e *Encoder, p unsafe.Pointer) error {
	if e.encodeNilSlice(p) {
		return nil
	}
	return encodeSliceFloat64(e, p)
}
func decodeSliceFloat64Nil(d *Decoder, p unsafe.Pointer) error {
	if d.decodeNilSlice(p) {
		return nil
	}
	return decodeSliceFloat64(d, p)
}
func encodeSliceBoolNil(e *Encoder, p unsafe.Pointer) error {
	if e.encodeNilSlice(p) {
		return nil
	}
	return encodeSliceBool(e, p)
}
func decodeSliceBoolNil(d *Decoder, p unsafe.Pointer) error {
	if d.decodeNilSlice(p) {
		return nil
	}
	return decodeSliceBool(d, p)
}

// ----- []string -----

func encodeSliceString(e *Encoder, p unsafe.Pointer) error {
	s := *(*[]string)(p)
	e.WriteArrayHeader(len(s))
	for i := range s {
		e.WriteString(s[i])
	}
	return nil
}
func decodeSliceString(d *Decoder, p unsafe.Pointer) error {
	n, err := d.ReadArrayHeader()
	if err != nil {
		return err
	}
	if err := d.CheckLength(n, 1); err != nil {
		return err
	}
	out := make([]string, n)
	for i := range n {
		v, err := d.ReadString()
		if err != nil {
			return err
		}
		out[i] = v
	}
	*(*[]string)(p) = out
	return nil
}

// ----- QPack codec helpers shared by 64-bit and widened 32-bit paths -----

// writeQPackUint64 runs the codec picker over s and writes the chosen QPack form.
func (e *Encoder) writeQPackUint64(s []uint64) {
	codec, mn, forBits, first, minDelta, deltaBits, pforBits, _ := pickU64Codec(s)
	e.emitQPackUint64(s, codec, mn, forBits, first, minDelta, deltaBits, pforBits)
}

// emitQPackUint64 writes s in the already-chosen codec form (picker output passed
// in, so a caller that needs to inspect the choice does not pick twice).
func (e *Encoder) emitQPackUint64(s []uint64, codec qpackCodec, mn uint64, forBits int, first uint64, minDelta int64, deltaBits, pforBits int) {
	switch codec {
	case qpackFor:
		e.writePackedForUint64Slice(s, mn, forBits)
	case qpackDeltaFor:
		e.writePackedDeltaForUint64Slice(s, first, minDelta, deltaBits)
	case qpackRLE:
		e.writePackedRLEUint64Slice(s)
	case qpackDict:
		e.writePackedDictUint64Slice(s)
	case qpackPFor:
		e.writePackedPForUint64Slice(s, mn, pforBits)
	default:
		e.writePackedUint64Slice(s)
	}
}

// writeQPackInt64 runs the codec picker over s and writes the chosen QPack form.
func (e *Encoder) writeQPackInt64(s []int64) {
	codec, mn, forBits, first, minDelta, deltaBits, pforBits, _ := pickI64Codec(s)
	e.emitQPackInt64(s, codec, mn, forBits, first, minDelta, deltaBits, pforBits)
}

// emitQPackInt64 writes s in the already-chosen codec form (picker output passed
// in, so a caller that needs to inspect the choice does not pick twice).
func (e *Encoder) emitQPackInt64(s []int64, codec qpackCodec, mn int64, forBits int, first int64, minDelta int64, deltaBits, pforBits int) {
	switch codec {
	case qpackFor:
		e.writePackedForInt64Slice(s, mn, forBits)
	case qpackDeltaFor:
		e.writePackedDeltaForInt64Slice(s, first, minDelta, deltaBits)
	case qpackRLE:
		e.writePackedRLEInt64Slice(s)
	case qpackDict:
		e.writePackedDictInt64Slice(s)
	case qpackPFor:
		e.writePackedPForInt64Slice(s, mn, pforBits)
	default:
		e.writePackedInt64Slice(s)
	}
}

// ----- []int / []int32 / []int64 / []uint32 / []uint64 -----

func encodeSliceInt(e *Encoder, p unsafe.Pointer) error {
	s := *(*[]int)(p)
	if e.qpack {
		// []int is platform-sized. On 64-bit platforms we re-view as
		// int64 and dispatch through pickI64Codec; on 32-bit we use the
		// raw int32 fast path. unsafe.Sizeof is a compile-time constant
		// so the dead branch is eliminated.
		if unsafe.Sizeof(int(0)) == 8 {
			s64 := unsafe.Slice((*int64)(unsafe.Pointer(unsafe.SliceData(s))), len(s))
			e.writeQPackInt64(s64)
			return nil
		}
		s32 := unsafe.Slice((*int32)(unsafe.Pointer(unsafe.SliceData(s))), len(s))
		e.writePackedInt32Slice(s32)
		return nil
	}
	e.WriteArrayHeader(len(s))
	for i := range s {
		e.WriteInt(int64(s[i]))
	}
	return nil
}
func decodeSliceInt(d *Decoder, p unsafe.Pointer) error {
	t, err := d.peekTag()
	if err != nil {
		return err
	}
	switch t {
	case tagPackRaw, tagPackFor, tagPackDeltaFor, tagPackRLE, tagPackDict, tagPackPFor:
		if unsafe.Sizeof(int(0)) == 8 {
			var dest []int64
			if err := decodeSliceInt64(d, unsafe.Pointer(&dest)); err != nil {
				return err
			}
			out := unsafe.Slice((*int)(unsafe.Pointer(unsafe.SliceData(dest))), len(dest))
			*(*[]int)(p) = out
			return nil
		}
		var dest []int32
		if err := decodeSliceInt32(d, unsafe.Pointer(&dest)); err != nil {
			return err
		}
		out := unsafe.Slice((*int)(unsafe.Pointer(unsafe.SliceData(dest))), len(dest))
		*(*[]int)(p) = out
		return nil
	}
	n, err := d.ReadArrayHeader()
	if err != nil {
		return err
	}
	if err := d.CheckLength(n, 1); err != nil {
		return err
	}
	out := make([]int, n)
	for i := range n {
		v, err := d.ReadInt()
		if err != nil {
			return err
		}
		out[i] = int(v)
	}
	*(*[]int)(p) = out
	return nil
}

// widenI64 promotes s into the encoder's reused int64 widening scratch and
// returns the filled slice. The result is valid only until the next widenI64
// call on the same encoder; the QPack int32 path consumes it (pick + emit)
// before any such call, so a single buffer serves every []int32 field.
func (e *Encoder) widenI64(s []int32) []int64 {
	if cap(e.wideI64) < len(s) {
		e.wideI64 = make([]int64, len(s))
	}
	w := e.wideI64[:len(s)]
	for i, v := range s {
		w[i] = int64(v)
	}
	return w
}

// widenU64 is the uint32→uint64 analogue of widenI64.
func (e *Encoder) widenU64(s []uint32) []uint64 {
	if cap(e.wideU64) < len(s) {
		e.wideU64 = make([]uint64, len(s))
	}
	w := e.wideU64[:len(s)]
	for i, v := range s {
		w[i] = uint64(v)
	}
	return w
}

func encodeSliceInt32(e *Encoder, p unsafe.Pointer) error {
	s := *(*[]int32)(p)
	if e.qpack {
		w := e.widenI64(s)
		codec, mn, forBits, first, minDelta, deltaBits, pforBits, bestCost := pickI64Codec(w)
		// Never-worse floor: native int32-raw is 4 B/elem vs the picker's 8 B/elem
		// uint64-raw baseline. Emit the widened codec only when it beats native
		// int32-raw; else native raw so incompressible 32-bit data isn't inflated.
		if bestCost >= 2+uvarintLen(uint64(len(s)))+4*len(s) {
			e.writePackedInt32Slice(s)
			return nil
		}
		e.emitQPackInt64(w, codec, mn, forBits, first, minDelta, deltaBits, pforBits)
		return nil
	}
	e.WriteArrayHeader(len(s))
	for i := range s {
		e.WriteInt(int64(s[i]))
	}
	return nil
}

// readQPackUint64 consumes a QPack uint64 codec body whose tag t was peeked.
// The caller must have already peeked (not consumed) the tag; this function
// increments d.i to consume it before reading the payload.
func (d *Decoder) readQPackUint64(t byte) ([]uint64, error) {
	d.i++
	switch t {
	case tagPackRaw:
		return d.readPackedUint64Slice()
	case tagPackFor:
		return d.readPackedForUint64Slice()
	case tagPackDeltaFor:
		return d.readPackedDeltaForUint64Slice()
	case tagPackRLE:
		return d.readPackedRLEUint64Slice()
	case tagPackDict:
		return d.readPackedDictUint64Slice()
	case tagPackPFor:
		return d.readPackedPForUint64Slice()
	}
	return nil, ErrBadTag
}

// readQPackInt64 consumes a QPack int64 codec body whose tag t was peeked.
// The caller must have already peeked (not consumed) the tag; this function
// increments d.i to consume it before reading the payload.
func (d *Decoder) readQPackInt64(t byte) ([]int64, error) {
	d.i++
	switch t {
	case tagPackRaw:
		return d.readPackedInt64Slice()
	case tagPackFor:
		return d.readPackedForInt64Slice()
	case tagPackDeltaFor:
		return d.readPackedDeltaForInt64Slice()
	case tagPackRLE:
		return d.readPackedRLEInt64Slice()
	case tagPackDict:
		return d.readPackedDictInt64Slice()
	case tagPackPFor:
		return d.readPackedPForInt64Slice()
	}
	return nil, ErrBadTag
}

func decodeSliceInt32(d *Decoder, p unsafe.Pointer) error {
	t, err := d.peekTag()
	if err != nil {
		return err
	}
	switch t {
	case tagPackRaw:
		// Could be qpackKindInt32 (legacy raw) or qpackKindInt64 (new QPack path).
		// Peek the kind byte (one position ahead of the tag) to decide.
		if d.i+1 < len(d.buf) && d.buf[d.i+1] == qpackKindInt64 {
			v64, err := d.readQPackInt64(t)
			if err != nil {
				return err
			}
			out := make([]int32, len(v64))
			for i, x := range v64 {
				out[i] = int32(x)
			}
			*(*[]int32)(p) = out
			return nil
		}
		d.i++
		v, err := d.readPackedInt32Slice()
		if err != nil {
			return err
		}
		*(*[]int32)(p) = v
		return nil
	case tagPackFor, tagPackDeltaFor, tagPackRLE, tagPackDict, tagPackPFor:
		v64, err := d.readQPackInt64(t)
		if err != nil {
			return err
		}
		out := make([]int32, len(v64))
		for i, x := range v64 {
			out[i] = int32(x)
		}
		*(*[]int32)(p) = out
		return nil
	}
	n, err := d.ReadArrayHeader()
	if err != nil {
		return err
	}
	if err := d.CheckLength(n, 1); err != nil {
		return err
	}
	out := make([]int32, n)
	for i := range n {
		v, err := d.ReadInt()
		if err != nil {
			return err
		}
		out[i] = int32(v)
	}
	*(*[]int32)(p) = out
	return nil
}
func encodeSliceInt64(e *Encoder, p unsafe.Pointer) error {
	s := *(*[]int64)(p)
	if e.qpack {
		e.writeQPackInt64(s)
		return nil
	}
	e.WriteArrayHeader(len(s))
	for i := range s {
		e.WriteInt(s[i])
	}
	return nil
}
func decodeSliceInt64(d *Decoder, p unsafe.Pointer) error {
	t, err := d.peekTag()
	if err != nil {
		return err
	}
	switch t {
	case tagPackRaw:
		d.i++
		v, err := d.readPackedInt64Slice()
		if err != nil {
			return err
		}
		*(*[]int64)(p) = v
		return nil
	case tagPackFor:
		d.i++
		v, err := d.readPackedForInt64Slice()
		if err != nil {
			return err
		}
		*(*[]int64)(p) = v
		return nil
	case tagPackDeltaFor:
		d.i++
		v, err := d.readPackedDeltaForInt64Slice()
		if err != nil {
			return err
		}
		*(*[]int64)(p) = v
		return nil
	case tagPackRLE:
		d.i++
		v, err := d.readPackedRLEInt64Slice()
		if err != nil {
			return err
		}
		*(*[]int64)(p) = v
		return nil
	case tagPackDict:
		d.i++
		v, err := d.readPackedDictInt64Slice()
		if err != nil {
			return err
		}
		*(*[]int64)(p) = v
		return nil
	case tagPackPFor:
		d.i++
		v, err := d.readPackedPForInt64Slice()
		if err != nil {
			return err
		}
		*(*[]int64)(p) = v
		return nil
	}
	n, err := d.ReadArrayHeader()
	if err != nil {
		return err
	}
	if err := d.CheckLength(n, 1); err != nil {
		return err
	}
	out := make([]int64, n)
	for i := range n {
		v, err := d.ReadInt()
		if err != nil {
			return err
		}
		out[i] = v
	}
	*(*[]int64)(p) = out
	return nil
}
func encodeSliceUint32(e *Encoder, p unsafe.Pointer) error {
	s := *(*[]uint32)(p)
	if e.qpack {
		w := e.widenU64(s)
		codec, mn, forBits, first, minDelta, deltaBits, pforBits, bestCost := pickU64Codec(w)
		// Never-worse floor: the picker scores codecs against the uint64-raw
		// 8 B/elem baseline, but the native form for a uint32 is 4 B/elem. Emit
		// the widened codec only when its cost beats native uint32-raw; otherwise
		// fall back to native raw so incompressible 32-bit data is never inflated.
		if bestCost >= 2+uvarintLen(uint64(len(s)))+4*len(s) {
			e.writePackedUint32Slice(s)
			return nil
		}
		e.emitQPackUint64(w, codec, mn, forBits, first, minDelta, deltaBits, pforBits)
		return nil
	}
	e.WriteArrayHeader(len(s))
	for i := range s {
		e.WriteUint(uint64(s[i]))
	}
	return nil
}
func decodeSliceUint32(d *Decoder, p unsafe.Pointer) error {
	t, err := d.peekTag()
	if err != nil {
		return err
	}
	switch t {
	case tagPackRaw:
		// Could be qpackKindUint32 (legacy raw) or qpackKindUint64 (new QPack path).
		// Peek the kind byte (one position ahead of the tag) to decide.
		if d.i+1 < len(d.buf) && d.buf[d.i+1] == qpackKindUint64 {
			v64, err := d.readQPackUint64(t)
			if err != nil {
				return err
			}
			out := make([]uint32, len(v64))
			for i, x := range v64 {
				out[i] = uint32(x)
			}
			*(*[]uint32)(p) = out
			return nil
		}
		d.i++
		v, err := d.readPackedUint32Slice()
		if err != nil {
			return err
		}
		*(*[]uint32)(p) = v
		return nil
	case tagPackFor, tagPackDeltaFor, tagPackRLE, tagPackDict, tagPackPFor:
		v64, err := d.readQPackUint64(t)
		if err != nil {
			return err
		}
		out := make([]uint32, len(v64))
		for i, x := range v64 {
			out[i] = uint32(x)
		}
		*(*[]uint32)(p) = out
		return nil
	}
	n, err := d.ReadArrayHeader()
	if err != nil {
		return err
	}
	if err := d.CheckLength(n, 1); err != nil {
		return err
	}
	out := make([]uint32, n)
	for i := range n {
		v, err := d.ReadUint()
		if err != nil {
			return err
		}
		out[i] = uint32(v)
	}
	*(*[]uint32)(p) = out
	return nil
}
func encodeSliceUint64(e *Encoder, p unsafe.Pointer) error {
	s := *(*[]uint64)(p)
	if e.qpack {
		e.writeQPackUint64(s)
		return nil
	}
	e.WriteArrayHeader(len(s))
	for i := range s {
		e.WriteUint(s[i])
	}
	return nil
}
func decodeSliceUint64(d *Decoder, p unsafe.Pointer) error {
	t, err := d.peekTag()
	if err != nil {
		return err
	}
	switch t {
	case tagPackRaw:
		d.i++
		v, err := d.readPackedUint64Slice()
		if err != nil {
			return err
		}
		*(*[]uint64)(p) = v
		return nil
	case tagPackFor:
		d.i++
		v, err := d.readPackedForUint64Slice()
		if err != nil {
			return err
		}
		*(*[]uint64)(p) = v
		return nil
	case tagPackDeltaFor:
		d.i++
		v, err := d.readPackedDeltaForUint64Slice()
		if err != nil {
			return err
		}
		*(*[]uint64)(p) = v
		return nil
	case tagPackRLE:
		d.i++
		v, err := d.readPackedRLEUint64Slice()
		if err != nil {
			return err
		}
		*(*[]uint64)(p) = v
		return nil
	case tagPackDict:
		d.i++
		v, err := d.readPackedDictUint64Slice()
		if err != nil {
			return err
		}
		*(*[]uint64)(p) = v
		return nil
	case tagPackPFor:
		d.i++
		v, err := d.readPackedPForUint64Slice()
		if err != nil {
			return err
		}
		*(*[]uint64)(p) = v
		return nil
	}
	n, err := d.ReadArrayHeader()
	if err != nil {
		return err
	}
	if err := d.CheckLength(n, 1); err != nil {
		return err
	}
	out := make([]uint64, n)
	for i := range n {
		v, err := d.ReadUint()
		if err != nil {
			return err
		}
		out[i] = v
	}
	*(*[]uint64)(p) = out
	return nil
}

// ----- []float32 / []float64 -----

func encodeSliceFloat32(e *Encoder, p unsafe.Pointer) error {
	s := *(*[]float32)(p)
	if e.qpack {
		// Mirror encodeSliceFloat64's Gorilla branch (float32 has no ALP). The
		// projection from pickF32Codec is only a hint, so emit Gorilla for real,
		// measure it, and keep it solely when it actually beat raw — a true
		// never-larger gate. The rollback re-emits raw only on the rare lose case,
		// so the common smooth-data path encodes Gorilla once.
		if e.gorillaFloat {
			if gorCodec, _ := pickF32Codec(s); gorCodec == qpackGorilla {
				rawExact := 2 + uvarintLen(uint64(len(s))) + len(s)*4
				start := len(e.buf)
				hdrBefore, flagBefore := e.headerOut, e.headerFlagAt
				e.writePackedGorillaFloat32Slice(s)
				if len(e.buf)-start < rawExact {
					return nil
				}
				// Gorilla did not win — roll back. writePackedGorilla* may have
				// emitted the stream header on a top-level first write; truncating to
				// start drops it, and writeHeader's headerOut latch would then
				// suppress the raw fallback's header and produce a headerless,
				// undecodable stream. Restore the pre-attempt header state so the raw
				// fallback re-emits the header when it was rolled away.
				e.buf = e.buf[:start]
				e.headerOut, e.headerFlagAt = hdrBefore, flagBefore
			}
		}
		e.writePackedFloat32Slice(s)
		return nil
	}
	return encodeSliceFloat32Impl(e, s)
}
func decodeSliceFloat32(d *Decoder, p unsafe.Pointer) error {
	t, err := d.peekTag()
	if err != nil {
		return err
	}
	if t == tagPackRaw {
		d.i++
		v, err := d.readPackedFloat32Slice()
		if err != nil {
			return err
		}
		*(*[]float32)(p) = v
		return nil
	}
	if t == tagPackGorilla {
		d.i++
		v, err := d.readPackedGorillaFloat32Slice()
		if err != nil {
			return err
		}
		*(*[]float32)(p) = v
		return nil
	}
	n, err := d.ReadArrayHeader()
	if err != nil {
		return err
	}
	if err := d.CheckLength(n, 1); err != nil {
		return err
	}
	out := make([]float32, n)
	for i := range n {
		v, err := d.ReadFloat32()
		if err != nil {
			return err
		}
		out[i] = v
	}
	*(*[]float32)(p) = out
	return nil
}
func encodeSliceFloat64(e *Encoder, p unsafe.Pointer) error {
	s := *(*[]float64)(p)
	if e.qpack {
		// Under OptCompression both Gorilla and ALP are enabled. Pick the
		// smallest of {raw-LE, Gorilla projection, ALP estimate}. ALP's
		// estimate is a conservative upper bound, so it is chosen only when it
		// strictly beats both alternatives — pure-smooth floats keep Gorilla,
		// and nothing grows the wire.
		if e.gorillaFloat {
			// Exact raw size (tag + kind + uvarint(n) + 8n), matching what
			// writePackedFloat64Slice emits. A looser fixed estimate (e.g.
			// 12+8n) over-counts raw by up to ~10 bytes and could keep Gorilla
			// when it is marginally LARGER than raw; the exact figure makes the
			// never-larger gate tight, mirroring the float32 path.
			rawEst := 2 + uvarintLen(uint64(len(s))) + len(s)*8
			plan, alpEst, alpOK := alpPlanFloat64(s) // ALP estimate is a safe upper bound
			alpWins := alpOK && alpEst < rawEst
			// pickF64Codec only projects Gorilla from a sample prefix, which can
			// be wildly optimistic on a smooth-prefix/high-entropy-tail slice.
			// Emit Gorilla for real and measure it; keep it only when it is
			// actually smaller than raw (and than ALP) — a true never-larger
			// gate. The rollback re-emits raw/ALP only on the rare lose case, so
			// the common smooth-data path still encodes Gorilla once.
			if gorCodec, _ := pickF64Codec(s); gorCodec == qpackGorilla {
				start := len(e.buf)
				hdrBefore, flagBefore := e.headerOut, e.headerFlagAt
				e.writePackedGorillaFloat64Slice(s)
				gorActual := len(e.buf) - start
				if gorActual < rawEst && (!alpWins || alpEst >= gorActual) {
					return nil
				}
				// Gorilla did not win — roll back. writePackedGorilla* may have
				// emitted the stream header (top-level first write); truncating to
				// `start` drops those bytes, but writeHeader's headerOut latch would
				// then suppress the fallback's header and produce a headerless,
				// undecodable stream. Restore the pre-attempt header state so the
				// raw/ALP fallback re-emits the header when it was rolled away.
				e.buf = e.buf[:start]
				e.headerOut, e.headerFlagAt = hdrBefore, flagBefore
			}
			if alpWins {
				e.writePackedALPFloat64Slice(s, plan)
				return nil
			}
		}
		e.writePackedFloat64Slice(s)
		return nil
	}
	return encodeSliceFloat64Impl(e, s)
}
func decodeSliceFloat64(d *Decoder, p unsafe.Pointer) error {
	t, err := d.peekTag()
	if err != nil {
		return err
	}
	if t == tagPackRaw {
		d.i++
		v, err := d.readPackedFloat64Slice()
		if err != nil {
			return err
		}
		*(*[]float64)(p) = v
		return nil
	}
	if t == tagPackGorilla {
		d.i++
		v, err := d.readPackedGorillaFloat64Slice()
		if err != nil {
			return err
		}
		*(*[]float64)(p) = v
		return nil
	}
	if t == tagPackALP {
		d.i++
		v, err := d.readPackedALPFloat64Slice()
		if err != nil {
			return err
		}
		*(*[]float64)(p) = v
		return nil
	}
	n, err := d.ReadArrayHeader()
	if err != nil {
		return err
	}
	if err := d.CheckLength(n, 1); err != nil {
		return err
	}
	out := make([]float64, n)
	for i := range n {
		v, err := d.ReadFloat64()
		if err != nil {
			return err
		}
		out[i] = v
	}
	*(*[]float64)(p) = out
	return nil
}

// ----- []bool -----

func encodeSliceBool(e *Encoder, p unsafe.Pointer) error {
	s := *(*[]bool)(p)
	if e.qpack {
		e.writePackedBool(s)
		return nil
	}
	e.WriteArrayHeader(len(s))
	for i := range s {
		e.WriteBool(s[i])
	}
	return nil
}
func decodeSliceBool(d *Decoder, p unsafe.Pointer) error {
	t, err := d.peekTag()
	if err != nil {
		return err
	}
	if t == tagPackBool {
		d.i++
		out, err := d.readPackedBool()
		if err != nil {
			return err
		}
		*(*[]bool)(p) = out
		return nil
	}
	n, err := d.ReadArrayHeader()
	if err != nil {
		return err
	}
	if err := d.CheckLength(n, 1); err != nil {
		return err
	}
	out := make([]bool, n)
	for i := range n {
		v, err := d.ReadBool()
		if err != nil {
			return err
		}
		out[i] = v
	}
	*(*[]bool)(p) = out
	return nil
}
