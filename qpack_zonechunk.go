package qdf

import "encoding/binary"

// Zone-chunked integer column with a min/max zonemap (wire tag tagZoneChunk,
// 0xF1, OptZoneMap).
//
// The column is split into fixed 256-row zones (same chunk size as the block
// codec, so the per-zone codec picks are shared via e.blkPlanI64). Each zone is
// an independent QPack int sub-slice; a uint32 offset table and a per-zone
// [min,max] zonemap precede the bodies. A bound-carrying predicate
// (WhereRange/GE/LE/Eq) skips zones whose [min,max] cannot intersect its bounds
// WITHOUT decoding them — the whole point of the feature. See decodeZoneSkip* in
// columnar.go for the query-side use.
//
// Unlike the block codec this is an explicit size-for-query-speed trade: it is
// emitted only under OptZoneMap, never auto, and carries no never-larger gate
// (chunking an ordered column costs more than a single whole-column codec — that
// is the price of zone-skippability). Without OptZoneMap the wire is unchanged.

const (
	zoneKindInt  = 0x00
	zoneKindUint = 0x01
	// zoneChunkMinLen: a column shorter than two zones has nothing to chunk.
	zoneChunkMinLen = 2 * blockSizeSmall // reuse the block codec's 256-row zone
)

// ---- encode (int64) ----

func (e *Encoder) writeZoneChunkInt64(s []int64) {
	e.writeHeader()
	n := len(s)
	zoneCount := (n + blockSizeSmall - 1) / blockSizeSmall
	e.planBlocksI64(s, blockSizeSmall) // cache per-zone codec picks (no re-pick on emit)
	plans := e.blkPlanI64[:zoneCount]

	e.buf = append(e.buf, tagZoneChunk, zoneKindInt, blkLogSmall)
	e.buf = appendUvarint(e.buf, uint64(n))
	offAt := len(e.buf)
	e.buf = append(e.buf, make([]byte, 4*zoneCount)...)
	// zonemap: per-zone min,max as zigzag-varint (variable length), before bodies.
	for i := 0; i < n; i += blockSizeSmall {
		j := min(i+blockSizeSmall, n)
		mn, mx := minMaxI64(s[i:j])
		e.buf = appendUvarint(e.buf, zigzagEncode64(mn))
		e.buf = appendUvarint(e.buf, zigzagEncode64(mx))
	}
	bodyStart := len(e.buf)
	zi := 0
	for i := 0; i < n; i += blockSizeSmall {
		j := min(i+blockSizeSmall, n)
		binary.LittleEndian.PutUint32(e.buf[offAt+4*zi:], uint32(len(e.buf)-bodyStart))
		p := &plans[zi]
		e.emitQPackInt64(s[i:j], p.codec, p.mn, p.forBits, p.first, p.minDelta, p.deltaBits, p.pforBits)
		zi++
	}
}

// ---- encode (uint64) ----

func (e *Encoder) writeZoneChunkUint64(s []uint64) {
	e.writeHeader()
	n := len(s)
	zoneCount := (n + blockSizeSmall - 1) / blockSizeSmall
	e.planBlocksU64(s, blockSizeSmall)
	plans := e.blkPlanU64[:zoneCount]

	e.buf = append(e.buf, tagZoneChunk, zoneKindUint, blkLogSmall)
	e.buf = appendUvarint(e.buf, uint64(n))
	offAt := len(e.buf)
	e.buf = append(e.buf, make([]byte, 4*zoneCount)...)
	for i := 0; i < n; i += blockSizeSmall {
		j := min(i+blockSizeSmall, n)
		mn, mx := minMaxU64(s[i:j])
		e.buf = appendUvarint(e.buf, mn)
		e.buf = appendUvarint(e.buf, mx)
	}
	bodyStart := len(e.buf)
	zi := 0
	for i := 0; i < n; i += blockSizeSmall {
		j := min(i+blockSizeSmall, n)
		binary.LittleEndian.PutUint32(e.buf[offAt+4*zi:], uint32(len(e.buf)-bodyStart))
		p := &plans[zi]
		e.emitQPackUint64(s[i:j], p.codec, p.mn, p.forBits, p.first, p.minDelta, p.deltaBits, p.pforBits)
		zi++
	}
}

// ---- decode header (shared) ----

// zoneChunkHeader is the decoded geometry + zonemap of a zone-chunked column.
// minI64/maxI64 (or minU64/maxU64) hold the per-zone bounds for predicate
// zone-skip; the other-kind slices stay nil.
type zoneChunkHeader struct {
	minI64, maxI64 []int64
	minU64, maxU64 []uint64
	offBase        int // byte offset of the uint32 offset table
	bodyStart      int // byte offset of the first zone body
	n              int
	zoneCount      int
	blk            int
	kind           byte
}

