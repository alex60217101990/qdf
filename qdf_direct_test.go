package qdf

import (
	"bytes"
	"reflect"
	"testing"
)

// TestWriteStringInline_DenseLastIDSync pins that WriteStringInline keeps the
// Dense state machine in lockstep with the decoder. The decoder resets lastID to
// invalid on every inline string read; the encoder must mirror that, otherwise a
// later repeated value emits tagStateRepeat against a lastID the decoder already
// dropped — desyncing into a wrong value or ErrUnknownStateID.
func TestWriteStringInline_DenseLastIDSync(t *testing.T) {
	enc := NewEncoder(Dense)
	enc.EnsureHeader()
	enc.WriteString("AAAAAAAA")       // intern id 0
	enc.WriteString("BBBBBBBB")       // intern id 1, becomes lastID
	enc.WriteStringInline("CCCCCCCC") // forced inline: must invalidate lastID
	enc.WriteString("BBBBBBBB")       // repeat of id 1
	buf := append([]byte(nil), enc.Bytes()...)

	dec := NewDecoderOnBuf(buf)
	want := []string{"AAAAAAAA", "BBBBBBBB", "CCCCCCCC", "BBBBBBBB"}
	for i, w := range want {
		got, err := dec.ReadString()
		if err != nil {
			t.Fatalf("ReadString[%d]: %v", i, err)
		}
		if got != w {
			t.Fatalf("ReadString[%d] = %q, want %q", i, got, w)
		}
	}
}

// TestInternCapClampedBelowSentinel pins that the intern-table cap can never
// admit id 0xFFFF, which the MRU ring and LRU links reserve as their
// empty/no-neighbor sentinel. A larger cap would let the 65536th interned
// string take id 0xFFFF and corrupt the chains.
func TestInternCapClampedBelowSentinel(t *testing.T) {
	e := NewEncoder(Dense)
	e.SetIntern(0, 1<<20) // absurd cap; must clamp
	if e.maxStateEntries > maxInternEntries {
		t.Fatalf("SetIntern cap not clamped: %d > %d", e.maxStateEntries, maxInternEntries)
	}
	if e.maxStateEntries-1 >= 0xFFFF {
		t.Fatalf("max assignable id %d collides with sentinel 0xFFFF", e.maxStateEntries-1)
	}
	// The stream encoder caps its own table and must obey the same ceiling.
	var buf bytes.Buffer
	se := NewStreamEncoder(&buf, Dense)
	if se.enc.maxStateEntries > maxInternEntries {
		t.Fatalf("stream encoder cap %d exceeds ceiling %d", se.enc.maxStateEntries, maxInternEntries)
	}
}

// lyingUnmarshaler reports consuming more bytes than the buffer holds.
type lyingUnmarshaler struct{}

func (lyingUnmarshaler) UnmarshalQDF(src []byte) (int, error) { return len(src) + 100, nil }

// negConsumeUnmarshaler reports a negative byte count.
type negConsumeUnmarshaler struct{}

func (negConsumeUnmarshaler) UnmarshalQDF([]byte) (int, error) { return -5, nil }

// TestUnmarshalNested_RejectsBadConsumeCount pins that UnmarshalNested rejects a
// nested Unmarshaler that over- or under-reports bytes consumed. Both the reflect
// path and generated code advance the parent cursor by this count, so a bogus
// value would push the cursor out of bounds and panic the next read.
func TestUnmarshalNested_RejectsBadConsumeCount(t *testing.T) {
	if _, err := UnmarshalNested(lyingUnmarshaler{}, []byte{1, 2, 3}, false); err == nil {
		t.Fatal("UnmarshalNested accepted an over-consume count")
	}
	if _, err := UnmarshalNested(negConsumeUnmarshaler{}, []byte{1, 2, 3}, false); err == nil {
		t.Fatal("UnmarshalNested accepted a negative consume count")
	}
	// A well-behaved count is still passed through.
	if n, err := UnmarshalNested(okUnmarshaler{}, []byte{1, 2, 3}, false); err != nil || n != 2 {
		t.Fatalf("well-behaved UnmarshalNested: n=%d err=%v, want 2,nil", n, err)
	}
}

// okUnmarshaler consumes a valid prefix.
type okUnmarshaler struct{}

func (okUnmarshaler) UnmarshalQDF(_ []byte) (int, error) { return 2, nil }

// marshalOnly implements ONLY Marshaler. Its MarshalQDF writes a sentinel
// string, distinct from the map a structural encode of the struct would emit.
type marshalOnly struct{ X int }

func (marshalOnly) MarshalQDF(dst []byte) ([]byte, error) {
	e := NewEncoderOnBuf(dst, Fast)
	if len(dst) >= 5 && dst[0] == Magic0 && dst[1] == Magic1 && dst[2] == Magic2 {
		e.MarkHeaderWritten()
	} else {
		e.EnsureHeader()
	}
	e.WriteString("SENTINEL-ENC")
	return e.Bytes(), nil
}

// unmarshalOnly implements ONLY Unmarshaler (pointer receiver). Its UnmarshalQDF
// ignores the wire and sets a sentinel, distinct from what a structural decode
// would read.
type unmarshalOnly struct{ X int }

func (u *unmarshalOnly) UnmarshalQDF(src []byte) (int, error) {
	u.X = 999
	return len(src), nil
}

