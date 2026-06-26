package qdf

import (
	"encoding/binary"
	"math"
	"slices"
	"unsafe"

	"github.com/alex60217101990/qdf/internal/fsst"
	"github.com/alex60217101990/qdf/internal/rans"
	"github.com/alex60217101990/qdf/internal/unsafestr"
	"github.com/alex60217101990/qdf/internal/vecquant"
)

// ransMinBytes is the smallest message body worth a rANS attempt. Below it
// the 256-entry frequency table dwarfs any entropy saving, so the picker
// would reject it anyway — this threshold just skips the wasted encode.
const ransMinBytes = 512

// maybeApplyRANS rewrites the message at e.buf[start:] in place to a
// rANS-compressed body when that is strictly smaller than the plain body,
// setting FlagRANS in the header. start is the message's header offset (0 for
// Marshal, len(dst) for AppendMarshal). No-op unless OptRANS is set.
func (e *Encoder) maybeApplyRANS(start int) {
	if !e.rans {
		return
	}
	if e.customFramed {
		// A top-level Marshaler forced Fast framing and its bytes are
		// opts-invariant by contract; do not reframe them with FlagRANS.
		return
	}
	const hdr = 5
	if len(e.buf)-start < hdr+ransMinBytes {
		return
	}
	body := e.buf[start+hdr:]
	cand := appendUvarint(make([]byte, 0, len(body)/2+512), uint64(len(body)))
	cand = rans.Encode(cand, body)
	if len(cand) >= len(body) {
		return // no win — keep the plain body
	}
	// Decline rANS if the frame we'd emit would trip the decoder's origLen
	// bound (decoder.go: origLen > len(buf)*64 + 1MiB rejects, to cap the decode
	// allocation). A body that compresses more than ~64x — e.g. a multi-MiB
	// low-entropy/repeated payload — has an origLen the decoder won't accept, so
	// rANS would make a wire its own Unmarshal rejects. Keep the plain body
	// (the decoder bounds it by remaining input as usual). The emitted frame is
	// hdr + len(cand) bytes, matching the decoder's len(d.buf) for this message.
	if uint64(len(body)) > uint64(hdr+len(cand))*64+(1<<20) {
		return
	}
	e.buf = append(e.buf[:start+hdr], cand...)
	e.buf[start+4] |= FlagRANS
}

// preInternEntry caches a caller-registered string by its backing
// pointer and length. id == preInternUnseen marks an entry that
// has not yet been interned on the wire — the first WriteString
// that matches still goes through the normal intern path so the
// decoder sees the tagInternStr record; subsequent matches reuse
// the resolved id and skip the hash + slot probe.
// See Encoder.PreIntern for the contract.
type preInternEntry struct {
	ptr unsafe.Pointer
	n   uintptr
	id  uint32
	// Naturally 24 B: ptr(8)+n(8)+id(4) rounds up to the 8 B struct alignment.
}

const preInternUnseen = ^uint32(0)

