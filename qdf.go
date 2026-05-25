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

var (
	fastEncPool = sync.Pool{
		New: func() any { return &Encoder{mode: Fast, buf: make([]byte, 0, initialEncBuf)} },
	}
	fastQPackEncPool = sync.Pool{
		New: func() any {
			return &Encoder{mode: Fast, buf: make([]byte, 0, initialEncBuf), qpack: true}
		},
	}
	denseEncPool = sync.Pool{
		New: func() any {
			return &Encoder{
				mode:            Dense,
				buf:             make([]byte, 0, initialEncBuf),
				state:           newEncState(),
				minIntern:       4,
				maxStateEntries: 1 << 14,
				qpack:           true,
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
