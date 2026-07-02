package qdf

import "testing"

// TestMapAnyLossyVecRoundTrip guards the schemaless map[K]any round trip under
// OptLossyVec. A []float32/[]float64 value (>= lossyVecMinElems) held in a
// map[K]any value slot used to be encoded through encodeReflect, which — unlike
// encodeIface — never raised e.ifaceDepth. With ifaceDepth==0 the lossy-vector
// gate fired and emitted a tagColVecLossy (0xFD) / tagVecBatchStruct (0xFE)
// block that decodeAny has no case for, so the value failed to decode with
// ErrBadTag ("unknown tag"). The generator now routes map-any values through
// encodeIface, matching the []any element path. Regression for the map-any gap.
func TestMapAnyLossyVecRoundTrip(t *testing.T) {
	const dim = 64
	vec32 := make([]float32, dim)
	vec64 := make([]float64, dim)
	for i := range vec32 {
		vec32[i] = float32(i) * 0.01
		vec64[i] = float64(i) * 0.01
	}
	opts := OptBalanced | OptLossyVec

	t.Run("map_string_any_f64", func(t *testing.T) {
		in := map[string]any{"v": vec64}
		data, err := Marshal(in, opts)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var out map[string]any
		if err := Unmarshal(data, &out); err != nil {
			t.Fatalf("unmarshal (map[string]any lossy value must round-trip): %v", err)
		}
		if _, ok := out["v"]; !ok {
			t.Fatalf("key v lost: %#v", out)
		}
	})

	t.Run("map_string_any_f32", func(t *testing.T) {
		in := map[string]any{"v": vec32}
		data, err := Marshal(in, opts)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var out map[string]any
		if err := Unmarshal(data, &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
	})

	t.Run("map_int_any", func(t *testing.T) {
		in := map[int]any{7: vec64}
		data, err := Marshal(in, opts)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var out map[int]any
		if err := Unmarshal(data, &out); err != nil {
			t.Fatalf("unmarshal (map[int]any): %v", err)
		}
	})

	t.Run("map_int64_any", func(t *testing.T) {
		in := map[int64]any{9: vec64}
		data, err := Marshal(in, opts)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var out map[int64]any
		if err := Unmarshal(data, &out); err != nil {
			t.Fatalf("unmarshal (map[int64]any): %v", err)
		}
	})

	t.Run("map_uint64_any", func(t *testing.T) {
		in := map[uint64]any{11: vec64}
		data, err := Marshal(in, opts)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var out map[uint64]any
		if err := Unmarshal(data, &out); err != nil {
			t.Fatalf("unmarshal (map[uint64]any): %v", err)
		}
	})

	// Batchable []struct-with-vector held as a map value (0xFE path).
	t.Run("map_string_any_batch_struct", func(t *testing.T) {
		rows := make([]vecOnlyRow, 32)
		for i := range rows {
			rows[i] = vecOnlyRow{Emb: sinVec32(i, 128)}
		}
		in := map[string]any{"rows": rows}
		data, err := Marshal(in, opts)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var out map[string]any
		if err := Unmarshal(data, &out); err != nil {
			t.Fatalf("unmarshal (map value batchable []struct): %v", err)
		}
	})
}
