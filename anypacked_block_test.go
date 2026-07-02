package qdf

import "testing"

// blockRegimeI64 builds an int64 slice that triggers the per-block adaptive
// codec (tagPackBlock 0xF0): alternating constant and wide-range 256-elem
// blocks so block-adaptive beats any single whole-column codec.
func blockRegimeI64() []int64 {
	const nb, bs = 8, 256
	s := make([]int64, 0, nb*bs)
	for b := range nb {
		for i := range bs {
			if b%2 == 0 {
				s = append(s, 0)
			} else {
				s = append(s, int64((b*bs+i)*1_000_003%9_000_000))
			}
		}
	}
	return s
}

// TestAnyPackBlockRoundTrip guards the schemaless round trip for a numeric
// slice encoded with the per-block adaptive codec (tagPackBlock 0xF0). decodeAny
// had cases for tagPackBool and tagPackRaw..tagPackALP but none for tagPackBlock,
// so a []int64/[]uint64 boxed in an any / map value / struct any-field that
// happened to pick the block codec under OptBalanced failed to decode with
// ErrBadTag ("unknown tag"). Same bug-class as the map-any lossy-vec gap: every
// tag reachable through a schemaless position must be decodeAny-readable.
func TestAnyPackBlockRoundTrip(t *testing.T) {
	s := blockRegimeI64()
	u := make([]uint64, len(s))
	for i, v := range s {
		u[i] = uint64(v)
	}

	t.Run("toplevel_any_int64", func(t *testing.T) {
		data, err := Marshal(any(s), OptBalanced)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if got := data[5]; got != tagPackBlock {
			t.Skipf("codec picked 0x%02X, not tagPackBlock — regime did not trigger block path", got)
		}
		var out any
		if err := Unmarshal(data, &out); err != nil {
			t.Fatalf("unmarshal any([]int64) with tagPackBlock: %v", err)
		}
	})

	t.Run("toplevel_any_uint64", func(t *testing.T) {
		data, err := Marshal(any(u), OptBalanced)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var out any
		if err := Unmarshal(data, &out); err != nil {
			t.Fatalf("unmarshal any([]uint64): %v", err)
		}
	})

	t.Run("struct_any_field", func(t *testing.T) {
		type S struct{ Payload any }
		data, err := Marshal(S{Payload: s}, OptBalanced)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var out S
		if err := Unmarshal(data, &out); err != nil {
			t.Fatalf("unmarshal struct any-field: %v", err)
		}
	})

	t.Run("map_string_any", func(t *testing.T) {
		data, err := Marshal(map[string]any{"nums": s}, OptBalanced)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var out map[string]any
		if err := Unmarshal(data, &out); err != nil {
			t.Fatalf("unmarshal map[string]any: %v", err)
		}
	})
}
