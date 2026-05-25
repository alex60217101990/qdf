package qdf

import (
	"bytes"
	"reflect"
	"strconv"
	"testing"
)

type genericS struct {
	A int      `qdf:"a"`
	B string   `qdf:"b"`
	C []uint64 `qdf:"c"`
	D []bool   `qdf:"d"`
}

func sampleGeneric() genericS {
	return genericS{A: 42, B: "hello", C: []uint64{1, 2, 3, 4, 5}, D: []bool{true, false, true}}
}

func TestMarshalT_WireMatchesMarshal(t *testing.T) {
	in := sampleGeneric()
	a, err := Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	b, err := MarshalT(in)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatalf("wire mismatch:\n any=%x\n gen=%x", a, b)
	}
}

func TestMarshalQPackT_WireMatchesMarshalQPack(t *testing.T) {
	in := sampleGeneric()
	a, _ := MarshalQPack(in)
	b, _ := MarshalQPackT(in)
	if !bytes.Equal(a, b) {
		t.Fatalf("qpack wire mismatch:\n any=%x\n gen=%x", a, b)
	}
}

func TestMarshalDenseT_WireMatchesMarshalDense(t *testing.T) {
	in := sampleGeneric()
	a, _ := MarshalDense(in)
	b, _ := MarshalDenseT(in)
	if !bytes.Equal(a, b) {
		t.Fatalf("dense wire mismatch:\n any=%x\n gen=%x", a, b)
	}
}

func TestUnmarshalT_RoundTrip(t *testing.T) {
	in := sampleGeneric()
	buf, _ := MarshalT(in)
	var out genericS
	if err := UnmarshalT(buf, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("round-trip mismatch")
	}
}

func TestMarshalT_PointerInput(t *testing.T) {
	in := sampleGeneric()
	a, err := Marshal(&in)
	if err != nil {
		t.Fatal(err)
	}
	b, err := MarshalT(&in)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatalf("pointer-input mismatch")
	}
}

func TestMarshalT_PrimitiveTypes(t *testing.T) {
	// Top-level primitives must produce the same wire.
	checkInt := func(v int64) {
		a, _ := Marshal(v)
		b, _ := MarshalT(v)
		if !bytes.Equal(a, b) {
			t.Fatalf("int %d: %x vs %x", v, a, b)
		}
	}
	checkInt(0)
	checkInt(1)
	checkInt(-1)
	checkInt(127)
	checkInt(-128)
	checkInt(1 << 40)

	checkStr := func(s string) {
		a, _ := Marshal(s)
		b, _ := MarshalT(s)
		if !bytes.Equal(a, b) {
			t.Fatalf("str %q: %x vs %x", s, a, b)
		}
	}
	checkStr("")
	checkStr("hi")
	checkStr("a-longer-string")
}

func TestUnmarshalT_NilPointer(t *testing.T) {
	if err := UnmarshalT[genericS](nil, nil); err == nil {
		t.Fatal("expected error on nil out")
	}
}

func BenchmarkMarshalT_VsAny(b *testing.B) {
	in := sampleGeneric()
	b.Run("any/Marshal", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, _ = Marshal(in)
		}
	})
	b.Run("generic/MarshalT", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, _ = MarshalT(in)
		}
	})
}

func BenchmarkUnmarshalT_VsAny(b *testing.B) {
	in := sampleGeneric()
	buf, _ := Marshal(in)
	b.Run("any/Unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var out genericS
			_ = Unmarshal(buf, &out)
		}
	})
	b.Run("generic/UnmarshalT", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var out genericS
			_ = UnmarshalT(buf, &out)
		}
	})
}

func BenchmarkMarshalT_Sizes(b *testing.B) {
	for _, n := range []int{0, 4, 64} {
		in := genericS{A: n, B: strconv.Itoa(n), C: make([]uint64, n)}
		for i := range in.C {
			in.C[i] = uint64(i)
		}
		b.Run("any/n="+strconv.Itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_, _ = Marshal(in)
			}
		})
		b.Run("gen/n="+strconv.Itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_, _ = MarshalT(in)
			}
		})
	}
}
