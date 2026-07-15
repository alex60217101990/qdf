package qdf

import (
	"hash/maphash"
	"reflect"
	"strings"
	"unsafe"

	"github.com/alex60217101990/qdf/internal/internarena"
	"github.com/alex60217101990/qdf/internal/unsafestr"
)

// internHashSeed is the shared maphash seed for the flat intern
// table. Per-process random; values are stable across goroutines
// but differ across binary invocations (no semantic impact — the
// intern table is per-encoder and reset between Marshals).
var internHashSeed = maphash.MakeSeed()

// internKeyHash hashes a single map key for the order-independent key-set hash
// used by OptMapShape. Same seed/function as the intern table so distribution
// matches; callers combine per-key results commutatively (sum) so key order
// does not affect the set identity.
func internKeyHash(s string) uint64 {
	return maphash.String(internHashSeed, s)
}

// internSlot is one entry in the flat hash table that replaces the
// old map[string]uint32 intern dictionary. Profiling on telemetry
// workloads showed Go's map[string]uint32 spending ~17 ns/op in
// mapaccess2_faststr (string hash + bucket walk + memequal).
// Open-addressing on a contiguous []internSlot with the hash
// precomputed and stored alongside lets the hot path read one
// cache line, compare the hash, then `==` the key — saving ~5 ns
// per lookup at the cost of one extra 8 B slot field.
//
// hash == 0 reserves the "empty slot" sentinel; computed hashes
// that fall on 0 are bumped to 1 before storage.
//
// The key (a pointer-bearing string) leads so the GC pointer-scan range is
// 8 B instead of 16 B for a hash-first layout; the total stays 32 B:
//
//	key 16 B  +  hash 8 B  +  id 4 B  +  pad 4 B  =  32 B  →  2 slots / cache line
type internSlot struct {
	key  string
	hash uint64
	id   uint32
	_    uint32
}

// Initial intern table size. Doubles when load > 0.5 so the
// linear-probing chain stays short. 64 covers the common
// telemetry / config payloads (a few dozen distinct values) with
// no resize at all.
const internTableInitSize = 64

// Intern table backing Dense mode. The encoder maintains a string→ID
// map; the decoder maintains the matching ID-ordered list of byte
// slices. IDs are assigned in encode order starting at 0. There is no
// eviction — callers bound lifetime by resetting or recycling.

// pairPredK is the wire bound on the rank varuint emitted after
// tagStatePair. The current predictor stores top-1 only (K=1) so a
// hit always emits rank=0 and the decoder rejects any rank ≥ 1 as
// malformed. The constant stays around so the wire-side validation
// reads the same on both sides.
const pairPredK = 1

// Pair predictor storage: []uint32 indexed by prev intern ID. The
// stored value is `succ+1` so an empty slot is zero, which lets
// reset() use the runtime `clear()` builtin (a memclr) instead of a
// per-element loop. Sentinel choice trades 1 bit of representable
// range we never use (max intern id ≤ maxStateEntries < ^uint32(0)-1)
// for a cleaner reset path.
//
// Top-1 vs the previous K=4 ring: 4 bytes per slot down from 17, so
// at default maxStateEntries=16384 the predictor's residual capacity
// drops from ~280 KiB to 64 KiB per encoder (-77 %). Hit rate on
// strictly cyclic workloads (A→{B,C,D,E,B,C,…}) falls to zero, but
// on the common stable-transition case (A→B→A→B) top-1 catches every
// hit the ring did. Real telemetry workloads sit close to the
// stable case.
const pairPredEmpty uint32 = 0

// mruRingSize is the side-cache covering recent emit history for fast
// MTF rank discovery. Scanning a contiguous 128-entry uint16 ring is
// dominated by sequential L1 hits (cache-prefetcher friendly) and
// completes in roughly N×1.5ns. The canonical LRU linked list still
// holds every ID; the ring just shortcuts the rank-walk on the hot
// path. 128 covers the full 1-byte rank space (uvarintLen(rank)==1
// for rank ≤ 127), which is exactly the range where MTF beats the
// raw 2-byte state-ref encoding most ids land in.
//
// uint16 not uint32: the encoder caps maxStateEntries at 1<<14 (=
// 16 384), well under the uint16 range. Halving the slot width packs
// the whole ring into 256 B (4 × 64 B cache lines) instead of 512 B
// — a sequential scan reads half as many lines and finishes faster
// on the same prefetcher budget. mruEmpty (0xFFFF) is reserved as
// the "no id here" sentinel; the runtime id space stays below it.
//
// Power of two so the modulo collapses to a single AND.
const (
	mruRingSize        = 128
	mruRingMask        = mruRingSize - 1
	mruEmpty    uint16 = 0xFFFF
)

