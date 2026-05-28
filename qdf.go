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
//	OptCompression — OptBalanced + Gorilla XOR for float slices +
//	                 column-repeat codec for repeating struct fields
//	                 (~70 % wire reduction on smooth time-series at
//	                 ~10× CPU per slice; future heavy codecs land here)
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
//	                WriteTimestampNano / WriteStringInline
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
//	                ReadMapHeader / ReadTimestampNano / Skip /
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
	"slices"
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
	// bit-pack; integer slices try Frame-of-Reference and Delta+FOR;
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
	// (tagStatePair, 0xEA). Per previous state-ref ID the encoder keeps
	// a ring of the last 4 successors; a hit emits the ring rank in a
	// single byte. Requires OptDense.
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
	// Bits 6..31 reserved for future codecs (rANS, LZ77, n-gram
	// dictionary, etc.).

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
	OptCompression Options = OptBalanced | OptGorillaFloat
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
			return &Encoder{
				buf:             make([]byte, 0, initialEncBuf),
				state:           newEncState(),
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
func Marshal(v any, opts Options) ([]byte, error) {
	enc := encPool.Get().(*Encoder)
	enc.Reset()
	enc.applyOpts(opts)
	if err := encodeReflect(enc, v); err != nil {
		putEnc(enc, &encPool)
		return nil, err
	}
	// Big-output detach: cloning a multi-megabyte buffer to hand
	// the caller their own copy used to dominate Large-payload
	// profiles (slices.Clone + runtime.memmove). At this size the
	// pool would drop the buffer in putEnc anyway (cap exceeds
	// maxPooledBuf), so handing the original to the caller and
	// leaving the pool encoder with a nil buf is strictly cheaper —
	// it skips the copy entirely. Small payloads stay on the
	// clone path so the warm 4 KiB pool buffer survives.
	var out []byte
	if cap(enc.buf) > marshalDetachThreshold {
		out = enc.buf
		enc.buf = nil
	} else {
		out = slices.Clone(enc.buf)
	}
	encPool.Put(enc)
	return out, nil
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
	enc := encPool.Get().(*Encoder)
	enc.Reset()
	enc.applyOpts(opts)
	enc.buf = dst
	if err := encodeReflect(enc, v); err != nil {
		putEnc(enc, &encPool)
		return dst, err
	}
	out := enc.buf
	enc.buf = nil
	encPool.Put(enc)
	return out, nil
}

// Unmarshal decodes data into out, which must be a non-nil pointer.
// The wire dialect is detected from the header — the same Unmarshal
// reads any output produced by Marshal regardless of the encode-side
// option bits.
func Unmarshal(data []byte, out any) error {
	dec := decPool.Get().(*Decoder)
	dec.buf = data
	dec.i = 0
	dec.headerRead = false
	dec.mode = Fast
	if dec.state != nil {
		dec.state.reset()
	}
	err := decodeReflect(dec, out)
	dec.buf = nil
	decPool.Put(dec)
	return err
}