// TestAsymmetricMarshaler_NotOverwritten pins that a type implementing only one
// of Marshaler/Unmarshaler keeps its custom codec for that direction. fillDesc's
// structural switch unconditionally set both encode and decode, clobbering the
// custom method unless both were present.
func TestAsymmetricMarshaler_NotOverwritten(t *testing.T) {
	// Marshaler-only: custom encode must run (writes a bare string), not the
	// structural map encode.
	b, err := Marshal(marshalOnly{X: 7}, OptSpeed)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var s string
	if err := Unmarshal(b, &s); err != nil {
		t.Fatalf("custom MarshalQDF was overwritten by structural encode: %v", err)
	}
	if s != "SENTINEL-ENC" {
		t.Fatalf("custom MarshalQDF output = %q, want SENTINEL-ENC", s)
	}

	// Unmarshaler-only: structural encode (writes {x:7}), custom decode must run
	// and set the sentinel 999 rather than reading 7 off the wire.
	eb, err := Marshal(unmarshalOnly{X: 7}, OptSpeed)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out unmarshalOnly
	if err := Unmarshal(eb, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.X != 999 {
		t.Fatalf("custom UnmarshalQDF was overwritten by structural decode: X=%d, want 999", out.X)
	}
}

// directSample is a minimal hand-rolled Marshaler / Unmarshaler used
// to exercise MarshalDirect / UnmarshalDirect without pulling in the
// separate codegen_test module. Wire layout: id varint, then a string.
type directSample struct {
	ID   int
	Name string
}

func (v *directSample) MarshalQDF(dst []byte) ([]byte, error) {
	enc := NewEncoderOnBuf(dst, Fast)
	enc.MarkHeaderWritten()
	// Encode as a 2-element struct-shaped map for round-trip parity
	// with the reflect path on tagged structs. Generated code uses the
	// same idea (fixmap with field-name keys).
	enc.WriteMapHeader(2)
	enc.WriteStringInline("id")
	enc.WriteInt(int64(v.ID))
	enc.WriteStringInline("name")
	enc.WriteStringInline(v.Name)
	return enc.Bytes(), nil
}

func (v *directSample) UnmarshalQDF(src []byte) (int, error) {
	dec := NewDecoderOnBuf(src)
	dec.MarkHeaderRead()
	n, err := dec.ReadMapHeader()
	if err != nil {
		return 0, err
	}
	for range n {
		key, err := dec.ReadString()
		if err != nil {
			return dec.Pos(), err
		}
		switch key {
		case "id":
			id, err := dec.ReadInt()
			if err != nil {
				return dec.Pos(), err
			}
			v.ID = int(id)
		case "name":
			s, err := dec.ReadString()
			if err != nil {
				return dec.Pos(), err
			}
			v.Name = s
		default:
			if err := dec.Skip(); err != nil {
				return dec.Pos(), err
			}
		}
	}
	return dec.Pos(), nil
}

func TestMarshalDirect_RoundTrip(t *testing.T) {
	in := directSample{ID: 42, Name: "hello"}
	buf, err := MarshalDirect(&in)
	if err != nil {
		t.Fatal(err)
	}
	var out directSample
	if err := UnmarshalDirect(buf, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("round-trip mismatch: %+v vs %+v", in, out)
	}
}

func TestMarshalDirect_WireMatchesInterfaceDispatch(t *testing.T) {
	// The generic shortcut must produce the same wire as Marshal(v, OptSpeed),
	// which goes through encodeMarshaler.
	in := directSample{ID: 7, Name: "qdf"}
	a, err := Marshal(&in, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	b, err := MarshalDirect(&in)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatalf("wire mismatch:\n any=%x\n dir=%x", a, b)
	}
}

func TestUnmarshalDirect_AcceptsBothEncoders(t *testing.T) {
	in := directSample{ID: 99, Name: "compat"}
	// Encode via Marshal (interface dispatch).
	a, _ := Marshal(&in, OptSpeed)
	// Encode via MarshalDirect (generic shortcut).
	b, _ := MarshalDirect(&in)

	for label, buf := range map[string][]byte{"any": a, "direct": b} {
		var out directSample
		if err := UnmarshalDirect(buf, &out); err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		if out != in {
			t.Fatalf("%s: %+v vs %+v", label, out, in)
		}
	}
}

func TestUnmarshalDirect_HeaderValidation(t *testing.T) {
	cases := map[string][]byte{
		"short":       {},
		"bad-magic":   {'X', 'Y', 'Z', 1, 0},
		"bad-version": {'Q', 'D', 'F', 0xEE, 0},
	}
	for name, buf := range cases {
		var out directSample
		if err := UnmarshalDirect(buf, &out); err == nil {
			t.Fatalf("%s: expected error", name)
		}
	}
}

func TestAppendMarshalDirect_AppendsToDst(t *testing.T) {
	in := directSample{ID: 1, Name: "append"}
	prefix := []byte("LEAD")
	out, err := AppendMarshalDirect(prefix, &in)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(out, prefix) {
		t.Fatalf("prefix lost: %x", out)
	}
	// Decode the tail past the prefix.
	var got directSample
	if err := UnmarshalDirect(out[len(prefix):], &got); err != nil {
		t.Fatal(err)
	}
	if got != in {
		t.Fatalf("decoded %+v", got)
	}
}

func BenchmarkMarshalDirect_VsReflect(b *testing.B) {
	in := directSample{ID: 12345, Name: "hello-world"}
	b.Run("reflect/Marshal", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, _ = Marshal(&in, OptSpeed)
		}
	})
	b.Run("direct/MarshalDirect", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, _ = MarshalDirect(&in)
		}
	})
}

func BenchmarkUnmarshalDirect_VsReflect(b *testing.B) {
	in := directSample{ID: 12345, Name: "hello-world"}
	buf, _ := MarshalDirect(&in)
	b.Run("reflect/Unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var out directSample
			_ = Unmarshal(buf, &out)
		}
	})
	b.Run("direct/UnmarshalDirect", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var out directSample
			_ = UnmarshalDirect(buf, &out)
		}
	})
}
