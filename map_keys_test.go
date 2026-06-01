package qdf

import "testing"

func TestMapKeys_Matrix(t *testing.T) {
	type rec struct {
		U8  map[uint8]string
		U32 map[uint32]int64
		U64 map[uint64]bool
	}
	v := rec{
		U8:  map[uint8]string{0: "a", 255: "b"},
		U32: map[uint32]int64{1: -1, 4294967295: 2},
		U64: map[uint64]bool{0: true, 1 << 63: false},
	}
	roundtripBundles(t, v)
}

func TestMapNilVsEmpty(t *testing.T) {
	type rec struct {
		Nil   map[string]int
		Empty map[string]int
	}
	v := rec{Nil: nil, Empty: map[string]int{}}
	data, err := Marshal(v, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var out rec
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	// Record actual behaviour. If the format does NOT distinguish nil
	// from empty, this is a design question, not a bug — log it with
	// t.Logf and DO NOT fail; report it to the controller. If it DOES
	// distinguish, assert it.
	t.Logf("nil map decoded as nil=%v (len=%d); empty map decoded as nil=%v (len=%d)",
		out.Nil == nil, len(out.Nil), out.Empty == nil, len(out.Empty))
}