type encState struct {
	// Hot scalars first so they share a single cache line with the
	// adjacent mruHead and the map header. lastID + lruHead + mruHead
	// are touched on every state-ref emit; co-locating them with
	// mruRing keeps the per-emit footprint at 1-2 cache lines.
	lastID     uint32
	lruHead    uint32
	mruHead    uint32
	internLoad uint32 // number of occupied slots in internTable

	// mruRing is a side-cache of the last mruRingSize state-ref emit
	// IDs. Each new emit pushes its ID at mruRing[mruHead&mask] and
	// bumps mruHead. Rank discovery scans backwards from mruHead: a
	// hit at offset r means r OTHER ids were emitted after this one,
	// which is exactly the LRU chain rank. Storing the ring as uint16
	// (id space fits) cuts the scan footprint to 256 B / 4 cache
	// lines.
	mruRing [mruRingSize]uint16

	// internTable is a flat open-addressed hash table replacing the
	// old map[string]uint32. See internSlot for layout. Hot —
	// touched on every WriteString. Linear probing keeps the access
	// pattern sequential and predictable; load is kept below 0.5 by
	// doubling on growth so probe chains stay short (typically 1).
	internTable []internSlot

	// Move-to-front LRU over intern IDs. lruHead is the ID at rank 0
	// (most recently emitted state-ref or freshly interned). lruLink
	// packs the previous + next ids for each chain slot into a single
	// uint32 (low 16 bits = prev, high 16 bits = next). IDs are
	// bounded by maxStateEntries (1 << 14) so they fit in 16 bits
	// with 0xFFFF reserved as the "no neighbour" sentinel. Packing
	// halves the cache lines a lruMoveFront has to touch (one array
	// instead of two) while keeping the unlink/insert update O(1).
	lruLink []uint32

	// Pair predictor: top-1 successor per prev intern ID. Slot stores
	// `succ+1` so zero = empty; this keeps Reset() on a memclr fast
	// path while preserving the ability to predict succ==0 cleanly.
	// See the pairPredEmpty / pairPredK constants and the
	// pairLookup / pairRecord methods for the lookup contract.
	pairPred []uint32

	// Shape table for tagMapShape. shapes is indexed by id-1 (id 0 is
	// reserved on the wire to mean "declare"). shapeBindings is a
	// small linear-scan registry of (typeDesc → wire ID). Typical
	// streams emit a handful of struct types, so a slice beats a map
	// on lookup cost AND keeps the race-detector instrumentation off
	// the hot path. lastShapeTd / lastShapeID memoise the
	// most-recent successful shape lookup; shapeCount is the running
	// total of declared shape IDs.
	lastShapeTd   *typeDesc
	lastShapeID   uint32
	shapeCount    uint32
	shapeBindings []shapeBinding

	// tokenShapes interns recurring struct shapes for code-generated types,
	// which have no *typeDesc — keyed on a stable per-type token address the
	// generated EncodeQDF passes (see Encoder.StructShape). Shares the shapeCount
	// ID space with shapeBindings / mapShapes so the decoder's single shape table
	// stays in lockstep. lastTokenPtr/lastTokenID memoise the most recent lookup
	// so a homogeneous slice (the common case: one token, N elements threaded
	// through one encoder) hits a single pointer compare, mirroring lastShapeTd.
	tokenShapes  []tokenShape
	lastTokenPtr *byte
	lastTokenID  uint32
	// lastMapShapeID memoises the most recently used map shape so a run of
	// homogeneous rows (the common case) verifies against it directly — no
	// set-hash recompute, no registry scan (see lastMapShapeKeys). 0 means none.
	// Packed adjacent to lastTokenID so the two cold uint32 memo IDs share one
	// 8-byte word instead of each stranding a 4-byte pad before the next slice.
	lastMapShapeID uint32

	// mapShapes interns recurring map key-sets (OptMapShape), parallel to
	// shapeBindings but keyed on the key-set rather than a *typeDesc. Shares
	// the shapeCount ID space with struct shapes (shapeDeclareEnc) so the
	// decoder's single shape table stays in lockstep.
	mapShapes []mapShapeBinding
	// lastMapShapeKeys holds the key order of the shape memoised by
	// lastMapShapeID, so a homogeneous run skips the registry scan.
	lastMapShapeKeys []string
	// mapEnc pools reflect holders for the generic (reflect) string-keyed map
	// encode path so it does not reflect.New per map (OptMapShape).
	mapEnc mapHolderCache

	// Columnar shape table (tagColStruct). Separate from shapeBindings
	// because columnar shapes carry field kinds. Keyed by structural
	// identity (names + kinds) since the same struct type always produces
	// the same columnar shape.
	colShapeNames [][]string
	colShapeKinds [][]colKind
	// Hybrid columnar shape table (tagHybridColStruct). Separate ID space from
	// colShapeNames/colShapeKinds so hybrid and pure-columnar shapes never alias
	// within a stream. kinds carry residualKind (0xFF) for residual fields.
	hybridShapeNames [][]string
	hybridShapeKinds [][]colKind
	// Pooled transpose scratch, reused across columns and across calls.
	colScratchI64  []int64
	colScratchU64  []uint64
	colScratchF64  []float64
	colScratchBool []bool
	colScratchF32  []float32 // codegen columnar float32 gather scratch
	colScratchStr  []string  // gathered string column values
	// Canonical map-key sort scratch (OptCanonical), reused across maps; adaptive
	// retention (dropped when oversized in the encoder pool reset, like
	// colScratch*). Pointer-free numeric scratch needs no clear; canonKeysStr
	// holds caller string headers, so it is cleared on reset to drop references.
	canonKeysStr []string
	canonKeysI64 []int64
	canonKeysU64 []uint64
	// Canonical float-slice normalization scratch (OptCanonical): when a
	// []float64/[]float32 contains -0.0 or NaN, the normalized copy lands here
	// (never mutating the caller's slice). Pointer-free, adaptive retention.
	canonFloat64 []float64
	canonFloat32 []float32
	// Columnar column-level diff scratch (delta_columnar.go). deltaColBitmap is
	// the per-row changed bitmap; deltaColRows the changed-row indices for one
	// column; deltaColBuf the built tagColSlicePatch body for the never-larger
	// compare. Row-scaled, retained/dropped like the colScratch* above.
	deltaColBitmap []uint64
	deltaColRows   []int
	deltaColBuf    []byte
	// deltaColAux* hold the OLD column gathered contiguously for the arithmetic
	// delta encode (new − old), so both operands are width-hoisted gathers fed to
	// a vectorizable subtract instead of two per-element width switches.
	deltaColAuxI64 []int64
	deltaColAuxU64 []uint64
	colDictTable   []string // distinct table for the string-dict codec
	colMaskScratch []byte   // presence bitmap for nullable columns
	// FSST codec scratch, reused across columns (same lifetime as colDictTable).
	fsstScratch []byte   // compressed bytes for all rows, concatenated
	fsstLens    []int    // per-row compressed lengths
	fsstSamples [][]byte // per-column []byte views fed to the FSST trainer
	// strDictMap maps a string column's distinct values to dense indices
	// while the string-dict codec decides/encodes. Reused (cleared) per
	// column to avoid a per-column map allocation.
	strDictMap map[string]uint32

	// canonKeysBusy guards the pooled canonKeys* scratch against re-entrancy: a
	// map whose values contain maps recurses into the gather mid-iteration and
	// would clobber the outer map's sorted-key slice. When busy, the gather falls
	// back to a fresh local slice (like mapHolderCache.busy). Flat maps — the
	// common case — keep the zero-alloc pooled path. Packed next to retainStreak
	// (both cold 1-byte flags) so the two share one word instead of each
	// stranding a 7-byte pad before the following 8-byte field.
	canonKeysBusy bool

	// retainStreak counts consecutive small (sub-cap) messages for the
	// adaptive-retention policy in reset(). Cold — touched once per reset.
	retainStreak uint8

	// arenaSmallStreak is the arena's own analogue of retainStreak, driven by
	// the byte volume interned per message (internarena.DefaultRetainBytes)
	// rather than the intern-id count (maxRetainedIDs). The two signals diverge:
	// a batch can intern < maxRetainedIDs distinct keys yet still push the arena
	// past its byte cap (long keys), so reusing retainStreak would shed the
	// arena's spike slabs every steady batch in that band. Cold — touched once
	// per reset. Packs into the same word as the two flags above (no size cost).
	arenaSmallStreak uint8

	// arena owns the byte storage that backs every intern key the
	// encoder allocates — accessed only on intern miss, kept at the
	// end so the hot fields above share earlier cache lines.
	arena internarena.Arena
}

// shapeBinding is a (typeDesc → wire shape ID) pair. Stored in a
// linear-scan slice on the encoder state so the hot path adds no map
// access (and no allocation under -race).
type shapeBinding struct {
	td *typeDesc
	id uint32
}

// tokenShape is a (per-type token address → wire shape ID) pair, the
// code-generated analogue of shapeBinding (which keys on *typeDesc). The token
// is a stable package-level address the generated EncodeQDF passes; a linear
// scan keeps the hot path map-free and allocation-free under -race.
type tokenShape struct {
	token *byte
	id    uint32
}

// mapShapeBinding maps a recurring map key-set to a shared shape ID.
// setHash is an order-independent hash of the (string) keys; n is the key
// count (disambiguates a setHash collision across different sizes); keys holds
// the canonical (sorted) key order, cloned so it survives the caller's map. id
// is drawn from the same sequential space as struct shapeBindings
// (shapeDeclareEnc).
type mapShapeBinding struct {
	keys    []string
	setHash uint64
	n       int
	id      uint32
}

// mapHolderCache pools the addressable reflect.Value scratch the generic
// (reflect) map encode/decode path needs — a key holder + a value holder —
// so it does not reflect.New on every map encoded/decoded. Reused across the
// rows of a []struct (same map type every row), so a 1000-row batch pays 2
// reflect.New total instead of 2 per row. A busy flag keeps it
// re-entrancy-safe: a nested map (e.g. map[string]map[string]T), or any
// acquire while the cache is already in use, falls back to a fresh local pair.
type mapHolderCache struct {
	kt, vt reflect.Type
	vp     unsafe.Pointer
	kh, vh reflect.Value
	busy   bool
}

func (c *mapHolderCache) acquire(kt, vt reflect.Type) (kh, vh reflect.Value, vp unsafe.Pointer, pooled bool) {
	if c.busy {
		vh = reflect.New(vt).Elem()
		return reflect.New(kt).Elem(), vh, unsafe.Pointer(vh.UnsafeAddr()), false
	}
	if c.kt != kt || c.vt != vt {
		c.kt, c.vt = kt, vt
		c.kh = reflect.New(kt).Elem()
		c.vh = reflect.New(vt).Elem()
		c.vp = unsafe.Pointer(c.vh.UnsafeAddr())
	}
	c.busy = true
	return c.kh, c.vh, c.vp, true
}

