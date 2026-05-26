package qdf

import "slices"

// Direct entry points for types whose MarshalQDF / UnmarshalQDF are
// emitted by cmd/qdfgen (or hand-written). They bypass the reflect
// descriptor cache entirely: no descOf lookup, no interface conversion,
// no reflect.Value materialisation. The type parameter is constrained
// to the Marshaler / Unmarshaler interface so the compiler can inline
// the method call directly.
//
// Output is byte-identical to Marshal / Unmarshal on the same value
// with OptSpeed. The path emits Fast-mode wire only because that is
// what the generated code produces. Dense / QPack are not available
// through these helpers — use Marshal(v, OptBalanced) or
// Marshal(v, OptQPack) (or the MarshalT generic counterpart) for
// those dialects.

// MarshalDirect serialises a value through its MarshalQDF method.
// Skips the public Marshal entry point's any-boxing, reflect.New, and
// descriptor lookup. Roughly 2-4× faster than Marshal for generated
// types on small payloads, with one fewer allocation per call.
func MarshalDirect[T Marshaler](v T) ([]byte, error) {
	enc := encPool.Get().(*Encoder)
	enc.Reset()
	enc.writeHeader()
	out, err := v.MarshalQDF(enc.buf)
	if err != nil {
		putEnc(enc, &encPool)
		return nil, err
	}
	cloned := slices.Clone(out)
	enc.buf = out[:0]
	encPool.Put(enc)
	return cloned, nil
}

// AppendMarshalDirect appends the serialisation of v to dst.
func AppendMarshalDirect[T Marshaler](dst []byte, v T) ([]byte, error) {
	// Header must be present before the receiver's MarshalQDF runs, so
	// emit it through a borrowed encoder (handles the headerOut flag and
	// any future header-byte changes).
	enc := encPool.Get().(*Encoder)
	enc.Reset()
	enc.buf = dst
	enc.writeHeader()
	out, err := v.MarshalQDF(enc.buf)
	if err != nil {
		enc.buf = nil
		encPool.Put(enc)
		return dst, err
	}
	enc.buf = nil
	encPool.Put(enc)
	return out, nil
}

// UnmarshalDirect dispatches to out.UnmarshalQDF after validating the
// 5-byte header. It is the inverse of MarshalDirect; the same wire
// constraints apply (Fast-mode only).
//
// If the input carries the FlagDense bit, UnmarshalDirect falls back
// to the full Unmarshal path because generated code cannot resolve
// state-ref tags without the decoder's intern table.
//
// Decode-side performance is bounded by the user's UnmarshalQDF
// implementation: the reflect path is heavily pooled (Decoder pool +
// per-decoder key intern cache) and tends to win against naive
// hand-rolled receivers. Generated code from cmd/qdfgen uses
// Decoder.InternKey for map / struct keys and matches or beats the
// reflect path; ad-hoc UnmarshalQDF methods that call NewDecoderOnBuf
// + ReadString in a tight loop will not.
func UnmarshalDirect[T Unmarshaler](data []byte, out T) error {
	if len(data) < 5 {
		return ErrShortBuffer
	}
	if data[0] != Magic0 || data[1] != Magic1 || data[2] != Magic2 {
		return ErrBadMagic
	}
	if data[3] != Version1 {
		return ErrBadVersion
	}
	if data[4]&FlagDense != 0 {
		// Dense buffers carry an intern table that generated code does
		// not maintain; fall back to the reflect path which does.
		return Unmarshal(data, out)
	}
	_, err := out.UnmarshalQDF(data[5:])
	return err
}
