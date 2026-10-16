package qdf

import (
	"sync"
	"unsafe"

	"github.com/alex60217101990/qdf/internal/fsst"
	"github.com/alex60217101990/qdf/internal/unsafestr"
)

const (
	// fsstMinElems: below this a per-column table cannot pay off.
	fsstMinElems = 16
	// fsstMaxDecompPerByte bounds decode expansion: each compressed byte
	// yields at most one symbol of ≤8 bytes.
	fsstMaxDecompPerByte = 8
	// fsstProbeRounds trains a coarser table for the columnar probe's size
	// estimate (a rough table is enough to decide columnar-vs-row-major); the
	// encoder still uses the full buildRounds for the table it emits. Keeps the
	// per-column probe training off the OptCompression hot path.
	fsstProbeRounds = 1
	// fsstReuseInterval is the number of consecutive batches that reuse the
	// cached FSST symbol table (fsstCache, per column) before retraining. Matches
	// retainReleaseStreak (8) so a steady workload holds its table across 8
	// batches before a full retrain checks whether the string distribution has
	// shifted. Only applies to streaming encoders (same Encoder reused across
	// calls); pool-backed Marshal resets the cache on every acquire.
	fsstReuseInterval = 8
)

// fsstCacheMax bounds the per-column table cache. A table is ~15 KB, so this
// caps what one encoder can pin; a message with more FSST columns than this
// simply trains the overflow fresh, which is what every column did before the
// cache existed.
const fsstCacheMax = 32

// strColScratch is the string-column picker's state, kept off the Encoder.
type strColScratch struct {
	// fsstCache holds the FSST tables trained on recent batches when no
	// user-supplied fsstDict is set, ONE PER COLUMN (keyed by fsstKey). Reused
	// across consecutive same-encoder batches to skip the per-batch Build cost
	// (~140–311 µs), retrained every fsstReuseInterval uses so a table follows a
	// drifting distribution. Keyed rather than single-slot because a single
	// slot handed column A's table to every later column of the same message:
	// measured +123% wire on the access corpus under OptCompression. Emptied by
	// Reset() on pool recycle; survives resetForReuse() so a streaming caller
	// amortizes training.
	fsstCache []fsstCacheEntry
	// fsstScratchTbl keeps a keyless column's freshly trained table past the
	// builder's return to the pool. fsstPicked / fsstDecompTotal carry the
	// measured block from measureStringColumnFSST to emitStringColumnFSST.
	fsstScratchTbl  *fsst.SymbolTable
	fsstPicked      *fsst.SymbolTable
	fsstDecompTotal int
	// alphaPlan carries a measured tagColStrAlpha block to its emit.
	alphaPlan alphaPlan
	// The rest of a measured tagColStrFrontDelta block (its length stream is
	// in Encoder.frontDeltaLens), carried to its emit.
	frontDeltaMid    int
	frontDeltaSize   int
	frontDeltaSuffix bool
}

// sc returns the picker state, allocating it on first use.
func (e *Encoder) sc() *strColScratch {
	if e.strCol == nil {
		e.strCol = new(strColScratch)
	}
	return e.strCol
}

// fsstCacheEntry is one column's cached table: key is the column's stable
// identity (columnarPlan.colIdentity), uses counts the reuses since the table was last trained.
type fsstCacheEntry struct {
	key  unsafe.Pointer
	tbl  *fsst.SymbolTable
	uses int
}

// fsstCacheSlot returns the cache entry for e.fsstKey, claiming a fresh one on
// first sight. nil when there is no key, or the cache is full.
func (e *Encoder) fsstCacheSlot() *fsstCacheEntry {
	k := e.fsstKey
	if k == nil {
		return nil
	}
	sc := e.sc()
	for i := range sc.fsstCache {
		if sc.fsstCache[i].key == k {
			return &sc.fsstCache[i]
		}
	}
	n := len(sc.fsstCache)
	if n == fsstCacheMax {
		return nil
	}
	if n < cap(sc.fsstCache) {
		// Reclaim a slot dropFSSTCache emptied: its tbl storage is reused by the
		// CloneInto that follows.
		sc.fsstCache = sc.fsstCache[:n+1]
	} else {
		sc.fsstCache = append(sc.fsstCache, fsstCacheEntry{})
	}
	ent := &sc.fsstCache[n]
	// uses at the interval forces the caller to train: a reclaimed slot still
	// holds the previous column's table (kept only for its storage).
	ent.key, ent.uses = k, fsstReuseInterval
	return ent
}

