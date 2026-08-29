package qdf

import (
	"reflect"
	"unsafe"
)

// Generic entry points. The non-generic Marshal / Unmarshal take any,
// which forces a runtime interface conversion of the caller's value
// (boxing) and, on the encode side, a reflect.New+Set copy for value-
// typed inputs. The generic forms below skip both: T is known at
// instantiation, so reflect.TypeFor[T]() resolves at compile time,
// and unsafe.Pointer(&v) points straight at the function parameter on
// the caller's stack.
//
// Wire output is byte-identical to the non-generic counterparts; the
// generic versions only change the calling convention.

// MarshalT is the generic equivalent of Marshal. T is fixed at the
// call site; opts copies by value, so the call adds zero heap
// allocations over MarshalT itself.
func MarshalT[T any](v T, opts Options) ([]byte, error) {
	enc := acquireEnc(opts, nil)
	if err := encodeT(enc, &v); err != nil {
		abortMarshal(enc)
		return nil, err
	}
	return finishMarshal(enc), nil
}

// AppendMarshalT is the generic equivalent of AppendMarshal.
func AppendMarshalT[T any](dst []byte, v T, opts Options) ([]byte, error) {
	enc := acquireEnc(opts, nil)
	start := len(dst)
	enc.buf = dst
	if err := encodeT(enc, &v); err != nil {
		abortAppend(enc)
		return dst, err
	}
	return finishAppend(enc, start), nil
}

// UnmarshalT is the generic equivalent of Unmarshal. The destination
// pointer must not be nil.
func UnmarshalT[T any](data []byte, out *T) error {
	if out == nil {
		return ErrTypeMismatch
	}
	dec := decPool.Get().(*Decoder)
	dec.buf = data
	dec.i = 0
	dec.depth = 0 // reset stale depth from a prior depth-overflow decode (see unmarshal)
	dec.headerRead = false
	dec.mode = Fast
	// Start from a clean pooled decoder: it is shared with Unmarshal /
	// UnmarshalColumns / query paths, any of which may have left noCopy,
	// selectFields, query, or colIndex set. Inheriting noCopy would silently
	// alias the input buffer; inheriting selectFields/query would mis-project.
	dec.noCopy = false
	dec.colIndex = false
	// A generated columnar DecodeQDF that errors between ReadColStructHeader and
	// ClearColMaxLen returns the pooled decoder with a stale colMaxLen; reset it
	// so this decode's slice codecs are not spuriously clamped (mirrors the reset
	// in unmarshal / unmarshalQuery — UnmarshalT shares the same decoder pool).
	dec.colMaxLen = 0
	dec.selectFields = nil
	dec.selectKeys = nil
	dec.query = nil
	dec.dropRecycledMaps() // drop maps recycled by a prior decode into a different target
	if dec.state != nil {
		dec.state.reset()
	}
	t := reflect.TypeFor[T]()
	td, err := descOf(t)
	if err != nil {
		dec.buf = nil
		decPool.Put(dec)
		return err
	}
	err = td.decode(dec, unsafe.Pointer(out))
	dec.buf = nil
	// Don't pin a spike-sized scratch buffer in the pool (mirrors unmarshal).
	if cap(dec.deltaScratch) > maxRetainedDeltaScratch {
		dec.deltaScratch = nil
	}
	decPool.Put(dec)
	return err
}

// encodeT is the shared body of every generic encode entry point. It
// resolves the type-T descriptor at compile time (via reflect.TypeFor)
// and points the encoder straight at vp, which the caller obtained as
// &v on its own stack. No interface boxing, no reflect.New copy.
//
// vp must point at a value of type T. For T = pointer-kind (e.g. *Foo)
// the descriptor's encodePtr will dereference once, matching the
// behavior of the non-generic Marshal(v any) on a *Foo argument.
func encodeT[T any](e *Encoder, vp *T) error {
	if vp == nil {
		e.WriteNil()
		return nil
	}
	t := reflect.TypeFor[T]()
	td, err := descOf(t)
	if err != nil {
		return err
	}
	return td.encode(e, unsafe.Pointer(vp))
}
