// Package qdf is a compact, streaming-friendly binary serialization
// format.
//
// # Quick start
//
// One encode entry point: Marshal(v, opts). The Options bit-mask
// picks which codecs run for that call. Convenience bundles cover
// the common tradeoffs:
//
//	OptSpeed       — Fast mode, no codecs (matches encoding/json shape)
//	OptBalanced    — Dense + QPack + shape interning + Markov-1 + MTF
//	OptCompression — OptBalanced + heavier codecs: Gorilla XOR and ALP for
//	                 float series, a final order-0 rANS entropy pass, and
//	                 FSST for columnar string columns (large wire reduction
//	                 at higher encode CPU)
//
// A single decoder handles every variant; the wire header self-
// describes the dialect.
//
//	b, err := qdf.Marshal(v, qdf.OptSpeed)        // hot path
//	b, err := qdf.Marshal(v, qdf.OptBalanced)     // telemetry / logs
//	b, err := qdf.Marshal(v, qdf.OptCompression)  // backup / archive
//
//	err := qdf.Unmarshal(b, &v)
//
//	enc := qdf.NewStreamEncoder(w, qdf.OptBalanced)
//	dec := qdf.NewStreamDecoder(r)
//
// Marshal returns a freshly-allocated slice owned by the caller.
// AppendMarshal is the zero-extra-copy variant.
//
// # Data shapes and codecs
//
// Under OptBalanced the encoder picks a codec per slice from the data
// itself: integer slices use Frame-of-Reference, delta, run-length or
// dictionary coding; bool slices bit-pack; repeated strings are interned
// and back-referenced. A slice of homogeneous flat structs is transposed
// and compressed column-by-column automatically — numeric columns get the
// integer codecs, an enum-like string column is dictionary-coded (distinct
// table + a bit-packed index per row) when that beats per-value interning,
// an optional (*T, T a scalar or string) field becomes a presence bitmap
// plus a dense present-only column instead of forcing the struct back to
// row-major,
// other repeated string columns collapse — with no flag, and a per-array
// probe falls back to the plain row encoding when columnar would not help. The float codecs that trade encode CPU for size live behind
// OptCompression: Gorilla XOR on smooth series, ALP on quantized/decimal
// float64 grids (quantized telemetry, prices, latencies).
//
// # Concurrency
//
// Marshal, Unmarshal and AppendMarshal are safe for concurrent use: each
// call leases its own encoder/decoder from a pool. A single Encoder,
// Decoder or StreamEncoder value is not safe to share across goroutines —
// use one per goroutine. A Dense document is a sequential stream, so one
// document cannot be encoded or decoded in parallel; to parallelize a
// large dataset, split it into independent shards and Marshal each
// separately.
//
// # Selective decode
//
// Under OptColumnIndex a columnar []struct payload carries a fixed-width
// uint32 column-length index, written after the shape declaration and before
// the column bodies (~4 bytes per column). A reader wanting only some columns
// then advances past the rest using the index instead of decoding them, so
// the cost becomes O(columns you read), not O(all columns). Two entry points:
// decode into a subset struct whose fields are a subset of the wire (matched
// by qdf tag / field name) with plain Unmarshal — wire columns absent from the
// target are skipped, target fields absent from the wire are left zero — or
// name the columns explicitly with UnmarshalColumns (also drives the dynamic
// *[]map[string]any form). The option is a true no-op on non-columnar
// payloads: the flag is backpatched only when the index is actually emitted,
// so the default columnar wire stays byte-identical when it is off. Without
// the index a subset decode still returns correct results by decoding and
// discarding unwanted columns. The index is a single-message feature and is
// not emitted in streaming mode.
//
// # Predicate pushdown
//
// Unmarshal accepts trailing QueryOption arguments that filter and project a
// columnar []struct payload before it is materialized. Where(field, pred)
// keeps only the rows for which a typed predicate holds — func(T) bool over
// the column's element type, AND-ed across multiple Where clauses, with zero
// per-value boxing — and Select(fields...) restricts the output to a named
// subset of columns. The filter field need not appear in the output type, and
// the result can be either *[]Struct or *[]map[string]any. The decoder reads
// each predicate column whole, evaluates it into a row bitmask, ANDs the
// masks, then compacts and materializes only the surviving rows of the
// projected columns; OptColumnIndex lets it seek past skipped columns instead
// of decode-and-discard. A nullable column's nil rows never match. Pushdown
// fires only on columnar payloads — a non-columnar payload (a single struct,
// or a batch the columnar probe declined) returns an error wrapping
// ErrUnsupported; a missing field yields ErrFieldNotFound and a predicate
// whose argument type mismatches the column yields ErrTypeMismatch, both as a
// *QueryError (errors.Is / errors.As). Versus a full decode followed by a
// manual loop, pushdown moves materially fewer bytes (about 4x less on a wide
// batch with 1% selectivity) and runs roughly 2x faster. No other Go
// serializer reads back a filtered, projected subset of a batch without
// decoding the whole thing — see docs/PREDICATE-PUSHDOWN.md for the full
// guide, rationale, and tutorial.
//
// # Structural delta
//
// Diff(old, new, opts) computes a patch carrying only the locations that
// changed; Apply(&base, patch) merges it back in place. The patch is
// self-describing — the receiver never has to know which Options produced it —
// and far smaller than a re-encode because unchanged fields, elements, and keys
// cost no bytes. Slices whose elements have a stable identity field tagged ",key"
// match by key (cheap reorder / middle edit) instead of by position, and an
// equal-length columnar batch is diffed column-by-column when that wins. Two
// fingerprints guard Apply against the wrong type or the wrong base.
// BaselineRegistry[T] applies a chain of patches in a state-sync stream without
// threading the previous value by hand. See docs/DELTA.md.
//
// # Canonical encoding
//
// OptCanonical makes the same logical value serialize to byte-identical output:
// map keys are emitted in sorted order (every key kind) and floats are
// normalized (-0.0 → +0.0, any NaN → one quiet NaN). The bytes are then safe to
// hash, sign, content-address, or deduplicate. It is encode-side only and
// decodes like any other qdf output; it is lossy for the sign of zero and the
// NaN payload, so use the default mode for a bit-exact float round-trip. See
// docs/CANONICAL.md.
//
// See docs/USAGE.md in the repository for a fuller guide.
//
// # Public API surface
//
// The package contract is split across four layers; everything else
// under internal/ is implementation detail and may change between
// releases without notice.
//
// Top-level entry points (file: qdf.go):
//
//	func Marshal(v any, opts Options) ([]byte, error)
//	func AppendMarshal(dst []byte, v any, opts Options) ([]byte, error)
//	func Unmarshal(data []byte, out any) error
//	type Options uint32   // bit-mask of OptDense, OptQPack, OptMTF, …
//
// Selective columnar decode (file: columnar_select.go):
//
//	func UnmarshalColumns(data []byte, out any, fields ...string) error
//
// Predicate pushdown — filter rows and project columns on a columnar
// []struct, AND-ed typed predicates with zero per-value boxing (file:
// query.go):
//
//	func Unmarshal(data []byte, out any, opts ...QueryOption) error
//	func Where[T Queryable](field string, pred func(T) bool) QueryOption
//	func Select(fields ...string) QueryOption
//
// Typed convenience wrappers — generic, zero-extra-reflection (file:
// qdf_generic.go):
//
//	func MarshalT[T any](v T, opts Options) ([]byte, error)
//	func AppendMarshalT[T any](dst []byte, v T, opts Options) ([]byte, error)
//	func UnmarshalT[T any](data []byte, out *T) error
//
// Structural delta — patch / merge two values, key-matched slices, and
// content-addressed baselines for state-sync streams (files: delta.go,
// delta_baseline.go):
//
//	func Diff[T any](old, new T, opts Options) ([]byte, error)
//	func AppendDiff[T any](dst []byte, old, new T, opts Options) ([]byte, error)
//	func Apply[T any](base *T, patch []byte) error
//	func ApplyArena[T any](base *T, patch []byte, arena *Arena) error
//	type BaselineRegistry[T any]   // NewBaselineRegistry / Register / Apply / Len
//
// Low-level encoder / decoder for callers driving the wire directly
// or interoperating with the qdfgen code generator (files:
// encoder.go, decoder.go, stream.go):
//
//	type Encoder struct{ … }
//	    NewEncoder(mode Mode) *Encoder
//	    NewEncoderWith(opts Options) *Encoder
//	    NewEncoderOnBuf(buf []byte, mode Mode) *Encoder
//	    (e *Encoder) WriteString / WriteBytes / WriteInt / WriteUint /
//	                WriteFloat32 / WriteFloat64 / WriteBool / WriteNil /
//	                WriteArrayHeader / WriteMapHeader /
//	                WriteTimestamp / WriteStringInline
//	    (e *Encoder) AppendBytes / EnsureHeader / Bytes / Take /
//	                Reset / SetBuffer / AdoptBuffer / SetIntern /
//	                SetMaxDepth / SetQPack / QPack / ApplyOpts /
//	                EncodeValue / PreIntern
//	type Decoder struct{ … }
//	    NewDecoder() *Decoder
//	    NewDecoderOnBuf(buf []byte) *Decoder
//	    (d *Decoder) ReadString / ReadStringBytes / ReadBytes /
//	                ReadBool / ReadInt / ReadUint / ReadFloat32 /
//	                ReadFloat64 / ReadNil / ReadArrayHeader /
//	                ReadMapHeader / ReadTimestamp / Skip /
//	                PeekTag / IsNil / Pos / Remaining / RemainingBytes /
//	                Advance / SetInput / SetNoCopy / MarkHeaderRead /
//	                CheckLength / InternKey
//	type StreamEncoder, type StreamDecoder         // io.Writer / io.Reader
//
// User-side hook points (file: marshaler.go):
//
//	type Marshaler interface   { MarshalQDF(dst []byte) ([]byte, error) }
//	type Unmarshaler interface { UnmarshalQDF(src []byte) (int, error) }
//
// The qdfgen code generator (cmd/qdfgen) emits MarshalQDF /
// UnmarshalQDF methods for user struct types; the codegen path
// uses the same Encoder / Decoder primitives listed above and
// requires no reflection at runtime.
package qdf

