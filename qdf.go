// Package qdf is a compact, streaming-friendly binary serialization
// format.
//
// Two wire dialects share the same tag space:
//
//   - Fast — tagged binary, no per-message dictionary. Used by Marshal.
//   - Dense — Fast plus an inline string-interning table. Repeated values
//     and keys collapse to small backward references. Used by MarshalDense
//     and by NewStreamEncoder when constructed with the Dense mode.
//
// A single decoder handles both. The wire header carries a flag bit that
// the decoder reads on first call.
//
//	b, err := qdf.Marshal(v)
//	err := qdf.Unmarshal(b, &v)
//
//	b, err := qdf.MarshalDense(v)        // smaller on repetitive payloads
//
//	enc := qdf.NewStreamEncoder(w, qdf.Dense)
//	dec := qdf.NewStreamDecoder(r)
//
// Marshal and MarshalDense return a freshly-allocated slice owned by the
// caller. AppendMarshal is the zero-extra-copy variant.
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
	maxPooledBuf  = 1 * 1024 * 1024
)

// Options is a bit-mask of per-call encoder feature toggles. A zero
// value (OptSpeed) gives the fastest path: Fast mode, no codecs, no
// predictors. Or-combine bits with | to opt in to individual codecs.
// Pre-built bundles cover the common "max speed", "balanced", and
// "max compression" use cases without having to remember the layout.
//
// Options carry by value (uint32) so MarshalWith / AppendMarshalWith
// add zero allocations over a plain Marshal call. The encoder checks
// each bit with a single AND on the hot path.
type Options uint32

const (
	// OptDense activates the inline intern table. First occurrences of
	// strings and []byte payloads are stored once and back-referenced
	// by ID (1-3 bytes per reuse). Required for any of the state-ref
	// codecs (Repeat / MTF / Pair) and for tagMapShape.
	OptDense Options = 1 << iota

	// OptQPack enables the numeric / boolean slice codecs. Bools
	// bit-pack; integer slices try Frame-of-Reference and Delta+FOR;
	// float slices use Gorilla XOR. Auto-selected per slice; the
	// encoder falls back to raw-LE when nothing wins.
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
	// Bits 5..31 reserved for future codecs (rANS, LZ77, n-gram
	// dictionary, etc.).

	// OptSpeed is the zero-bit preset: Fast mode, no codecs, no
	// predictors. Maximum throughput, smallest CPU footprint.
	OptSpeed Options = 0

	// OptBalanced bundles every codec that does not trade CPU for
	// compression beyond its sweet spot. Wire-format match for the
	// default MarshalDense entry point.
	OptBalanced Options = OptDense | OptQPack | OptShapeIntern | OptPairPred | OptMTF

	// OptCompression is an alias for OptBalanced today. The constant
	// is reserved so future heavy-CPU codecs (rANS, dictionary
	// preloading) can opt in without breaking the bundle name.
	OptCompression Options = OptBalanced
)

// Has reports whether the named bit is set. Compiles to a single AND
// + compare. Use on the encoder hot path to gate codec emission.
//
//go:nosplit
func (o Options) Has(bit Options) bool { return o&bit != 0 }

