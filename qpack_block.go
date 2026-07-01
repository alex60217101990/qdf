package qdf

import (
	"encoding/binary"
	"math"
	"sync/atomic"

	"github.com/alex60217101990/qdf/internal/bitpack"
)

// Per-block adaptive integer codec (wire tag tagPackBlock, 0xF0).
//
// A long integer column is split into fixed-size blocks; each block is encoded
// independently with the existing whole-column picker (pickI64Codec /
// pickU64Codec → FOR / Delta+FOR / RLE / dict / PFOR / raw). A column whose
// statistics shift along its length (sorted-then-random, bursty timestamps,
// mixed-cardinality IDs, counters with resets) packs each region optimally
// instead of paying one codec for the whole. An offset table prefixes the
// blocks so a predicate query can seek to and decode only the blocks covering
// its matched rows.
//
// Strictly never-larger: the block form is emitted only when its exact byte
// cost is smaller than the whole-column codec. The block-cost estimate is exact
// (each block's pickXXXCodec.bestCost is exact for the codec it picks, and the
// header + offset table are sized exactly), and the whole-column cost is exact
// for non-constant columns — constant columns are filtered out by the flat
// pre-gate before any per-block work — so the comparison cannot be wrong in a
// way that inflates the wire.

const (
	// blockCodecMinLen is the smallest column the block codec considers: two
	// blocks of the smaller size, below which there is nothing to adapt.
	blockCodecMinLen = 2 * blockSizeSmall
	blockSizeSmall   = 256
	blockSizeLarge   = 1024
	blkLogSmall      = 8  // log2(256)
	blkLogLarge      = 10 // log2(1024)

	blockKindInt  = 0x00
	blockKindUint = 0x01

	// The cheap pre-gate samples blockGateSamples blocks, measures each one's FOR
	// width over just a blockGateWindow-element window (one tiny min/max pass —
	// orders of magnitude cheaper than a full pickXXXCodec), and fires when the
	// spread between the widest and narrowest sampled block is >= blockGateBitsSpread.
	// It keys on block-to-block HETEROGENEITY, not block-vs-whole tightness, so a
	// globally monotonic column (small per-block FOR width but delta-optimal as a
	// whole) is correctly skipped instead of triggering a wasted full probe. The
	// gate only filters probe CPU — the full probe + never-larger still decide, so
	// a gate miss costs size, never correctness.
	blockGateSamples    = 8
	blockGateWindow     = 64
	blockGateBitsSpread = 4

	// blockGateWideBits is the cheapest gate: a column whose whole-column FOR and
	// delta widths are BOTH below this is already compact (delta/RLE/dict or a
	// narrow FOR — the common flat column), so blocks have no headroom and the
	// sampling pass is skipped entirely. Only genuinely wide columns (the kind a
	// regime shift can shrink) ever reach the sampling gate.
	blockGateWideBits = 12
)

// blockSizeFor maps a wire blkLog byte to its block size, rejecting any value
// other than the two the encoder emits.
func blockSizeFor(blkLog byte) (int, bool) {
	switch blkLog {
	case blkLogSmall:
		return blockSizeSmall, true
	case blkLogLarge:
		return blockSizeLarge, true
	}
	return 0, false
}

// blockRegimeLikelyI64 cheaply tests whether s is heterogeneous enough to be
// worth a full per-block probe. It samples up to blockGateSamples blocks and
// compares the average per-block FOR width (one min/max pass each — far cheaper
// than a full pickI64Codec) to the whole-column width: blocks that pack markedly
// tighter mean local structure a single codec can't exploit. Catches trend
// (sorted-then-random), periodic (bursty, mixed-card), and constant-run
// (rle-then-noise) regimes; misses only the rare case where width is uniform but
// codec choice shifts — a size miss, never a correctness one.
func blockRegimeLikelyI64(s []int64, wholeForBits int) bool {
	if wholeForBits <= 2 {
		return false // already near-incompressible by FOR; blocks can't help
	}
	n := len(s)
	nb := (n + blockSizeSmall - 1) / blockSizeSmall
	step := max(nb/blockGateSamples, 1)
	minB, maxB := 64, -1
	for b := 0; b < nb; b += step {
		i := b * blockSizeSmall
		j := min(i+blockGateWindow, n)
		mn, mx := minMaxI64(s[i:j])
		w := bitpack.BitsForDelta(uint64(mx) - uint64(mn))
		minB, maxB = min(minB, w), max(maxB, w)
	}
	return maxB-minB >= blockGateBitsSpread
}