import (
	"sync"
)

// 4 KiB initial buffer covers most messages without a growth realloc.
// Larger payloads grow once and the grown buffer is recycled through the
// pool. Buffers that pass maxPooledBuf are dropped before being returned
// so idle memory stays bounded.
const (
	initialEncBuf = 4 * 1024
	// maxPooledBuf bounds how big a buffer the encoder pool will
	// retain between Marshal calls. Profiling on large batches
	// (50 k+ records, ~60 MiB output) showed the previous 1 MiB cap
	// caused every iteration to re-grow the buffer from 4 KiB to
	// the final size — runtime.memmove + runtime.madvise dominated
	// the trace. Retaining up to 16 MiB lets large encoders reuse
	// the high-water buffer once it warms up, while still releasing
	// truly outlier payloads instead of pinning them to the pool.
	maxPooledBuf = 16 * 1024 * 1024
	// maxHintedBuf bounds the sizes worth pre-allocating from the hint. The hint
	// pays where the buffer is assembled from many modest appends and the
	// doubling chain from initialEncBuf is long relative to the output — a
	// 1.26 MB IoT batch allocated 7.2 MB to deliver 1.26. It does not pay above
	// that: slices.Grow serves a large request with a single right-sized
	// allocation, so the chain is already short (a 17.9 MB dense batch allocates
	// 18.3 MB) and any pre-allocation is either overshoot or an undershoot that
	// doubles on top. Measured on that batch: 18.3 MB/op unhinted, 21.9 with a
	// cap-sized hint, 39.3 with a len-sized one.
	maxHintedBuf = 4 * 1024 * 1024
	// maxRetainedDeltaScratch bounds the Delta+FOR decode scratch a pooled
	// decoder keeps between calls. Retains it for columns up to ~16 k rows
	// (the maxStateEntries scale) and drops it after a one-off larger decode so
	// the pool never pins an outlier buffer.
	maxRetainedDeltaScratch = 1 << 14
	// maxRetainedWideScratch bounds the int32→int64 / uint32→uint64 widening
	// scratch a pooled encoder keeps between calls (same ~16 k-element scale as
	// the delta scratch); a one-off larger narrow-int slice is dropped so the
	// pool never pins an outlier buffer.
	maxRetainedWideScratch = 1 << 17
)