func (c *mapHolderCache) release(pooled bool) {
	if pooled {
		c.busy = false
	}
}

const lruInvalidID = ^uint32(0)

// maxInternEntries is the hard ceiling on the intern-table size. Intern ids are
// packed into uint16 fields in the MRU ring and the LRU prev/next links, with
// 0xFFFF reserved as the "empty / no-neighbour" sentinel (mruEmpty,
// lruLink16Invalid). The largest assignable id must therefore stay below 0xFFFF.
// The assign gate is `internLoad < maxStateEntries`, so capping maxStateEntries
// at 0xFFFF yields a max id of 0xFFFE — one below the sentinel. A larger cap
// would let id 0xFFFF collide with the sentinel and corrupt the LRU/MRU chains
// (silent wrong-string resolution on later state-refs).
const maxInternEntries = 0xFFFF

func newEncState() *encState {
	// arena is zero-value initialised here — its slab is lazily
	// allocated on first Put (see internarena.Arena.Put).
	e := &encState{
		internTable: make([]internSlot, internTableInitSize),
		lruHead:     lruInvalidID,
		lastID:      lruInvalidID,
	}
	// Prime ring with sentinels so a scan never matches id 0 by
	// accident before the ring has been written.
	for i := range e.mruRing {
		e.mruRing[i] = mruEmpty
	}
	return e
}

// Soft caps on per-encoder state retention across Reset(). A single
// payload that pushes any of these past their threshold is dropped
// rather than pinned to the pooled encoder forever — long-running
// services with bursty traffic keep a bounded resident set instead
// of growing to peak and staying there.
//
// Numbers picked so that a "typical" telemetry batch (≤ a few
// thousand interned strings, ≤ a few thousand state-ref ids) stays
// under every cap; only outlier payloads trigger the shrink.
const (
	maxRetainedIDs      = 4096
	maxRetainedLRUCap   = 4096
	maxRetainedPairCap  = 4096
	maxRetainedShapeCap = 1024

	// retainReleaseStreak governs adaptive retention (see encState.reset /
	// decState.reset). A pooled state RETAINS oversized backing arrays —
	// clearing them in place instead of dropping them to nil — while
	// consecutive messages stay large, so a steady high-cardinality /
	// large-batch workload amortizes the table allocation instead of
	// reallocating every single message (the dominant encode-alloc cost on
	// AD/log/telemetry batches). Only after this many consecutive SMALL
	// (sub-cap) messages does it conclude the burst subsided and release the
	// memory. sync.Pool's GC-driven eviction bounds idle retention regardless.
	retainReleaseStreak = 8

	// maxRetainedColScratch hard-caps the row-count-scaled columnar scratch
	// arrays (colScratch*, colDictTable, colMaskScratch, fsstScratch): a
	// backing larger than this is dropped, bounding worst-case pooled memory
	// after a one-off giant columnar batch. 1<<17 (131072 rows) covers any
	// realistic batch while capping the per-encoder pin at ~2 MB for the
	// []string scratch (16 B/elem). The intern/LRU/pair arrays need no such
	// ceiling — maxStateEntries (≤ 0xFFFF) already bounds them to ~1-4 MB.
	maxRetainedColScratch = 1 << 17
)