func blockRegimeLikelyU64(s []uint64, wholeForBits int) bool {
	if wholeForBits <= 2 {
		return false
	}
	n := len(s)
	nb := (n + blockSizeSmall - 1) / blockSizeSmall
	step := max(nb/blockGateSamples, 1)
	minB, maxB := 64, -1
	for b := 0; b < nb; b += step {
		i := b * blockSizeSmall
		j := min(i+blockGateWindow, n)
		mn, mx := minMaxU64(s[i:j])
		w := bitpack.BitsForDelta(mx - mn)
		minB, maxB = min(minB, w), max(maxB, w)
	}
	return maxB-minB >= blockGateBitsSpread
}

// ---- encode (int64) ----

// tryWriteBlockInt64 emits the per-block adaptive form of s when it is strictly
// smaller than the whole-column codec, returning true. It returns false (having
// written nothing) when the column is too short, statistically flat, or the
// block form would not win — the caller then falls back to writeQPackInt64.
// blockCodecEnabled gates the per-block codec. Always true in production; tests
// flip it to obtain a whole-column baseline wire for never-larger assertions.
var blockCodecEnabled = true

// blockSelectiveBlocksDecoded counts blocks materialized by the selective decode
// path. Test-only instrumentation (to assert untouched blocks are skipped);
// atomic so concurrent Decoders running selective decode do not race on it.
var blockSelectiveBlocksDecoded atomic.Int64

// tryWriteBlockInt64 emits the smaller of the per-block form and the whole-column
// form when the cheap gates say the block form is plausible, returning true (the
// column is fully written). It returns false when the gates reject the column —
// the caller then emits the whole-column form itself.
//
// The whole-column codec is passed in (picked once by the caller). The size
// decision is made by EMITTING BOTH forms and keeping the smaller, not by
// comparing predicted costs: pickI64Codec's PFOR cost is a conservative UPPER
// bound, so a predicted-cost compare could keep a block form that is actually
// larger than the real whole-column emit (a never-larger violation). Emitting and
// measuring is exact. The gates ensure this double-emit only happens on the rare
// regime column, never on flat columns.
func (e *Encoder) tryWriteBlockInt64(s []int64, codec qpackCodec, mn int64, forBits int, first, minDelta int64, deltaBits, pforBits int) bool {
	if !blockCodecEnabled || e.rans {
		// Skip under the outer rANS pass: the per-block offset table (high-entropy
		// uint32s) and repeated sub-headers rANS-compress worse than the homogeneous
		// whole-column form, so even a raw-smaller block can inflate the final wire.
		// Keeping the whole-column form leaves OptCompression byte-identical to before.
		return false
	}
	if len(s) < blockCodecMinLen {
		return false
	}
	// Cheapest gate first: a compact column (narrow FOR and narrow delta) has no
	// headroom — skip before sampling.
	if forBits < blockGateWideBits && deltaBits < blockGateWideBits {
		return false
	}
	if !blockRegimeLikelyI64(s, forBits) {
		return false
	}
	// Emit whole first (the safe fallback), then the block form, and keep whichever
	// is actually smaller — block usually wins when the gates fire, so it is emitted
	// last and re-emission only happens on a gate false-positive. planBlocksI64
	// caches the per-block picks so writeBlockInt64 does not re-pick.
	start := len(e.buf)
	hdr, flag := e.headerOut, e.headerFlagAt
	e.emitQPackInt64(s, codec, mn, forBits, first, minDelta, deltaBits, pforBits)
	wholeSz := len(e.buf) - start
	e.buf = e.buf[:start]
	e.headerOut, e.headerFlagAt = hdr, flag
	e.planBlocksI64(s, blockSizeSmall)
	e.writeBlockInt64(s, blockSizeSmall)
	if len(e.buf)-start >= wholeSz {
		// Block not smaller — roll back and re-emit the whole-column form. Restore
		// the header latch so a rolled-away top-level stream header is re-emitted
		// (mirrors the Gorilla never-worse rollback).
		e.buf = e.buf[:start]
		e.headerOut, e.headerFlagAt = hdr, flag
		e.emitQPackInt64(s, codec, mn, forBits, first, minDelta, deltaBits, pforBits)
	}
	return true
}