// Options is a bit-mask of per-call encoder feature toggles. A zero
// value (OptSpeed) gives the fastest path: Fast mode, no codecs, no
// predictors. Or-combine bits with | to opt in to individual codecs.
// Pre-built bundles cover the common "max speed", "balanced", and
// "max compression" use cases without having to remember the layout.
//
// Options carry by value (uint32) so Marshal / AppendMarshal add zero
// allocations over the pool / output-clone overhead. The encoder
// checks each bit with a single AND on the hot path.
type Options uint32

const (
	// OptDense activates the inline intern table. First occurrences of
	// strings and []byte payloads are stored once and back-referenced
	// by ID (1-3 bytes per reuse). Required for any of the state-ref
	// codecs (Repeat / MTF / Pair) and for tagMapShape.
	OptDense Options = 1 << iota

	// OptQPack enables the numeric / boolean slice codecs. Bools
	// bit-pack; 32- and 64-bit integer slices ([]int/int32/int64,
	// []uint/uint32/uint64) run the codec picker — Frame-of-Reference,
	// Delta+FOR, RLE, dictionary, and Patched FOR — widening 32-bit
	// values to 64-bit for selection and narrowing back on decode;
	// float slices stay on raw-LE. Auto-selected per slice; the
	// encoder falls back to raw-LE when nothing wins. Gorilla XOR
	// for floats lives behind OptGorillaFloat (bit 5) — see
	// OptCompression for the bundle that turns it on.
	OptQPack

	// OptShapeIntern routes struct emissions through tagMapShape: each
	// distinct struct type declares its field ordering on first emit;
	// subsequent emits write only the shape ID + values. Requires
	// OptDense (the shape table lives on the intern table side).
	OptShapeIntern

	// OptPairPred enables the Markov-1 successor predictor
	// (tagStatePair, 0xEA). Per previous state-ref ID the encoder stores
	// its most-recent successor (top-1, K=1); a hit emits the transition
	// rank in a single byte (always rank 0). Requires OptDense.
	OptPairPred

	// OptMTF enables Move-to-Front rank coding (tagStateMTF, 0xE9).
	// When the LRU rank of a state-ref ID needs fewer varuint bytes
	// than the raw id, the rank is emitted instead. Requires OptDense.
	OptMTF

	// OptGorillaFloat opts in to the Gorilla XOR codec for []float64
	// (and []float32) slices. Gorilla collapses smooth time-series
	// data dramatically (~75 % wire reduction on quantised metric
	// streams) but trades that for ~10× more CPU on the encode/decode
	// path because the body is bit-level. Off by default in
	// OptBalanced for that reason; included in OptCompression for
	// archive-style workloads where wire size dominates. Requires
	// OptQPack.
	OptGorillaFloat

	// OptRANS adds a final static order-0 rANS entropy pass over the whole
	// encoded body. Opt-in (bundled into OptCompression) because it trades
	// encode/decode CPU for wire size. The encoder applies it only when the
	// rANS form is strictly smaller than the plain body, so it never makes a
	// buffer larger. Whole-buffer, so the streaming encoder ignores it.
	OptRANS

	// OptColumnIndex makes a columnar []struct payload carry a column-length
	// index so readers can decode a subset of columns without decoding the
	// rest. Opt-in, ~4 bytes per column. Only affects payloads the encoder
	// transposes into the columnar container (tagColStruct); has no effect on
	// other shapes. Sets FlagColIndex on the header.
	OptColumnIndex

	// OptFSST opts in to the FSST string codec for columnar string columns
	// (high-cardinality, substring-sharing text: log lines, URLs, paths). A
	// CPU-for-size trade (trains a per-column symbol table), so it is excluded
	// from OptBalanced and bundled into OptCompression. Never-larger: emitted
	// only when strictly smaller than the dictionary / per-value forms.
	OptFSST

	// OptMapShape interns map key-sets the way OptShapeIntern interns struct
	// field-names: the first map with a given set of (string) keys declares the
	// key ordering via tagMapShape; subsequent maps with the same key-set emit
	// only the shape ID + values in canonical (sorted) key order. Cuts encode
	// CPU and wire on maps with recurring keys (telemetry tags, log labels).
	// Requires OptDense (the shape table lives on the intern side). Opt-in in
	// v1; never-worse fallback to a plain map header when a key-set does not
	// recur.
	OptMapShape

	// OptDeltaNoBaseFingerprint disables the patch base fingerprint that Diff/Apply
	// use to detect a mismatched base. With it set, Diff omits the fingerprint and
	// Apply performs no base check — skipping a full reflect walk of the value on
	// BOTH sides (a large speedup for a big base with a tiny patch), at the cost of
	// the wrong-base safety guard. Use only when the caller guarantees Apply's base
	// is exactly the value Diff was computed against. No effect outside Diff/Apply.
	OptDeltaNoBaseFingerprint // bit 10

	// OptCanonical guarantees byte-identical output for logically-equal values:
	// map keys are emitted in sorted order (all key kinds) and floats are
	// normalized (-0.0 → +0.0, any NaN → a canonical quiet NaN). Encode-side
	// only; the bytes are ordinary qdf and decode normally. Intended for hashing
	// / signing / dedup of serialized output. Lossy for the sign of zero and NaN
	// payloads (use the default mode for bit-exact float round-trip).
	OptCanonical // bit 11

	// OptLossyVec enables the opt-in lossy float-vector codec (tag 0xFD) for
	// []float32/[]float64 columns. LOSSY: reconstruction is approximate within
	// the VectorBudget set on the Encoder (default MinCosine(0.999)). Never
	// fires unless this bit is set; never larger than the raw/lossless form.
	OptLossyVec // bit 12

	// OptZoneMap stores eligible columnar integer columns as zone-chunked
	// containers (tag 0xF1): the column is split into 256-row zones, each zone
	// independently codec-picked, prefixed by a per-zone [min,max] zonemap. A
	// bound-carrying predicate (WhereRange / WhereGE / WhereLE / WhereEq) then
	// skips zones that provably cannot match WITHOUT decoding them — a large
	// speedup for range/equality queries over positionally-ordered columns
	// (timestamps, sorted IDs). An explicit size-for-query-speed trade (chunking
	// + zonemap cost wire), so it is opt-in; without the bit the wire is
	// unchanged. v1 covers int/uint columns; string/float are a follow-up.
	OptZoneMap // bit 13

	// OptStringAlphabet packs a struct string field against the alphabet that
	// field actually uses (tagStrAlpha, 0xF5), at ceil(log2 distinct) bits per
	// character instead of eight. A field of hex ids halves; one of decimal
	// digits halves; a field whose character set is restricted but not standard
	// declares its table once and references it thereafter.
	//
	// Opt-in, and the measurements say why. The gain is real but narrow: on the
	// bench corpora it is -11.6% wire for OpenTelemetry spans and -7.1% for
	// OpenRTB bid requests, and 0.0% for access logs, event streams and IoT
	// samples, whose string fields have no restricted alphabet to exploit.
	// Under OptCompression it nearly vanishes — -1.8% and -0.7% — because the
	// entropy coder already removes the same redundancy. Against that, encoding
	// a corpus where it fires hard costs about 2.4% CPU; nothing else moves.
	//
	// So it is worth having and not worth defaulting to: a payload of trace ids
	// or numeric keys serialized without entropy coding gains a tenth of its
	// wire, and everyone else would pay for a codec that cannot help them. Set
	// the bit when the wire matters more than the CPU and the fields are shaped
	// for it. Requires OptDense.
	OptStringAlphabet // bit 14

	// Bits 15..31 reserved for future codecs (LZ77, n-gram dictionary, etc.).

	// OptSpeed is the zero-bit preset: Fast mode, no codecs, no
	// predictors. Maximum throughput, smallest CPU footprint.
	OptSpeed Options = 0

	// OptBalanced bundles every codec that does not trade CPU for
	// compression beyond its sweet spot. The right default for
	// telemetry, log batches, and any payload with repetitive
	// strings or numeric slices. Notably excludes OptGorillaFloat —
	// reach for OptCompression when the float slices in the payload
	// are smooth time-series and wire size matters more than encode
	// latency.
	OptBalanced Options = OptDense | OptQPack | OptShapeIntern | OptPairPred | OptMTF

	// OptCompression bundles every codec that the encoder will spend
	// CPU on for wire-size gains. On top of OptBalanced it adds
	// OptGorillaFloat (Gorilla XOR for float slices), OptRANS (a final
	// order-0 rANS entropy pass over the whole body), and OptFSST (the
	// FSST string codec for columnar string columns). Each trades
	// encode/decode CPU for wire size, which is why they stay out of the
	// OptBalanced default; future heavy codecs land in this bundle without
	// breaking the name.
	OptCompression Options = OptBalanced | OptGorillaFloat | OptRANS | OptFSST
)

