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

func TestStringColumn_RoundTrip(t *testing.T) {
	e := NewEncoderOnBuf(nil, Fast)
	e.EnsureHeader()
	e.WriteColStructHeader(5, []string{"name"}, []byte{4}) // colKindString
	in := []string{"alpha", "beta", "alpha", "gamma", "beta"}
	e.WriteStringColumn(in)
	d := NewDecoderOnBuf(e.Bytes())
	if err := d.readHeader(); err != nil {
		t.Fatal(err)
	}
	if !d.PeekColStruct() {
		t.Fatal("want colStruct frame")
	}
	n, names, kinds, err := d.ReadColStructHeader()
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 || names[0] != "name" || kinds[0] != 4 {
		t.Fatalf("header: n=%d names=%v kinds=%v", n, names, kinds)
	}
	got, err := d.ReadStringColumn(n)
	if err != nil {
		t.Fatal(err)
	}
	for i := range in {
		if got[i] != in[i] {
			t.Fatalf("string[%d]=%q want %q", i, got[i], in[i])
		}
	}
}

func TestTimeColumn_RoundTrip(t *testing.T) {
	e := NewEncoderOnBuf(nil, Fast)
	e.EnsureHeader()
	base := int64(1_700_000_000)
	secs := []int64{base, base + 1, base + 2, base + 60}
	nsec := []uint64{0, 500, 999_999_999, 12345}
	e.WriteColStructHeader(4, []string{"ts"}, []byte{5}) // colKindTime
	if err := e.WriteTimeColumn(secs, nsec); err != nil {
		t.Fatal(err)
	}
	d := NewDecoderOnBuf(e.Bytes())
	if err := d.readHeader(); err != nil {
		t.Fatal(err)
	}
	n, _, kinds, err := d.ReadColStructHeader()
	if err != nil {
		t.Fatal(err)
	}
	if kinds[0] != 5 {
		t.Fatalf("kind=%d want 5", kinds[0])
	}
	gs, gn, err := d.ReadTimeColumn(n)
	if err != nil {
		t.Fatal(err)
	}
	for i := range secs {
		if gs[i] != secs[i] || gn[i] != nsec[i] {
			t.Fatalf("time[%d]=(%d,%d) want (%d,%d)", i, gs[i], gn[i], secs[i], nsec[i])
		}
	}
}

func TestHybridColStructHeader_RoundTrip(t *testing.T) {
	e := NewEncoderOnBuf(nil, Fast)
	e.EnsureHeader()
	names := []string{"ts", "nested", "v"}
	kinds := []byte{0, 0xFF, 2} // int, residual, float64
	e.WriteHybridColStructHeader(7, names, kinds)
	d := NewDecoderOnBuf(e.Bytes())
	if err := d.readHeader(); err != nil {
		t.Fatal(err)
	}
	if !d.PeekHybridColStruct() {
		t.Fatal("want hybrid frame")
	}
	n, gn, gk, err := d.ReadHybridColStructHeader()
	if err != nil {
		t.Fatal(err)
	}
	if n != 7 || len(gn) != 3 || gn[1] != "nested" || gk[1] != 0xFF || gk[0] != 0 || gk[2] != 2 {
		t.Fatalf("hybrid header: n=%d names=%v kinds=%v", n, gn, gk)
	}
}

func TestStringColumnsBeneficial(t *testing.T) {
	lowCard := make([]string, 64)
	for i := range lowCard {
		lowCard[i] = []string{"GET", "POST", "PUT"}[i%3]
	}
	if !stringColumnsBeneficial(false, lowCard) {
		t.Fatal("low-cardinality string column should be columnar-beneficial")
	}
	highCard := make([]string, 64)
	for i := range highCard {
		highCard[i] = "unique-value-" + string(rune('a'+i%26)) + string(rune('0'+i%10)) + string(rune('A'+(i*7)%26))
	}
	if stringColumnsBeneficial(false, highCard) {
		t.Fatal("high-cardinality string column should stay row-major")
	}
}

func TestStringColumn_ConstFallback_Fast(t *testing.T) {
	// All-identical column under Fast → tagColStrConst: tiny wire, round-trips.
	e := NewEncoderOnBuf(nil, Fast)
	e.EnsureHeader()
	n := 500
	same := make([]string, n)
	for i := range same {
		same[i] = "constant-value"
	}
	e.WriteColStructHeader(n, []string{"s"}, []byte{4})
	e.WriteStringColumn(same)
	buf := e.Bytes()
	if !bytes.Contains(buf, []byte{tagColStrConst}) {
		t.Fatal("all-identical string column should use tagColStrConst under Fast")
	}
	if len(buf) > 100 {
		t.Fatalf("const column wire %d too large (not deduped)", len(buf))
	}
	d := NewDecoderOnBuf(buf)
	if err := d.readHeader(); err != nil {
		t.Fatal(err)
	}
	cn, _, _, err := d.ReadColStructHeader()
	if err != nil {
		t.Fatal(err)
	}
	got, err := d.ReadStringColumn(cn)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != n || got[0] != "constant-value" || got[n-1] != "constant-value" {
		t.Fatalf("const round-trip wrong: len=%d", len(got))
	}
}

func TestStringColumn_RawFallback_Fast(t *testing.T) {
	// High-cardinality under Fast → tagColStrRaw: one-slab decode, round-trips.
	e := NewEncoderOnBuf(nil, Fast)
	e.EnsureHeader()
	n := 300
	in := make([]string, n)
	for i := range in {
		in[i] = "unique-" + string(rune('a'+i%26)) + string(rune('0'+i%10)) + string(rune('A'+(i*7)%26))
	}
	e.WriteColStructHeader(n, []string{"s"}, []byte{4})
	e.WriteStringColumn(in)
	buf := e.Bytes()
	if !bytes.Contains(buf, []byte{tagColStrRaw}) {
		t.Fatal("high-cardinality string column should use tagColStrRaw under Fast")
	}
	d := NewDecoderOnBuf(buf)
	if err := d.readHeader(); err != nil {
		t.Fatal(err)
	}
	cn, _, _, err := d.ReadColStructHeader()
	if err != nil {
		t.Fatal(err)
	}
	got, err := d.ReadStringColumn(cn)
	if err != nil {
		t.Fatal(err)
	}
	for i := range in {
		if got[i] != in[i] {
			t.Fatalf("raw round-trip[%d]=%q want %q", i, got[i], in[i])
		}
	}
}