var (
	fastEncPool = sync.Pool{
		New: func() any {
			return &Encoder{mode: Fast, opts: OptSpeed, buf: make([]byte, 0, initialEncBuf), maxDepth: DefaultMaxDepth}
		},
	}
	fastQPackEncPool = sync.Pool{
		New: func() any {
			return &Encoder{mode: Fast, opts: OptQPack, buf: make([]byte, 0, initialEncBuf), qpack: true, maxDepth: DefaultMaxDepth}
		},
	}
	denseEncPool = sync.Pool{
		New: func() any {
			return &Encoder{
				mode:            Dense,
				opts:            OptBalanced,
				buf:             make([]byte, 0, initialEncBuf),
				state:           newEncState(),
				minIntern:       4,
				maxStateEntries: 1 << 14,
				qpack:           true,
				maxDepth:        DefaultMaxDepth,
			}
		},
	}
	decPool = sync.Pool{
		New: func() any { return &Decoder{} },
	}
	// customEncPool serves MarshalWith / AppendMarshalWith. Encoders
	// are reconfigured via applyOpts on each acquire so a single pool
	// covers any Options combination; the buffer + intern table are
	// reused across calls when the cap fits maxPooledBuf.
	customEncPool = sync.Pool{
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
)

//go:nosplit
func putEnc(enc *Encoder, pool *sync.Pool) {
	if cap(enc.buf) > maxPooledBuf {
		enc.buf = nil
	}
	pool.Put(enc)
}

// Marshal encodes v in Fast mode. The returned slice is freshly allocated
// and owned by the caller.
func Marshal(v any) ([]byte, error) {
	enc := fastEncPool.Get().(*Encoder)
	enc.Reset()
	if err := encodeReflect(enc, v); err != nil {
		putEnc(enc, &fastEncPool)
		return nil, err
	}
	out := slices.Clone(enc.buf)
	fastEncPool.Put(enc)
	return out, nil
}

// MarshalDense encodes v in Dense mode. Repeated strings and bytes are
// emitted once and back-referenced by ID; numeric and bool slices are
// auto-packed via QPack codecs (bitpacked bools, raw-LE/FOR/Delta-FOR
// for integers, raw-LE for floats). Output is smaller on repetitive and
// numeric payloads at a small CPU cost.
func MarshalDense(v any) ([]byte, error) {
	enc := denseEncPool.Get().(*Encoder)
	enc.Reset()
	if err := encodeReflect(enc, v); err != nil {
		putEnc(enc, &denseEncPool)
		return nil, err
	}
	out := slices.Clone(enc.buf)
	denseEncPool.Put(enc)
	return out, nil
}

// MarshalQPack encodes v in Fast mode with QPack codecs enabled. Wire
// is identical to Marshal for scalars and strings, but []bool is
// bit-packed and numeric slices use raw-LE / FOR / Delta-FOR auto-
// selected by minimum size. The output is a strict superset of the
// wire surface accepted by Unmarshal; a Marshal-only consumer that has
// not been updated to handle the QPack tags will fail with ErrBadTag.
func MarshalQPack(v any) ([]byte, error) {
	enc := fastQPackEncPool.Get().(*Encoder)
	enc.Reset()
	if err := encodeReflect(enc, v); err != nil {
		putEnc(enc, &fastQPackEncPool)
		return nil, err
	}
	out := slices.Clone(enc.buf)
	fastQPackEncPool.Put(enc)
	return out, nil
}

// MarshalWith encodes v with the given option bit-mask. Combine bits
// with | to opt into individual codecs, or use one of the OptSpeed /
// OptBalanced / OptCompression bundles. Zero allocations beyond the
// existing pool / output-clone overhead.
//
//	b, _ := qdf.MarshalWith(snapshot, qdf.OptCompression)   // backup
//	b, _ := qdf.MarshalWith(event,    qdf.OptSpeed)         // hot path
//	b, _ := qdf.MarshalWith(payload,                        // tuned
//	    qdf.OptDense|qdf.OptQPack|qdf.OptShapeIntern)
func MarshalWith(v any, opts Options) ([]byte, error) {
	enc := customEncPool.Get().(*Encoder)
	enc.Reset()
	enc.applyOpts(opts)
	if err := encodeReflect(enc, v); err != nil {
		putEnc(enc, &customEncPool)
		return nil, err
	}
	out := slices.Clone(enc.buf)
	customEncPool.Put(enc)
	return out, nil
}

// AppendMarshalWith is the AppendMarshal equivalent of MarshalWith.
// dst is reused as the output buffer; the slice header may be
// reseated if dst's capacity is insufficient.
func AppendMarshalWith(dst []byte, v any, opts Options) ([]byte, error) {
	enc := customEncPool.Get().(*Encoder)
	enc.Reset()
	enc.applyOpts(opts)
	enc.buf = dst
	if err := encodeReflect(enc, v); err != nil {
		putEnc(enc, &customEncPool)
		return dst, err
	}
	out := enc.buf
	enc.buf = nil
	customEncPool.Put(enc)
	return out, nil
}

// AppendMarshal encodes v in Fast mode and appends the result to dst.
// Reuse the returned slice as dst on the next call to avoid per-message
// allocations.
func AppendMarshal(dst []byte, v any) ([]byte, error) {
	enc := fastEncPool.Get().(*Encoder)
	enc.Reset()
	enc.buf = dst
	if err := encodeReflect(enc, v); err != nil {
		putEnc(enc, &fastEncPool)
		return dst, err
	}
	out := enc.buf
	enc.buf = nil
	fastEncPool.Put(enc)
	return out, nil
}

// Unmarshal decodes data into out, which must be a non-nil pointer. The
// dialect (Fast or Dense) is detected from the wire header.
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