// fsstRetainSlots is how many slots keep their table storage across Reset. A
// table is ~15 KB (255 symbols of 32 B, 255 candidates of 24 B, a 1 KB index),
// so keeping all fsstCacheMax would let a pooled encoder pin ~480 KB that the
// encoder before this cache never held past Reset.
const fsstRetainSlots = 4

// dropFSSTCache forgets every cached table. The key goes so a recycled encoder
// never pairs one caller's column with another's table (fsstCacheSlot marks a
// reclaimed slot untrained); the storage stays for the first fsstRetainSlots
// slots, so their next training reuses it, and is released for the rest.
func (e *Encoder) dropFSSTCache() {
	sc := e.strCol
	if sc == nil {
		return // never used: nothing to drop, and nothing to allocate
	}
	all := sc.fsstCache[:cap(sc.fsstCache)]
	for i := range all {
		if i < fsstRetainSlots {
			all[i].key = nil
		} else {
			all[i] = fsstCacheEntry{}
		}
	}
	sc.fsstCache = sc.fsstCache[:0]
}

// fsstBuilderPool reuses FSST training scratch across the per-column probe
// estimates and encode attempts, so a column where FSST is evaluated but not
// chosen does not re-allocate the trainer's counter + first-byte index on every
// call (the source of the OptCompression encode-allocation regression).
var fsstBuilderPool = sync.Pool{New: func() any { return fsst.NewBuilder() }}

// fsstTablePool reuses SymbolTable objects across per-column decodes. Each
// pooled table retains its symbols/cands backing arrays; UnmarshalInto resets
// them and refills in-place, avoiding the three allocations that a fresh
// newSymbolTable call would otherwise make.
var fsstTablePool = sync.Pool{New: func() any { return &fsst.SymbolTable{} }}

// fsstEstimateSample is how many evenly spaced rows estimateStringColumnFSST
// trains and compresses. Big enough that the coarse table finds a column's
// common substrings, small enough to cost a few percent of a full Build.
const fsstEstimateSample = 64

// estimateStringColumnFSST predicts the tagColStrFSST block size for strs from
// a coarse table trained on an evenly spaced sample, scaling the sample's
// compressed bytes to the column. It is PESSIMISTIC on text — measured 1.1–2.6x
// the real block, because a table trained on 64 rows misses substrings that
// recur across the whole column — and close to exact on random IDs. The picker
// relies on that direction: it skips the full Build only when even this
// estimate cannot beat a codec already measured exactly.
func (e *Encoder) estimateStringColumnFSST(strs []string) int {
	n := len(strs)
	k := min(n, fsstEstimateSample)
	var sample [fsstEstimateSample][]byte
	for i := range k {
		sample[i] = unsafestr.Bytes(strs[i*n/k])
	}
	tbl := e.fsstDict
	var bld *fsst.Builder
	if tbl == nil {
		bld = fsstBuilderPool.Get().(*fsst.Builder)
		tbl = bld.BuildRounds(sample[:k], fsstProbeRounds)
	}
	body := 0
	for i := range k {
		cl := tbl.CompressedLen(sample[i])
		body += cl + uvarintLen(uint64(cl))
	}
	est := body*n/k + tbl.SerializedSize()
	if bld != nil {
		fsstBuilderPool.Put(bld)
	}
	return est
}

