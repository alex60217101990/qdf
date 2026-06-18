package qdf

import "testing"

// shapeToken is a stable per-"type" address the encoder keys the struct shape on
// (the generated code uses a package-level var per type).
var shapeToken byte

// TestStructShapeDeclareReuse drives Encoder.StructShape directly: the first
// emit for a token declares the field-name shape (tagMapShape, id 0, count,
// names); subsequent emits for the same token on the same encoder reuse it
// (tagMapShape + id, no names). A decoder reading the stream sees the same field
// names for every record.
func TestStructShapeDeclareReuse(t *testing.T) {
	// Two pre-encoded fixstr field-name headers, as qdfgen emits (qdfFieldHdr_*).
	hdrA := append([]byte{tagFixstr | 1}, 'A')
	hdrB := append([]byte{tagFixstr | 1}, 'B')
	fieldHdrs := [][]byte{hdrA, hdrB}

	e := NewEncoderOnBuf(nil, Fast)
	e.EnsureHeader()
	// Three "records" of the same shape, each: shape header + 2 int values.
	for i := range 3 {
		e.StructShape(&shapeToken, fieldHdrs)
		e.WriteInt(int64(i * 10))
		e.WriteInt(int64(i*10 + 1))
	}
	buf := e.Bytes()

	// Decode: every record's shape must resolve to the same names ["A","B"].
	d := NewDecoderOnBuf(buf)
	if err := d.readHeader(); err != nil { // consume the 5-byte stream header
		t.Fatal(err)
	}
	for i := range 3 {
		names, err := decodeMapStringShapeHeader(d)
		if err != nil {
			t.Fatalf("record %d shape header: %v", i, err)
		}
		if len(names) != 2 || names[0] != "A" || names[1] != "B" {
			t.Fatalf("record %d names = %v, want [A B]", i, names)
		}
		v0, err := d.ReadInt()
		if err != nil {
			t.Fatal(err)
		}
		v1, err := d.ReadInt()
		if err != nil {
			t.Fatal(err)
		}
		if v0 != int64(i*10) || v1 != int64(i*10+1) {
			t.Fatalf("record %d values = %d,%d", i, v0, v1)
		}
	}

	// The reuse records must be far smaller than the declaration: only the first
	// carries the field names. A second encoder emitting the SAME single record
	// must be larger per-record than the amortized reuse, proving the names are
	// written once.
	eDecl := NewEncoderOnBuf(nil, Fast)
	eDecl.EnsureHeader()
	eDecl.StructShape(&shapeToken2, fieldHdrs)
	declLen := len(eDecl.Bytes())
	eReuse := NewEncoderOnBuf(nil, Fast)
	eReuse.EnsureHeader()
	eReuse.StructShape(&shapeToken2, fieldHdrs) // declare
	afterDecl := len(eReuse.Bytes())
	eReuse.StructShape(&shapeToken2, fieldHdrs) // reuse
	reuseDelta := len(eReuse.Bytes()) - afterDecl
	if reuseDelta >= declLen {
		t.Fatalf("reuse delta %d not smaller than declaration %d — names not interned", reuseDelta, declLen)
	}
}

var shapeToken2 byte
var shapeToken3 byte

// TestReadStructHeader covers both forms ReadStructHeader must accept: a
// shape-interned header (from StructShape) and a plain map header.
func TestReadStructHeader(t *testing.T) {
	hdrA := append([]byte{tagFixstr | 1}, 'A')
	fieldHdrs := [][]byte{hdrA}

	// Shaped form.
	es := NewEncoderOnBuf(nil, Fast)
	es.EnsureHeader()
	es.StructShape(&shapeToken3, fieldHdrs)
	es.WriteInt(42)
	ds := NewDecoderOnBuf(es.Bytes())
	if err := ds.readHeader(); err != nil {
		t.Fatal(err)
	}
	names, plainN, shaped, err := ds.ReadStructHeader()
	if err != nil {
		t.Fatal(err)
	}
	if !shaped || len(names) != 1 || names[0] != "A" || plainN != 0 {
		t.Fatalf("shaped: names=%v plainN=%d shaped=%v", names, plainN, shaped)
	}
	if v, _ := ds.ReadInt(); v != 42 {
		t.Fatalf("shaped value=%d", v)
	}

	// Plain map form (interop: a non-shaped producer).
	ep := NewEncoderOnBuf(nil, Fast)
	ep.EnsureHeader()
	ep.WriteMapHeader(1)
	ep.WriteString("A")
	ep.WriteInt(7)
	dp := NewDecoderOnBuf(ep.Bytes())
	if err := dp.readHeader(); err != nil {
		t.Fatal(err)
	}
	names, plainN, shaped, err = dp.ReadStructHeader()
	if err != nil {
		t.Fatal(err)
	}
	if shaped || plainN != 1 || names != nil {
		t.Fatalf("plain: names=%v plainN=%d shaped=%v", names, plainN, shaped)
	}
	kb, _ := dp.ReadStringBytes()
	if string(kb) != "A" {
		t.Fatalf("plain key=%q", kb)
	}
	if v, _ := dp.ReadInt(); v != 7 {
		t.Fatalf("plain value=%d", v)
	}
}