func (e *encState) reset() {
	// Adaptive retention. A pooled encoder that just handled a large
	// (high-cardinality / wide-batch) message would, under a fixed cap, drop
	// its grown backing arrays and reallocate them from scratch on the very
	// next message — so a STEADY large workload (AD / log / telemetry sync)
	// never amortizes the table allocation and pays a full table regrow every
	// message (the dominant encode-alloc cost measured on such batches).
	// Instead we keep the intern/LRU/pair/shape backings (clearing them in
	// place) while messages stay large, releasing only after
	// retainReleaseStreak consecutive small messages. The decision is driven
	// by internLoad — the count of strings interned THIS message, a true
	// per-message demand signal (the retained cap would stay large forever
	// and never let the streak advance). The row-scaled columnar scratch is
	// governed separately by a hard ceiling (maxRetainedColScratch): retained
	// up to it, dropped above, so a one-off giant batch can't pin unbounded
	// memory. The intern/LRU/pair arrays are already bounded by
	// maxStateEntries; sync.Pool's GC eviction caps idle retention regardless.
	if int(e.internLoad) > maxRetainedIDs {
		e.retainStreak = 0
	} else if e.retainStreak < retainReleaseStreak {
		e.retainStreak++
	}
	release := e.retainStreak >= retainReleaseStreak

	// Intern table. Clear (zero every slot — internSlot{} is the empty
	// sentinel) to reuse in place; drop the backing only when releasing.
	//
	// Order: clear / rebuild BEFORE arena.Reset. The slot.key fields alias
	// arena bytes; arena.Reset rolls cursors back and the next Put overwrites
	// the prior payload area, so any surviving aliased key would read garbage.
	if cap(e.internTable) > maxRetainedIDs*2 && release {
		e.internTable = make([]internSlot, internTableInitSize)
	} else {
		clear(e.internTable)
	}
	e.internLoad = 0
	// Adaptive arena retention. The default Reset() soft cap (256 KiB) drops the
	// spike slabs a high-cardinality AD/log batch just grew, forcing a full arena
	// regrow every batch — the dominant streaming-encode allocation (~181 KB/value
	// measured on high-card AD data, where each per-batch StreamEncoder.Reset
	// sheds and the next batch rebuilds the slabs). While a steady large-volume
	// workload keeps filling the arena past the cap, retain every slab —
	// ResetWithLimit(0) rolls the cursor back to chunks[0] and keeps the spike
	// chunks so the next same-shaped batch reuses them in place. Resident memory
	// stays bounded by the single-batch peak (the cursor resets each batch and
	// grow() walks the existing slabs before allocating).
	//
	// The trigger is the arena's OWN per-message byte demand (BytesPut, the
	// payload volume interned THIS batch — an O(1) counter, read before the
	// cursor rolls back), not the intern-id streak above: a batch can intern
	// fewer than maxRetainedIDs keys yet still exceed the byte cap with long
	// keys, so the id-based `release` would shed the arena's slabs every steady
	// batch in that band. Only after retainReleaseStreak consecutive sub-cap
	// batches (burst genuinely subsided) do we fall back to the default cap and
	// shed the spike memory, so a one-shot/bursty pool encoder still bounds its
	// resident set. Under a sustained streak the resident set is bounded by the
	// streak-window peak (plus doubling slack), reclaimed by the sub-cap shed and
	// sync.Pool's GC eviction. The intern table was already cleared above,
	// dropping every aliased slot.key, so rolling the cursor back is safe at any
	// retain limit.
	if e.arena.BytesPut() > internarena.DefaultRetainBytes {
		e.arenaSmallStreak = 0
	} else if e.arenaSmallStreak < retainReleaseStreak {
		e.arenaSmallStreak++
	}
	if e.arenaSmallStreak >= retainReleaseStreak {
		e.arena.Reset() // default 256 KiB soft cap — burst subsided, shrink.
	} else {
		e.arena.ResetWithLimit(0) // large-volume streak: keep slabs warm.
	}

	e.lastID = lruInvalidID
	e.lruHead = lruInvalidID
	if cap(e.lruLink) > maxRetainedLRUCap && release {
		e.lruLink = nil
	} else {
		e.lruLink = e.lruLink[:0]
	}

	// pairPred slice: clear in place (memclr-fast because the empty sentinel
	// is zero); drop the backing array only when releasing.
	if cap(e.pairPred) > maxRetainedPairCap && release {
		e.pairPred = nil
	} else {
		clear(e.pairPred)
	}

	e.shapeCount = 0
	if cap(e.shapeBindings) > maxRetainedShapeCap && release {
		e.shapeBindings = nil
	} else {
		e.shapeBindings = e.shapeBindings[:0]
	}
	e.lastShapeTd = nil
	e.lastShapeID = 0
	if cap(e.tokenShapes) > maxRetainedShapeCap && release {
		e.tokenShapes = nil
	} else {
		e.tokenShapes = e.tokenShapes[:0]
	}
	e.lastTokenPtr = nil
	e.lastTokenID = 0
	if cap(e.mapShapes) > maxRetainedShapeCap && release {
		e.mapShapes = nil
	} else {
		// Each binding holds keys []string aliasing caller strings; a bare [:0]
		// leaves those headers live in the backing array and pins the strings
		// until overwritten. Clear the FULL backing first (mirrors colDictTable).
		clear(e.mapShapes[:cap(e.mapShapes)])
		e.mapShapes = e.mapShapes[:0]
	}
	e.lastMapShapeID = 0
	e.lastMapShapeKeys = nil
	e.mapEnc = mapHolderCache{}

	if cap(e.colShapeNames) > maxRetainedShapeCap && release {
		e.colShapeNames = nil
		e.colShapeKinds = nil
	} else {
		e.colShapeNames = e.colShapeNames[:0]
		e.colShapeKinds = e.colShapeKinds[:0]
	}
	if cap(e.hybridShapeNames) > maxRetainedShapeCap && release {
		e.hybridShapeNames = nil
		e.hybridShapeKinds = nil
	} else {
		e.hybridShapeNames = e.hybridShapeNames[:0]
		e.hybridShapeKinds = e.hybridShapeKinds[:0]
	}
	// Row-scaled columnar scratch: retained across batches (amortizes a steady
	// columnar workload, whose row count is independent of internLoad), dropped
	// only past the hard ceiling so a one-off giant batch can't pin unbounded
	// memory. sync.Pool's GC eviction reclaims it when the encoder goes idle.
	// Each backing grows on its own per-column-type demand (a batch of only
	// float64 columns grows colScratchF64 while colScratchI64 stays at 0), so
	// gate each independently — a single check keyed on colScratchI64 would miss
	// an oversized U64/F64/Bool backing and pin double the intended scratch (same
	// class as the deltaColAux* fix below).
	if cap(e.colScratchI64) > maxRetainedColScratch {
		e.colScratchI64 = nil
	}
	if cap(e.colScratchU64) > maxRetainedColScratch {
		e.colScratchU64 = nil
	}
	if cap(e.colScratchF64) > maxRetainedColScratch {
		e.colScratchF64 = nil
	}
	if cap(e.colScratchBool) > maxRetainedColScratch {
		e.colScratchBool = nil
	}
	if cap(e.colScratchF32) > maxRetainedColScratch {
		e.colScratchF32 = nil
	}
	// deltaColAux* are swapped with colScratchI64/U64 in encodeDeltaColumn, so
	// after a large columnar-delta batch one of the two grown backings lives here
	// and is missed by the colScratchI64 check above — cap them independently or
	// the pooled encoder retains double the intended scratch.
	if cap(e.deltaColAuxI64) > maxRetainedColScratch {
		e.deltaColAuxI64 = nil
	}
	if cap(e.deltaColAuxU64) > maxRetainedColScratch {
		e.deltaColAuxU64 = nil
	}
	// Column-diff scratch (delta_columnar.go): same row-scaled hard-ceiling
	// retention as colScratch* (pointer-free, no clear needed).
	if cap(e.deltaColBitmap) > maxRetainedColScratch {
		e.deltaColBitmap = nil
	}
	if cap(e.deltaColRows) > maxRetainedColScratch {
		e.deltaColRows = nil
	}
	if cap(e.deltaColBuf) > maxRetainedColScratch {
		e.deltaColBuf = nil
	}
	if cap(e.fsstScratch) > maxRetainedColScratch {
		e.fsstScratch = nil
		e.fsstLens = nil
	}
	// fsstSamples holds []byte views into caller strings; clear the headers to
	// drop those references across a pool recycle, drop backing past the ceiling.
	if cap(e.fsstSamples) > maxRetainedColScratch {
		e.fsstSamples = nil
	} else {
		clear(e.fsstSamples)
		e.fsstSamples = e.fsstSamples[:0]
	}
	// String-column scratch: []string slices retain header references that pin
	// the caller's string memory across a pool recycle. clear() drops those
	// headers while keeping the backing; drop the backing only past the ceiling.
	if cap(e.colScratchStr) > maxRetainedColScratch {
		e.colScratchStr = nil
	} else {
		// Clear across the FULL backing, not just len: the gather reslices via
		// [:0] per column, so a column with fewer rows than an earlier one leaves
		// a high-water tail of headers aliasing the caller's (now possibly dead)
		// struct strings, pinning them from GC across the pool recycle.
		clear(e.colScratchStr[:cap(e.colScratchStr)])
		e.colScratchStr = e.colScratchStr[:0]
	}
	// Canonical map-key sort scratch (OptCanonical): numeric scratch is
	// pointer-free (drop only past the ceiling); canonKeysStr holds caller
	// string headers, so clear them to drop references across a pool recycle.
	// canonKeysI64/U64 grow independently (a uint64-key-map workload grows
	// canonKeysU64 while canonKeysI64 stays empty), so gate each on its own cap —
	// a single check keyed on canonKeysI64 would never drop an oversized
	// canonKeysU64. Both are pointer-free: drop only past the ceiling.
	if cap(e.canonKeysI64) > maxRetainedColScratch {
		e.canonKeysI64 = nil
	}
	if cap(e.canonKeysU64) > maxRetainedColScratch {
		e.canonKeysU64 = nil
	}
	if cap(e.canonKeysStr) > maxRetainedColScratch {
		e.canonKeysStr = nil
	} else {
		// Clear across the FULL backing, not just len: a shorter key-set leaves a
		// high-water tail of headers aliasing the caller's map key strings.
		clear(e.canonKeysStr[:cap(e.canonKeysStr)])
		e.canonKeysStr = e.canonKeysStr[:0]
	}
	// Canonical float-slice scratch is pointer-free: drop only past the ceiling.
	// canonFloat64/canonFloat32 grow independently (a dirty []float32 workload
	// grows canonFloat32 while canonFloat64 stays empty), so gate each on its own
	// cap — a single check keyed on canonFloat64 would never drop an oversized
	// canonFloat32.
	if cap(e.canonFloat64) > maxRetainedColScratch {
		e.canonFloat64 = nil
	}
	if cap(e.canonFloat32) > maxRetainedColScratch {
		e.canonFloat32 = nil
	}
	if cap(e.colDictTable) > maxRetainedColScratch {
		e.colDictTable = nil
	} else {
		// []string resliced via [:0] per column, so a shorter column leaves a
		// high-water tail of headers aliasing caller strings; clear the FULL
		// backing (mirrors the decoder's colDictTableScr reset).
		clear(e.colDictTable[:cap(e.colDictTable)])
		e.colDictTable = e.colDictTable[:0]
	}
	if cap(e.colMaskScratch) > maxRetainedColScratch {
		e.colMaskScratch = nil
	} else {
		e.colMaskScratch = e.colMaskScratch[:0]
	}
	if len(e.strDictMap) > 0 {
		clear(e.strDictMap)
	}

	// Ring side-cache: re-prime with sentinels so post-reset emits
	// can't false-match a stale id 0.
	for i := range e.mruRing {
		e.mruRing[i] = mruEmpty
	}
	e.mruHead = 0
}

// shapeForType returns the wire shape ID bound to t in this encoder's
// state, or 0 if none. Pair with shapeBindType after a declaration.
//
//go:nosplit
func (e *encState) shapeForType(t *typeDesc) uint32 {
	if e.lastShapeTd == t && e.lastShapeID != 0 {
		return e.lastShapeID
	}
	for i := range e.shapeBindings {
		if e.shapeBindings[i].td == t {
			id := e.shapeBindings[i].id
			e.lastShapeTd = t
			e.lastShapeID = id
			return id
		}
	}
	return 0
}