// measureStringColumnFSST trains (or reuses) a table for strs, compresses them
// into the encoder's FSST staging, and returns the exact block size. ok is false
// when FSST cannot beat the raw per-value size (sum(len)+n framing) — a sound
// floor because FSST is only reached on high-cardinality columns, where the
// per-value path is dominated by the raw bytes. On ok the staging and
// e.sc().fsstPicked stay valid until the next measure, for emitStringColumnFSST.
func (e *Encoder) measureStringColumnFSST(strs []string) (int, bool) {
	n := len(strs)
	if n < fsstMinElems {
		return 0, false
	}

	// A pre-trained dictionary (FSSTDict.Marshal) skips the per-batch training,
	// which is the dominant FSST encode cost; otherwise train on this column.
	tbl := e.fsstDict
	if tbl == nil {
		// Streaming cache: reuse the table this same column trained on a recent
		// batch to skip the Build() cost (~140–311 µs), retraining every
		// fsstReuseInterval uses so the table follows a drifting distribution.
		// The wire format is unchanged: the table is always fully serialized, so
		// caching is encoder-side only.
		ent := e.fsstCacheSlot()
		if ent != nil && ent.uses < fsstReuseInterval {
			tbl = ent.tbl
			ent.uses++
		} else {
			samples := e.state.fsstSamples[:0]
			for _, s := range strs {
				samples = append(samples, unsafestr.Bytes(s))
			}
			e.state.fsstSamples = samples
			// bld.Build returns a table aliasing bld's internal storage, which the
			// next Build on that Builder invalidates — and the builder goes back to
			// the pool before the emit — so the table is copied out, into storage
			// the encoder keeps (no allocation once warm).
			bld := fsstBuilderPool.Get().(*fsst.Builder)
			trained := bld.Build(samples)
			if ent != nil {
				ent.tbl = trained.CloneInto(ent.tbl)
				ent.uses = 0
				tbl = ent.tbl
			} else {
				e.sc().fsstScratchTbl = trained.CloneInto(e.sc().fsstScratchTbl)
				tbl = e.sc().fsstScratchTbl
			}
			fsstBuilderPool.Put(bld)
		}
	}

	// Compress all rows into one scratch buffer, recording per-row lengths.
	comp := e.state.fsstScratch[:0]
	compLens := e.state.fsstLens[:0]
	decompTotal := 0
	for _, s := range strs {
		before := len(comp)
		comp = tbl.Compress(unsafestr.Bytes(s), comp)
		compLens = append(compLens, len(comp)-before)
		decompTotal += len(s)
	}
	e.state.fsstScratch = comp
	e.state.fsstLens = compLens

	// Block size = tag + table + uvarint(n) + uvarint(decompTotal)
	//            + sum( uvarint(compLen) + compLen ). SerializedSize avoids
	//            materializing the table just to measure it.
	size := 1 + tbl.SerializedSize() + uvarintLen(uint64(n)) + uvarintLen(uint64(decompTotal))
	for _, cl := range compLens {
		size += uvarintLen(uint64(cl)) + cl
	}
	if size >= decompTotal+n {
		return 0, false
	}
	e.sc().fsstPicked = tbl
	e.sc().fsstDecompTotal = decompTotal
	return size, true
}

// emitStringColumnFSST writes the block the last successful
// measureStringColumnFSST staged, without compressing again.
func (e *Encoder) emitStringColumnFSST(n int) {
	comp, compLens := e.state.fsstScratch, e.state.fsstLens
	e.writeHeader()
	out := e.buf
	out = append(out, tagColStrFSST)
	out = e.sc().fsstPicked.MarshalTo(out) // serialize the table straight into the output
	out = appendUvarint(out, uint64(n))
	out = appendUvarint(out, uint64(e.sc().fsstDecompTotal))
	pos := 0
	for _, cl := range compLens {
		out = appendUvarint(out, uint64(cl))
		out = append(out, comp[pos:pos+cl]...)
		pos += cl
	}
	e.buf = out
	e.sc().fsstPicked = nil // the table may be a cache slot; do not hold it past the emit
}

// readStringColumnFSST decodes a tagColStrFSST block (tag at d.i) into n
// strings, all backed by a single per-column slab. Every length is validated
// before allocation.
func (d *Decoder) readStringColumnFSST(n int) ([]string, error) {
	d.i++ // consume tagColStrFSST
	tbl := fsstTablePool.Get().(*fsst.SymbolTable)
	defer fsstTablePool.Put(tbl)
	used, err := fsst.UnmarshalInto(d.buf[d.i:], tbl)
	if err != nil {
		return nil, err
	}
	d.i += used

	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	if int(n64) != n {
		return nil, ErrTypeMismatch
	}

	dt64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	rem := uint64(len(d.buf) - d.i)
	if dt64 > rem*fsstMaxDecompPerByte {
		return nil, ErrShortBuffer
	}
	// Hard cap: decompTotal must not exceed maxColumnarElems * maxSymLen (8).
	// This prevents a hostile varint from driving a huge slab alloc.
	if dt64 > uint64(maxColumnarElems)*8 {
		return nil, ErrShortBuffer
	}

	slab := make([]byte, 0, int(dt64))
	out := d.colStrScratch(n)
	for i := range n {
		cl64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return nil, ErrInvalidLength
		}
		d.i += nr
		if cl64 > uint64(len(d.buf)-d.i) {
			return nil, ErrShortBuffer
		}
		cl := int(cl64)
		start := len(slab)
		var ok bool
		// Bounded decode: never let the slab grow past its pre-sized capacity
		// (dt64), so a malformed row cannot trigger a transient over-allocation.
		slab, ok = tbl.DecompressN(d.buf[d.i:d.i+cl], slab, int(dt64))
		if !ok {
			return nil, ErrShortBuffer
		}
		d.i += cl
		out[i] = unsafestr.String(slab[start:])
	}
	return out, nil
}

