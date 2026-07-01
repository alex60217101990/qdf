package qdf

import (
	"encoding/binary"
	"math"
	"sync/atomic"
)

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
	zoneKindInt   = 0x00
	zoneKindUint  = 0x01
	zoneKindFloat = 0x02 // float64
	// zoneChunkMinLen: a column shorter than two zones has nothing to chunk.
	zoneChunkMinLen = 2 * blockSizeSmall // reuse the block codec's 256-row zone
)

// Zonemap encoding (the byte after kind). minmax is the per-zone [min,max] index;
// linear is a single learned model pos≈c·value+d (±epsP) over a SORTED int/uint
// column, which collapses the whole per-zone index to ~18 bytes. The encoder
// picks linear only when the column is monotonic, the fit holds within ~one zone
// (so zone-skip is preserved), and the model is strictly smaller than the minmax
// index (never-worse). See fitLinearZmap.
const (
	zmapMinMax = 0x00
	zmapLinear = 0x01
)

// linearFit is a learned zonemap: the row position of a value v is predicted by
// c*v+d within ±epsP positions. A query range [lo,hi] therefore maps to the row
// range [c*lo+d-epsP, c*hi+d+epsP], hence to a contiguous zone range — every
// matching row provably lands inside it, so no zone with a match is ever skipped.
type linearFit struct {
	c, d float64
	epsP int
}

// fitLinearZmap fits a single line pos≈c*value+d over a monotonic non-decreasing
// integer column (values widened to float64) and returns it iff (a) the column is
// sorted, (b) the values are not all equal (c>0), and (c) the worst-case position
// residual is at most blk — i.e. the model locates any value to within one zone,
// so zone-skip stays tight. epsP carries a +1 margin to absorb float rounding in
// the query-side re-evaluation. The caller still compares the encoded size
// against the minmax index and keeps whichever is smaller.
func fitLinearZmap[T int64 | uint64](s []T, blk int) (linearFit, bool) {
	n := len(s)
	if n < 2 {
		return linearFit{}, false
	}
	// Monotonic non-decreasing? (position == rank requires a sorted column.)
	for i := 1; i < n; i++ {
		if s[i] < s[i-1] {
			return linearFit{}, false
		}
	}
	// Least-squares regression of position (y=i) on value (x=s[i]), in float64.
	var sx, sy, sxx, sxy float64
	fn := float64(n)
	for i := range n {
		x := float64(s[i])
		y := float64(i)
		sx += x
		sy += y
		sxx += x * x
		sxy += x * y
	}
	denom := fn*sxx - sx*sx
	if denom <= 0 { // all values equal (or degenerate) → no usable slope
		return linearFit{}, false
	}
	c := (fn*sxy - sx*sy) / denom
	d := (sy - c*sx) / fn
	// Reject any non-finite model. The decoder (readZoneChunkHeader) rejects
	// IsInf(d) too, so emitting d=±Inf here (reachable when c*sx overflows) would
	// write a column its own decoder refuses to read.
	if !(c > 0) || math.IsInf(c, 0) || math.IsNaN(d) || math.IsInf(d, 0) {
		return linearFit{}, false
	}
	// Worst-case position residual.
	var maxRes float64
	for i := range n {
		r := math.Abs(float64(i) - (c*float64(s[i]) + d))
		if r > maxRes {
			maxRes = r
		}
	}
	epsP := int(math.Ceil(maxRes)) + 1 // +1: float-rounding margin for query re-eval
	if epsP > blk {
		return linearFit{}, false // fit too loose → zone-skip would degrade
	}
	return linearFit{c: c, d: d, epsP: epsP}, true
}