// Encoder writes a single QDF value into a growing internal buffer. Reset
// drops the buffer contents and (in Dense mode) the intern table so the
// encoder can be reused.
// Field order groups the pointer-bearing / slice / map fields and the
// 8-byte counters first, then the 1-byte flags (mode + the many bools) last
// so the interspersed bools do not each force a padding word (200 bytes vs
// 232 for the source order).
type Encoder struct {
	state *encState

	// fsstDict, when non-nil, is a pre-trained FSST symbol table supplied via
	// FSSTDict.Marshal. The encoder uses it instead of training a table per
	// string column (the dominant FSST encode cost), so train-once-reuse-many
	// collapses per-batch encode to compression only. The table is bounded and
	// immutable; it is never mutated by the encoder.
	fsstDict *fsst.SymbolTable

	// keyIdx is a reused (clear-not-realloc) old-key→index map for keyed slice
	// diff. Lives on the Encoder so a many-element keyed slice builds its match
	// table once per pool acquire; dropped past a spike cap in Reset(). Single
	// pointer word — grouped with state/fsstDict ahead of the slices so the GC
	// pointer-scan range stays tight (the slices' len/cap words trail).
	keyIdx map[string]int

	buf []byte

	// alpScratch is a reused FOR-mantissa staging buffer for the ALP float
	// writer (mirrors the decoder's deltaScratch). Lives on the Encoder, not
	// encState, so the row-major float path reuses it without needing a state.
	alpScratch []uint64

	// vecScratch reuses the lossy vector codec's rotate/row/coord buffers across
	// encodes, retained between batches and dropped past the ceiling in
	// resetForReuse. Mirrors alpScratch.
	vecScratch vecquant.Scratch

	// wideF64 reuses the []float32 -> []float64 widening buffer for the lossy
	// vector codec, retained across encodes and dropped past the ceiling in
	// resetForReuse. For f32 fields this avoids one allocation per encode;
	// for f64 fields it holds a defensive copy so appendLossyVec cannot mutate
	// the caller's slice.
	wideF64 []float64

	// vecBatchFlat / vecBatchRows reuse the gather scratch for the batched lossy
	// vector-column codec: vecBatchFlat is the flat [n*dim]float64 backing and
	// vecBatchRows the [n] row views into it. Retained across encodes, dropped
	// past the ceiling in Reset.
	vecBatchFlat []float64
	vecBatchRows [][]float64
	// vecBatchNorms reuses the per-vector L2 norm scratch for the polar-split
	// variant of the batched vector codec.
	vecBatchNorms []float64

	// preIntern is an opt-in identity cache populated by PreIntern.
	// When non-empty WriteString does a linear scan against it
	// before falling to the intern table — a pointer-and-length
	// match means we already know the intern id and can skip the
	// hash + slot probe. The slice is empty (and the check is a
	// single branch-predicted compare) when no caller has opted
	// in, so it does not regress the default Marshal path.
	preIntern []preInternEntry

	// wideI64 / wideU64 are reused widening scratch for the QPack slice
	// encoders: []int32 / []uint32 must be promoted to []int64 / []uint64 so
	// the int64/uint64 codec pickers can score them. The widen → pick → emit
	// sequence is atomic (no nested slice encode runs between fill and the last
	// read), so a single scratch per element type is safely reused across every
	// narrow-int slice field in a message — turning one make per field into
	// zero. Bounded on return to the encoder pool (putEnc).
	wideI64 []int64
	wideU64 []uint64

	// minIntern is the minimum string length eligible for interning;
	// shorter values go in line.
	minIntern int

	// maxStateEntries caps the intern table. Past the cap, new strings go
	// in line; existing IDs still resolve.
	maxStateEntries int

	// headerFlagAt is the byte offset of the header flag byte in e.buf,
	// recorded by writeHeader so encodeColumnar can backpatch FlagColIndex
	// only when it actually emits a column index.
	headerFlagAt int

	// depth tracks nested pointer/struct traversal. Pointer cycles do
	// not crash the process; encodePtr increments depth on entry and
	// returns ErrCycleDetected when it exceeds maxDepth. Lightweight
	// alternative to a per-pointer set (no allocation per call).
	depth    int
	maxDepth int

	// vecBudget is the fidelity target used when OptLossyVec is active.
	// No effect unless OptLossyVec is set. Zero value resolves to MinCosine(0.999).
	// Placed among the 8-byte fields (it is pointer-free, 8-byte aligned) so the
	// trailing 1-byte flags pack contiguously without a padding gap on either side.
	vecBudget VectorBudget

	// opts is the bit-mask of feature toggles. mode and qpack are
	// derived from it at configure time so the hot path can stay on
	// fast bool / Mode compares; the rest of the codecs (MTF, Pair,
	// ShapeIntern) check the corresponding bit directly via opts. Placed
	// next to the 1-byte flags so the uint32 packs without a padding gap.
	opts Options

	mode Mode

	headerOut bool
	// customFramed is set when a top-level Marshaler emitted the body and forced
	// a Fast (flag 0) header. maybeApplyRANS must then leave the wire alone — a
	// Marshaler's bytes are opts-invariant by contract, so the entropy pass must
	// not reframe them with FlagRANS (see TestMarshaler_AlwaysFastFraming).
	customFramed bool

	// qpack switches the slice fast paths to QPack codecs (bitpack, FOR,
	// Gorilla, raw-LE bulk). When set, the header's FlagQPack bit is
	// emitted as an early hint so a reader that does not implement the
	// codec tags fails fast on the header rather than mid-stream.
	qpack bool

	// gorillaFloat lets encodeSliceFloat64 / encodeSliceFloat32 probe
	// the input and pick Gorilla XOR for smooth time-series. Costs
	// ~10× more CPU per slice than raw-LE bulk, so it stays off in
	// OptBalanced; flipped on by OptCompression. Implies qpack.
	gorillaFloat bool

	// rans enables the order-0 rANS entropy post-pass over the finished
	// body at the top-level Marshal entry points (never per nested value,
	// never in streaming). Set from OptRANS; applied only when it shrinks.
	rans bool

	// fsst enables the FSST string codec for columnar string columns. Set
	// from OptFSST; applied only when strictly smaller than dict/per-value.
	// Implies qpack (columnar path requires OptQPack).
	fsst bool

	// colIndex makes encodeColumnar emit a fixed-width uint32 column-length
	// table after the shape declaration and before the column bodies, and
	// backpatch FlagColIndex onto the header. Set from OptColumnIndex. Lets a
	// reader skip columns without decoding them.
	colIndex bool

	// pairPred / mtf cache OptPairPred / OptMTF (both on in OptBalanced) so the
	// hot Dense state-ref emit path tests a bool field instead of re-running
	// opts.Has() several times per repeated value. Set in applyOpts, cleared in
	// Reset — same pattern as qpack/rans/fsst/colIndex.
	pairPred bool
	mtf      bool

	// keyIdxBusy marks keyIdx as borrowed by an in-progress keyed-slice diff so a
	// nested keyed slice routes to a fresh local map instead of clobbering it.
	keyIdxBusy bool

	// stateSuspended turns off every wire-stateful encoding — string/[]byte
	// interning, struct-shape interning, map-shape interning — for the duration
	// of a keyed-slice delta's never-larger trial (delta_keyed.go). A keyed patch
	// is compared against the positional alternative by building both bodies and
	// keeping the smaller; a stateful DISCARDED body leaks shared-state ids whose
	// wire definitions are thrown away, so a later reference into the kept wire
	// dangles (ErrUnknownStateID). Suspending makes both candidate bodies — and
	// the emitted winner — wire-stateless, so the trial is pollution-free, the
	// size comparison is exact, and the kept body is self-consistent. The only
	// substate a suspended body mutates is lastID (the inline-string reset, which
	// the decoder mirrors). Saved/restored across nested keyed slices so
	// re-entrancy keeps the outer suspension.
	stateSuspended bool
}

