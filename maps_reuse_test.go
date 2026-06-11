package qdf

import (
	"reflect"
	"strconv"
	"testing"
)

func TestMapReuseRoundTrip(t *testing.T) {
	type rec struct {
		ID   int               `qdf:"id"`
		Tags map[string]string `qdf:"tags"`
	}
	in := []rec{
		{ID: 1, Tags: map[string]string{"a": "x", "b": "y"}},
		{ID: 2, Tags: map[string]string{"c": "z"}},
	}
	data, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var out []rec
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("fresh: got %v", out)
	}
	// decode AGAIN into the SAME out (reuse path) — must still equal in,
	// no stale keys from the first decode.
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("reused: got %v", out)
	}
}

func TestMapReuseNoStaleKeys(t *testing.T) {
	type rec struct {
		Tags map[string]string `qdf:"tags"`
	}
	a := []rec{{Tags: map[string]string{"old1": "1", "old2": "2"}}}
	b := []rec{{Tags: map[string]string{"new": "3"}}}
	da, _ := Marshal(a, OptBalanced)
	db, _ := Marshal(b, OptBalanced)
	var out []rec
	if err := Unmarshal(da, &out); err != nil {
		t.Fatal(err)
	}
	if err := Unmarshal(db, &out); err != nil {
		t.Fatal(err)
	} // reuse the map
	if !reflect.DeepEqual(b, out) {
		t.Fatalf("stale keys leaked: got %v want %v", out, b)
	}
	if _, ok := out[0].Tags["old1"]; ok {
		t.Fatal("stale key old1 survived")
	}
}

func TestMapReuseCutsAllocs(t *testing.T) {
	type rec struct {
		Tags map[string]string `qdf:"tags"`
	}
	recs := make([]rec, 64)
	for i := range recs {
		m := make(map[string]string, 8)
		for j := 0; j < 8; j++ {
			m["k"+strconv.Itoa(j)] = "v" + strconv.Itoa(i*8+j)
		}
		recs[i] = rec{Tags: m}
	}
	data, _ := Marshal(recs, OptBalanced)
	var out []rec
	_ = Unmarshal(data, &out) // warm: allocates the maps
	reuse := testing.AllocsPerRun(20, func() { _ = Unmarshal(data, &out) })
	var fresh []rec
	freshAllocs := testing.AllocsPerRun(20, func() { fresh = nil; _ = Unmarshal(data, &fresh) })
	if reuse >= freshAllocs {
		t.Fatalf("reuse (%v) did not cut allocs vs fresh (%v)", reuse, freshAllocs)
	}
	t.Logf("allocs/op: fresh=%v reuse=%v", freshAllocs, reuse)
}