// zoneRangeFor maps a predicate value range [lo,hi] to the inclusive zone range
// [zlo,zhi] that can contain a matching row, per the linear model. Returns
// ok=false when the range cannot intersect the column (no zone needed).
func (lf linearFit) zoneRangeFor(lo, hi float64, n, blk int) (zlo, zhi int, ok bool) {
	pLo := lf.c*lo + lf.d - float64(lf.epsP)
	pHi := lf.c*hi + lf.d + float64(lf.epsP)
	if math.IsNaN(pLo) || math.IsNaN(pHi) || pHi < 0 || pLo > float64(n-1) {
		return 0, 0, false
	}
	// Clamp to [0, n-1] in the FLOAT domain before the int conversion: an
	// open-ended GE/LE bound makes pHi/pLo overflow int64 range, and an
	// out-of-range float→int yields MinInt64 on amd64, which would wrap to a
	// bogus negative zone and silently drop every matching row.
	if pLo < 0 {
		pLo = 0
	}
	if pHi > float64(n-1) {
		pHi = float64(n - 1)
	}
	loI := int(math.Floor(pLo))
	hiI := int(math.Floor(pHi))
	return loI / blk, hiI / blk, true
}

// finiteMinMaxF64 returns the min and max of s over non-NaN values. ±Inf are
// ordered and included. An all-NaN (or empty) slice yields the empty interval
// (+Inf, -Inf), which intersects no finite predicate range — so such a zone is
// correctly skipped (NaN never matches a comparison). Caller passes len(s) > 0.
func finiteMinMaxF64(s []float64) (mn, mx float64) {
	mn, mx = math.Inf(1), math.Inf(-1)
	for _, v := range s {
		if v != v { // NaN: builtin min/max would propagate it, so skip explicitly
			continue
		}
		mn, mx = min(mn, v), max(mx, v)
	}
	return mn, mx
}

// ---- encode (int64) ----

// linearZmapBytes is the encoded size of a linear zonemap (2 float64 + epsP).
func linearZmapBytes(f linearFit) int { return 16 + uvarintLen(uint64(f.epsP)) }

func zmapByte(linear bool) byte {
	if linear {
		return zmapLinear
	}
	return zmapMinMax
}