// readZoneChunkHeader consumes the tag-less header (caller consumed the tag),
// validates it, decodes the offset table position and the zonemap, and returns
// the geometry. It leaves d.i at bodyStart.
func (d *Decoder) readZoneChunkHeader() (zoneChunkHeader, error) {
	var h zoneChunkHeader
	if d.i+2 > len(d.buf) {
		return h, ErrShortBuffer
	}
	h.kind = d.buf[d.i]
	if h.kind != zoneKindInt && h.kind != zoneKindUint {
		return h, ErrTypeMismatch
	}
	d.i++
	blk, ok := blockSizeFor(d.buf[d.i])
	if !ok {
		return h, ErrBadTag
	}
	h.blk = blk
	d.i++
	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return h, ErrInvalidLength
	}
	d.i += nr
	if !d.colLenOK(n64) || n64 == 0 || n64 > uint64(int(^uint(0)>>1)) {
		return h, ErrInvalidLength
	}
	h.n = int(n64)
	if checkColumnarBytes(h.n, 8) != nil {
		return h, ErrInvalidLength
	}
	h.zoneCount = (h.n + blk - 1) / blk

	// offset table
	h.offBase = d.i
	if d.i+4*h.zoneCount > len(d.buf) {
		return h, ErrShortBuffer
	}
	d.i += 4 * h.zoneCount

	// zonemap (zoneCount x min,max varints)
	if h.kind == zoneKindInt {
		h.minI64 = make([]int64, h.zoneCount)
		h.maxI64 = make([]int64, h.zoneCount)
		for z := 0; z < h.zoneCount; z++ {
			mnZ, nr := readUvarint(d.buf[d.i:])
			if nr <= 0 {
				return h, ErrInvalidLength
			}
			d.i += nr
			mxZ, nr2 := readUvarint(d.buf[d.i:])
			if nr2 <= 0 {
				return h, ErrInvalidLength
			}
			d.i += nr2
			h.minI64[z] = zigzagDecode64(mnZ)
			h.maxI64[z] = zigzagDecode64(mxZ)
			if h.minI64[z] > h.maxI64[z] {
				return h, ErrInvalidLength
			}
		}
	} else {
		h.minU64 = make([]uint64, h.zoneCount)
		h.maxU64 = make([]uint64, h.zoneCount)
		for z := 0; z < h.zoneCount; z++ {
			mnZ, nr := readUvarint(d.buf[d.i:])
			if nr <= 0 {
				return h, ErrInvalidLength
			}
			d.i += nr
			mxZ, nr2 := readUvarint(d.buf[d.i:])
			if nr2 <= 0 {
				return h, ErrInvalidLength
			}
			d.i += nr2
			h.minU64[z] = mnZ
			h.maxU64[z] = mxZ
			if h.minU64[z] > h.maxU64[z] {
				return h, ErrInvalidLength
			}
		}
	}

	h.bodyStart = d.i
	// Validate the offset table: offsets[0]==0, strictly increasing, in-buffer.
	prev := -1
	for z := 0; z < h.zoneCount; z++ {
		off := int(binary.LittleEndian.Uint32(d.buf[h.offBase+4*z:]))
		if z == 0 && off != 0 {
			return h, ErrInvalidLength
		}
		if off <= prev {
			return h, ErrInvalidLength
		}
		if h.bodyStart+off > len(d.buf) {
			return h, ErrShortBuffer
		}
		prev = off
	}
	return h, nil
}

// zoneOff returns the absolute byte offset of zone z's body.
func (h *zoneChunkHeader) zoneOff(d *Decoder, z int) int {
	return h.bodyStart + int(binary.LittleEndian.Uint32(d.buf[h.offBase+4*z:]))
}

// zoneLen returns the element count of zone z.
func (h *zoneChunkHeader) zoneLen(z int) int {
	if z == h.zoneCount-1 {
		return h.n - z*h.blk
	}
	return h.blk
}

// zoneSkippedZones counts zones the query path proved cannot match and did not
// decode. Test-only instrumentation; otherwise inert.
var zoneSkippedZones int