func (e *encState) shapeBindType(t *typeDesc, id uint32) {
	e.shapeBindings = append(e.shapeBindings, shapeBinding{td: t, id: id})
	e.lastShapeTd = t
	e.lastShapeID = id
}

// shapeForToken returns the wire shape ID bound to a code-generated type's token
// address in this encoder's state, or 0 if none. Pair with shapeBindToken after
// a declaration.
func (e *encState) shapeForToken(token *byte) uint32 {
	if e.lastTokenPtr == token && e.lastTokenID != 0 {
		return e.lastTokenID
	}
	for i := range e.tokenShapes {
		if e.tokenShapes[i].token == token {
			e.lastTokenPtr = token
			e.lastTokenID = e.tokenShapes[i].id
			return e.tokenShapes[i].id
		}
	}
	return 0
}

func (e *encState) shapeBindToken(token *byte, id uint32) {
	e.tokenShapes = append(e.tokenShapes, tokenShape{token: token, id: id})
	e.lastTokenPtr = token
	e.lastTokenID = id
}

// shapeDeclareEnc reserves the next sequential wire ID and returns
// it. Caller emits the keys on the wire; this side only tracks the
// count to keep IDs aligned with the decoder.
func (e *encState) shapeDeclareEnc() uint32 {
	e.shapeCount++
	return e.shapeCount
}

// mapShapeRegister binds a key-set to a shape ID. keys must be the canonical
// (sorted) order and is taken over by the binding — the caller must not reuse or
// mutate it afterward. Both callers pass a freshly-allocated per-declare slice
// (the strings alias the caller's map keys, exactly as a clone would have), so
// ownership transfer drops one []string allocation per first-sight key-set.
func (e *encState) mapShapeRegister(setHash uint64, n int, keys []string, id uint32) {
	e.mapShapes = append(e.mapShapes, mapShapeBinding{setHash: setHash, n: n, keys: keys, id: id})
}

// pairLookup reports whether the top-1 predicted successor of prev
// is curr. The wire emits a rank byte after tagStatePair that is
// always 0 in the top-1 design — callers hand-write the literal
// instead of consuming a rank return value.
//
//go:nosplit
func (e *encState) pairLookup(prev, curr uint32) bool {
	if int(prev) >= len(e.pairPred) {
		return false
	}
	return e.pairPred[prev] == curr+1
}

// pairEnsure grows the predictor slice so prev is a valid index. New
// slots default to pairPredEmpty (zero) via the runtime's append
// zero-fill — no extra initialisation needed.
//
//go:nosplit
func (e *encState) pairEnsure(prev uint32) {
	for uint32(len(e.pairPred)) <= prev {
		e.pairPred = append(e.pairPred, pairPredEmpty)
	}
}

// pairRecord installs curr as the top-1 successor of prev. Always
// overwrites — the predictor remembers the most recent transition
// only.
//
//go:nosplit
func (e *encState) pairRecord(prev, curr uint32) {
	e.pairEnsure(prev)
	e.pairPred[prev] = curr + 1
}

// mruPush records id as the newest entry in the side-cache ring.
// Overwrites the slot at mruHead and advances the head. The ring is
// power-of-two sized so the modulo collapses to an AND. Caller
// guarantees id < mruEmpty (maxStateEntries cap ensures this).
//
//go:nosplit
func (e *encState) mruPush(id uint32) {
	e.mruRing[e.mruHead&mruRingMask] = uint16(id)
	e.mruHead++
}

// mruRank scans the ring from newest to oldest looking for id. If
// found at offset r from the head, r equals the current LRU chain
// rank (since every state-ref emit is recorded in the ring in
// order). Returns (rank, true) on hit, (0, false) when id is not in
// the last mruRingSize emissions — in which case the caller falls
// back to the raw state-ref encoding (chain rank is necessarily
// ≥ mruRingSize and would need a multi-byte varuint anyway).
//
// Hand-unrolled 4-way: profiling on telemetry workloads showed the
// scalar loop at ~17 % flat (top hotspot post the May 2026
// series). The unroll amortises the back-edge branch, lets the CPU
// issue 4 independent loads per iteration, and keeps the typical
// low-rank early-exit semantics. Falls back to scalar for the
// final partial iteration when mruRingSize is not a multiple of 4
// (it is at 128, but the guard keeps the function correct under
// future ring-size changes).
//
//go:nosplit
func (e *encState) mruRank(id uint32) (uint32, bool) {
	// IDs above the uint16 representable range can never appear in
	// the uint16 ring; short-circuit so an oversized id (only
	// reachable if maxStateEntries was bumped) never false-hits the
	// mruEmpty sentinel.
	if id >= uint32(mruEmpty) {
		return 0, false
	}
	target := uint16(id)
	h := e.mruHead - 1 // newest emission lives at h after this offset
	r := uint32(0)
	for ; r+3 < mruRingSize; r += 4 {
		if e.mruRing[(h-r)&mruRingMask] == target {
			return r, true
		}
		if e.mruRing[(h-r-1)&mruRingMask] == target {
			return r + 1, true
		}
		if e.mruRing[(h-r-2)&mruRingMask] == target {
			return r + 2, true
		}
		if e.mruRing[(h-r-3)&mruRingMask] == target {
			return r + 3, true
		}
	}
	for ; r < mruRingSize; r++ {
		if e.mruRing[(h-r)&mruRingMask] == target {
			return r, true
		}
	}
	return 0, false
}

// lruLinkInvalid encodes (prev=0xFFFF, next=0xFFFF) — an isolated
// slot with no neighbours. Used as the append default when growing
// the lruLink slice.
const lruLinkInvalid uint32 = 0xFFFF | (0xFFFF << 16)
const lruLink16Invalid uint32 = 0xFFFF // 16-bit sentinel masked into a uint32

//go:nosplit
func linkPrev(link uint32) uint32 { return link & 0xFFFF }

//go:nosplit
func linkNext(link uint32) uint32 { return link >> 16 }

//go:nosplit
func setLinkPrev(link, prev uint32) uint32 { return (link &^ 0xFFFF) | (prev & 0xFFFF) }

//go:nosplit
func setLinkNext(link, next uint32) uint32 { return (link & 0xFFFF) | ((next & 0xFFFF) << 16) }

// lruAddFresh inserts a brand-new ID (just assigned) at the head of
// the LRU. Caller must have ensured id == len(ids)-1 (i.e. ids assigns
// sequentially starting from 0). Also records the emit in the MRU
// ring so the rank-discovery side-cache reflects the new chain head.
func (e *encState) lruAddFresh(id uint32) {
	for uint32(len(e.lruLink)) <= id {
		e.lruLink = append(e.lruLink, lruLinkInvalid)
	}
	head := e.lruHead
	// id.prev = invalid, id.next = head
	if head == lruInvalidID {
		e.lruLink[id] = lruLinkInvalid
	} else {
		e.lruLink[id] = lruLink16Invalid | (head << 16)
		// head.prev = id
		e.lruLink[head] = setLinkPrev(e.lruLink[head], id)
	}
	e.lruHead = id
	e.mruPush(id)
}

// lruMoveFront performs the unlink+insert-at-head update of the LRU
// but skips the rank walk. Use when the caller does not need the
// rank (e.g. raw state-ref where MTF cannot win). Also records the
// emit in the MRU ring so the rank side-cache mirrors the chain
// head update.
//
//go:nosplit
func (e *encState) lruMoveFront(id uint32) {
	if e.lruHead == id {
		e.mruPush(id)
		return
	}
	link := e.lruLink[id]
	p := linkPrev(link)
	n := linkNext(link)
	// p is always valid here (id was not head). Patch p.next = n.
	e.lruLink[p] = setLinkNext(e.lruLink[p], n)
	if n != lruLink16Invalid {
		// Patch n.prev = p.
		e.lruLink[n] = setLinkPrev(e.lruLink[n], p)
	}
	// Insert id at head: id.prev=invalid, id.next=head.
	head := e.lruHead
	e.lruLink[id] = lruLink16Invalid | (head << 16)
	// head.prev = id (head is always valid here — id was in chain).
	e.lruLink[head] = setLinkPrev(e.lruLink[head], id)
	e.lruHead = id
	e.mruPush(id)
}

