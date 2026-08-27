package qdf

import (
	"slices"

	"github.com/alex60217101990/qdf/internal/fsst"
	"github.com/alex60217101990/qdf/internal/unsafestr"
)

// FSSTDict is a pre-trained, immutable FSST symbol table.
//
// Training the per-column symbol table is the dominant cost of FSST encoding.
// Train one FSSTDict over representative samples and reuse it across many
// FSSTDict.Marshal calls: each encode then skips training and pays only
// compression — the train-once, encode-many pattern (the "Static" in FSST).
//
// An FSSTDict is bounded (≤255 symbols, a few KB) and never mutated after
// training, so it is safe for concurrent use and holds a fixed, small amount of
// memory. It cannot grow, so reusing one across the life of a process does not
// leak. The wire stays self-describing: each FSST column still carries its
// table, so output produced with a dictionary decodes with a plain Unmarshal.
type FSSTDict struct {
	tbl *fsst.SymbolTable
}

// TrainFSSTDict learns an FSST symbol table from representative byte samples
// (typically a slice of the string column you will encode). Deterministic: the
// same samples always yield the same dictionary.
func TrainFSSTDict(samples [][]byte) *FSSTDict {
	return &FSSTDict{tbl: fsst.BuildSymbolTable(samples)}
}

// TrainFSSTDictStrings is TrainFSSTDict over []string, viewing each string's
// bytes without copying (the strings are only read during training).
func TrainFSSTDictStrings(samples []string) *FSSTDict {
	b := make([][]byte, len(samples))
	for i, s := range samples {
		b[i] = unsafestr.Bytes(s)
	}
	return &FSSTDict{tbl: fsst.BuildSymbolTable(b)}
}

// fsstRequired is the option set FSST needs to actually fire: the columnar
// prerequisites (OptDense + OptShapeIntern gate encodeColumnar; OptQPack gates
// e.fsst) plus OptFSST itself. FSSTDict.Marshal ORs this into the caller's opts
// so a bare FSSTDict.Marshal(v, 0) still compresses instead of silently being a
// no-op. OptCompression / OptBalanced already include the columnar bits.
const fsstRequired = OptDense | OptQPack | OptShapeIntern | OptFSST

// Marshal encodes v using this pre-trained dictionary for FSST string columns.
// It enables the FSST codec and its columnar prerequisites (Dense, QPack,
// ShapeIntern), so even FSSTDict.Marshal(v, OptSpeed) compresses; combine with
// OptCompression to add the float codecs and rANS.
func (d *FSSTDict) Marshal(v any, opts Options) ([]byte, error) {
	return marshalDict(v, opts|fsstRequired, d.tbl)
}

// AppendMarshal encodes v with this dictionary and appends to dst (see
// AppendMarshal). Enables FSST and its columnar prerequisites.
func (d *FSSTDict) AppendMarshal(dst []byte, v any, opts Options) ([]byte, error) {
	return appendMarshalDict(dst, v, opts|fsstRequired, d.tbl)
}

// marshalDict is Marshal with an optional pre-trained FSST table (nil = train
// per batch). Marshal delegates here with a nil dict.
// The pooled-encoder lifecycle, shared by the any-taking and the type-parameter
// entry points.
//
// Those two differ in exactly one step — how the value reaches the encoder: the
// any path unpacks an eface, the typed path hands over a pointer whose type is
// known at the call site. Everything around that step is identical, and it is the
// part where a divergence would be a memory bug rather than a cosmetic one: when
// the caller's backing array may stay aliased in a pooled encoder, and when a
// spike-sized buffer is handed over rather than copied.
//
// It is split into acquire/finish rather than wrapped around a closure so the
// hot path keeps its shape: a closure passed to a non-inlined helper can escape,
// and this is the allocation-sensitive path in the library.

// acquireEnc takes a pooled encoder and prepares it for one message.
func acquireEnc(opts Options, dict *fsst.SymbolTable) *Encoder {
	enc := encPool.Get().(*Encoder)
	enc.Reset()
	enc.applyOpts(opts)
	enc.fsstDict = dict
	return enc
}

// finishMarshal completes a from-scratch encode: it applies the entropy pass,
// then either hands the encoder's buffer to the caller or clones it, and returns
// the encoder to the pool.
//
// A buffer past marshalDetachThreshold is handed over rather than copied —
// putEnc would drop one that large anyway, so cloning it would cost a second
// large allocation for nothing.
func finishMarshal(enc *Encoder) []byte {
	enc.maybeApplyRANS(0)
	var out []byte
	if cap(enc.buf) > marshalDetachThreshold {
		out = enc.buf
		enc.noteDetached(out)
		enc.buf = nil
	} else {
		out = slices.Clone(enc.buf)
	}
	putEnc(enc, &encPool) // cap a spike-sized buffer / widening scratch before pooling
	return out
}

// abortMarshal returns the encoder after a failed from-scratch encode. The
// buffer is the encoder's own, so there is nothing to detach.
func abortMarshal(enc *Encoder) { putEnc(enc, &encPool) }

// finishAppend completes an append-to-dst encode. The buffer belongs to the
// caller, so it is always detached rather than cloned.
func finishAppend(enc *Encoder, start int) []byte {
	enc.maybeApplyRANS(start)
	out := enc.buf
	enc.buf = nil
	putEnc(enc, &encPool) // cap a spike-sized widening scratch before pooling
	return out
}

// abortAppend returns the encoder after a failed append-to-dst encode.
//
// Detaching the caller's dst first is load-bearing: putEnc only nils buf past
// maxPooledBuf, so a normal-sized dst would stay aliased in the pooled encoder
// and the next encode would overwrite the caller's backing array.
func abortAppend(enc *Encoder) {
	enc.buf = nil
	putEnc(enc, &encPool)
}

func marshalDict(v any, opts Options, dict *fsst.SymbolTable) ([]byte, error) {
	enc := acquireEnc(opts, dict)
	if err := encodeReflect(enc, v); err != nil {
		abortMarshal(enc)
		return nil, err
	}
	return finishMarshal(enc), nil
}

// appendMarshalDict is AppendMarshal with an optional pre-trained FSST table.
func appendMarshalDict(dst []byte, v any, opts Options, dict *fsst.SymbolTable) ([]byte, error) {
	enc := acquireEnc(opts, dict)
	start := len(dst)
	enc.buf = dst
	if err := encodeReflect(enc, v); err != nil {
		abortAppend(enc)
		return dst, err
	}
	return finishAppend(enc, start), nil
}