// applyOpts mirrors the options bitmask onto the cached mode / qpack
// fields so hot-path checks compile to a single bool / Mode compare
// instead of a bit-test. It is safe to call on a pooled encoder;
// pair with state setup as needed (callers that need OptDense must
// also point state at a non-nil encState).
//
//go:nosplit
func (e *Encoder) applyOpts(opts Options) {
	e.opts = opts
	if opts.Has(OptDense) {
		e.mode = Dense
		// Lazily allocate the Dense intern state. A pooled encoder starts with
		// state == nil (set in the pool's New) and never allocates it while it
		// only serves OptSpeed; the first Dense encode brings it up. A reused
		// encoder keeps its state — Reset() clears it before this runs.
		if e.state == nil {
			e.state = newEncState()
		}
	} else {
		e.mode = Fast
	}
	e.qpack = opts.Has(OptQPack)
	e.gorillaFloat = e.qpack && opts.Has(OptGorillaFloat)
	e.rans = opts.Has(OptRANS)
	e.colIndex = opts.Has(OptColumnIndex)
	e.fsst = e.qpack && opts.Has(OptFSST)
	e.pairPred = opts.Has(OptPairPred)
	e.mtf = opts.Has(OptMTF)
}

// DefaultMaxDepth caps reflect-path pointer/struct recursion. Set
// large enough for any legitimate payload (10 000) while still
// rejecting genuine cycles before the goroutine stack overflows.
const DefaultMaxDepth = 10_000

// Mode selects the wire dialect.
type Mode uint8

const (
	// Fast writes strings and []byte in line. No intern bookkeeping.
	Fast Mode = 0

	// Dense writes repeated strings/bytes once and references them by ID
	// thereafter. Smaller output on repetitive payloads; slightly slower
	// per call.
	Dense Mode = 1
)

// NewEncoder returns an Encoder. The internal buffer is allocated lazily
// on first write.
func NewEncoder(mode Mode) *Encoder {
	e := &Encoder{
		mode:            mode,
		minIntern:       4,
		maxStateEntries: 1 << 14,
		maxDepth:        DefaultMaxDepth,
	}
	if mode == Dense {
		e.state = newEncState()
		e.opts = OptBalanced
		e.qpack = true
		// OptBalanced includes OptPairPred | OptMTF; mirror them onto the cached
		// flags this constructor sets by hand (it bypasses applyOpts).
		e.pairPred = true
		e.mtf = true
	} else {
		e.opts = OptSpeed
	}
	return e
}

// NewEncoderWith returns an Encoder configured by the option bit-mask
// directly. Dense state machinery is allocated only when OptDense is
// set. Defaults for intern threshold / depth match NewEncoder.
func NewEncoderWith(opts Options) *Encoder {
	e := &Encoder{
		minIntern:       4,
		maxStateEntries: 1 << 14,
		maxDepth:        DefaultMaxDepth,
	}
	e.applyOpts(opts) // allocates Dense state when OptDense is set
	return e
}

// NewEncoderOnBuf returns an Encoder that appends to buf at its current
// length. The buffer is NOT truncated; pass an empty slice for a fresh
// encoding. Call SetBuffer afterwards to truncate.
func NewEncoderOnBuf(buf []byte, mode Mode) *Encoder {
	e := NewEncoder(mode)
	e.buf = buf
	return e
}

// Reset truncates the buffer and resets the intern table. Capacities
// are preserved. opts / mode / qpack are also reset to OptSpeed so a
// pooled encoder does not leak its previous configuration into the
// next caller — apply the desired options explicitly via applyOpts /
// SetQPack after Reset.
func (e *Encoder) Reset() {
	e.resetForReuse()
	e.opts = OptSpeed
	e.mode = Fast
	e.qpack = false
	e.gorillaFloat = false
	e.rans = false
	e.colIndex = false
	e.fsst = false
	e.pairPred = false
	e.mtf = false
	e.fsstDict = nil
}