// Has reports whether the named bit is set. Compiles to a single AND
// + compare. Use on the encoder hot path to gate codec emission.
//
//go:nosplit
func (o Options) Has(bit Options) bool { return o&bit != 0 }

var (
	// encPool serves every Marshal / AppendMarshal call. Encoders are
	// reconfigured via applyOpts on each acquire so a single pool
	// covers any Options combination; the buffer + intern table are
	// reused across calls when the cap fits maxPooledBuf.
	encPool = sync.Pool{
		New: func() any {
			// state is left nil: Fast/OptSpeed encodes never touch the intern
			// table, so a pooled encoder serving only OptSpeed pays no state
			// allocation and no per-call state.reset(). applyOpts lazily
			// allocates it the first time the encoder runs in Dense mode.
			return &Encoder{
				buf:             make([]byte, 0, initialEncBuf),
				minIntern:       4,
				maxStateEntries: 1 << 14,
				maxDepth:        DefaultMaxDepth,
			}
		},
	}
	decPool = sync.Pool{
		New: func() any { return &Decoder{} },
	}
)

// noteDetached records, on a pooled encoder whose output buffer was just handed
// to the caller, how many bytes that message actually produced. resetForReuse
// allocates the next buffer at that size instead of doubling up from
// initialEncBuf, so a steady workload pays one allocation per message.
//
// The delivered length, not the capacity the buffer reached. cap carries
// whatever slack the doubling chain left behind, and pre-allocating that slack
// again every message costs more than it saves on payloads whose columns are
// written in a few large appends: a 17.9 MB dense batch measured 21.9 MB/op
// from a cap-sized hint against 18.3 MB/op with no hint at all. Sizing to the
// delivered bytes tracks the workload in both directions with no decay rule and
// no margin to tune.
//
// The codec picker writes candidate encodings and rewinds the losers, so a
// message whose peak sits above its output still grows once past this hint —
// that is the pre-hint behaviour, never worse, and it is still far below the
// doubling chain it replaces.
//
// Not clamped to maxPooledBuf: that ceiling bounds what the pool RETAINS
// between calls, while this sizes an allocation handed straight to the caller.
// Clamping was worse than no hint at all above the ceiling — the encode
// allocated the clamp, outgrew it by construction, then doubled on top.
//
//go:nosplit
func (e *Encoder) noteDetached(out []byte) {
	e.bufHint = cap(out)
}