func (e *Encoder) writeZoneChunkInt64(s []int64) {
	e.writeHeader()
	n := len(s)
	zoneCount := (n + blockSizeSmall - 1) / blockSizeSmall
	e.planBlocksI64(s, blockSizeSmall) // cache per-zone codec picks (no re-pick on emit)
	plans := e.blkPlanI64[:zoneCount]

	// Pick the learned linear zonemap over per-zone min/max only when it is both
	// valid (monotonic, tight fit) and strictly smaller (never-worse).
	fit, linOK := fitLinearZmap(s, blockSizeSmall)
	if linOK {
		var mmBytes int
		for i := 0; i < n; i += blockSizeSmall {
			j := min(i+blockSizeSmall, n)
			mn, mx := minMaxI64(s[i:j])
			mmBytes += uvarintLen(zigzagEncode64(mn)) + uvarintLen(zigzagEncode64(mx))
		}
		linOK = linearZmapBytes(fit) < mmBytes
	}

	e.buf = append(e.buf, tagZoneChunk, zoneKindInt, zmapByte(linOK), blkLogSmall)
	e.buf = appendUvarint(e.buf, uint64(n))
	offAt := len(e.buf)
	e.buf = append(e.buf, make([]byte, 4*zoneCount)...)
	if linOK {
		e.buf = binary.LittleEndian.AppendUint64(e.buf, math.Float64bits(fit.c))
		e.buf = binary.LittleEndian.AppendUint64(e.buf, math.Float64bits(fit.d))
		e.buf = appendUvarint(e.buf, uint64(fit.epsP))
	} else {
		// zonemap: per-zone min,max as zigzag-varint (variable length), before bodies.
		for i := 0; i < n; i += blockSizeSmall {
			j := min(i+blockSizeSmall, n)
			mn, mx := minMaxI64(s[i:j])
			e.buf = appendUvarint(e.buf, zigzagEncode64(mn))
			e.buf = appendUvarint(e.buf, zigzagEncode64(mx))
		}
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

	fit, linOK := fitLinearZmap(s, blockSizeSmall)
	if linOK {
		var mmBytes int
		for i := 0; i < n; i += blockSizeSmall {
			j := min(i+blockSizeSmall, n)
			mn, mx := minMaxU64(s[i:j])
			mmBytes += uvarintLen(mn) + uvarintLen(mx)
		}
		linOK = linearZmapBytes(fit) < mmBytes
	}

	e.buf = append(e.buf, tagZoneChunk, zoneKindUint, zmapByte(linOK), blkLogSmall)
	e.buf = appendUvarint(e.buf, uint64(n))
	offAt := len(e.buf)
	e.buf = append(e.buf, make([]byte, 4*zoneCount)...)
	if linOK {
		e.buf = binary.LittleEndian.AppendUint64(e.buf, math.Float64bits(fit.c))
		e.buf = binary.LittleEndian.AppendUint64(e.buf, math.Float64bits(fit.d))
		e.buf = appendUvarint(e.buf, uint64(fit.epsP))
	} else {
		for i := 0; i < n; i += blockSizeSmall {
			j := min(i+blockSizeSmall, n)
			mn, mx := minMaxU64(s[i:j])
			e.buf = appendUvarint(e.buf, mn)
			e.buf = appendUvarint(e.buf, mx)
		}
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

// ---- encode (float64) ----

func (e *Encoder) writeZoneChunkFloat64(s []float64) {
	e.writeHeader()
	n := len(s)
	zoneCount := (n + blockSizeSmall - 1) / blockSizeSmall
	// float64 uses the per-zone min/max index only (no linear model in v1).
	e.buf = append(e.buf, tagZoneChunk, zoneKindFloat, zmapMinMax, blkLogSmall)
	e.buf = appendUvarint(e.buf, uint64(n))
	offAt := len(e.buf)
	e.buf = append(e.buf, make([]byte, 4*zoneCount)...)
	// zonemap: per-zone finite min,max as 8-byte IEEE-754 bits (LE).
	for i := 0; i < n; i += blockSizeSmall {
		j := min(i+blockSizeSmall, n)
		mn, mx := finiteMinMaxF64(s[i:j])
		e.buf = binary.LittleEndian.AppendUint64(e.buf, math.Float64bits(mn))
		e.buf = binary.LittleEndian.AppendUint64(e.buf, math.Float64bits(mx))
	}
	bodyStart := len(e.buf)
	zi := 0
	for i := 0; i < n; i += blockSizeSmall {
		j := min(i+blockSizeSmall, n)
		binary.LittleEndian.PutUint32(e.buf[offAt+4*zi:], uint32(len(e.buf)-bodyStart))
		ss := s[i:j]
		_ = encodeSliceFloat64Lossless(e, ss) // never errors for a plain []float64
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
	minF64, maxF64 []float64
	lin            linearFit // valid when zmap == zmapLinear (int/uint only)
	offBase        int       // byte offset of the uint32 offset table
	bodyStart      int       // byte offset of the first zone body
	n              int
	zoneCount      int
	blk            int
	kind           byte
	zmap           byte // zmapMinMax | zmapLinear
}

// readZoneChunkHeader consumes the tag-less header (caller consumed the tag),
// validates it, decodes the offset table position and (when loadZonemap) the
// per-zone min/max zonemap, and returns the geometry. It leaves d.i at bodyStart.
//
// Only the predicate zone-skip path needs the zonemap; the full-decode and
// selective paths read all/matched zones regardless, so they pass loadZonemap
// false — the zonemap bytes are still walked to locate bodyStart (the int/uint
// entries are variable-length varints), but no slices are allocated and no
// values are decoded.
func (d *Decoder) readZoneChunkHeader(loadZonemap bool) (zoneChunkHeader, error) {
	var h zoneChunkHeader
	if d.i+3 > len(d.buf) {
		return h, ErrShortBuffer
	}
	h.kind = d.buf[d.i]
	if h.kind != zoneKindInt && h.kind != zoneKindUint && h.kind != zoneKindFloat {
		return h, ErrTypeMismatch
	}
	d.i++
	h.zmap = d.buf[d.i]
	// A linear zonemap is only defined for int/uint columns.
	if h.zmap != zmapMinMax && h.zmap != zmapLinear {
		return h, ErrBadTag
	}
	if h.zmap == zmapLinear && h.kind == zoneKindFloat {
		return h, ErrBadTag
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
	if !d.colLenOK(n64) || n64 == 0 || n64 > uint64(math.MaxInt) {
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

	// Linear zonemap: a single model (2 float64 + epsP varint) replaces the whole
	// per-zone min/max index. Read it (or skip its fixed bytes) and return.
	if h.zmap == zmapLinear {
		if d.i+16 > len(d.buf) {
			return h, ErrShortBuffer
		}
		if loadZonemap {
			h.lin.c = math.Float64frombits(binary.LittleEndian.Uint64(d.buf[d.i:]))
			h.lin.d = math.Float64frombits(binary.LittleEndian.Uint64(d.buf[d.i+8:]))
		}
		d.i += 16
		eps64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return h, ErrInvalidLength
		}
		d.i += nr
		if eps64 > uint64(math.MaxInt) {
			return h, ErrInvalidLength
		}
		if loadZonemap {
			h.lin.epsP = int(eps64)
			// A non-finite model or non-positive slope cannot bound positions.
			if !(h.lin.c > 0) || math.IsInf(h.lin.c, 0) || math.IsNaN(h.lin.d) || math.IsInf(h.lin.d, 0) {
				return h, ErrInvalidLength
			}
		}
		h.bodyStart = d.i
		if err := h.validateOffsets(d); err != nil {
			return h, err
		}
		return h, nil
	}

	// zonemap (per-kind min,max)
	switch h.kind {
	case zoneKindInt:
		if loadZonemap {
			h.minI64 = make([]int64, h.zoneCount)
			h.maxI64 = make([]int64, h.zoneCount)
		}
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
			if loadZonemap {
				h.minI64[z] = zigzagDecode64(mnZ)
				h.maxI64[z] = zigzagDecode64(mxZ)
				if h.minI64[z] > h.maxI64[z] {
					return h, ErrInvalidLength
				}
			}
		}
	case zoneKindUint:
		if loadZonemap {
			h.minU64 = make([]uint64, h.zoneCount)
			h.maxU64 = make([]uint64, h.zoneCount)
		}
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
			if loadZonemap {
				h.minU64[z] = mnZ
				h.maxU64[z] = mxZ
				if h.minU64[z] > h.maxU64[z] {
					return h, ErrInvalidLength
				}
			}
		}
	default: // zoneKindFloat: 8-byte min + 8-byte max per zone (IEEE-754 bits LE)
		if d.i+16*h.zoneCount > len(d.buf) {
			return h, ErrShortBuffer
		}
		if loadZonemap {
			h.minF64 = make([]float64, h.zoneCount)
			h.maxF64 = make([]float64, h.zoneCount)
			for z := 0; z < h.zoneCount; z++ {
				h.minF64[z] = math.Float64frombits(binary.LittleEndian.Uint64(d.buf[d.i:]))
				h.maxF64[z] = math.Float64frombits(binary.LittleEndian.Uint64(d.buf[d.i+8:]))
				d.i += 16
				// No min<=max check: an all-NaN zone stores the empty interval
				// (+Inf, -Inf) on purpose, and NaN bounds are not ordered.
			}
		} else {
			d.i += 16 * h.zoneCount
		}
	}

	h.bodyStart = d.i
	if err := h.validateOffsets(d); err != nil {
		return h, err
	}
	return h, nil
}

// validateOffsets checks the offset table: offsets[0]==0, strictly increasing,
// each in-buffer. Call after h.bodyStart is set.
func (h *zoneChunkHeader) validateOffsets(d *Decoder) error {
	prev := -1
	for z := 0; z < h.zoneCount; z++ {
		off := int(binary.LittleEndian.Uint32(d.buf[h.offBase+4*z:]))
		if z == 0 && off != 0 {
			return ErrInvalidLength
		}
		if off <= prev {
			return ErrInvalidLength
		}
		if h.bodyStart+off > len(d.buf) {
			return ErrShortBuffer
		}
		prev = off
	}
	return nil
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

// ---- decode-all (float64) ----

func (d *Decoder) readZoneChunkFloat64() ([]float64, error) {
	var s []float64
	if err := d.readZoneChunkFloat64Into(&s); err != nil {
		return nil, err
	}
	return s, nil
}

func (d *Decoder) readZoneChunkFloat64Into(dst *[]float64) error {
	if err := d.descend(); err != nil {
		return err
	}
	defer d.ascend()
	h, err := d.readZoneChunkHeader(false)
	if err != nil {
		return err
	}
	if h.kind != zoneKindFloat {
		return ErrTypeMismatch
	}
	growF64(dst, h.n)
	out := *dst
	var tmp []float64
	for z := range h.zoneCount {
		d.i = h.zoneOff(d, z)
		if err := decodeSliceFloat64Into(d, &tmp); err != nil {
			return err
		}
		if len(tmp) != h.zoneLen(z) {
			return ErrInvalidLength
		}
		copy(out[z*h.blk:], tmp)
	}
	return nil
}

// zoneSkippedZones counts zones the query path proved cannot match and did not
// decode. Test-only instrumentation; atomic so concurrent Decoders zone-skipping
// on different goroutines do not race on it.
var zoneSkippedZones atomic.Int64

// decodeZoneChunkQuery decodes only the zones of a zone-chunked column that
// intersect at least one of the referencing leaves' bounds and evaluates each
// leaf's predicate over those zones into the leaf's precompT mask (zones that
// cannot match are skipped — their rows stay FALSE). It is FILTER-only: it never
// materialises projected values, because a row matched via an OR/NOT sibling can
// live in a zone this column's bounds skipped, so projection must follow the
// FINAL matched set (decodeZoneChunkSelective), not the leaf bounds. The column
// starts at byte offset start (pointing at the tag); the caller repositions d.i
// afterwards via the column-length index.
func (d *Decoder) decodeZoneChunkQuery(start int, leaves []*cnode, n int) error {
	d.i = start + 1 // skip tag
	h, err := d.readZoneChunkHeader(true)
	if err != nil {
		return err
	}
	if h.n != n {
		return ErrTypeMismatch
	}
	for _, lf := range leaves {
		lf.precompT = newBitset(n)
	}
	// For the linear zonemap, each leaf's value range maps once to a contiguous
	// zone range; a zone is then needed iff it falls inside some leaf's range.
	type zr struct {
		zlo, zhi int
		ok       bool
	}
	var lzr []zr
	if h.zmap == zmapLinear {
		lzr = make([]zr, len(leaves))
		for li, lf := range leaves {
			var lo, hi float64
			if h.kind == zoneKindInt {
				lo, hi = float64(lf.term.loI64), float64(lf.term.hiI64)
			} else {
				lo, hi = float64(lf.term.loU64), float64(lf.term.hiU64)
			}
			a, b, ok := h.lin.zoneRangeFor(lo, hi, h.n, h.blk)
			lzr[li] = zr{a, b, ok}
		}
	}
	for z := range h.zoneCount {
		needed := false
		for li, lf := range leaves {
			if h.zmap == zmapLinear {
				if lzr[li].ok && z >= lzr[li].zlo && z <= lzr[li].zhi {
					needed = true
				}
				if needed {
					break
				}
				continue
			}
			// Zone z intersects the leaf's bounds iff zoneMax >= lo && zoneMin <= hi.
			switch h.kind {
			case zoneKindInt:
				if h.maxI64[z] >= lf.term.loI64 && h.minI64[z] <= lf.term.hiI64 {
					needed = true
				}
			case zoneKindUint:
				if h.maxU64[z] >= lf.term.loU64 && h.minU64[z] <= lf.term.hiU64 {
					needed = true
				}
			default: // float: empty-interval (all-NaN) zones never intersect → skipped
				if h.maxF64[z] >= lf.term.loF64 && h.minF64[z] <= lf.term.hiF64 {
					needed = true
				}
			}
			if needed {
				break
			}
		}
		if !needed {
			zoneSkippedZones.Add(1)
			continue
		}
		d.i = h.zoneOff(d, z)
		tg, err := d.peekTag()
		if err != nil {
			return err
		}
		base := z * h.blk
		want := h.zoneLen(z)
		switch h.kind {
		case zoneKindInt:
			v, err := d.readQPackInt64(tg)
			if err != nil {
				return err
			}
			if len(v) != want {
				return ErrInvalidLength
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
		case zoneKindUint:
			v, err := d.readQPackUint64(tg)
			if err != nil {
				return err
			}
			if len(v) != want {
				return ErrInvalidLength
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
		default: // float64
			var v []float64
			if err := decodeSliceFloat64Into(d, &v); err != nil {
				return err
			}
			if len(v) != want {
				return ErrInvalidLength
			}
			for _, lf := range leaves {
				p := lf.term.pF64
				m := lf.precompT
				for r, x := range v {
					if p(x) {
						setBit(m, base+r)
					}
				}
			}
		}
	}
	return nil
}

// decodeZoneChunkSelective decodes only the zones spanning at least one matched
// row and fills a length-n colVals with their values (rows in undecoded zones
// stay zero — they are not matched, so are never scattered). It is the projection
// counterpart to decodeZoneChunkQuery's filter pass: a row matched via an OR/NOT
// sibling can fall in a zone this column's bounds skipped, so projection must
// follow the FINAL matched set rather than the leaf bounds. start points at the
// tag; matched holds the surviving row indices.
func (d *Decoder) decodeZoneChunkSelective(start, n int, matched []int) (*colVals, error) {
	d.i = start + 1 // skip tag
	h, err := d.readZoneChunkHeader(false)
	if err != nil {
		return nil, err
	}
	if h.n != n {
		return nil, ErrTypeMismatch
	}
	var cv *colVals
	switch h.kind {
	case zoneKindInt:
		cv = &colVals{kind: colKindInt, i64: make([]int64, n)}
	case zoneKindUint:
		cv = &colVals{kind: colKindUint, u64: make([]uint64, n)}
	default:
		cv = &colVals{kind: colKindFloat, f64: make([]float64, n)}
	}
	need := make([]bool, h.zoneCount)
	for _, r := range matched {
		need[r/h.blk] = true
	}
	var ftmp []float64
	for z := range h.zoneCount {
		if !need[z] {
			continue
		}
		d.i = h.zoneOff(d, z)
		base := z * h.blk
		want := h.zoneLen(z)
		switch h.kind {
		case zoneKindInt:
			tg, err := d.peekTag()
			if err != nil {
				return nil, err
			}
			v, err := d.readQPackInt64(tg)
			if err != nil {
				return nil, err
			}
			if len(v) != want {
				return nil, ErrInvalidLength
			}
			copy(cv.i64[base:], v)
		case zoneKindUint:
			tg, err := d.peekTag()
			if err != nil {
				return nil, err
			}
			v, err := d.readQPackUint64(tg)
			if err != nil {
				return nil, err
			}
			if len(v) != want {
				return nil, ErrInvalidLength
			}
			copy(cv.u64[base:], v)
		default: // float64
			if err := decodeSliceFloat64Into(d, &ftmp); err != nil {
				return nil, err
			}
			if len(ftmp) != want {
				return nil, ErrInvalidLength
			}
			copy(cv.f64[base:], ftmp)
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
	// A zone body could itself carry tagZoneChunk; bound the nesting so a hostile
	// payload of nested zone chunks cannot overflow the goroutine stack.
	if err := d.descend(); err != nil {
		return err
	}
	defer d.ascend()
	h, err := d.readZoneChunkHeader(false)
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
	if err := d.descend(); err != nil {
		return err
	}
	defer d.ascend()
	h, err := d.readZoneChunkHeader(false)
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