// resetForReuse clears the per-message / per-stream encoder state — the output
// buffer, frame header flag, dense intern/shape state, and the row-scaled
// scratch — while KEEPING the configured options/mode/codec flags. Encoder.Reset
// layers the option reset on top (a pooled encoder is reconfigured per acquire);
// StreamEncoder.Reset calls this directly so one encoder + its grown intern table
// is reused across independent streams without a fresh newEncState allocation.
func (e *Encoder) resetForReuse() {
	e.buf = e.buf[:0]
	e.customFramed = false
	// Row-scaled ALP staging scratch: retain across batches, drop only past the
	// hard ceiling so a one-off giant float slice can't pin unbounded memory.
	// Pure []uint64 (no pointers) so no clear is needed.
	if cap(e.alpScratch) > maxRetainedColScratch {
		e.alpScratch = nil
	}
	e.vecScratch.Reset()
	if cap(e.wideF64) > maxRetainedColScratch {
		e.wideF64 = nil
	}
	if cap(e.vecBatchFlat) > maxRetainedColScratch {
		e.vecBatchFlat = nil
	}
	// vecBatchRows holds slice headers into vecBatchFlat; drop it independently
	// (many rows × small dim keeps flat under the ceiling) and clear first so the
	// retained headers do not GC-pin the prior flat backing.
	if cap(e.vecBatchRows) > maxRetainedColScratch || e.vecBatchFlat == nil {
		clear(e.vecBatchRows[:cap(e.vecBatchRows)])
		e.vecBatchRows = nil
	}
	if cap(e.vecBatchNorms) > maxRetainedColScratch {
		e.vecBatchNorms = nil
	}
	// Keyed-diff match map: drop a spike-sized backing, else clear() it in place.
	// The keys are unsafe.String / string-header aliases into the caller's prior
	// key strings or []struct element backing (keyTokenAt), so leaving them
	// populated across a pool recycle GC-pins that caller data until the next
	// keyed build — mirror the clear-on-reset policy already applied to the other
	// aliasing string-keyed pooled state (colScratchStr, d.values, strDictMap).
	if len(e.keyIdx) > 1<<16 {
		e.keyIdx = nil
	} else {
		clear(e.keyIdx)
	}
	e.keyIdxBusy = false
	e.headerOut = false
	if e.state != nil {
		e.state.reset()
	}
	e.depth = 0
	if e.maxDepth == 0 {
		e.maxDepth = DefaultMaxDepth
	}
	// Drop any PreIntern entries — they reference caller-supplied backing
	// pointers that are not safe to assume valid across a pool / stream recycle.
	if e.preIntern != nil {
		e.preIntern = e.preIntern[:0]
	}
}

// ApplyOpts reconfigures the encoder via the options bit-mask.
// Equivalent to recreating the encoder with NewEncoderWith but
// preserves the existing pool buffer and (Dense-mode) intern
// state. The pool-backed Marshal* entry points call this on every
// acquire; callers driving an Encoder directly use it to switch
// between OptSpeed and OptBalanced without re-allocating.
//
//go:nosplit
func (e *Encoder) ApplyOpts(opts Options) { e.applyOpts(opts) }

// EncodeValue runs the reflect-driven Marshal pipeline on v
// against this Encoder. Convenience for callers driving an
// Encoder directly (e.g. with PreIntern) — equivalent to what
// the pool-backed Marshal does internally.
func (e *Encoder) EncodeValue(v any) error { return encodeReflect(e, v) }

// PreIntern registers the given strings against the encoder's
// intern table up front. Subsequent WriteString / WriteBytes
// calls that pass the SAME backing string header (i.e. the same
// underlying byte pointer and length) skip the hash + slot probe
// and emit a state-ref directly against the cached intern id.
//
// Intended for power users who know their hot string pool ahead
// of time — service names, region codes, enum-like values drawn
// from a fixed slice. Real telemetry payloads draw 90 %+ of
// their dense intern hits from such pools. Skipping the
// hash/probe on those calls is the main per-emit cost left on
// the encode hot path.
//
// Requires OptDense to be applied first (otherwise the call is a
// no-op). The registered identities are dropped on Reset so a
// pooled encoder does not carry caller-supplied pointers across
// recycles.
//
// Safety: the caller must keep the backing memory of every
// PreIntern'd string alive for the lifetime of the next encode
// call. For literals embedded in a slice / global / struct
// field this is automatic; for short-lived stack strings the
// caller is responsible.
func (e *Encoder) PreIntern(strs ...string) {
	if e.state == nil || !e.opts.Has(OptDense) {
		return
	}
	if e.preIntern == nil {
		e.preIntern = make([]preInternEntry, 0, len(strs))
	}
	for _, s := range strs {
		if len(s) < e.minIntern {
			continue
		}
		// Record the caller's backing pointer + length without
		// touching the intern table yet. The first WriteString
		// that matches will run the regular intern path
		// (emitting tagInternStr on the wire so the decoder can
		// register the entry) and back-fill the id here.
		e.preIntern = append(e.preIntern, preInternEntry{
			ptr: unsafe.Pointer(unsafe.StringData(s)),
			n:   uintptr(len(s)),
			id:  preInternUnseen,
		})
	}
}

// SetMaxDepth caps reflect-path pointer/struct recursion. The default
// (DefaultMaxDepth = 10000) is sufficient for any normal payload and
// rejects pointer cycles before they stack-overflow the goroutine.
// Set to 0 to disable the check entirely — only safe when the caller
// can prove the input graph is acyclic.
func (e *Encoder) SetMaxDepth(d int) { e.maxDepth = d }

// Bytes returns the encoded payload. It aliases the encoder's buffer and
// is only valid until the next write or Reset.
func (e *Encoder) Bytes() []byte { return e.buf }

// Take returns the encoded payload and detaches it from the encoder. The
// caller takes ownership.
func (e *Encoder) Take() []byte {
	out := e.buf
	e.buf = nil
	e.headerOut = false
	if e.state != nil {
		e.state.reset()
	}
	return out
}

// SetBuffer installs a caller-owned buffer (truncated to length 0).
func (e *Encoder) SetBuffer(b []byte) {
	e.buf = b[:0]
	e.headerOut = false
}

// AdoptBuffer installs b as the working buffer, preserving its current
// length. Used to continue writing into a buffer that already carries
// data — for example, after a nested type returned its extended slice.
// headerOut is left unchanged.
func (e *Encoder) AdoptBuffer(b []byte) {
	e.buf = b
}

