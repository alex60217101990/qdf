package qdf

// Marshaler is implemented by types that know how to serialize themselves
// into the QDF wire format. Implementations should append exactly one
// value to dst and return the extended slice.
type Marshaler interface {
	MarshalQDF(dst []byte) ([]byte, error)
}

// EncoderMarshaler is an optional extension of Marshaler that writes directly
// into a shared Encoder. The buffer-based MarshalQDF forces a nested value to
// build its own Encoder on the parent's bytes (one *Encoder heap allocation per
// nested value); EncodeQDF lets a parent thread one Encoder through the whole
// graph. Generated code (cmd/qdfgen) implements it; EncodeNested prefers it.
type EncoderMarshaler interface {
	Marshaler
	EncodeQDF(e *Encoder) error
}

// EncodeNested encodes a nested Marshaler m into the shared encoder e. When m
// implements EncoderMarshaler it writes directly (no new Encoder); otherwise it
// falls back to the buffer-based MarshalQDF + AdoptBuffer. Exported for
// cmd/qdfgen-generated code.
func EncodeNested(e *Encoder, m Marshaler) error {
	if em, ok := m.(EncoderMarshaler); ok {
		return em.EncodeQDF(e)
	}
	b, err := m.MarshalQDF(e.Bytes())
	if err != nil {
		return err
	}
	e.AdoptBuffer(b)
	e.MarkHeaderWritten()
	return nil
}

// DecoderUnmarshaler is the decode counterpart of EncoderMarshaler: it reads its
// value from a SHARED Decoder, advancing it, so a parent can thread one decoder
// through a whole value graph instead of opening one per nested value (one
// *Decoder plus its scratch / intern state per nested value otherwise).
// Generated code (cmd/qdfgen) implements it; DecodeNested prefers it. noCopy and
// arena live on the shared decoder, so a threaded nested decode inherits them
// with no extra arguments.
type DecoderUnmarshaler interface {
	Unmarshaler
	DecodeQDF(d *Decoder) error
}

// DecodeNested decodes one nested value from the shared decoder d. When u
// implements DecoderUnmarshaler it reads directly from d (no new decoder, and it
// inherits d's noCopy / arena). Otherwise it falls back to the buffer-based
// UnmarshalQDF over d's remaining bytes — honoring d's noCopy / arena via the
// Opts / Arena extensions — and advances d by the bytes consumed. Exported for
// cmd/qdfgen-generated code.
func DecodeNested(d *Decoder, u Unmarshaler) error {
	if du, ok := u.(DecoderUnmarshaler); ok {
		return du.DecodeQDF(d)
	}
	src := d.RemainingBytes()
	var n int
	var err error
	switch {
	case d.arena != nil:
		if ua, ok := u.(UnmarshalerArena); ok {
			n, err = ua.UnmarshalQDFArena(src, d.noCopy, d.arena)
			break
		}
		if uo, ok := u.(UnmarshalerOpts); ok && d.noCopy {
			n, err = uo.UnmarshalQDFOpts(src, true)
		} else {
			n, err = u.UnmarshalQDF(src)
		}
	default:
		if uo, ok := u.(UnmarshalerOpts); ok && d.noCopy {
			n, err = uo.UnmarshalQDFOpts(src, true)
		} else {
			n, err = u.UnmarshalQDF(src)
		}
	}
	if err != nil {
		return err
	}
	// Guard the shared cursor against a misbehaving Unmarshaler (mirrors
	// UnmarshalNested): a count past the remaining bytes would push the cursor
	// out of bounds and panic the next read.
	if n < 0 || n > d.Remaining() {
		return ErrShortBuffer
	}
	d.Advance(n)
	return nil
}