// decodeZoneChunkQuery decodes only the zones of a zone-chunked column that
// intersect at least one of the referencing leaves' bounds, evaluates each
// leaf's predicate over those zones into the leaf's precompT mask (zones that
// cannot match are skipped — their rows stay FALSE), and, when proj is set, fills
// a colVals of length n with the decoded zones' values (skipped zones left zero;
// they can never be matched, so are never scattered). The column starts at byte
// offset start (pointing at the tag); the caller repositions d.i afterwards via
// the column-length index.
func (d *Decoder) decodeZoneChunkQuery(start int, leaves []*cnode, n int, proj bool) (*colVals, error) {
	d.i = start + 1 // skip tag
	h, err := d.readZoneChunkHeader()
	if err != nil {
		return nil, err
	}
	if h.n != n {
		return nil, ErrTypeMismatch
	}
	for _, lf := range leaves {
		lf.precompT = newBitset(n)
	}
	var cv *colVals
	if proj {
		if h.kind == zoneKindInt {
			cv = &colVals{kind: colKindInt, i64: make([]int64, n)}
		} else {
			cv = &colVals{kind: colKindUint, u64: make([]uint64, n)}
		}
	}
	for z := range h.zoneCount {
		needed := false
		for _, lf := range leaves {
			// Zone z intersects the leaf's bounds iff zoneMax >= lo && zoneMin <= hi.
			if h.kind == zoneKindInt {
				if h.maxI64[z] >= lf.term.loI64 && h.minI64[z] <= lf.term.hiI64 {
					needed = true
					break
				}
			} else {
				if h.maxU64[z] >= lf.term.loU64 && h.minU64[z] <= lf.term.hiU64 {
					needed = true
					break
				}
			}
		}
		if !needed {
			zoneSkippedZones++
			continue
		}
		d.i = h.zoneOff(d, z)
		tg, err := d.peekTag()
		if err != nil {
			return nil, err
		}
		base := z * h.blk
		want := h.zoneLen(z)
		if h.kind == zoneKindInt {
			v, err := d.readQPackInt64(tg)
			if err != nil {
				return nil, err
			}
			if len(v) != want {
				return nil, ErrInvalidLength
			}
			for _, lf := range leaves {
				p := lf.term.pI64
				m := lf.precompT
				for r, x := range v {
					if p(x) {
						setBit(m, base+r)
					}
				}
			}
			if proj {
				copy(cv.i64[base:], v)
			}
		} else {
			v, err := d.readQPackUint64(tg)
			if err != nil {
				return nil, err
			}
			if len(v) != want {
				return nil, ErrInvalidLength
			}
			for _, lf := range leaves {
				p := lf.term.pU64
				m := lf.precompT
				for r, x := range v {
					if p(x) {
						setBit(m, base+r)
					}
				}
			}
			if proj {
				copy(cv.u64[base:], v)
			}
		}
	}
	return cv, nil
}

// ---- decode-all (int64) ----

func (d *Decoder) readZoneChunkInt64() ([]int64, error) {
	var s []int64
	if err := d.readZoneChunkInt64Into(&s); err != nil {
		return nil, err
	}
	return s, nil
}

func (d *Decoder) readZoneChunkInt64Into(dst *[]int64) error {
	h, err := d.readZoneChunkHeader()
	if err != nil {
		return err
	}
	if h.kind != zoneKindInt {
		return ErrTypeMismatch
	}
	growI64(dst, h.n)
	out := *dst
	for z := range h.zoneCount {
		d.i = h.zoneOff(d, z)
		t, err := d.peekTag()
		if err != nil {
			return err
		}
		v, err := d.readQPackInt64(t)
		if err != nil {
			return err
		}
		if len(v) != h.zoneLen(z) {
			return ErrInvalidLength
		}
		copy(out[z*h.blk:], v)
	}
	return nil
}

// ---- decode-all (uint64) ----

func (d *Decoder) readZoneChunkUint64() ([]uint64, error) {
	var s []uint64
	if err := d.readZoneChunkUint64Into(&s); err != nil {
		return nil, err
	}
	return s, nil
}

func (d *Decoder) readZoneChunkUint64Into(dst *[]uint64) error {
	h, err := d.readZoneChunkHeader()
	if err != nil {
		return err
	}
	if h.kind != zoneKindUint {
		return ErrTypeMismatch
	}
	growU64(dst, h.n)
	out := *dst
	for z := range h.zoneCount {
		d.i = h.zoneOff(d, z)
		t, err := d.peekTag()
		if err != nil {
			return err
		}
		v, err := d.readQPackUint64(t)
		if err != nil {
			return err
		}
		if len(v) != h.zoneLen(z) {
			return ErrInvalidLength
		}
		copy(out[z*h.blk:], v)
	}
	return nil
}