// SetIntern overrides the Dense-mode tuning knobs. Zero values keep the
// current setting.
func (e *Encoder) SetIntern(min int, cap int) {
	if min > 0 {
		e.minIntern = min
	}
	if cap > 0 {
		if cap > maxInternEntries {
			cap = maxInternEntries // keep max id below the 0xFFFF uint16 sentinel
		}
		e.maxStateEntries = cap
	}
}

// streamHeaderLen is the fixed byte length writeHeader emits: the 3 magic bytes,
// the version byte, and the flag byte.
const streamHeaderLen = 5

func (e *Encoder) writeHeader() {
	if e.headerOut {
		return
	}
	flag := byte(0)
	if e.mode == Dense {
		flag |= FlagDense
	}
	if e.qpack {
		flag |= FlagQPack
	}
	// FlagColIndex is NOT set here: encodeColumnar backpatches it only when it
	// actually emits a column index, so OptColumnIndex on a non-columnar
	// payload stays a true no-op (header byte unchanged).
	e.headerFlagAt = len(e.buf) + 4
	e.buf = append(e.buf, Magic0, Magic1, Magic2, Version1, flag)
	e.headerOut = true
}

// SetQPack toggles QPack codec emission. When true, slice fast paths
// produce packed/encoded forms (bitpacked bools, FOR-packed integers,
// Gorilla-encoded floats, raw-LE bulk) instead of per-element tag streams.
// Setting must happen before the first write of the value (the header is
// emitted lazily and carries the FlagQPack hint when this is on).
func (e *Encoder) SetQPack(v bool) { e.qpack = v }

// SetVectorBudget sets the fidelity target used when OptLossyVec is active.
// No effect unless OptLossyVec is in the encoder's options.
func (e *Encoder) SetVectorBudget(b VectorBudget) { e.vecBudget = b }

// QPack reports whether QPack codec emission is enabled.
func (e *Encoder) QPack() bool { return e.qpack }

// EnsureHeader forces a header write if one has not been emitted yet.
func (e *Encoder) EnsureHeader() { e.writeHeader() }

// MarkHeaderWritten tells the encoder the QDF header is already present
// in its buffer (e.g. left there by a parent encoder). The next write
// will skip the header emission.
func (e *Encoder) MarkHeaderWritten() { e.headerOut = true }

// AppendBytes appends raw, already-valid wire bytes. Bypasses tag
// dispatch; used by generated code to emit pre-encoded field-name
// prefixes.
func (e *Encoder) AppendBytes(p []byte) {
	e.writeHeader()
	e.buf = append(e.buf, p...)
}

// ----- primitives -----

func (e *Encoder) WriteNil() {
	e.writeHeader()
	e.buf = append(e.buf, tagNil)
}

func (e *Encoder) WriteBool(v bool) {
	e.writeHeader()
	if v {
		e.buf = append(e.buf, tagTrue)
	} else {
		e.buf = append(e.buf, tagFalse)
	}
}

func (e *Encoder) WriteUint(v uint64) {
	e.writeHeader()
	switch {
	case v <= tagFixintMax:
		e.buf = append(e.buf, byte(v))
	case v <= math.MaxUint8:
		e.buf = append(e.buf, tagUint8, byte(v))
	case v <= math.MaxUint16:
		e.buf = appendU16(append(e.buf, tagUint16), uint16(v))
	case v <= math.MaxUint32:
		e.buf = appendU32(append(e.buf, tagUint32), uint32(v))
	default:
		e.buf = appendU64(append(e.buf, tagUint64), v)
	}
}

func (e *Encoder) WriteInt(v int64) {
	e.writeHeader()
	if v >= 0 {
		e.WriteUint(uint64(v))
		return
	}
	switch {
	case v >= -negfixintMaxAbs:
		// negfixint packs -1..-8 as 0xD8..0xDF; decoder mirrors as
		// -(int8(tag & 0x07) + 1).
		e.buf = append(e.buf, tagNegfixint|byte(-v-1))
	case v >= math.MinInt8:
		e.buf = append(e.buf, tagInt8, byte(int8(v)))
	case v >= math.MinInt16:
		e.buf = appendU16(append(e.buf, tagInt16), uint16(int16(v)))
	case v >= math.MinInt32:
		e.buf = appendU32(append(e.buf, tagInt32), uint32(int32(v)))
	default:
		e.buf = appendU64(append(e.buf, tagInt64), uint64(v))
	}
}

func (e *Encoder) WriteFloat32(v float32) {
	e.writeHeader()
	if e.opts.Has(OptCanonical) {
		v = canonicalizeFloat32(v)
	}
	e.buf = appendU32(append(e.buf, tagFloat32), math.Float32bits(v))
}

func (e *Encoder) WriteFloat64(v float64) {
	e.writeHeader()
	if e.opts.Has(OptCanonical) {
		v = canonicalizeFloat64(v)
	}
	e.buf = appendU64(append(e.buf, tagFloat64), math.Float64bits(v))
}

