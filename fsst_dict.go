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
func marshalDict(v any, opts Options, dict *fsst.SymbolTable) ([]byte, error) {
	enc := encPool.Get().(*Encoder)
	enc.Reset()
	enc.applyOpts(opts)
	enc.fsstDict = dict
	if err := encodeReflect(enc, v); err != nil {
		putEnc(enc, &encPool)
		return nil, err
	}
	enc.maybeApplyRANS(0)
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

// appendMarshalDict is AppendMarshal with an optional pre-trained FSST table.
func appendMarshalDict(dst []byte, v any, opts Options, dict *fsst.SymbolTable) ([]byte, error) {
	enc := encPool.Get().(*Encoder)
	enc.Reset()
	enc.applyOpts(opts)
	enc.fsstDict = dict
	start := len(dst)
	enc.buf = dst
	if err := encodeReflect(enc, v); err != nil {
		putEnc(enc, &encPool)
		return dst, err
	}
	enc.maybeApplyRANS(start)
	out := enc.buf
	enc.buf = nil
	encPool.Put(enc)
	return out, nil
}