// blockPlanI64 caches one block's picked codec so the emit pass does not re-pick.
type blockPlanI64 struct {
	mn, first, minDelta          int64
	codec                        qpackCodec
	forBits, deltaBits, pforBits int
}

// planBlocksI64 picks the codec for every block of s (size blk) and caches the
// choices in e.blkPlanI64 so writeBlockInt64 emits without re-picking (pickI64Codec
// is the dominant encode cost).
func (e *Encoder) planBlocksI64(s []int64, blk int) {
	n := len(s)
	nBlocks := (n + blk - 1) / blk
	if cap(e.blkPlanI64) < nBlocks {
		e.blkPlanI64 = make([]blockPlanI64, nBlocks)
	}
	plans := e.blkPlanI64[:nBlocks]
	bi := 0
	for i := 0; i < n; i += blk {
		j := min(i+blk, n)
		codec, mn, forBits, first, minDelta, deltaBits, pforBits, _ := pickI64Codec(s[i:j])
		plans[bi] = blockPlanI64{mn: mn, first: first, minDelta: minDelta, codec: codec, forBits: forBits, deltaBits: deltaBits, pforBits: pforBits}
		bi++
	}
	e.blkPlanI64 = plans
}

func (e *Encoder) writeBlockInt64(s []int64, blk int) {
	// Emit the stream header before appending directly to e.buf, exactly as the
	// other QPack writers do — a bare top-level slice has no header yet, and the
	// per-block emitQPackInt64 calls would otherwise insert it after the tag.
	// Idempotent (no-op for the struct-field case where it already exists).
	e.writeHeader()
	n := len(s)
	nBlocks := (n + blk - 1) / blk
	plans := e.blkPlanI64[:nBlocks] // filled by planBlocksI64 (same s, same blk)
	e.buf = append(e.buf, tagPackBlock, blockKindInt, blkLogSmall)
	e.buf = appendUvarint(e.buf, uint64(n))
	offAt := len(e.buf)
	e.buf = append(e.buf, make([]byte, 4*nBlocks)...)
	bodyStart := len(e.buf)
	bi := 0
	for i := 0; i < n; i += blk {
		j := min(i+blk, n)
		binary.LittleEndian.PutUint32(e.buf[offAt+4*bi:], uint32(len(e.buf)-bodyStart))
		p := &plans[bi]
		e.emitQPackInt64(s[i:j], p.codec, p.mn, p.forBits, p.first, p.minDelta, p.deltaBits, p.pforBits)
		bi++
	}
}

// ---- encode (uint64) ----