// WriteString writes s. In Dense mode (OptDense set), eligible strings
// are intern-encoded.
func (e *Encoder) WriteString(s string) {
	e.writeHeader()
	st := e.state
	dense := e.opts.Has(OptDense)
	if dense && !e.stateSuspended && st != nil && len(s) >= e.minIntern && int(st.internLoad) < e.maxStateEntries {
		// PreIntern identity fast path: if the caller registered
		// this exact backing pointer + length via Encoder.PreIntern,
		// the cached id (after first sight on the wire) lets us
		// skip the hash + slot probe and emit a state-ref directly.
		// The slice is empty for default callers, so the gating
		// `len > 0` check is a single branch-predicted compare
		// that does not regress the default path.
		//
		// On the first WriteString of a PreIntern'd string the
		// entry's id is still preInternUnseen — we fall through
		// to the normal intern path (which emits tagInternStr on
		// the wire so the decoder can register the entry) and
		// back-fill the id below.
		preInternIdx := -1
		if len(e.preIntern) > 0 {
			sp := unsafe.Pointer(unsafe.StringData(s))
			sn := uintptr(len(s))
			for i := range e.preIntern {
				if e.preIntern[i].ptr == sp && e.preIntern[i].n == sn {
					if e.preIntern[i].id != preInternUnseen {
						id := e.preIntern[i].id
						if st.lastID == id {
							e.buf = append(e.buf, tagStateRepeat)
							if e.pairPred {
								st.pairRecord(id, id)
							}
							return
						}
						e.emitStateRef(id)
						return
					}
					preInternIdx = i
					break
				}
			}
		}
		id, ok := st.lookupOrAssign(s)
		if preInternIdx >= 0 {
			e.preIntern[preInternIdx].id = id
		}
		if ok {
			// Repeat hot path: hand-inlined out of emitStateRef so the
			// most common Dense hit avoids the non-inlinable call.
			if st.lastID == id {
				e.buf = append(e.buf, tagStateRepeat)
				if e.pairPred {
					st.pairRecord(id, id)
				}
				return
			}
			e.emitStateRef(id)
			return
		}
		e.buf = append(e.buf, tagInternStr)
		e.buf = appendUvarint(e.buf, uint64(len(s)))
		e.buf = appendString(e.buf, s)
		if st.lastID != lruInvalidID && e.pairPred {
			st.pairRecord(st.lastID, id)
		}
		st.lastID = id
		return
	}
	if dense && st != nil {
		st.lastID = lruInvalidID
	}
	e.writeStringInline(s)
}

// emitStateRef writes a state-ref to id. Four forms are possible, the
// encoder picks the smallest:
//
//	tagStateRepeat                   1 byte total, when id == lastID
//	tagStatePair  + varuint(pairR)   1 + uvarintLen(pairR) bytes,
//	                                 when id is in lastID's predictor ring
//	tagStateMTF   + varuint(mtfR)    1 + uvarintLen(mtfR) bytes
//	tagStateRef   + varuint(id)      1 + uvarintLen(id) bytes
//
// MTF rank comes from the encState LRU. The pair rank comes from the
// per-prev successor ring (Markov-1 predictor). The wire never grows
// over the plain tagStateRef encoding because we only pick the
// alternative when its varuint is strictly shorter than the raw id
// varuint.
//
// Every successful emit moves id to the LRU head AND records the
// (prev, id) transition in the pair predictor so the decoder's mirror
// chain stays in sync.
func (e *Encoder) emitStateRef(id uint32) {
	st := e.state
	pairOn := e.pairPred
	if st.lastID == id {
		e.buf = append(e.buf, tagStateRepeat)
		if pairOn {
			st.pairRecord(id, id)
		}
		return
	}
	prev := st.lastID
	prevValid := prev != lruInvalidID
	// Small-id fast path: id<128 means the raw state-ref is already a
	// 2-byte literal (tag + 1-byte varuint). Neither MTF (rank varuint
	// ≥1 byte) nor Pair (rank varuint =1 byte) can be strictly shorter,
	// so we write the raw form directly and skip the LRU walk entirely.
	if id < 0x80 {
		st.lruMoveFront(id)
		e.buf = append(e.buf, tagStateRef, byte(id))
		if prevValid && pairOn {
			st.pairRecord(prev, id)
		}
		st.lastID = id
		return
	}
	// Pair predictor: when (prev, id) is in the predictor ring the
	// payload is always a single byte (ranks 0..3), so the pair form
	// is strictly shorter than the raw state-ref whenever id needs a
	// multi-byte varuint — which the small-id branch above just
	// excluded.
	if prevValid && pairOn {
		if st.pairLookup(prev, id) {
			st.lruMoveFront(id)
			// Top-1 predictor: rank is always 0 (see encState.pairLookup
			// comment), so the rank byte is a hard-coded literal here.
			e.buf = append(e.buf, tagStatePair, 0)
			st.pairRecord(prev, id)
			st.lastID = id
			return
		}
	}
	// MTF — fall back on raw if the rank varuint is not shorter than
	// the id varuint. When OptMTF is off the LRU is still maintained
	// (the chain must stay in sync for the rest of the codec) but the
	// rank is never emitted; the raw state-ref form is used instead.
	//
	// Rank discovery used to walk the LRU linked list head→tail
	// looking for id (state.go's old lruMoveToFront). Profiling on
	// telemetry workloads showed that pointer-chase walk consumed
	// >50% of encode CPU because each step is a cache-cold random
	// index into lruNext[]. The MRU ring side-cache (mruRank) gives
	// O(1) rank for the recent-128 emits — a contiguous, cache-warm
	// scan that covers exactly the rank range where MTF beats the
	// 2-byte raw state-ref (rank ≤ 127). On a ring miss the chain
	// rank is necessarily ≥ 128 so the raw form would be picked
	// anyway; we skip the walk entirely and emit raw.
	idLen := uvarintLen(uint64(id))
	if e.mtf {
		if rank, ok := st.mruRank(id); ok {
			st.lruMoveFront(id)
			if rankLen := uvarintLen(uint64(rank)); rankLen < idLen {
				e.buf = append(e.buf, tagStateMTF)
				e.buf = appendUvarint(e.buf, uint64(rank))
			} else {
				e.buf = append(e.buf, tagStateRef)
				e.buf = appendUvarint(e.buf, uint64(id))
			}
		} else {
			st.lruMoveFront(id)
			e.buf = append(e.buf, tagStateRef)
			e.buf = appendUvarint(e.buf, uint64(id))
		}
	} else {
		st.lruMoveFront(id)
		e.buf = append(e.buf, tagStateRef)
		e.buf = appendUvarint(e.buf, uint64(id))
	}
	if prevValid && pairOn {
		st.pairRecord(prev, id)
	}
	st.lastID = id
}