func putEnc(enc *Encoder, pool *sync.Pool) {
	if cap(enc.buf) > maxPooledBuf {
		enc.buf = nil
	}
	// Don't pin a spike-sized widening scratch in the pool: a single huge
	// []int32/[]uint32 field would otherwise keep its widened []int64/[]uint64
	// resident on every pooled encoder forever (mirrors the buffer / delta cap).
	if cap(enc.wideI64) > maxRetainedWideScratch {
		enc.wideI64 = nil
	}
	if cap(enc.wideU64) > maxRetainedWideScratch {
		enc.wideU64 = nil
	}
	if cap(enc.blkPlanI64) > maxRetainedWideScratch {
		enc.blkPlanI64 = nil
	}
	if cap(enc.blkPlanU64) > maxRetainedWideScratch {
		enc.blkPlanU64 = nil
	}
	if cap(enc.zoneMM) > maxRetainedWideScratch {
		enc.zoneMM = nil
	}
	pool.Put(enc)
}

// Marshal encodes v with the given option bit-mask. Combine bits with
// | to opt into individual codecs, or use one of the OptSpeed /
// OptBalanced / OptCompression bundles.
//
//	b, _ := qdf.Marshal(event,    qdf.OptSpeed)         // hot path
//	b, _ := qdf.Marshal(batch,    qdf.OptBalanced)      // telemetry
//	b, _ := qdf.Marshal(snapshot, qdf.OptCompression)   // backup
//	b, _ := qdf.Marshal(payload,                        // tuned
//	    qdf.OptDense|qdf.OptQPack|qdf.OptShapeIntern)
//
// Big-output detach (in marshalDict): cloning a multi-megabyte buffer to hand
// the caller their own copy used to dominate Large-payload profiles
// (slices.Clone + runtime.memmove). Above marshalDetachThreshold the pool would
// drop the buffer in putEnc anyway (cap exceeds maxPooledBuf), so handing the
// original to the caller and leaving the pool encoder with a nil buf is
// strictly cheaper. Small payloads stay on the clone path so the warm 4 KiB
// pool buffer survives.
func Marshal(v any, opts Options) ([]byte, error) {
	return marshalDict(v, opts, nil)
}

