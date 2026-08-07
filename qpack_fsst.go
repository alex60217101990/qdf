package qdf

import (
	"sync"

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
	// cached FSST symbol table (fsstCachedTbl) before retraining. Matches
	// retainReleaseStreak (8) so a steady workload holds its table across 8
	// batches before a full retrain checks whether the string distribution has
	// shifted. Only applies to streaming encoders (same Encoder reused across
	// calls); pool-backed Marshal resets the cache on every acquire.
	fsstReuseInterval = 8
)

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

// tryWriteStringColumnFSST trains an FSST table on strs, compresses them, and
// emits a tagColStrFSST block — but only when the block is strictly smaller
// than the raw per-value size (sum(len)+n framing). It is invoked only after
// the dictionary codec has bailed (i.e. high-cardinality columns), where the
// per-value path is dominated by the raw bytes, so raw size is a sound
// never-larger baseline. Returns true iff the block was written.
func (e *Encoder) tryWriteStringColumnFSST(strs []string) bool {
	n := len(strs)
	if n < fsstMinElems {
		return false
	}

	// A pre-trained dictionary (FSSTDict.Marshal) skips the per-batch training,
	// which is the dominant FSST encode cost; otherwise train on this column.
	tbl := e.fsstDict
	if tbl == nil {
		// Streaming cache: reuse the table trained on a recent batch to skip
		// the Build() cost (~140–311 µs) when the same Encoder is reused across
		// consecutive batches. We retrain every fsstReuseInterval batches so the
		// table adapts to drifting distributions. The wire format is unchanged:
		// the table is always fully serialised (MarshalTo below) so the decoder
		// needs no protocol changes — caching is encoder-side only.
		if e.fsstCachedTbl != nil && e.fsstBatchCount < fsstReuseInterval {
			tbl = e.fsstCachedTbl
			e.fsstBatchCount++
		} else {
			// (Re)train: build via the pooled builder (reuses counter/cand scratch),
			// then clone to an independent copy before the builder is returned to
			// the pool. bld.Build returns a table aliasing bld's internal storage;
			// that alias is invalidated the next time any goroutine calls Build on
			// the same Builder. Clone() copies the three fields (symbols, cands,
			// first) into a fresh allocation that survives beyond this call.
			samples := e.state.fsstSamples[:0]
			for _, s := range strs {
				samples = append(samples, unsafestr.Bytes(s))
			}
			e.state.fsstSamples = samples
			bld := fsstBuilderPool.Get().(*fsst.Builder)
			defer fsstBuilderPool.Put(bld)
			tbl = bld.Build(samples)
			e.fsstCachedTbl = tbl.Clone() // independent copy; bld safe to pool
			e.fsstBatchCount = 0
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
	// Never-larger baseline: raw per-value bytes + one framing byte each.
	// decompTotal is the sum of len(s) over strs (accumulated in the encode
	// loop above), identical to a separate raw-total pass.
	if size >= decompTotal+n {
		return false
	}

	e.writeHeader()
	out := e.buf
	out = append(out, tagColStrFSST)
	out = tbl.MarshalTo(out) // serialize the table straight into the output
	out = appendUvarint(out, uint64(n))
	out = appendUvarint(out, uint64(decompTotal))
	pos := 0
	for _, cl := range compLens {
		out = appendUvarint(out, uint64(cl))
		out = append(out, comp[pos:pos+cl]...)
		pos += cl
	}
	e.buf = out
	return true
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
