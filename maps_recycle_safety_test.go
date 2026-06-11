package qdf

import (
	"bytes"
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

// TestStreamMapRecycleFrames checks (a) intra-target streaming recycle works
// across frames into the SAME reused target, and (b) frames into DIFFERENT
// targets don't leak a recycled map across frames (the per-frame clear).
func TestStreamMapRecycleFrames(t *testing.T) {
	type rec struct {
		Tags map[string]string `qdf:"tags"`
	}
	frames := [][]rec{
		{{Tags: map[string]string{"a": "1", "b": "2"}}},
		{{Tags: map[string]string{"c": "3"}}}, // fewer keys → would expose stale
		{{Tags: map[string]string{"d": "4", "e": "5"}}},
	}
	var buf bytes.Buffer
	enc := NewStreamEncoder(&buf, Dense)
	for _, f := range frames {
		if err := enc.Encode(f); err != nil {
			t.Fatal(err)
		}
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}
	// (a) reuse the SAME target across frames.
	dec := NewStreamDecoder(bytes.NewReader(buf.Bytes()))
	var out []rec
	for i, want := range frames {
		if err := dec.Decode(&out); err != nil {
			t.Fatalf("frame %d: %v", i, err)
		}
		if !reflect.DeepEqual(want, out) {
			t.Fatalf("frame %d (reused target): got %v want %v", i, out, want)
		}
	}
	// (b) fresh target per frame.
	dec2 := NewStreamDecoder(bytes.NewReader(buf.Bytes()))
	for i, want := range frames {
		var fresh []rec
		if err := dec2.Decode(&fresh); err != nil {
			t.Fatalf("frame %d fresh: %v", i, err)
		}
		if !reflect.DeepEqual(want, fresh) {
			t.Fatalf("frame %d (fresh target): got %v want %v", i, fresh, want)
		}
	}
}