func (e *Encoder) tryWriteBlockUint64(s []uint64, codec qpackCodec, mn uint64, forBits int, first uint64, minDelta int64, deltaBits, pforBits int) bool {
	if !blockCodecEnabled || e.rans {
		return false // see tryWriteBlockInt64: offset table is rANS-hostile
	}
	if len(s) < blockCodecMinLen {
		return false
	}
	if forBits < blockGateWideBits && deltaBits < blockGateWideBits {
		return false
	}
	if !blockRegimeLikelyU64(s, forBits) {
		return false
	}
	// Emit-measure-keep-smaller (see tryWriteBlockInt64 for why predicted costs
	// are not trustworthy for the never-larger decision).
	start := len(e.buf)
	hdr, flag := e.headerOut, e.headerFlagAt
	e.emitQPackUint64(s, codec, mn, forBits, first, minDelta, deltaBits, pforBits)
	wholeSz := len(e.buf) - start
	e.buf = e.buf[:start]
	e.headerOut, e.headerFlagAt = hdr, flag
	e.planBlocksU64(s, blockSizeSmall)
	e.writeBlockUint64(s, blockSizeSmall)
	if len(e.buf)-start >= wholeSz {
		e.buf = e.buf[:start]
		e.headerOut, e.headerFlagAt = hdr, flag
		e.emitQPackUint64(s, codec, mn, forBits, first, minDelta, deltaBits, pforBits)
	}
	return true
}

// blockPlanU64 caches one block's picked codec so the emit pass does not re-pick.
type blockPlanU64 struct {
	mn, first                    uint64
	minDelta                     int64
	codec                        qpackCodec
	forBits, deltaBits, pforBits int
}

func (e *Encoder) planBlocksU64(s []uint64, blk int) {
	n := len(s)
	nBlocks := (n + blk - 1) / blk
	if cap(e.blkPlanU64) < nBlocks {
		e.blkPlanU64 = make([]blockPlanU64, nBlocks)
	}
	plans := e.blkPlanU64[:nBlocks]
	bi := 0
	for i := 0; i < n; i += blk {
		j := min(i+blk, n)
		codec, mn, forBits, first, minDelta, deltaBits, pforBits, _ := pickU64Codec(s[i:j])
		plans[bi] = blockPlanU64{mn: mn, first: first, minDelta: minDelta, codec: codec, forBits: forBits, deltaBits: deltaBits, pforBits: pforBits}
		bi++
	}
	e.blkPlanU64 = plans
}

func (e *Encoder) writeBlockUint64(s []uint64, blk int) {
	e.writeHeader()
	n := len(s)
	nBlocks := (n + blk - 1) / blk
	plans := e.blkPlanU64[:nBlocks]
	e.buf = append(e.buf, tagPackBlock, blockKindUint, blkLogSmall)
	e.buf = appendUvarint(e.buf, uint64(n))
	offAt := len(e.buf)
	e.buf = append(e.buf, make([]byte, 4*nBlocks)...)
	bodyStart := len(e.buf)
	bi := 0
	for i := 0; i < n; i += blk {
		j := min(i+blk, n)
		binary.LittleEndian.PutUint32(e.buf[offAt+4*bi:], uint32(len(e.buf)-bodyStart))
		p := &plans[bi]
		e.emitQPackUint64(s[i:j], p.codec, p.mn, p.forBits, p.first, p.minDelta, p.deltaBits, p.pforBits)
		bi++
	}
}

// ---- decode header (shared) ----

