package qdf

import "math"

// Skip advances past one value without materializing it.
//
// Skip recurses through nested arrays and maps, so it bounds nesting depth via
// descend/ascend exactly like the reflect decode path: an unknown struct field
// whose value is a deeply-nested array is skipped here, and without this guard a
// hostile payload of N nested arrays would overflow the goroutine stack (an
// unrecoverable fatal error). The recursive d.Skip() calls below re-enter this
// guard, so every level is counted.
func (d *Decoder) Skip() error {
	if err := d.descend(); err != nil {
		return err
	}
	defer d.ascend()
	t, err := d.peekTag()
	if err != nil {
		return err
	}
	switch {
	case t <= tagFixintMax:
		d.i++
		return nil
	case t >= tagFixstr && t <= tagFixstr|tagFixstrMask:
		n := int(t & tagFixstrMask)
		if d.i+1+n > len(d.buf) {
			return ErrShortBuffer
		}
		d.i += 1 + n
		return nil
	case t >= tagFixarr && t <= tagFixarr|tagFixarrMask:
		n := int(t & tagFixarrMask)
		d.i++
		for range n {
			if err := d.Skip(); err != nil {
				return err
			}
		}
		return nil
	case t >= tagNegfixint && t <= tagNegfixint|tagNegfixintMask:
		d.i++
		return nil
	}
	switch t {
	case tagNil, tagTrue, tagFalse:
		d.i++
		return nil
	case tagUint8, tagInt8:
		if d.i+2 > len(d.buf) {
			return ErrShortBuffer
		}
		d.i += 2
		return nil
	case tagUint16, tagInt16:
		if d.i+3 > len(d.buf) {
			return ErrShortBuffer
		}
		d.i += 3
		return nil
	case tagUint32, tagInt32, tagFloat32:
		if d.i+5 > len(d.buf) {
			return ErrShortBuffer
		}
		d.i += 5
		return nil
	case tagUint64, tagInt64, tagFloat64:
		if d.i+9 > len(d.buf) {
			return ErrShortBuffer
		}
		d.i += 9
		return nil
	case tagTimestamp:
		// New wire format: tag + uvarint(zigzag(sec)) + uvarint(nsec).
		// Skip two uvarints.
		d.i++ // consume tag
		for range 2 {
			_, n := readUvarint(d.buf[d.i:])
			if n <= 0 {
				return ErrShortBuffer
			}
			d.i += n
		}
		return nil
	case tagStr8, tagBin8, tagMap8:
		// Map8 is a count, not a byte length; handle separately below.
		if t == tagMap8 {
			n, err := d.ReadMapHeader()
			if err != nil {
				return err
			}
			for range n {
				if err := d.Skip(); err != nil {
					return err
				}
				if err := d.Skip(); err != nil {
					return err
				}
			}
			return nil
		}
		d.i++ // tag
		if d.i >= len(d.buf) {
			return ErrShortBuffer
		}
		n := int(d.buf[d.i])
		d.i++
		if d.i+n > len(d.buf) {
			return ErrShortBuffer
		}
		d.i += n
		return nil
	case tagStr16, tagBin16, tagArr16, tagMap16:
		d.i++
		if t == tagArr16 {
			if d.i+2 > len(d.buf) {
				return ErrShortBuffer
			}
			n := int(readU16(d.buf[d.i:]))
			d.i += 2
			for range n {
				if err := d.Skip(); err != nil {
					return err
				}
			}
			return nil
		}
		if t == tagMap16 {
			if d.i+2 > len(d.buf) {
				return ErrShortBuffer
			}
			n := int(readU16(d.buf[d.i:]))
			d.i += 2
			for range n {
				if err := d.Skip(); err != nil {
					return err
				}
				if err := d.Skip(); err != nil {
					return err
				}
			}
			return nil
		}
		if d.i+2 > len(d.buf) {
			return ErrShortBuffer
		}
		n := int(readU16(d.buf[d.i:]))
		d.i += 2
		if d.i+n > len(d.buf) {
			return ErrShortBuffer
		}
		d.i += n
		return nil
	case tagStr32, tagBin32, tagArr32, tagMap32:
		d.i++
		if t == tagArr32 {
			if d.i+4 > len(d.buf) {
				return ErrShortBuffer
			}
			// Compare in uint64 before narrowing: on a 32-bit int a count near 2^32
			// wraps negative, making `for range n` run zero iterations and Skip
			// silently succeed without consuming the elements (parse desync). Each
			// element is >= 1 byte, so a count exceeding the remaining bytes is
			// impossible. Mirrors the tagStr32/tagBin32 guard below.
			n64 := uint64(readU32(d.buf[d.i:]))
			d.i += 4
			if n64 > uint64(len(d.buf)-d.i) {
				return ErrShortBuffer
			}
			for range int(n64) {
				if err := d.Skip(); err != nil {
					return err
				}
			}
			return nil
		}
		if t == tagMap32 {
			if d.i+4 > len(d.buf) {
				return ErrShortBuffer
			}
			// Same 32-bit wrap guard as tagArr32; each entry is >= 2 bytes (key +
			// value, each >= 1), so a count exceeding the remaining bytes is invalid.
			n64 := uint64(readU32(d.buf[d.i:]))
			d.i += 4
			if n64 > uint64(len(d.buf)-d.i) {
				return ErrShortBuffer
			}
			for range int(n64) {
				if err := d.Skip(); err != nil {
					return err
				}
				if err := d.Skip(); err != nil {
					return err
				}
			}
			return nil
		}
		if d.i+4 > len(d.buf) {
			return ErrShortBuffer
		}
		n64 := uint64(readU32(d.buf[d.i:]))
		d.i += 4
		// Compare in uint64 before narrowing: on a 32-bit int a length near 2^32
		// becomes negative, slips past `d.i+n > len`, and `d.i += n` rewinds the
		// cursor (parse desync). Mirrors the read-path tagStr32/tagBin32 guard.
		if n64 > uint64(len(d.buf)-d.i) {
			return ErrShortBuffer
		}
		d.i += int(n64)
		return nil
	case tagInternStr, tagInternBin:
		// Read+register; Skip semantics still need the state table to stay
		// in sync with the stream.
		_, err := d.readStringBytes()
		return err
	case tagStateRef, tagStateRepeat, tagStateMTF, tagStatePair:
		_, err := d.readStringBytes()
		return err
	case tagMapShape:
		// Wire form mirrors the decode path: either a declaration
		// (shapeID==0, varuint(N), N keys, N values) or a reuse
		// (shapeID>0, looked-up N, N values). Skipping must still
		// advance the shape table on declaration so subsequent
		// references stay consistent.
		d.i++
		shapeID, n := readUvarint(d.buf[d.i:])
		if n <= 0 {
			return ErrInvalidLength
		}
		if shapeID > uint64(math.MaxUint32) {
			return ErrUnknownStateID // would truncate on the uint32 cast below
		}
		d.i += n
		if d.state == nil {
			d.state = newDecState()
		}
		var cnt int
		if shapeID == 0 {
			cnt64, n := readUvarint(d.buf[d.i:])
			if n <= 0 {
				return ErrInvalidLength
			}
			d.i += n
			if cnt64 > uint64(math.MaxInt) { // 32-bit: int(cnt64) would wrap before CheckLength
				return ErrInvalidLength
			}
			if err := d.CheckLength(int(cnt64), 1); err != nil {
				return err
			}
			cnt = int(cnt64)
			sh := d.state.shapeDeclare()
			sh.names = make([]string, 0, cnt)
			for i := 0; i < cnt; i++ {
				kb, err := d.readStringBytes()
				if err != nil {
					return err
				}
				sh.names = append(sh.names, d.keyCache.Make(kb))
			}
		} else {
			sh := d.state.shapeLookup(uint32(shapeID))
			if sh == nil {
				return ErrUnknownStateID
			}
			cnt = len(sh.names)
		}
		for i := 0; i < cnt; i++ {
			if err := d.Skip(); err != nil {
				return err
			}
		}
		return nil
	case tagPackBool:
		d.i++
		n64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		// Validate the element count in uint64 BEFORE the signed cast so
		// a hostile varuint cannot drive nBytes negative and corrupt
		// d.i. n64 elements need ceil(n64/8) bytes.
		rem := uint64(len(d.buf) - d.i)
		if n64 > rem*8 {
			return ErrShortBuffer
		}
		d.i += int((n64 + 7) >> 3)
		return nil
	case tagPackRaw:
		d.i++
		if d.i >= len(d.buf) {
			return ErrShortBuffer
		}
		k := d.buf[d.i]
		d.i++
		w := qpackRawWidthBytes(k)
		if w == 0 {
			return ErrBadTag
		}
		n64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		if n64 > uint64(len(d.buf)-d.i)/uint64(w) {
			return ErrShortBuffer
		}
		d.i += int(n64) * w
		return nil
	case tagPackFor:
		d.i++
		if d.i+2 > len(d.buf) {
			return ErrShortBuffer
		}
		// kind, bits
		d.i++ // skip kind
		bitsPer := int(d.buf[d.i])
		d.i++
		if bitsPer > qpackForMaxBits {
			return ErrBadTag
		}
		// min varuint
		_, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		n64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		// Same overflow guard as tagPackBool. n64 elements times
		// bitsPer bits => ceil/8 bytes; validate in uint64 to avoid the
		// sign-bit pitfall on a hostile varuint.
		rem := uint64(len(d.buf) - d.i)
		if bitsPer > 0 && n64 > rem*8/uint64(bitsPer) {
			return ErrShortBuffer
		}
		d.i += int((n64*uint64(bitsPer) + 7) / 8)
		return nil
	case tagPackGorilla:
		d.i++
		if d.i >= len(d.buf) {
			return ErrShortBuffer
		}
		k := d.buf[d.i]
		d.i++
		w := qpackRawWidthBytes(k)
		if w != 4 && w != 8 {
			return ErrBadTag
		}
		n64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		if n64 == 0 {
			return nil
		}
		if d.i+w > len(d.buf) {
			return ErrShortBuffer
		}
		d.i += w
		// numBits varuint
		nb64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		rem := uint64(len(d.buf) - d.i)
		if nb64 > rem*8 {
			return ErrShortBuffer
		}
		d.i += int((nb64 + 7) >> 3)
		return nil
	case tagPackALP:
		d.i++
		if d.i >= len(d.buf) {
			return ErrShortBuffer
		}
		excValBytes := 8 // float64 exception value width
		switch d.buf[d.i] {
		case qpackKindFloat64:
		case qpackKindFloat32:
			excValBytes = 4
		case qpackKindALPRD64:
			return d.skipALPRD()
		default:
			return ErrTypeMismatch
		}
		d.i++ // kind
		n64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		if n64 == 0 {
			return nil
		}
		if n64 > alpMaxElems {
			return ErrInvalidLength
		}
		if d.i >= len(d.buf) {
			return ErrShortBuffer
		}
		d.i++                                           // d exponent
		if _, nr := readUvarint(d.buf[d.i:]); nr <= 0 { // forMin
			return ErrInvalidLength
		} else {
			d.i += nr
		}
		if d.i >= len(d.buf) {
			return ErrShortBuffer
		}
		width := int(d.buf[d.i])
		d.i++
		if width > qpackForMaxBits {
			return ErrBadTag
		}
		if width > 0 {
			rem := uint64(len(d.buf) - d.i)
			if n64 > rem*8/uint64(width) {
				return ErrShortBuffer
			}
			d.i += int((n64*uint64(width) + 7) / 8)
		}
		excN, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		if excN > n64 {
			return ErrInvalidLength
		}
		for range excN {
			_, nr := readUvarint(d.buf[d.i:])
			if nr <= 0 {
				return ErrInvalidLength
			}
			d.i += nr
			if d.i+excValBytes > len(d.buf) {
				return ErrShortBuffer
			}
			d.i += excValBytes
		}
		return nil
	case tagPackDeltaFor:
		d.i++
		if d.i+2 > len(d.buf) {
			return ErrShortBuffer
		}
		d.i++ // skip kind
		bitsPer := int(d.buf[d.i])
		d.i++
		if bitsPer > qpackForMaxBits {
			return ErrBadTag
		}
		// firstVal varuint
		_, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		// minDelta varuint
		_, nr = readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		n64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		if n64 >= 2 {
			// Body holds (n-1) elements at bitsPer each. Bound (n-1) by the
			// remaining bytes via DIVISION before the multiply — n64 is an
			// unbounded varuint, so `(n64-1)*bitsPer` would overflow uint64 for a
			// large n64 (wrapping to a small bodyBytes that passes the size check
			// and desyncs the skip cursor). Mirrors readPackedDeltaForHeader.
			rem := uint64(len(d.buf) - d.i)
			if bitsPer > 0 && n64-1 > rem*8/uint64(bitsPer) {
				return ErrShortBuffer
			}
			bodyBytes := ((n64-1)*uint64(bitsPer) + 7) >> 3
			d.i += int(bodyBytes)
		}
		return nil
	case tagPackRLE:
		d.i++
		if d.i >= len(d.buf) {
			return ErrShortBuffer
		}
		k := d.buf[d.i]
		d.i++
		if k != qpackKindUint64 && k != qpackKindInt64 {
			return ErrBadTag
		}
		n64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		// Body is a sequence of (value-varuint, runLen-varuint)
		// pairs whose runLen sum equals n. Walk until n elements
		// consumed; the loop also catches truncated bodies.
		var produced uint64
		for produced < n64 {
			_, nr := readUvarint(d.buf[d.i:])
			if nr <= 0 {
				return ErrInvalidLength
			}
			d.i += nr
			runLen, nr := readUvarint(d.buf[d.i:])
			if nr <= 0 {
				return ErrInvalidLength
			}
			d.i += nr
			if runLen == 0 || produced+runLen > n64 {
				return ErrInvalidLength
			}
			produced += runLen
		}
		return nil
	case tagPackDict:
		d.i++
		if d.i >= len(d.buf) {
			return ErrShortBuffer
		}
		k := d.buf[d.i]
		d.i++
		if k != qpackKindUint64 && k != qpackKindInt64 {
			return ErrBadTag
		}
		distinct64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		if distinct64 == 0 || distinct64 > qpackDictMaxDistinct {
			return ErrBadTag
		}
		for range distinct64 {
			_, nr := readUvarint(d.buf[d.i:])
			if nr <= 0 {
				return ErrInvalidLength
			}
			d.i += nr
		}
		n64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		bp := bitsForDistinct(int(distinct64))
		if bp == 0 {
			return nil
		}
		rem := uint64(len(d.buf) - d.i)
		if n64 > rem*8/uint64(bp) {
			return ErrShortBuffer
		}
		d.i += int((n64*uint64(bp) + 7) / 8)
		return nil
	case tagPackPFor:
		d.i++
		if d.i >= len(d.buf) {
			return ErrShortBuffer
		}
		k := d.buf[d.i]
		d.i++
		if k != qpackKindUint64 && k != qpackKindInt64 {
			return ErrBadTag
		}
		n64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		if d.i >= len(d.buf) {
			return ErrShortBuffer
		}
		b := int(d.buf[d.i])
		d.i++
		if b > qpackForMaxBits {
			return ErrBadTag
		}
		// min varuint
		if _, nr := readUvarint(d.buf[d.i:]); nr <= 0 {
			return ErrInvalidLength
		} else {
			d.i += nr
		}
		// body: n*b bits
		rem := uint64(len(d.buf) - d.i)
		if b > 0 && n64 > rem*8/uint64(b) {
			return ErrShortBuffer
		}
		d.i += int((n64*uint64(b) + 7) / 8)
		// exception list: excN pairs of (dPos, delta) varuints
		excN64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		if excN64 > n64 {
			return ErrInvalidLength
		}
		for range excN64 {
			if _, nr := readUvarint(d.buf[d.i:]); nr <= 0 {
				return ErrInvalidLength
			} else {
				d.i += nr
			}
			if _, nr := readUvarint(d.buf[d.i:]); nr <= 0 {
				return ErrInvalidLength
			} else {
				d.i += nr
			}
		}
		return nil
	case tagColStruct:
		// A columnar []struct payload reached Skip — an unknown struct-slice
		// field under OptBalanced/OptCompression (schema evolution). Decode it
		// via the any path and discard the result: that advances the cursor
		// exactly and replays the shape-table (and any per-column) state a real
		// decode would, keeping later state-refs in sync — a byte-only skip
		// could not. Uses the map-path row ceiling (maxColumnarAnyElems); a
		// skipped columnar field with more rows than that is rejected rather
		// than skipped, which is far above any realistic schema-evolution batch.
		if _, err := decodeColumnarAny(d); err != nil {
			return err
		}
		return nil
	case tagHybridColStruct:
		// Hybrid columnar []struct (unknown mixed-struct-slice field — emitted
		// under OptFSST/OptCompression, or under Balanced when the intern-aware
		// probe predicts a win). Same rationale as tagColStruct: decode-and-
		// discard via the any path to advance the cursor and replay the intern /
		// shape state, keeping later state-refs in sync.
		if _, err := decodeHybridColumnarAny(d); err != nil {
			return err
		}
		return nil
	case tagPackBlock:
		// Block-coded int/uint column reached Skip (an unknown []int*/[]uint*
		// field under OptBalanced — writeQPackInt64 emits 0xF0 at top level).
		// Decode-and-discard via the real reader to advance the cursor exactly;
		// the kind byte (after the tag) selects int vs uint.
		d.i++
		if d.i >= len(d.buf) {
			return ErrShortBuffer
		}
		switch d.buf[d.i] {
		case blockKindInt:
			_, err := d.readBlockInt64()
			return err
		case blockKindUint:
			_, err := d.readBlockUint64()
			return err
		}
		return ErrBadTag
	case tagZoneChunk:
		// Zone-chunked column reached Skip. Defensive: today 0xF1 is only emitted
		// inside tagColStruct columns, but the top-level slice decoders accept it,
		// so a future top-level emission must skip cleanly too. Decode-and-discard
		// via the kind-selected reader.
		d.i++
		if d.i >= len(d.buf) {
			return ErrShortBuffer
		}
		switch d.buf[d.i] {
		case zoneKindInt:
			_, err := d.readZoneChunkInt64()
			return err
		case zoneKindUint:
			_, err := d.readZoneChunkUint64()
			return err
		case zoneKindFloat:
			_, err := d.readZoneChunkFloat64()
			return err
		}
		return ErrBadTag
	case tagColVecLossy:
		// Lossy float-vector column reached Skip (an unknown []float32/[]float64
		// field under OptLossyVec — 0xFD at top level). The block self-describes
		// its element kind and length; advance by the consumed byte count.
		_, _, used, err := readLossyVec(d.buf[d.i:])
		if err != nil {
			return err
		}
		d.i += used
		return nil
	case tagVecBatchStruct:
		// Batched lossy vector-column ([]struct with vector fields under
		// OptLossyVec) reached Skip. Not byte-skippable: the per-row non-batched
		// fields can carry intern/shape state-refs, so each must be walked via
		// d.Skip() to replay that state (a raw byte advance would desync later
		// state-refs — same rationale as tagColStruct). The header carries the
		// struct field count so the schemaless walk knows how many fields per row.
		d.i++
		n64, k := readUvarint(d.buf[d.i:])
		if k <= 0 {
			return ErrInvalidLength
		}
		d.i += k
		if n64 > uint64(maxColumnarElems) {
			return ErrInvalidLength
		}
		n := int(n64)
		if d.i+3 > len(d.buf) {
			return ErrShortBuffer
		}
		nv := int(d.buf[d.i])
		mask := d.buf[d.i+1]
		polarMask := d.buf[d.i+2]
		d.i += 3
		if nv > 8 || polarMask&^mask != 0 { // nv fits one mask byte; polar ⊆ mask
			return ErrBadTag
		}
		nf64, kf := readUvarint(d.buf[d.i:])
		if kf <= 0 {
			return ErrInvalidLength
		}
		d.i += kf
		// Skip each batched column block (each is self-describing in length).
		for vi := range nv {
			if mask&(1<<uint(vi)) == 0 {
				continue
			}
			if polarMask&(1<<uint(vi)) != 0 {
				_, used, err := readNormStream(d.buf[d.i:], n)
				if err != nil {
					return err
				}
				d.i += used
			}
			_, _, used, err := readLossyVec(d.buf[d.i:])
			if err != nil {
				return err
			}
			d.i += used
		}
		// Skip per-row non-batched fields, replaying their intern/shape state.
		batched := 0
		for m := mask; m != 0; m &= m - 1 {
			batched++
		}
		perRow := int(nf64) - batched
		if nf64 > uint64(maxColumnarElems) || perRow < 0 {
			return ErrBadTag
		}
		for range n {
			for range perRow {
				if err := d.Skip(); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return ErrBadTag
}