// lookupOrAssign returns (id, hit). On a miss a fresh entry is
// installed and (id, false) is returned; the caller is expected to
// emit an intern record. The key bytes are copied into the encState
// arena so the table is independent of the caller's buffer.
//
// Uses the flat open-addressed hash table (internTable) — a single
// memhash + a couple of cache-line loads instead of Go's
// mapaccess2_faststr (hash + bucket walk + tophash + memequal).
// The stored slot.key aliases the arena copy via unsafestr.String
// so the encoder owns the bytes and the caller's buffer can be
// reused immediately after the call.
//
// For payloads longer than the arena's per-string limit
// (internarena.MaxStringLen, 65 535 bytes), fall back to
// strings.Clone. Such oversized intern attempts are not expected
// on real workloads; the path exists so a hostile input cannot
// crash the encoder.
//
//go:nosplit
func (e *encState) lookupOrAssign(key string) (uint32, bool) {
	// Hot-path fast lookup: hash + one slot probe. Inlinable (no
	// loop, no allocs); the slow tail (collision probing, miss-
	// install, grow) is split out so the linear-probing loop does
	// not pollute the inline budget. Hit rate at slot 0 is high
	// because the table is kept under 0.5 load.
	h := maphash.String(internHashSeed, key)
	if h == 0 {
		h = 1 // reserve 0 as the empty-slot sentinel
	}
	i := h & uint64(len(e.internTable)-1)
	slot := &e.internTable[i]
	if slot.hash == h && slot.key == key {
		return slot.id, true
	}
	if slot.hash == 0 {
		// Empty first slot: direct install.
		return e.installInternSlot(slot, h, key), false
	}
	// Collision at first slot — fall to the probing loop.
	return e.lookupOrAssignSlow(h, key, i)
}

// lookupOrAssignSlow handles the collision case: probe past startIdx
// looking for either an empty slot (install) or a matching entry
// (hit). Separated from lookupOrAssign so the inliner keeps the
// fast path tight.
func (e *encState) lookupOrAssignSlow(h uint64, key string, startIdx uint64) (uint32, bool) {
	mask := uint64(len(e.internTable) - 1)
	for i := (startIdx + 1) & mask; ; i = (i + 1) & mask {
		slot := &e.internTable[i]
		if slot.hash == 0 {
			return e.installInternSlot(slot, h, key), false
		}
		if slot.hash == h && slot.key == key {
			return slot.id, true
		}
	}
}

// installInternSlot writes a fresh entry into slot, copies the key
// into the encoder arena (so it survives the caller's buffer
// lifetime), bumps the LRU + intern counters, and grows the table
// when the load crosses 3/4. The slot pointer can be invalidated
// by the grow; callers must not touch it after this returns.
func (e *encState) installInternSlot(slot *internSlot, h uint64, key string) uint32 {
	id := e.internLoad
	var stored string
	if len(key) <= internarena.MaxStringLen {
		arenaID := e.arena.Put(key)
		stored = unsafestr.String(e.arena.Get(arenaID))
	} else {
		stored = strings.Clone(key)
	}
	slot.hash = h
	slot.key = stored
	slot.id = id
	e.internLoad++
	e.lruAddFresh(id)
	// Grow at 3/4 load, not 1/2. A denser table is smaller (better cache)
	// and rehashes less often; with the well-distributed maphash the longer
	// linear-probe chains cost less than the cache + rehash savings.
	// Measured -12.6% encode on the large-payload Archive profile (thousands
	// of interned strings), neutral on small/medium payloads, wire unchanged.
	if e.internLoad*4 >= uint32(len(e.internTable))*3 {
		e.internTableGrow()
	}
	return id
}

// internTableGrow doubles the flat hash table and rehashes every
// occupied slot. Called from installInternSlot when the load factor
// reaches 3/4. Amortised insert stays O(1); the denser table trades
// slightly longer probe chains for fewer rehashes and a smaller cache
// footprint (a net encode win on large, intern-heavy payloads).
func (e *encState) internTableGrow() {
	old := e.internTable
	newSize := len(old) * 2
	if newSize == 0 {
		newSize = internTableInitSize
	}
	e.internTable = make([]internSlot, newSize)
	mask := uint64(newSize - 1)
	for i := range old {
		if old[i].hash == 0 {
			continue
		}
		for j := old[i].hash & mask; ; j = (j + 1) & mask {
			if e.internTable[j].hash == 0 {
				e.internTable[j] = old[i]
				break
			}
		}
	}
}

// decShape is the decoder-side mirror of an encShape. names is the
// resolved Go-string for each field in declaration order — used to
// dispatch values to struct fields when the shape is re-used.
type decShape struct {
	names []string
}

// decColShape is the decoder-side descriptor for a columnar struct shape
// (tagColStruct). Parallel to encState's colShapeNames/colShapeKinds entries.
type decColShape struct {
	names []string
	kinds []colKind
}