// readBlockHeader consumes the block-container header (the tag is already
// consumed by the caller), validates it, and returns the block size, total
// element count, the byte offset where the offset table begins, and the byte
// offset where the block bodies begin. It does NOT consume the offset table.
func (d *Decoder) readBlockHeader(expectKind byte) (blk, n, offBase, bodyStart, nBlocks int, err error) {
	if d.i+2 > len(d.buf) {
		return 0, 0, 0, 0, 0, ErrShortBuffer
	}
	if d.buf[d.i] != expectKind {
		return 0, 0, 0, 0, 0, ErrTypeMismatch
	}
	d.i++
	blk, ok := blockSizeFor(d.buf[d.i])
	if !ok {
		return 0, 0, 0, 0, 0, ErrBadTag
	}
	d.i++
	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return 0, 0, 0, 0, 0, ErrInvalidLength
	}
	d.i += nr
	if !d.colLenOK(n64) || n64 == 0 || n64 > uint64(math.MaxInt) {
		return 0, 0, 0, 0, 0, ErrInvalidLength
	}
	n = int(n64)
	// Bound the make([]T, n) below by the same 256 MiB ceiling the columnar path
	// uses. A standalone block wire (top-level []int64) has no colMaxLen, so n is
	// otherwise capped only by the offset table (4 B/block) — constant sub-blocks
	// pack 256 elems into ~5 B, so a hostile tiny buffer could request a huge
	// slice. This caps the pre-block allocation regardless. (In the columnar path
	// colMaxLen already bounds n to the row count; this is redundant but cheap.)
	if checkColumnarBytes(n, 8) != nil {
		return 0, 0, 0, 0, 0, ErrInvalidLength
	}
	nBlocks = (n + blk - 1) / blk
	offBase = d.i
	if d.i+4*nBlocks > len(d.buf) {
		return 0, 0, 0, 0, 0, ErrShortBuffer
	}
	d.i += 4 * nBlocks
	bodyStart = d.i
	// Validate the offset table once: offsets[0]==0, strictly increasing, each
	// within the buffer. Selective and decode-all both rely on this.
	prev := -1
	for b := 0; b < nBlocks; b++ {
		off := int(binary.LittleEndian.Uint32(d.buf[offBase+4*b:]))
		if b == 0 && off != 0 {
			return 0, 0, 0, 0, 0, ErrInvalidLength
		}
		if off <= prev {
			return 0, 0, 0, 0, 0, ErrInvalidLength
		}
		if bodyStart+off > len(d.buf) {
			return 0, 0, 0, 0, 0, ErrShortBuffer
		}
		prev = off
	}
	return blk, n, offBase, bodyStart, nBlocks, nil
}

// blockLen returns the element count of block b (the last block holds the
// remainder).
func blockLen(b, nBlocks, n, blk int) int {
	if b == nBlocks-1 {
		return n - b*blk
	}
	return blk
}

// ---- decode-all (int64) ----

func (d *Decoder) readBlockInt64() ([]int64, error) {
	var s []int64
	if err := d.readBlockInt64Into(&s); err != nil {
		return nil, err
	}
	return s, nil
}

// readBlockInt64Into is the scratch-reusing form: it grows *dst to the column
// length and decodes every block into it. The tag is already consumed.
func (d *Decoder) readBlockInt64Into(dst *[]int64) error {
	// A block body could itself carry tagPackBlock or tagZoneChunk (both dispatched
	// by readQPackInt64); bound the nesting so a hostile payload of nested blocks
	// cannot overflow the goroutine stack.
	if err := d.descend(); err != nil {
		return err
	}
	defer d.ascend()
	blk, n, offBase, bodyStart, nBlocks, err := d.readBlockHeader(blockKindInt)
	if err != nil {
		return err
	}
	// Bound each inner sub-block's constant-codec count to the block length so a
	// malformed sub-block header cannot drive an oversized make() before the
	// per-block length check rejects it (the standalone path has colMaxLen==0).
	oldMax := d.colMaxLen
	d.colMaxLen = blk
	defer func() { d.colMaxLen = oldMax }()
	growI64(dst, n)
	out := *dst
	for b := range nBlocks {
		off := int(binary.LittleEndian.Uint32(d.buf[offBase+4*b:]))
		d.i = bodyStart + off
		t, err := d.peekTag()
		if err != nil {
			return err
		}
		v, err := d.readQPackInt64(t)
		if err != nil {
			return err
		}
		if len(v) != blockLen(b, nBlocks, n, blk) {
			return ErrInvalidLength
		}
		copy(out[b*blk:], v)
	}
	return nil
}

// ---- selective decode (predicate-matched rows only) ----

