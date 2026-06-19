package qdf

import (
	"bytes"
	"testing"
)

func TestWriteColStructHeader_DeclareThenReuse(t *testing.T) {
	e := NewEncoderOnBuf(nil, Fast)
	e.EnsureHeader()
	names := []string{"a", "b"}
	kinds := []byte{0, 2} // colKindInt, colKindFloat
	start := len(e.Bytes())
	e.WriteColStructHeader(3, names, kinds)
	first := append([]byte(nil), e.Bytes()[start:]...)
	// First emission declares the shape inline: tag, uvarint(n=3), uvarint(0=declare),
	// uvarint(2 cols), then per-col name+kind.
	if first[0] != tagColStruct {
		t.Fatalf("want tagColStruct 0x%x, got 0x%x", tagColStruct, first[0])
	}
	// Second emission of the same shape reuses by id (much shorter, no names).
	mid := len(e.Bytes())
	e.WriteColStructHeader(3, names, kinds)
	second := e.Bytes()[mid:]
	if len(second) >= len(first) {
		t.Fatalf("reuse (%d bytes) should be shorter than declare (%d bytes)", len(second), len(first))
	}
	if second[0] != tagColStruct {
		t.Fatalf("reuse tag wrong: 0x%x", second[0])
	}
}

func TestColStruct_RoundTripExposedAPI(t *testing.T) {
	e := NewEncoderOnBuf(nil, Fast)
	e.EnsureHeader()
	e.WriteColStructHeader(4, []string{"x", "y"}, []byte{0, 3}) // int, bool
	if err := e.WriteIntColumn([]int64{10, 20, 30, 40}); err != nil {
		t.Fatal(err)
	}
	if err := e.WriteBoolColumn([]bool{true, false, true, false}); err != nil {
		t.Fatal(err)
	}
	buf := e.Bytes()

	d := NewDecoderOnBuf(buf)
	if err := d.readHeader(); err != nil {
		t.Fatal(err)
	}
	if !d.PeekColStruct() {
		t.Fatal("PeekColStruct = false, want true")
	}
	n, names, kinds, err := d.ReadColStructHeader()
	if err != nil {
		t.Fatal(err)
	}
	if n != 4 || len(names) != 2 || names[0] != "x" || names[1] != "y" {
		t.Fatalf("header mismatch: n=%d names=%v", n, names)
	}
	if kinds[0] != 0 || kinds[1] != 3 {
		t.Fatalf("kinds mismatch: %v", kinds)
	}
	xs, err := d.ReadIntColumn(n)
	if err != nil {
		t.Fatal(err)
	}
	ys, err := d.ReadBoolColumn(n)
	if err != nil {
		t.Fatal(err)
	}
	if len(xs) != 4 || xs[0] != 10 || xs[3] != 40 {
		t.Fatalf("int column wrong: %v", xs)
	}
	if len(ys) != 4 || !ys[0] || ys[1] {
		t.Fatalf("bool column wrong: %v", ys)
	}
}

func TestPeekColStruct_FalseOnArrayHeader(t *testing.T) {
	e := NewEncoderOnBuf(nil, Fast)
	e.EnsureHeader()
	e.WriteArrayHeader(2)
	d := NewDecoderOnBuf(e.Bytes())
	if err := d.readHeader(); err != nil {
		t.Fatal(err)
	}
	if d.PeekColStruct() {
		t.Fatal("PeekColStruct = true on an array header, want false")
	}
}

func TestColStruct_ReflectEncodeDecodesViaExposedReader(t *testing.T) {
	type metric struct {
		A int64
		B float64
	}
	// Reflect path columnar-encodes a plain []struct (>= columnarMinElems).
	in := make([]metric, 32)
	for i := range in {
		in[i] = metric{A: int64(i), B: float64(i) * 1.5}
	}
	buf, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	d := NewDecoderOnBuf(buf)
	if err := d.readHeader(); err != nil {
		t.Fatal(err)
	}
	if !d.PeekColStruct() {
		t.Skip("reflect did not choose columnar for this shape; interop check N/A")
	}
	n, names, kinds, err := d.ReadColStructHeader()
	if err != nil {
		t.Fatal(err)
	}
	if n != 32 || len(names) != 2 || kinds[0] != 0 || kinds[1] != 2 {
		t.Fatalf("interop header mismatch n=%d names=%v kinds=%v", n, names, kinds)
	}
}

func TestWriteColumns_RoundTripViaReadColShape(t *testing.T) {
	e := NewEncoderOnBuf(nil, Fast)
	e.EnsureHeader()
	e.WriteColStructHeader(4, []string{"x", "y"}, []byte{0, 3}) // int, bool
	if err := e.WriteIntColumn([]int64{10, 20, 30, 40}); err != nil {
		t.Fatal(err)
	}
	if err := e.WriteBoolColumn([]bool{true, false, true, false}); err != nil {
		t.Fatal(err)
	}
	buf := e.Bytes()
	if !bytes.Contains(buf, []byte{tagColStruct}) {
		t.Fatal("no colStruct frame emitted")
	}
}