// marshalDetachThreshold is the buffer capacity above which Marshal
// hands the encoder buffer directly to the caller instead of cloning
// it. Chosen at 256 KiB — well above typical message sizes (so the
// hot pool path keeps reusing its buffer) but small enough that any
// payload heading toward the maxPooledBuf cap skips the copy.
const marshalDetachThreshold = 256 * 1024

// AppendMarshal encodes v and appends the result to dst. Reuse the
// returned slice as dst on the next call to avoid per-message
// allocations.
func AppendMarshal(dst []byte, v any, opts Options) ([]byte, error) {
	return appendMarshalDict(dst, v, opts, nil)
}

// Unmarshal decodes data into out, which must be a non-nil pointer.
// The wire dialect is detected from the header — the same Unmarshal
// reads any output produced by Marshal regardless of the encode-side
// option bits.
//
// Slice-backing reuse: when out is a *[]T whose slice already has enough
// capacity for the decoded element count, the decoder reuses that backing
// array instead of allocating a new one — eliminating the result allocation
// (the dominant decode cost) on a decode into a pooled / pre-sized slice. The
// reused backing is overwritten in place (its elements are zeroed first, so no
// stale data leaks), so do not share it with other live slices across the call.
// A nil or too-small destination allocates fresh, so default usage is
// unaffected.
//
// The optional QueryOptions (Where / Select) turn the call into a
// filtering/projecting columnar decode: predicates are AND-ed and the
// named columns projected. They apply only to a columnar []struct,
// []map[string]any or []any payload; on any other shape a query call
// returns a *QueryError wrapping ErrUnsupported. With no options the
// behavior is exactly the plain decode above.
//
// Round-trip fidelity: decoding into the same concrete Go type reproduces the
// value exactly, including the nil-vs-empty distinction for slices, maps and
// pointers (a nil []T decodes as nil, an empty []T{} as empty — like
// encoding/json's null vs []). Two deliberate normalizations are NOT bit-exact:
// decoding into an interface (any / map[string]any) canonicalizes integers to
// int64 / uint64 and structs to map[string]any (the schemaless type contract),
// and a time.Time loses any monotonic-clock reading on encode (as the standard
// library strips it for transport). Values nested deeper than the configured
// max depth, and maps with both int and uint keys of the same small value, are
// the documented exceptions.
func Unmarshal(data []byte, out any, opts ...QueryOption) error {
	if len(opts) == 0 {
		return unmarshal(data, out, nil, false, nil)
	}
	qp, err := buildQueryPlan(opts)
	if err != nil {
		return err
	}
	// A noCopy-only plan (no predicate, no projection) is a plain decode — do
	// NOT route it through the columnar query path, which assumes a columnar
	// container and would reject a row payload.
	if qp.root == nil && qp.selectFields == nil {
		return unmarshal(data, out, nil, qp.noCopy, qp.arena)
	}
	return unmarshalQuery(data, out, qp)
}