// decodeBlockColumnSelective decodes only the blocks of a block-tagged column
// that cover the matched row indices, scattering their values into a colVals of
// length n (matched positions filled, the rest left zero). matched must be
// ascending (matchedIndices guarantees this), so each needed block is decoded
// at most once and in wire order. The column starts at byte offset start;
// d.i is restored to its entry value on return. kind must be colKindInt or
// colKindUint and the column must be non-nullable (the caller gates this).
func (d *Decoder) decodeBlockColumnSelective(start int, kind colKind, n int, matched []int) (colVals, error) {
	save := d.i
	defer func() { d.i = save }()
	d.i = start + 1 // skip the tagPackBlock byte (recorded start points at it)

	expect := byte(blockKindInt)
	if kind == colKindUint {
		expect = blockKindUint
	}
	blk, hn, offBase, bodyStart, nBlocks, err := d.readBlockHeader(expect)
	if err != nil {
		return colVals{}, err
	}
	if hn != n {
		return colVals{}, ErrTypeMismatch
	}
	log := blkLogSmall
	if blk == blockSizeLarge {
		log = blkLogLarge
	}

	// Tighten the sub-block allocation bound to blk for the nested codec reads
	// (a constant sub-block otherwise inherits colMaxLen==n, the whole column).
	// The decode-all paths (readBlockInt64Into/readBlockUint64Into) do the same.
	oldMax := d.colMaxLen
	d.colMaxLen = blk
	defer func() { d.colMaxLen = oldMax }()

	cv := colVals{kind: kind}
	if kind == colKindInt {
		cv.i64 = make([]int64, n)
	} else {
		cv.u64 = make([]uint64, n)
	}

	mi := 0
	for mi < len(matched) {
		b := matched[mi] >> log
		if b >= nBlocks { // defensive; matched indices are < n so this cannot trip
			return colVals{}, ErrInvalidLength
		}
		off := int(binary.LittleEndian.Uint32(d.buf[offBase+4*b:]))
		d.i = bodyStart + off
		blockSelectiveBlocksDecoded.Add(1)
		t, err := d.peekTag()
		if err != nil {
			return colVals{}, err
		}
		want := blockLen(b, nBlocks, n, blk)
		bstart := b * blk
		if kind == colKindInt {
			v, err := d.readQPackInt64(t)
			if err != nil {
				return colVals{}, err
			}
			if len(v) != want {
				return colVals{}, ErrInvalidLength
			}
			for mi < len(matched) && matched[mi]>>log == b {
				cv.i64[matched[mi]] = v[matched[mi]-bstart]
				mi++
			}
		} else {
			v, err := d.readQPackUint64(t)
			if err != nil {
				return colVals{}, err
			}
			if len(v) != want {
				return colVals{}, ErrInvalidLength
			}
			for mi < len(matched) && matched[mi]>>log == b {
				cv.u64[matched[mi]] = v[matched[mi]-bstart]
				mi++
			}
		}
	}
	return cv, nil
}

// ---- decode-all (uint64) ----

func (d *Decoder) readBlockUint64() ([]uint64, error) {
	var s []uint64
	if err := d.readBlockUint64Into(&s); err != nil {
		return nil, err
	}
	return s, nil
}

func (d *Decoder) readBlockUint64Into(dst *[]uint64) error {
	if err := d.descend(); err != nil {
		return err
	}
	defer d.ascend()
	blk, n, offBase, bodyStart, nBlocks, err := d.readBlockHeader(blockKindUint)
	if err != nil {
		return err
	}
	// Bound each inner sub-block's constant-codec count to the block length (see
	// readBlockInt64Into).
	oldMax := d.colMaxLen
	d.colMaxLen = blk
	defer func() { d.colMaxLen = oldMax }()
	growU64(dst, n)
	out := *dst
	for b := range nBlocks {
		off := int(binary.LittleEndian.Uint32(d.buf[offBase+4*b:]))
		d.i = bodyStart + off
		t, err := d.peekTag()
		if err != nil {
			return err
		}
		v, err := d.readQPackUint64(t)
		if err != nil {
			return err
		}
		if len(v) != blockLen(b, nBlocks, n, blk) {
			return ErrInvalidLength
		}
		copy(out[b*blk:], v)
	}
	return nil
}
