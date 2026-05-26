package qdf

import (
	"bytes"
	"reflect"
	"testing"
)

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
