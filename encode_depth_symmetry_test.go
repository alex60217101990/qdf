package qdf

import "testing"

type depthRecS struct {
	V        int         `qdf:"v"`
	Children []depthRecS `qdf:"children"` // recursion carried by a SLICE field
}

// Regression: the encoder counted recursion depth only at pointer/interface
// hops, while the decoder caps EVERY slice/array/map level at maxDepth. So a
// recursion carried by slices/maps encoded unbounded but failed to decode
// (ErrCycleDetected) — a value qdf produced but could not read back. The encoder
// now guards slice/array/map levels too, so a too-deep value is refused AT
// ENCODE rather than emitted as an undecodable blob (symmetric encode/decode).
func TestEncodeDecodeDepthSymmetry(t *testing.T) {
	// Slice-carried nesting beyond DefaultMaxDepth must be rejected at encode.
	leaf := depthRecS{V: 0}
	cur := &leaf
	for i := 1; i < DefaultMaxDepth+2000; i++ {
		cur.Children = []depthRecS{{V: i}}
		cur = &cur.Children[0]
	}
	if _, err := Marshal(leaf, OptSpeed); err == nil {
		t.Fatal("expected encode to reject slice-carried nesting beyond maxDepth")
	}

	// A modest depth still round-trips (the guard does not over-reject).
	shallow := depthRecS{V: 0}
	c := &shallow
	for i := 1; i < 50; i++ {
		c.Children = []depthRecS{{V: i}}
		c = &c.Children[0]
	}
	b, err := Marshal(shallow, OptSpeed)
	if err != nil {
		t.Fatalf("shallow encode: %v", err)
	}
	var out depthRecS
	if err := Unmarshal(b, &out); err != nil {
		t.Fatalf("shallow decode (must round-trip): %v", err)
	}
	if out.Children[0].Children[0].V != 2 {
		t.Fatalf("shallow round-trip wrong: %+v", out)
	}
}
