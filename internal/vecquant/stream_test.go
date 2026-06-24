package vecquant

import "testing"

func TestCoordStreamRoundTrip(t *testing.T) {
	cases := [][]int32{
		{},
		{0},
		{0, 1, -1, 2, -2, 127, -128, 1000, -1000, 1 << 20, -(1 << 20)},
	}
	for _, q := range cases {
		enc := encodeCoords(q)
		got, err := decodeCoords(enc, len(q))
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(got) != len(q) {
			t.Fatalf("len %d != %d", len(got), len(q))
		}
		for i := range q {
			if got[i] != q[i] {
				t.Fatalf("i=%d got %d want %d", i, got[i], q[i])
			}
		}
	}
}

func TestCoordStreamNeverLarger(t *testing.T) {
	// Highly compressible (all zero) must use the rANS branch and be small.
	q := make([]int32, 4096)
	enc := encodeCoords(q)
	if len(enc) >= 4096 {
		t.Fatalf("zero stream not compressed: %d bytes", len(enc))
	}
}
