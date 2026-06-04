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
//	OptCompression — OptBalanced + heavier float codecs: Gorilla XOR for
//	                 smooth time-series and ALP for quantized/decimal
//	                 float64 (large wire reduction at higher encode CPU)
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
//	func UnmarshalT[T any](data []byte) (T, error)
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
	// maxRetainedDeltaScratch bounds the Delta+FOR decode scratch a pooled
	// decoder keeps between calls. Retains it for columns up to ~16 k rows
	// (the maxStateEntries scale) and drops it after a one-off larger decode so
	// the pool never pins an outlier buffer.
	maxRetainedDeltaScratch = 1 << 14
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

	// Bits 11..31 reserved for future codecs (LZ77, n-gram dictionary, etc.).

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
	// OptGorillaFloat (Gorilla XOR for float slices), which trades
	// encode/decode CPU for wire size, so it stays out of the
	// OptBalanced default; future heavy codecs (rANS, dictionary
	// preloading) will land in this bundle without breaking the name.
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

//go:nosplit
func putEnc(enc *Encoder, pool *sync.Pool) {
	if cap(enc.buf) > maxPooledBuf {
		enc.buf = nil
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
// The optional QueryOptions (Where / Select) turn the call into a
// filtering/projecting columnar decode: predicates are AND-ed and the
// named columns projected. They apply only to a columnar []struct,
// []map[string]any or []any payload; on any other shape a query call
// returns a *QueryError wrapping ErrUnsupported. With no options the
// behavior is exactly the plain decode above.
func Unmarshal(data []byte, out any, opts ...QueryOption) error {
	if len(opts) == 0 {
		return unmarshal(data, out, nil, false)
	}
	qp, err := buildQueryPlan(opts)
	if err != nil {
		return err
	}
	// A noCopy-only plan (no predicate, no projection) is a plain decode — do
	// NOT route it through the columnar query path, which assumes a columnar
	// container and would reject a row payload.
	if qp.root == nil && qp.selectFields == nil {
		return unmarshal(data, out, nil, qp.noCopy)
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
	dec.headerRead = false
	dec.mode = Fast
	// colIndex is set fresh by readHeader; reset defensively so a pooled decoder never carries a stale flag.
	dec.colIndex = false
	dec.selectFields = qp.selectFields
	dec.query = qp
	dec.noCopy = qp.noCopy
	if dec.state != nil {
		dec.state.reset()
	}
	err := decodeReflect(dec, out)
	dec.buf = nil
	dec.selectFields = nil
	dec.query = nil
	dec.noCopy = false
	if cap(dec.deltaScratch) > maxRetainedDeltaScratch {
		dec.deltaScratch = nil
	}
	decPool.Put(dec)
	return err
}

// unmarshal is the shared pooled-decoder dispatch behind Unmarshal and
// UnmarshalColumns. When fields is non-nil it restricts the columnar map
// (any) decode to those columns (see Decoder.selectFields).
func unmarshal(data []byte, out any, fields []string, noCopy bool) error {
	dec := decPool.Get().(*Decoder)
	dec.buf = data
	dec.i = 0
	dec.headerRead = false
	dec.mode = Fast
	dec.colIndex = false
	dec.noCopy = noCopy
	dec.selectFields = fields
	if dec.state != nil {
		dec.state.reset()
	}
	err := decodeReflect(dec, out)
	dec.buf = nil
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