// unmarshalQuery is the pooled-decoder dispatch for a filtering/projecting
// decode. The plan's selectFields double as the column projection for the map
// (any) path, reusing the v1 selectFields mechanism.
func unmarshalQuery(data []byte, out any, qp *queryPlan) error {
	dec := decPool.Get().(*Decoder)
	dec.buf = data
	dec.i = 0
	dec.depth = 0 // a prior depth-overflow decode leaks depth>0 (descend errors before its defer ascend); reset so a pooled decoder never carries a stale depth into the next decode
	dec.headerRead = false
	dec.mode = Fast
	// colIndex is set fresh by readHeader; reset defensively so a pooled decoder never carries a stale flag.
	dec.colIndex = false
	// colMaxLen is normally cleared by the reflect columnar defer / generated
	// ClearColMaxLen, but a generated DecodeQDF that errors between
	// ReadColStructHeader and ClearColMaxLen returns the decoder with a stale
	// bound; reset so it can't spuriously clamp the next pooled decode's lengths.
	dec.colMaxLen = 0
	dec.selectFields = qp.selectFields
	dec.query = qp
	dec.noCopy = qp.noCopy
	dec.arena = qp.arena
	clear(dec.mapFreeList) // drop maps recycled by a prior decode into a different target
	if dec.state != nil {
		dec.state.reset()
	}
	err := decodeReflect(dec, out)
	dec.buf = nil
	dec.selectFields = nil
	dec.query = nil
	dec.noCopy = false
	dec.arena = nil // never pin the caller's arena across pooled reuse
	if cap(dec.deltaScratch) > maxRetainedDeltaScratch {
		dec.deltaScratch = nil
	}
	decPool.Put(dec)
	return err
}

