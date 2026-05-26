package qdf

import (
	"reflect"
	"slices"
	"unsafe"
)

// Generic entry points. Marshal / Unmarshal take any, which forces a
// runtime interface conversion of the caller's value (boxing) and, on
// the encode side, a reflect.New+Set copy for value-typed inputs. The
// generic forms below skip both: T is known at instantiation, so
// reflect.TypeFor[T]() resolves at compile time, and unsafe.Pointer(&v)
// points straight at the function parameter on the caller's stack.
//
// Wire output is byte-identical to the non-generic counterparts; the
// generic versions only change the calling convention.

// MarshalT is the generic equivalent of Marshal.
func MarshalT[T any](v T) ([]byte, error) {
	enc := fastEncPool.Get().(*Encoder)
	enc.Reset()
	if err := encodeT(enc, &v); err != nil {
		putEnc(enc, &fastEncPool)
		return nil, err
	}
	out := slices.Clone(enc.buf)
	fastEncPool.Put(enc)
	return out, nil
}

// MarshalQPackT is the generic equivalent of MarshalQPack.
func MarshalQPackT[T any](v T) ([]byte, error) {
	enc := fastQPackEncPool.Get().(*Encoder)
	enc.Reset()
	if err := encodeT(enc, &v); err != nil {
		putEnc(enc, &fastQPackEncPool)
		return nil, err
	}
	out := slices.Clone(enc.buf)
	fastQPackEncPool.Put(enc)
	return out, nil
}

// MarshalDenseT is the generic equivalent of MarshalDense.
func MarshalDenseT[T any](v T) ([]byte, error) {
	enc := denseEncPool.Get().(*Encoder)
	enc.Reset()
	if err := encodeT(enc, &v); err != nil {
		putEnc(enc, &denseEncPool)
		return nil, err
	}
	out := slices.Clone(enc.buf)
	denseEncPool.Put(enc)
	return out, nil
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
	dec.headerRead = false
	dec.mode = Fast
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
	decPool.Put(dec)
	return err
}

// AppendMarshalT is the generic equivalent of AppendMarshal.
func AppendMarshalT[T any](dst []byte, v T) ([]byte, error) {
	enc := fastEncPool.Get().(*Encoder)
	enc.Reset()
	enc.buf = dst
	if err := encodeT(enc, &v); err != nil {
		putEnc(enc, &fastEncPool)
		return dst, err
	}
	out := enc.buf
	enc.buf = nil
	fastEncPool.Put(enc)
	return out, nil
}

// MarshalTWith is the generic option-bitmask entry point. It carries
// the same zero-allocation guarantees as MarshalT — T is fixed at
// the call site, opts copies by value — while gaining per-call codec
// selection via the Options bit-mask.
func MarshalTWith[T any](v T, opts Options) ([]byte, error) {
	enc := customEncPool.Get().(*Encoder)
	enc.Reset()
	enc.applyOpts(opts)
	if err := encodeT(enc, &v); err != nil {
		putEnc(enc, &customEncPool)
		return nil, err
	}
	out := slices.Clone(enc.buf)
	customEncPool.Put(enc)
	return out, nil
}

// AppendMarshalTWith is the generic, option-aware AppendMarshal.
func AppendMarshalTWith[T any](dst []byte, v T, opts Options) ([]byte, error) {
	enc := customEncPool.Get().(*Encoder)
	enc.Reset()
	enc.applyOpts(opts)
	enc.buf = dst
	if err := encodeT(enc, &v); err != nil {
		putEnc(enc, &customEncPool)
		return dst, err
	}
	out := enc.buf
	enc.buf = nil
	customEncPool.Put(enc)
	return out, nil
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