// WriteStringInline forces an in-line encoding even when Dense intern would
// be eligible. Use for fields known to be unique per message.
func (e *Encoder) WriteStringInline(s string) {
	e.writeHeader()
	// In Dense mode the decoder resets lastID to invalid on every inline string
	// read, so the encoder must do the same here — otherwise a later repeated
	// value emits tagStateRepeat against a lastID the decoder has already
	// dropped, desyncing the state tables (silent wrong value or
	// ErrUnknownStateID). Mirrors WriteString's own inline fallthrough.
	if e.opts.Has(OptDense) && e.state != nil {
		e.state.lastID = lruInvalidID
	}
	e.writeStringInline(s)
}

// stringInlineHeaderLen returns the number of header bytes writeStringInline
// emits for a string of length n (fixstr 1, str8 2, str16 3, str32 5). Kept
// next to writeStringInline so the two stay in lockstep; used by size
// estimators (e.g. the tagColStrRaw never-larger gate).
func stringInlineHeaderLen(n int) int {
	switch {
	case n <= int(tagFixstrMask):
		return 1
	case n <= math.MaxUint8:
		return 2
	case n <= math.MaxUint16:
		return 3
	default:
		return 5
	}
}

func (e *Encoder) writeStringInline(s string) {
	n := len(s)
	// One Grow up front for worst-case header (5 B) + body beats append's
	// amortized growth on the hot path.
	b := slices.Grow(e.buf, 5+n)
	switch {
	case n <= int(tagFixstrMask):
		b = append(b, tagFixstr|byte(n))
	case n <= math.MaxUint8:
		b = append(b, tagStr8, byte(n))
	case n <= math.MaxUint16:
		b = appendU16(append(b, tagStr16), uint16(n))
	default:
		b = appendU32(append(b, tagStr32), uint32(n))
	}
	b = append(b, s...)
	e.buf = b
}

// WriteBytes writes a []byte. In Dense mode, eligible payloads are
// intern-encoded. The intern table is keyed on the byte sequence, so a
// string and a []byte with identical content share an ID.
func (e *Encoder) WriteBytes(b []byte) {
	e.writeHeader()
	st := e.state
	dense := e.opts.Has(OptDense)
	if dense && !e.stateSuspended && st != nil && len(b) >= e.minIntern && int(st.internLoad) < e.maxStateEntries {
		key := unsafestr.String(b)
		id, ok := st.lookupOrAssign(key)
		if ok {
			if st.lastID == id {
				e.buf = append(e.buf, tagStateRepeat)
				if e.pairPred {
					st.pairRecord(id, id)
				}
				return
			}
			e.emitStateRef(id)
			return
		}
		e.buf = append(e.buf, tagInternBin)
		e.buf = appendUvarint(e.buf, uint64(len(b)))
		e.buf = append(e.buf, b...)
		if st.lastID != lruInvalidID && e.pairPred {
			st.pairRecord(st.lastID, id)
		}
		st.lastID = id
		return
	}
	if dense && st != nil {
		st.lastID = lruInvalidID
	}
	e.writeBytesInline(b)
}

func (e *Encoder) writeBytesInline(p []byte) {
	n := len(p)
	out := slices.Grow(e.buf, 5+n)
	switch {
	case n <= math.MaxUint8:
		out = append(out, tagBin8, byte(n))
	case n <= math.MaxUint16:
		out = appendU16(append(out, tagBin16), uint16(n))
	default:
		out = appendU32(append(out, tagBin32), uint32(n))
	}
	out = append(out, p...)
	e.buf = out
}

// WriteArrayHeader writes the header for an array of n elements. The
// caller must follow with exactly n element writes. n must not exceed
// math.MaxUint32 — the wire array count is a uint32, so a larger n is
// unrepresentable and panics rather than silently truncating the count
// (which would desync the decoder against the n bodies that follow).
func (e *Encoder) WriteArrayHeader(n int) {
	e.writeHeader()
	if n < 0 {
		// A negative count would narrow to a garbage byte (byte(-1)==0xFF) and
		// desync the decoder. Treat it as the empty header the caller's
		// for-i<n loop would actually emit (zero iterations).
		n = 0
	}
	if uint64(n) > math.MaxUint32 {
		// The wire count is a uint32 (there is no tagArr64). Truncating would
		// emit a count smaller than the n bodies the caller writes next, an
		// undecodable desync; fail loud on this unrepresentable input instead.
		panic("qdf: array length exceeds uint32 wire limit")
	}
	switch {
	case n <= int(tagFixarrMask):
		e.buf = append(e.buf, tagFixarr|byte(n))
	case n <= math.MaxUint16:
		e.buf = appendU16(append(e.buf, tagArr16), uint16(n))
	default:
		e.buf = appendU32(append(e.buf, tagArr32), uint32(n))
	}
}