// unmarshal is the shared pooled-decoder dispatch behind Unmarshal and
// UnmarshalColumns. When fields is non-nil it restricts the columnar map
// (any) decode to those columns (see Decoder.selectFields).
// unmarshalKeys mirrors unmarshal but arms the root-map key projection
// (UnmarshalKeys) rather than the columnar field filter. The two are separate
// decoder fields on purpose: one filter must never be read as the other.
func unmarshalKeys(data []byte, out any, keys []string) error {
	dec := decPool.Get().(*Decoder)
	dec.buf = data
	dec.i = 0
	dec.depth = 0
	dec.headerRead = false
	dec.mode = Fast
	dec.colIndex = false
	dec.colMaxLen = 0
	dec.noCopy = false
	dec.arena = nil
	dec.selectFields = nil
	dec.selectKeys = keys
	dec.query = nil
	clear(dec.mapFreeList)
	if dec.state != nil {
		dec.state.reset()
	}
	err := decodeReflect(dec, out)
	dec.buf = nil
	dec.selectKeys = nil
	dec.i = 0
	decPool.Put(dec)
	return err
}

func unmarshal(data []byte, out any, fields []string, noCopy bool, arena *Arena) error {
	dec := decPool.Get().(*Decoder)
	dec.buf = data
	dec.i = 0
	dec.depth = 0 // a prior depth-overflow decode leaks depth>0 (descend errors before its defer ascend); reset so a pooled decoder never carries a stale depth into the next decode
	dec.headerRead = false
	dec.mode = Fast
	dec.colIndex = false
	dec.colMaxLen = 0 // see unmarshalQuery: a generated DecodeQDF can leak a stale bound on error
	dec.noCopy = noCopy
	dec.arena = arena
	dec.selectFields = fields
	dec.selectKeys = nil
	dec.query = nil        // parity with UnmarshalT / SetInput: never inherit a prior query decode's plan from the shared pool
	clear(dec.mapFreeList) // drop maps recycled by a prior decode into a different target
	if dec.state != nil {
		dec.state.reset()
	}
	err := decodeReflect(dec, out)
	dec.buf = nil
	dec.arena = nil // never pin the caller's arena across pooled reuse
	dec.selectFields = nil
	// Reset noCopy so a WithNoCopy() decode never leaves the pooled decoder in
	// aliasing mode for the next acquirer (e.g. UnmarshalT) — that would return
	// buffer-aliased strings the caller never opted into (a silent
	// use-after-free, undetectable by the race detector).
	dec.noCopy = false
	if cap(dec.deltaScratch) > maxRetainedDeltaScratch {
		dec.deltaScratch = nil
	}
	decPool.Put(dec)
	return err
}
