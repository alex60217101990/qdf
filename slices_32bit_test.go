package qdf

import (
	"math/rand"
	"testing"
)

func TestQPack32BitCompresses(t *testing.T) {
	const N = 1024
	// monotonic uint32 — must compress far below raw 4*N via delta/FOR
	u := make([]uint32, N)
	for i := range u {
		u[i] = uint32(1_000_000 + i)
	}
	enc, err := Marshal(u, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if len(enc) > 512 {
		t.Fatalf("uint32 monotonic not compressed: %d B (want << 4096)", len(enc))
	}
	var back []uint32
	if err := Unmarshal(enc, &back); err != nil {
		t.Fatal(err)
	}
	if len(back) != N {
		t.Fatalf("len %d != %d", len(back), N)
	}
	for i := range u {
		if back[i] != u[i] {
			t.Fatalf("u32 roundtrip row %d: %d != %d", i, back[i], u[i])
		}
	}

	// int32 with negatives, low cardinality
	s := make([]int32, N)
	for i := range s {
		s[i] = int32(-5 + i%4)
	}
	enc2, err := Marshal(s, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if len(enc2) > 512 {
		t.Fatalf("int32 low-card not compressed: %d B", len(enc2))
	}
	var back2 []int32
	if err := Unmarshal(enc2, &back2); err != nil {
		t.Fatal(err)
	}
	for i := range s {
		if back2[i] != s[i] {
			t.Fatalf("i32 roundtrip row %d: %d != %d", i, back2[i], s[i])
		}
	}

	// edge: full-range uint32 values (max) must round-trip exactly (no truncation bug)
	w := []uint32{0, 1, 4294967295, 2147483648, 42}
	enc3, _ := Marshal(w, OptBalanced)
	var back3 []uint32
	if err := Unmarshal(enc3, &back3); err != nil {
		t.Fatal(err)
	}
	for i := range w {
		if back3[i] != w[i] {
			t.Fatalf("u32 max roundtrip %d: %d != %d", i, back3[i], w[i])
		}
	}

	// edge: int32 min/max
	wi := []int32{-2147483648, 2147483647, 0, -1, 1}
	enc4, _ := Marshal(wi, OptBalanced)
	var back4 []int32
	if err := Unmarshal(enc4, &back4); err != nil {
		t.Fatal(err)
	}
	for i := range wi {
		if back4[i] != wi[i] {
			t.Fatalf("i32 min/max %d: %d != %d", i, back4[i], wi[i])
		}
	}

	// empty + single
	for _, e := range [][]uint32{{}, {7}} {
		eb, _ := Marshal(e, OptBalanced)
		var bk []uint32
		if err := Unmarshal(eb, &bk); err != nil {
			t.Fatal(err)
		}
		if len(bk) != len(e) {
			t.Fatalf("empty/single u32")
		}
	}
}

// TestQPack32BitNeverWorse: incompressible full-range 32-bit data must not
// exceed the native 4 B/elem raw form — widening to 64-bit must never inflate.
func TestQPack32BitNeverWorse(t *testing.T) {
	const N = 1024
	r := rand.New(rand.NewSource(1))
	u := make([]uint32, N)
	for i := range u {
		u[i] = r.Uint32()
	}
	enc, _ := Marshal(u, OptBalanced)
	if len(enc) > 4*N+16 { // 4 B/elem + small header, never the 8 B/elem widened form
		t.Fatalf("incompressible uint32 inflated to %d B (native floor ~%d)", len(enc), 4*N)
	}
	var back []uint32
	if err := Unmarshal(enc, &back); err != nil {
		t.Fatal(err)
	}
	for i := range u {
		if back[i] != u[i] {
			t.Fatalf("u32 nw roundtrip %d", i)
		}
	}
	s := make([]int32, N)
	for i := range s {
		s[i] = int32(r.Uint32())
	}
	enc2, _ := Marshal(s, OptBalanced)
	if len(enc2) > 4*N+16 {
		t.Fatalf("incompressible int32 inflated to %d B", len(enc2))
	}

	// Small-N adversarial: alternating near-max-range values where a widened
	// DeltaFor would be chosen over FOR yet still exceed native 4 B/elem raw.
	for _, small := range [][]uint32{
		{1450029205, 1650547297, 1650967836, 2572537883, 3887843792, 3947857014, 4268614616},
		{0, 2147483648, 0, 2147483648, 0, 2147483648, 0},
	} {
		eb, _ := Marshal(small, OptBalanced)
		// native total = 5B message header + tag(1)+kind(1)+nVarUint+4n.
		native := 5 + 2 + 1 + 4*len(small)
		if len(eb) > native {
			t.Fatalf("small-N uint32 inflated to %d B (native %d)", len(eb), native)
		}
		var bk []uint32
		if err := Unmarshal(eb, &bk); err != nil {
			t.Fatal(err)
		}
		for i := range small {
			if bk[i] != small[i] {
				t.Fatalf("small-N u32 roundtrip %d", i)
			}
		}
	}
	var back2 []int32
	if err := Unmarshal(enc2, &back2); err != nil {
		t.Fatal(err)
	}
	for i := range s {
		if back2[i] != s[i] {
			t.Fatalf("i32 nw roundtrip %d", i)
		}
	}
}
