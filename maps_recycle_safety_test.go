package qdf

import (
	"reflect"
	"runtime"
	"strconv"
	"testing"
)

// Verifies BUG-1 (claimed GC-liveness use-after-free on free-list shrink) is FALSE:
// hammer GC during 2000 recycle decodes; if harvested hmap pointers were freed
// between pop and use, this corrupts/crashes.
func TestMapRecycleGCSafe(t *testing.T) {
	if testing.Short() {
		t.Skip("slow GC stress")
	}
	type rec struct {
		Tags map[string]string `qdf:"tags"`
	}
	in := make([]rec, 64)
	for i := range in {
		in[i].Tags = map[string]string{"k" + strconv.Itoa(i): "v" + strconv.Itoa(i), "s": "x"}
	}
	data, _ := Marshal(in, OptBalanced)
	var out []rec
	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				runtime.GC()
			}
		}
	}()
	defer close(stop)
	for iter := 0; iter < 2000; iter++ {
		if err := Unmarshal(data, &out); err != nil {
			t.Fatalf("iter %d: %v", iter, err)
		}
		if !reflect.DeepEqual(in, out) {
			t.Fatalf("iter %d: CORRUPTION %v", iter, out)
		}
		runtime.GC()
	}
}

// Verifies the UnmarshalT mapFreeList clear: a leftover map from a prior
// Unmarshal must not surface in an UnmarshalT target.
func TestUnmarshalTNoCrossTarget(t *testing.T) {
	type rec struct {
		Tags map[string]string `qdf:"tags"`
	}
	// Prime: a long batch then a short batch leaves surplus maps in the free-list.
	long := make([]rec, 16)
	for i := range long {
		long[i].Tags = map[string]string{"x": strconv.Itoa(i)}
	}
	dl, _ := Marshal(long, OptBalanced)
	var b []rec
	_ = Unmarshal(dl, &b)
	short := []rec{{Tags: map[string]string{"y": "9"}}}
	ds, _ := Marshal(short, OptBalanced)
	_ = Unmarshal(ds, &b) // surplus maps now linger on the pooled decoder
	// UnmarshalT must clear them and decode cleanly.
	in := []rec{{Tags: map[string]string{"z": "1"}}, {Tags: map[string]string{"w": "2"}}}
	din, _ := Marshal(in, OptBalanced)
	var out []rec
	if err := UnmarshalT(din, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("cross-target leak: got %v want %v", out, in)
	}
}