// Field order: every pointer-bearing field (the slices and mapDec, which
// holds reflect handles) is grouped FIRST so the GC pointer-scan range stays
// tight (440 pointer bytes vs 720 for a hot-scalars-first layout); the
// non-pointer hot scalars + mruRing + retainStreak trail at the end. The
// scalars stay packed together (lastID/lruHead/mruHead/mruRing) so the
// per-emit MTF update still touches one contiguous span.
type decState struct {
	// values holds the decoded byte slices indexed by intern id —
	// each entry aliases the wire buffer (zero-copy).
	values [][]byte

	// stringValues caches a heap-allocated `string` copy per intern
	// record. Populated once at append time (one `string(b)` alloc
	// per first occurrence); subsequent state-ref / MTF / pair /
	// repeat reads return the cached string from Decoder.ReadString
	// without paying another `string(b)` copy each time. On dense
	// payloads with repeated values (telemetry, archive,
	// LargePayload) this collapses N-1 of every N reads to zero
	// alloc.
	stringValues []string

	// boxValues is the interface-boxing analogue of stringValues, indexed
	// by the same intern id and grown in lockstep (one nil placeholder per
	// record). decodeAny boxes a repeated string's `any(s)` exactly once and
	// caches it here; every later state-ref / MTF / pair / repeat occurrence
	// of that value returns the shared box with zero allocation. A unique
	// (never-referenced) string arrives inline with lastID == lruInvalidID and
	// never touches this cache, so high-cardinality payloads pay no lookup
	// overhead — the wire's own intern encoding is the adaptivity signal.
	// Sharing an immutable boxed scalar across map/slice slots is safe.
	boxValues []any

	// zeroTimeBox caches the boxed `any` of the zero time.Time (the value
	// unset time fields — DeletedAt, ExpiresAt, … — decode to en masse). It is
	// a universal, immutable constant, so it is boxed once and shared across
	// every occurrence AND across decodes on this pooled state (never reset):
	// one alloc replaces N. decodeAny gates on time.IsZero(), a single cheap
	// branch, so non-zero timestamps pay nothing.
	zeroTimeBox any

	// LRU mirror of encState's. Decoder maintains the same MTF chain
	// the encoder did so tagStateMTF + rank resolves to the same ID.
	// See encState.lruLink for the packing layout.
	lruLink []uint32

	// Pair predictor mirror. See encState.pairPred for the storage
	// layout (succ+1 packed into a uint32, 0 = empty). The decoder
	// only ever reads rank 0; any rank ≥ pairPredK on the wire is
	// rejected upstream as malformed.
	pairPred []uint32

	// Shape table mirror. shapes[i] is the shape with wire-ID i+1.
	shapes []decShape

	// Columnar shape table (tagColStruct). colShapes[i] is the columnar
	// shape with wire-ID i+1. Parallel to encState's colShapeNames/colShapeKinds.
	colShapes []decColShape
	// Hybrid columnar shape table (tagHybridColStruct), separate ID space from
	// colShapes. kinds carry residualKind (0xFF) for residual fields.
	hybridShapes   []decColShape
	colScratchI64  []int64
	colScratchU64  []uint64
	colScratchF64  []float64
	colScratchBool []bool
	colScratchF32  []float32 // codegen columnar float32 scatter scratch
	colScratchStr  []string  // codegen columnar plain string scatter scratch
	colStrHandles  []Str     // string column handle scratch (mirrors colScratchI64 pattern)
	// String-dict column scratch, reused across columns. Both are transient: the
	// dispatcher copies table[idx[i]] into the column result before reading the
	// next column, so neither is aliased past one column read.
	colDictTableScr []string // distinct-value table for a string-dict column
	colDictIdxScr   []uint32 // per-row dictionary index for a string-dict column

	// deltaColRows is reused storage for a sparse column's decoded ascending row
	// indices during a tagColSlicePatch apply (delta_columnar.go).
	deltaColRows []int

	// colLenScratch is reused storage for the column-length index parsed from
	// a FlagColIndex columnar payload (one uint32 byte-length per column).
	colLenScratch []uint32

	// mapDec pools reflect holders for the generic (reflect) string-keyed map
	// decode path so it does not reflect.New per map entry (OptMapShape).
	mapDec mapHolderCache

	// Hot scalars — touched on every tagState* read. Packing them with the
	// mruRing/head update keeps the per-emit footprint contiguous.
	lastID  uint32
	lruHead uint32
	mruHead uint32
	_       uint32 // align mruRing on 8-byte boundary

	// mruRing mirrors the encoder's side-cache: the last mruRingSize
	// state-ref ids in emission order, stored as uint16 (the id
	// space is < 2^14 by encoder cap). For tagStateMTF the wire
	// carries rank — direct index into the ring resolves the id in
	// O(1) (mruRing[(mruHead-1-rank)&mask]) instead of walking the
	// LRU chain. Pure decoder-side optimization; the wire format is
	// unchanged.
	mruRing [mruRingSize]uint16

	// retainStreak counts consecutive small (sub-cap) messages for the
	// adaptive-retention policy in reset(). Cold — touched once per reset.
	retainStreak uint8
}

func newDecState() *decState {
	d := &decState{
		values:       make([][]byte, 0, 64),
		stringValues: make([]string, 0, 64),
		boxValues:    make([]any, 0, 64),
		lruHead:      lruInvalidID,
		lastID:       lruInvalidID,
	}
	for i := range d.mruRing {
		d.mruRing[i] = mruEmpty
	}
	return d
}

func (d *decState) reset() {
	// Symmetric to encState.reset's adaptive-retention policy: retain grown
	// backing arrays (reuse in place) while messages stay large so a steady
	// large-batch decode workload amortizes the table allocation, releasing
	// only after retainReleaseStreak consecutive small messages. The decision
	// is driven by len(d.values) — the count of intern records decoded THIS
	// message, a true per-message demand signal (the retained cap would stay
	// large forever and never let the streak advance). Row-scaled columnar
	// scratch is governed by the hard ceiling; the id-bounded tables are
	// already bounded by maxStateEntries. See encState.reset for rationale.
	if len(d.values) > maxRetainedIDs {
		d.retainStreak = 0
	} else if d.retainStreak < retainReleaseStreak {
		d.retainStreak++
	}
	release := d.retainStreak >= retainReleaseStreak

	if cap(d.values) > maxRetainedIDs && release {
		d.values = nil
		d.stringValues = nil
		d.boxValues = nil
	} else {
		// Retain the backing, but clear() first: d.values entries are []byte
		// aliasing the previous message's input (wire) buffer, and
		// stringValues entries are prior decoded-string headers. A bare [:0]
		// keeps those headers live in the tail, pinning the previous caller's
		// input buffer (and decoded strings) from GC for the lifetime of the
		// pooled decoder. clear() drops the headers before reuse — mirrors the
		// encoder's colScratchStr treatment. memclr only, no allocation.
		clear(d.values)
		clear(d.stringValues)
		clear(d.boxValues)
		d.values = d.values[:0]
		d.stringValues = d.stringValues[:0]
		d.boxValues = d.boxValues[:0]
	}
	d.lastID = lruInvalidID
	d.lruHead = lruInvalidID
	if cap(d.lruLink) > maxRetainedLRUCap && release {
		d.lruLink = nil
	} else {
		d.lruLink = d.lruLink[:0]
	}
	if cap(d.pairPred) > maxRetainedPairCap && release {
		d.pairPred = nil
	} else {
		clear(d.pairPred)
	}
	if cap(d.shapes) > maxRetainedShapeCap && release {
		d.shapes = nil
	} else {
		d.shapes = d.shapes[:0]
	}
	if cap(d.colShapes) > maxRetainedShapeCap && release {
		d.colShapes = nil
	} else {
		d.colShapes = d.colShapes[:0]
	}
	if cap(d.hybridShapes) > maxRetainedShapeCap && release {
		d.hybridShapes = nil
	} else {
		d.hybridShapes = d.hybridShapes[:0]
	}
	// Row-scaled columnar scratch: ceiling-only (independent of the intern
	// streak), reclaimed when idle by sync.Pool GC. Each backing grows on its own
	// per-column-type demand, so gate each independently — a single check keyed on
	// colScratchI64 would miss an oversized U64/F64/Bool backing (mirrors the
	// encode-side fix).
	if cap(d.colScratchI64) > maxRetainedColScratch {
		d.colScratchI64 = nil
	}
	if cap(d.colScratchU64) > maxRetainedColScratch {
		d.colScratchU64 = nil
	}
	if cap(d.colScratchF64) > maxRetainedColScratch {
		d.colScratchF64 = nil
	}
	if cap(d.colScratchBool) > maxRetainedColScratch {
		d.colScratchBool = nil
	}
	if cap(d.colScratchF32) > maxRetainedColScratch {
		d.colScratchF32 = nil
	}
	if cap(d.colScratchStr) > maxRetainedColScratch {
		d.colScratchStr = nil
	} else {
		// Drop retained string headers across the FULL backing, not just len:
		// colScratchStr is resliced via [:n] (ReadStringColumn) / [:0] (gather),
		// so nested columnar structs of differing row counts leave a high-water
		// tail of live headers that would pin prior-message decoded strings from
		// GC for the pooled decoder's lifetime. Sibling to d.stringValues' clear.
		clear(d.colScratchStr[:cap(d.colScratchStr)])
		d.colScratchStr = d.colScratchStr[:0]
	}
	if cap(d.colStrHandles) > maxRetainedColScratch { // pointer-free (Str={off,len uint32}), no clear
		d.colStrHandles = nil
	} else {
		d.colStrHandles = d.colStrHandles[:0]
	}
	if cap(d.colDictTableScr) > maxRetainedColScratch {
		d.colDictTableScr = nil
	} else {
		// Drop retained string headers across the full backing (sibling to
		// colScratchStr) so a prior message's dict values cannot be pinned.
		clear(d.colDictTableScr[:cap(d.colDictTableScr)])
		d.colDictTableScr = d.colDictTableScr[:0]
	}
	if cap(d.colDictIdxScr) > maxRetainedColScratch { // pointer-free, no clear
		d.colDictIdxScr = nil
	}
	if cap(d.colLenScratch) > maxRetainedColScratch {
		d.colLenScratch = nil
	}
	// Column-diff sparse-row scratch (delta_columnar.go): same ceiling policy.
	if cap(d.deltaColRows) > maxRetainedColScratch {
		d.deltaColRows = nil
	}
	for i := range d.mruRing {
		d.mruRing[i] = mruEmpty
	}
	d.mruHead = 0
	d.mapDec = mapHolderCache{}
}