// readStringColumnFSSTInto decodes a tagColStrFSST block (tag at d.i) DIRECTLY
// into the batch slab, writing a per-row Str handle into out[0:n]. It mirrors
// readStringColumnFSST's header parse and EVERY bounds guard verbatim (the
// dt64 caps, per-row compressed-length checks, and DecompressN's absolute-length
// limit), but instead of a temp scratch buffer it decompresses each row straight
// onto slab.buf — DecompressN appends in place — eliminating the ~dt64-byte temp
// alloc and the redundant second copy the general readStringColumnHandles path
// pays for FSST columns.
//
// Bounds parity is load-bearing here (this is decompress + unsafe slab writes):
// the slab reservation is bounded by dt64, which is itself bounded by the input;
// the slab's uint32 offset overflow guard (maxBatchSlabBytes, matching
// slab.append) is honored BEFORE any reservation. A malformed/hostile column
// errors — never OOM, never OOB write into the slab.
func readStringColumnFSSTInto(d *Decoder, n int, slab *batchSlab, out []Str) error {
	d.i++ // consume tagColStrFSST
	tbl := fsstTablePool.Get().(*fsst.SymbolTable)
	defer fsstTablePool.Put(tbl)
	used, err := fsst.UnmarshalInto(d.buf[d.i:], tbl)
	if err != nil {
		return err
	}
	d.i += used

	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return ErrInvalidLength
	}
	d.i += nr
	if int(n64) != n {
		return ErrTypeMismatch
	}

	dt64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return ErrInvalidLength
	}
	d.i += nr
	rem := uint64(len(d.buf) - d.i)
	if dt64 > rem*fsstMaxDecompPerByte {
		return ErrShortBuffer
	}
	// Hard cap: decompTotal must not exceed maxColumnarElems * maxSymLen (8).
	// This prevents a hostile varint from driving a huge slab reservation.
	if dt64 > uint64(maxColumnarElems)*8 {
		return ErrShortBuffer
	}

	// Honor the slab's uint32 offset cap BEFORE reserving: the reserved region
	// grows slab.buf by dt64, and a Str handle's offset must fit uint32. This is
	// the same overflow guard slab.append enforces (maxBatchSlabBytes) — a direct
	// write that would exceed it errors instead of silently wrapping the offset.
	base := len(slab.buf)
	if uint64(base)+dt64 > maxBatchSlabBytes {
		return ErrInvalidLength
	}
	// Reserve dt64 bytes ONCE so per-row DecompressN appends land in place with
	// no mid-column reallocation (offsets stay stable). limit is the ABSOLUTE
	// len(dst) cap DecompressN enforces; base+dt64 mirrors readStringColumnFSST's
	// int(dt64) cap on a slab that started empty.
	slab.grow(int(dt64))
	limit := base + int(dt64)
	for i := range n {
		cl64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		if cl64 > uint64(len(d.buf)-d.i) {
			return ErrShortBuffer
		}
		cl := int(cl64)
		start := len(slab.buf)
		var ok bool
		// Bounded decode: never let slab.buf grow past base+dt64, so a malformed
		// row cannot trigger a transient over-allocation or an OOB slab write.
		slab.buf, ok = tbl.DecompressN(d.buf[d.i:d.i+cl], slab.buf, limit)
		if !ok {
			return ErrShortBuffer
		}
		d.i += cl
		ln := len(slab.buf) - start
		if ln == 0 {
			// Match slab.append's empty-string convention: the zero handle.
			out[i] = Str{}
			continue
		}
		out[i] = Str{off: uint32(start), len: uint32(ln)}
	}
	return nil
}
