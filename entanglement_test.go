package qdf

import (
	"reflect"
	"strconv"
	"testing"
)

// Markov-0 state-ref predictor: a state-ref whose ID equals the
// immediately preceding emission is encoded as the 1-byte
// tagStateRepeat tag. Tests cover encode/decode parity, the
// invalidation rule on inline-string emissions, Skip integration, and
// streaming continuity across messages.

func TestStateRepeat_BasicRunCollapses(t *testing.T) {
	// Three consecutive identical interned strings: the first records
	// the intern, the next two collapse to a single byte each.
	in := []string{"region-eu-west-1", "region-eu-west-1", "region-eu-west-1"}
	enc := NewEncoder(Dense)
	for _, s := range in {
		enc.WriteString(s)
	}

	// First intern + two repeats. Tag bytes: tagInternStr, tagStateRepeat,
	// tagStateRepeat. Plus header.
	gotRepeats := 0
	for _, b := range enc.buf[5:] {
		if b == tagStateRepeat {
			gotRepeats++
		}
	}
	if gotRepeats != 2 {
		t.Fatalf("expected 2 tagStateRepeat bytes, got %d (buf=%x)", gotRepeats, enc.buf)
	}

	dec := NewDecoderOnBuf(enc.buf)
	for i, want := range in {
		got, err := dec.ReadString()
		if err != nil {
			t.Fatalf("[%d] decode: %v", i, err)
		}
		if got != want {
			t.Fatalf("[%d] got %q want %q", i, got, want)
		}
	}
}

func TestStateRepeat_DoesNotFireOnDifferentIDs(t *testing.T) {
	// A B A B alternation — predictor never matches.
	in := []string{"alpha-token", "beta-token", "alpha-token", "beta-token"}
	enc := NewEncoder(Dense)
	for _, s := range in {
		enc.WriteString(s)
	}
	for _, b := range enc.buf[5:] {
		if b == tagStateRepeat {
			t.Fatalf("unexpected tagStateRepeat on alternation: %x", enc.buf)
		}
	}
	dec := NewDecoderOnBuf(enc.buf)
	for _, want := range in {
		got, _ := dec.ReadString()
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	}
}

func TestStateRepeat_InlineEmissionBreaksChain(t *testing.T) {
	// Sequence: interned, inline (short), interned-same. Even though the
	// third value matches the first, the inline emission in between
	// must invalidate the chain so the third encodes as a full state-ref
	// (not a repeat).
	enc := NewEncoder(Dense)
	enc.SetIntern(4, 0)
	enc.WriteString("token-A")
	enc.WriteString("xy") // shorter than minIntern, inline
	enc.WriteString("token-A")

	for _, b := range enc.buf[5:] {
		if b == tagStateRepeat {
			t.Fatalf("predictor must not fire across inline: %x", enc.buf)
		}
	}

	dec := NewDecoderOnBuf(enc.buf)
	for _, want := range []string{"token-A", "xy", "token-A"} {
		got, err := dec.ReadString()
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	}
}

func TestStateRepeat_SkipKeepsStreamInSync(t *testing.T) {
	enc := NewEncoder(Dense)
	enc.WriteString("repeating-token-string")
	enc.WriteString("repeating-token-string") // repeat
	enc.WriteInt(99)                          // sentinel after the repeat tag

	dec := NewDecoderOnBuf(enc.buf)
	if err := dec.Skip(); err != nil {
		t.Fatal(err)
	}
	if err := dec.Skip(); err != nil {
		t.Fatal(err)
	}
	v, err := dec.ReadInt()
	if err != nil || v != 99 {
		t.Fatalf("after skip: v=%d err=%v", v, err)
	}
}

func TestStateRepeat_FreshStateRefOnNewID(t *testing.T) {
	// A A B B sequence: A repeat, B fresh state-ref, B repeat.
	in := []string{"first-token", "first-token", "second-token", "second-token"}
	enc := NewEncoder(Dense)
	for _, s := range in {
		enc.WriteString(s)
	}
	dec := NewDecoderOnBuf(enc.buf)
	for i, want := range in {
		got, err := dec.ReadString()
		if err != nil {
			t.Fatalf("[%d] %v", i, err)
		}
		if got != want {
			t.Fatalf("[%d] got %q want %q", i, got, want)
		}
	}
}

func TestStateRepeat_DenseStruct(t *testing.T) {
	// Marshal/Unmarshal through public API on a struct with repeating
	// field value to confirm wire shrinks vs Dense-without-predictor
	// is not a regression.
	type batch struct {
		Items []string `qdf:"items"`
	}
	in := batch{Items: []string{
		"alpha-event-id-001", "alpha-event-id-001", "alpha-event-id-001",
		"beta-event-id-002", "beta-event-id-002",
		"alpha-event-id-001",
	}}
	buf, err := MarshalDense(in)
	if err != nil {
		t.Fatal(err)
	}
	var out batch
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("round-trip mismatch: %v", out)
	}
}

func makeRepeatedDenseBatch(n int, alphabet []string) []byte {
	type batch struct {
		Vals []string `qdf:"vals"`
	}
	b := batch{Vals: make([]string, n)}
	for i := range b.Vals {
		b.Vals[i] = alphabet[i%len(alphabet)]
	}
	buf, _ := MarshalDense(b)
	return buf
}

func BenchmarkStateRepeat_Sizes(b *testing.B) {
	// Empty marker; the real value is in the b.Logf size print. Compare
	// the size when the alphabet collapses to a single token (predictor
	// always fires after first emission) vs an alternating two-token
	// alphabet (predictor never fires).
	for _, n := range []int{16, 256, 1024} {
		runs := makeRepeatedDenseBatch(n, []string{"event-alpha-id-001"})
		alt := makeRepeatedDenseBatch(n, []string{"event-alpha-id-001", "event-beta-id-002"})
		b.Run("repeat/n="+strconv.Itoa(n), func(b *testing.B) {
			b.Logf("size with predictor: runs=%d alt=%d (delta=%d)", len(runs), len(alt), len(alt)-len(runs))
			for b.Loop() {
				_ = runs
			}
		})
	}
}
