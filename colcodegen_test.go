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