// WriteMapHeader writes the header for a map of n entries. The caller
// must follow with exactly n key/value pairs. n must not exceed
// math.MaxUint32 — the wire map count is a uint32, so a larger n is
// unrepresentable and panics rather than silently truncating the count.
func (e *Encoder) WriteMapHeader(n int) {
	e.writeHeader()
	if n < 0 {
		// See WriteArrayHeader: a negative count narrows to a garbage byte and
		// desyncs the decoder; emit the empty header instead.
		n = 0
	}
	if uint64(n) > math.MaxUint32 {
		// See WriteArrayHeader: the wire count is a uint32 (no tagMap64), so a
		// larger count is unrepresentable; fail loud rather than truncate.
		panic("qdf: map length exceeds uint32 wire limit")
	}
	switch {
	case n <= math.MaxUint8:
		e.buf = append(e.buf, tagMap8, byte(n))
	case n <= math.MaxUint16:
		e.buf = appendU16(append(e.buf, tagMap16), uint16(n))
	default:
		e.buf = appendU32(append(e.buf, tagMap32), uint32(n))
	}
}

// StructShape begins a shape-interned struct emission for a code-generated type
// (the decode-time counterpart is Decoder.ReadStructHeader). token is a stable
// per-type address — a package-level var the generated EncodeQDF passes;
// fieldHdrs are the pre-encoded fixstr/strN field-name headers in field order.
//
// On the first emit for token on this encoder it DECLARES the shape — tagMapShape,
// id 0, the field count, then the names — so the decoder registers it; every
// later emit writes only tagMapShape + the shape ID. The caller then writes the
// field VALUES in field order (no names). Across a slice of the same struct
// threaded through one encoder, the field names are written once instead of per
// record. Reuses the shared shape-ID space, so a generated buffer stays decodable
// by the reflection path (tagMapShape is standard wire).
func (e *Encoder) StructShape(token *byte, fieldHdrs [][]byte) {
	e.writeHeader()
	if e.state == nil {
		e.state = newEncState()
	}
	st := e.state
	if e.stateSuspended {
		// Inside a never-larger trial (diffKeyedSlice / diffColumnar): emit a
		// self-contained declaration every time and never bind or reference a
		// token. A bound id whose declaration is thrown away with the losing
		// candidate would dangle (ErrUnknownStateID) — the same leak the trial
		// suspends interning to avoid. Still advance the shape counter so it
		// tracks the decoder (which registers a shape per declaration it reads);
		// the trial re-bases the counter for the discarded candidate.
		st.shapeDeclareEnc()
		e.buf = append(e.buf, tagMapShape)
		e.buf = appendUvarint(e.buf, 0) // 0 ⇒ declaration follows
		e.buf = appendUvarint(e.buf, uint64(len(fieldHdrs)))
		for _, h := range fieldHdrs {
			e.buf = append(e.buf, h...)
		}
		return
	}
	if id := st.shapeForToken(token); id != 0 {
		e.buf = append(e.buf, tagMapShape)
		e.buf = appendUvarint(e.buf, uint64(id))
		return
	}
	id := st.shapeDeclareEnc()
	st.shapeBindToken(token, id)
	e.buf = append(e.buf, tagMapShape)
	e.buf = appendUvarint(e.buf, 0) // 0 ⇒ declaration follows
	e.buf = appendUvarint(e.buf, uint64(len(fieldHdrs)))
	for _, h := range fieldHdrs {
		e.buf = append(e.buf, h...)
	}
}

// WriteTimestamp writes a full-range timestamp as two uvarints:
// sec (zigzag-encoded signed int64 seconds since Unix epoch) and
// nsec (unsigned uint32 nanoseconds in [0, 999_999_999]).
// This replaces the old fixed-8-byte UnixNano encoding (clean break).
func (e *Encoder) WriteTimestamp(sec int64, nsec uint32) {
	e.writeHeader()
	e.buf = append(e.buf, tagTimestamp)
	e.buf = appendUvarint(e.buf, zigzagEncode64(sec))
	e.buf = appendUvarint(e.buf, uint64(nsec))
}

// ----- helpers -----

func appendU16(b []byte, v uint16) []byte { return binary.LittleEndian.AppendUint16(b, v) }
func appendU32(b []byte, v uint32) []byte { return binary.LittleEndian.AppendUint32(b, v) }
func appendU64(b []byte, v uint64) []byte { return binary.LittleEndian.AppendUint64(b, v) }

// appendString uses the runtime's append-string fast path to avoid the
// implicit []byte(s) copy.
func appendString(b []byte, s string) []byte { return append(b, s...) }

func readU16(b []byte) uint16 { return binary.LittleEndian.Uint16(b) }
func readU32(b []byte) uint32 { return binary.LittleEndian.Uint32(b) }
func readU64(b []byte) uint64 { return binary.LittleEndian.Uint64(b) }