// Unmarshaler is implemented by types that know how to deserialize themselves
// from a QDF wire-format slice. Implementations should consume exactly one
// value from src and return the number of bytes consumed.
//
// A custom UnmarshalQDF reads the Fast wire format, and Marshal honors that
// for a HAND-WRITTEN Marshaler: such a type emits its own Fast body and is
// framed as Fast regardless of the Options passed, so it round-trips under any
// Options and its wire may be read by stripping the five-byte header and
// calling UnmarshalQDF directly.
//
// An EncoderMarshaler — generated code — is different, and the difference
// matters to anyone reading its bytes by hand. It writes its body into the
// caller's encoder and honors that encoder's Options, so its framing follows
// them too: under OptRANS or OptCompression the body is entropy-coded and
// everything after the header is a compressed blob, not a tag stream. Read it
// through Unmarshal or UnmarshalDirect, which unframe first. Stripping five
// bytes and calling UnmarshalQDF on the remainder was never guaranteed and now
// fails outright on a framed body — usually with an error, but a compressed
// blob is arbitrary bytes and a parser can be unlucky.
//
// A type that implements Unmarshaler WITHOUT
// also implementing Marshaler is encoded structurally; under a Dense/QPack
// tier that produces a Dense body its Fast-only UnmarshalQDF cannot read.
// Implement both interfaces (or neither) to avoid this — generated code from
// cmd/qdfgen always implements both.
type Unmarshaler interface {
	UnmarshalQDF(src []byte) (n int, err error)
}

// UnmarshalerOpts is an optional extension of Unmarshaler that accepts the
// noCopy flag. When noCopy is true, the implementation should decode string and
// []byte fields as aliases of src (see WithNoCopy) instead of copying. Generated
// code from cmd/qdfgen implements this; the plain UnmarshalQDF delegates to it
// with noCopy=false. Decoders honor it only when the caller opted into noCopy.
type UnmarshalerOpts interface {
	Unmarshaler
	UnmarshalQDFOpts(src []byte, noCopy bool) (n int, err error)
}

// UnmarshalerArena is an optional extension that also accepts a decode Arena, so
// the implementation packs copied string/[]byte fields into it (see WithArena)
// instead of one allocation per field. Generated code from cmd/qdfgen implements
// this; UnmarshalQDFOpts delegates to it with a nil arena. Decoders honor it only
// when the caller passed an arena.
type UnmarshalerArena interface {
	UnmarshalerOpts
	UnmarshalQDFArena(src []byte, noCopy bool, a *Arena) (n int, err error)
}

// UnmarshalNestedArena is UnmarshalNested threading a decode arena: when a is
// non-nil and u implements UnmarshalerArena, the nested decode packs its strings
// into a. Otherwise it falls back to the plain (copying) nested decode.
// Exported for cmd/qdfgen-generated code.
func UnmarshalNestedArena(u Unmarshaler, src []byte, noCopy bool, a *Arena) (int, error) {
	if a != nil {
		if ua, ok := u.(UnmarshalerArena); ok {
			n, err := ua.UnmarshalQDFArena(src, noCopy, a)
			if err != nil {
				return n, err
			}
			if n < 0 || n > len(src) {
				return 0, ErrShortBuffer
			}
			return n, nil
		}
	}
	return UnmarshalNested(u, src, noCopy)
}

// UnmarshalNested decodes one nested Unmarshaler value from src, honoring noCopy
// when u also implements UnmarshalerOpts. External Unmarshalers without the Opts
// method fall back to a copying decode. Used by decodeUnmarshaler and by
// cmd/qdfgen-generated code; exported for the latter.
func UnmarshalNested(u Unmarshaler, src []byte, noCopy bool) (int, error) {
	var n int
	var err error
	if uo, ok := u.(UnmarshalerOpts); ok && noCopy {
		n, err = uo.UnmarshalQDFOpts(src, true)
	} else {
		n, err = u.UnmarshalQDF(src)
	}
	if err != nil {
		return n, err
	}
	// Guard the parent cursor against a misbehaving Unmarshaler. Both the
	// reflect path (decodeUnmarshaler) and the generated code advance the
	// parent decoder by the returned count; a count larger than the nested
	// buffer (or negative) would push the cursor out of bounds and panic the
	// next read. Reject it as a short/invalid buffer instead.
	if n < 0 || n > len(src) {
		return 0, ErrShortBuffer
	}
	return n, nil
}
