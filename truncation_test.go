package qdf

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// Truncation matrix: for every representative encoded payload, decode
// every prefix `payload[:i]` for i in [0, len(payload)). Each must
// return an error rather than panic. This is the deterministic
// counterpart to FuzzDecoder_NeverPanics — same property,
// reproducible.

// Single-byte mutation matrix: for every representative payload, flip
// each byte to a small set of "bad" alternatives and assert the
// decoder either accepts the change (rare, only for value bytes) or
// returns a clean error. Never panic.

func truncationPayloads(t *testing.T) map[string][]byte {
	t.Helper()
	type small struct {
		ID   int    `qdf:"id"`
		Name string `qdf:"name"`
	}
	type vec struct {
		Vals []float64 `qdf:"vals"`
	}
	type batch struct {
		Bools []bool   `qdf:"bools"`
		IDs   []uint64 `qdf:"ids"`
		Tags  []string `qdf:"tags"`
		When  int64    `qdf:"when"`
	}

	payloads := map[string]any{
		"primitive":   small{ID: 42, Name: "alice"},
		"vec_qpack":   vec{Vals: []float64{1, 2, 3, 4, 5, 6, 7, 8}},
		"batch_dense": batch{Bools: []bool{true, false, true}, IDs: []uint64{1700000000, 1700000001}, Tags: []string{"prod", "prod"}, When: time.Now().Unix()},
		"map_string":  map[string]string{"a": "1", "b": "2", "c": "3"},
		"slice_int":   []int{0, 1, 127, -1, 32767, -32768, 1 << 30},
	}

	encoders := map[string]Options{
		"fast":  OptSpeed,
		"qpack": OptQPack,
		"dense": OptBalanced,
	}

	out := map[string][]byte{}
	for name, v := range payloads {
		for dialect, opts := range encoders {
			buf, err := Marshal(v, opts)
			if err != nil {
				t.Fatalf("encode %s/%s: %v", name, dialect, err)
			}
			out[name+"."+dialect] = buf
		}
	}
	return out
}

// decodeIntoEvery dispatches the buffer at out-pointers of every
// commonly-encountered destination kind so the matrix exercises every
// reflect path, not just the one matching the original type.
func decodeIntoEvery(t *testing.T, buf []byte) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("decoder panicked: %v", r)
		}
	}()
	// Each Unmarshal is independent; any return error is fine, panic is not.
	var s string
	_ = Unmarshal(buf, &s)
	var i int
	_ = Unmarshal(buf, &i)
	var f float64
	_ = Unmarshal(buf, &f)
	var m map[string]any
	_ = Unmarshal(buf, &m)
	var slc []any
	_ = Unmarshal(buf, &slc)
	type generic struct {
		A int       `qdf:"a"`
		B string    `qdf:"b"`
		C []float64 `qdf:"c"`
	}
	var g generic
	_ = Unmarshal(buf, &g)
}

func TestTruncation_NeverPanics(t *testing.T) {
	for name, buf := range truncationPayloads(t) {
		for i := range buf {
			t.Run(fmt.Sprintf("%s/prefix=%d", name, i), func(t *testing.T) {
				decodeIntoEvery(t, buf[:i])
			})
		}
	}
}

func TestByteMutation_NeverPanics(t *testing.T) {
	// For each byte position, flip the byte to a known-bad value and
	// to a random nearby value. The decoder must handle every mutation
	// gracefully. Combinatorial explosion is kept in check by sampling
	// every 4th byte beyond the header so a 200-byte payload stays
	// under ~50 t.Runs.
	mutants := []byte{0x00, 0xFF, 0x80, 0x55, 0xAA, tagPackBool, tagPackFor, tagStateRepeat, tagPackGorilla}
	for name, original := range truncationPayloads(t) {
		buf := append([]byte(nil), original...)
		step := 1
		if len(buf) > 16 {
			step = 4
		}
		for i := 0; i < len(buf); i += step {
			orig := buf[i]
			for _, m := range mutants {
				if m == orig {
					continue
				}
				buf[i] = m
				t.Run(fmt.Sprintf("%s/pos=%d/mut=%02x", name, i, m), func(t *testing.T) {
					decodeIntoEvery(t, buf)
				})
			}
			buf[i] = orig
		}
	}
}

func TestHeader_RejectsAllBadVariants(t *testing.T) {
	// Every malformed 5-byte header must yield a typed error, not a
	// panic and not a silent accept.
	body := []byte{0xC0} // tagNil — a trivially valid body
	cases := map[string][]byte{
		"empty":          {},
		"one_byte":       {'Q'},
		"two_bytes":      {'Q', 'D'},
		"three_bytes":    {'Q', 'D', 'F'},
		"four_bytes":     {'Q', 'D', 'F', 0x01},
		"bad_magic":      {'X', 'Y', 'Z', 0x01, 0x00},
		"bad_version_0":  {'Q', 'D', 'F', 0x00, 0x00},
		"bad_version_ff": {'Q', 'D', 'F', 0xFF, 0x00},
	}
	for name, header := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("decoder panicked on header=%q: %v", name, r)
				}
			}()
			input := append(append([]byte(nil), header...), body...)
			var v any
			err := Unmarshal(input, &v)
			if err == nil {
				t.Fatalf("expected error on header=%q", name)
			}
		})
	}
}

func TestSkip_NeverPanics(t *testing.T) {
	// Skip() must be panic-free on every truncated/mutated prefix the
	// public Unmarshal path is panic-free on. This exercises the skip
	// dispatch table directly so a missing case in Skip cannot lurk
	// behind a "field was unknown" code path.
	for name, buf := range truncationPayloads(t) {
		for i := range buf {
			t.Run(fmt.Sprintf("%s/prefix=%d", name, i), func(t *testing.T) {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("Skip panicked: %v", r)
					}
				}()
				d := NewDecoderOnBuf(buf[:i])
				// Either header parse fails, or Skip returns an error; both fine.
				_ = d.Skip()
			})
		}
	}
}

// Sanity check that the truncation table actually exercises a wide
// range of tag bytes — if every payload turns out to use the same six
// tags, the coverage claim is weaker than it looks.
func TestTruncation_TagCoverage(t *testing.T) {
	seen := map[byte]struct{}{}
	for _, buf := range truncationPayloads(t) {
		for _, b := range buf {
			seen[b] = struct{}{}
		}
	}
	if len(seen) < 12 {
		got := make([]string, 0, len(seen))
		for b := range seen {
			got = append(got, fmt.Sprintf("%02x", b))
		}
		t.Fatalf("truncation matrix touches only %d distinct bytes: %s", len(seen), strings.Join(got, ","))
	}
}