// pairAtRank returns the predicted successor of prev. With top-1
// storage the only valid wire rank is 0; any rank ≥ pairPredK was
// already rejected upstream. ok=false marks an empty slot.
//
//go:nosplit
func (d *decState) pairAtRank(prev uint32, rank uint8) (uint32, bool) {
	if int(prev) >= len(d.pairPred) || rank != 0 {
		return 0, false
	}
	v := d.pairPred[prev]
	if v == pairPredEmpty {
		return 0, false
	}
	return v - 1, true
}

//go:nosplit
func (d *decState) pairEnsure(prev uint32) {
	for uint32(len(d.pairPred)) <= prev {
		d.pairPred = append(d.pairPred, pairPredEmpty)
	}
}

// pairRecord mirrors encState.pairRecord exactly. Overwrites the
// stored successor — top-1 keeps only the most recent transition.
//
//go:nosplit
func (d *decState) pairRecord(prev, curr uint32) {
	d.pairEnsure(prev)
	d.pairPred[prev] = curr + 1
}

// shapeDeclare appends a new shape with the next sequential wire ID
// and returns a pointer to its slot. The wire ID equals
// len(d.shapes) after the append; callers do not need it returned
// because the encoder hands shape IDs out in the same order.
func (d *decState) shapeDeclare() *decShape {
	d.shapes = append(d.shapes, decShape{})
	return &d.shapes[len(d.shapes)-1]
}

// shapeLookup returns the shape with the given wire ID (≥ 1). nil
// means an unknown ID — the stream is malformed.
func (d *decState) shapeLookup(id uint32) *decShape {
	if id == 0 || id > uint32(len(d.shapes)) {
		return nil
	}
	return &d.shapes[id-1]
}

// mruPush records id as the newest entry in the decoder side-cache.
// Mirrors encState.mruPush so the ring sees every emit the encoder
// recorded. id must fit in uint16 (id space is < 2^14 by encoder
// cap).
//
//go:nosplit
func (d *decState) mruPush(id uint32) {
	d.mruRing[d.mruHead&mruRingMask] = uint16(id)
	d.mruHead++
}

// mruIDAtRank returns the id stored rank positions back from the head
// of the ring. Provided the encoder emitted tagStateMTF only for
// ranks ≤ mruRingSize-1 (which is the only range where MTF beats raw
// for the common 2-byte id wire form), this resolves the id in O(1)
// without walking the LRU chain.
//
//go:nosplit
func (d *decState) mruIDAtRank(rank uint32) (uint32, bool) {
	if rank >= mruRingSize {
		return 0, false
	}
	id := d.mruRing[(d.mruHead-1-rank)&mruRingMask]
	if id == mruEmpty {
		return 0, false
	}
	return uint32(id), true
}

func (d *decState) lruAddFresh(id uint32) {
	for uint32(len(d.lruLink)) <= id {
		d.lruLink = append(d.lruLink, lruLinkInvalid)
	}
	head := d.lruHead
	if head == lruInvalidID {
		d.lruLink[id] = lruLinkInvalid
	} else {
		d.lruLink[id] = lruLink16Invalid | (head << 16)
		d.lruLink[head] = setLinkPrev(d.lruLink[head], id)
	}
	d.lruHead = id
	d.mruPush(id)
}

// lruIDAtRank returns the ID currently at the given MTF rank (head
// = 0) by walking the linked-list chain. Used as a fallback when the
// MRU ring side-cache misses (rank ≥ mruRingSize, which the encoder
// never emits today but the decoder must still handle for forward
// compatibility with larger ring sizes on the encoder side).
func (d *decState) lruIDAtRank(rank uint32) (uint32, bool) {
	cur := d.lruHead
	for range rank {
		if cur == lruInvalidID {
			return 0, false
		}
		cur = linkNext(d.lruLink[cur])
		if cur == lruLink16Invalid {
			cur = lruInvalidID
		}
	}
	if cur == lruInvalidID {
		return 0, false
	}
	return cur, true
}

func (d *decState) lruMoveToFront(id uint32) {
	if d.lruHead == id {
		d.mruPush(id)
		return
	}
	link := d.lruLink[id]
	p := linkPrev(link)
	n := linkNext(link)
	d.lruLink[p] = setLinkNext(d.lruLink[p], n)
	if n != lruLink16Invalid {
		d.lruLink[n] = setLinkPrev(d.lruLink[n], p)
	}
	head := d.lruHead
	d.lruLink[id] = lruLink16Invalid | (head << 16)
	d.lruLink[head] = setLinkPrev(d.lruLink[head], id)
	d.lruHead = id
	d.mruPush(id)
}

// append registers a fresh intern record with the decoder. b
// aliases the wire buffer; the cached string slot is left empty so
// the first ReadString of this record pays the string(b) copy
// exactly once, and every later state-ref / MTF / pair / repeat
// read returns the cached value without alloc.
//
// Eager materialisation would punish single-shot decodes (Config-
// shaped workloads, ~10 distinct interns, each read once) where
// the cache slot never gets re-read; lazy population matches what
// the old direct-string(b) path did on first sight and adds zero
// alloc on subsequent reads.
func (d *decState) append(b []byte) uint32 {
	id := uint32(len(d.values))
	d.values = append(d.values, b)
	d.stringValues = append(d.stringValues, "")
	d.boxValues = append(d.boxValues, nil)
	d.lruAddFresh(id)
	return id
}

func (d *decState) get(id uint32) ([]byte, bool) {
	if id >= uint32(len(d.values)) {
		return nil, false
	}
	return d.values[id], true
}

// getString returns the cached string copy of the intern record at
// id, populating the slot on first call. Empty interned bytes
// resolve to "" without an extra alloc — the `len(b) == 0` branch
// short-circuits before the materialisation. Used on the state-ref
// / MTF / pair / repeat decode paths so ReadString skips the
// string(b) heap copy after the first sight.
//
// When arena is non-nil the first-sight materialisation is packed into the
// bump arena instead of its own heap allocation, so a Dense/intern decode of a
// high-cardinality string column amortises its copies the same way the plain
// (Speed-mode) string path already does. Each interned id is still copied at
// most once — the cached aliasing string is reused on every later reference.
//
//go:nosplit
func (d *decState) getString(id uint32, arena *Arena) (string, bool) {
	if id >= uint32(len(d.stringValues)) {
		return "", false
	}
	s := d.stringValues[id]
	if s != "" {
		return s, true
	}
	b := d.values[id]
	if len(b) == 0 {
		return "", true
	}
	if arena != nil {
		s = arena.appendStr(b)
	} else {
		s = string(b)
	}
	d.stringValues[id] = s
	return s, true
}

// getBoxStr returns the string at id boxed into an `any`, cached so repeated
// occurrences of the same interned value share ONE box (zero alloc after the
// first). Called only when the string arrived as an intern/state reference
// (lastID valid) — a unique inline string never reaches here, so the boxing
// cache never adds overhead on high-cardinality data. The shared box is
// immutable, so handing it to many map/slice slots is safe.
func (d *decState) getBoxStr(id uint32, arena *Arena) (any, bool) {
	if id >= uint32(len(d.boxValues)) {
		return nil, false
	}
	if b := d.boxValues[id]; b != nil {
		return b, true
	}
	s, ok := d.getString(id, arena)
	if !ok {
		return nil, false
	}
	box := any(s)
	d.boxValues[id] = box
	return box, true
}
