package qdf

import (
	"math"
	"testing"
)

// f32NaNBits are float32 values whose bit patterns are NOT preserved by a
// float32→float64→float32 round trip: signaling NaNs (quiet bit clear) and
// quiet NaNs with custom payloads. The old columnar path widened f32 columns
// to a []float64 column, canonicalizing these. A correct codec preserves the
// exact 32 bits.
var f32NaNBits = []uint32{
	0x7f800001, // sNaN, payload 1
	0xff800001, // negative sNaN
	0x7fbfffff, // sNaN, max payload
	0x7fc00000, // canonical qNaN
	0x7fc00001, // qNaN, payload 1
	0xffaaaaaa, // negative qNaN, custom payload
	0x7f7fffff, // max finite (control)
	0x00000001, // smallest subnormal (control)
	0x80000000, // -0.0 (control)
}

type f32Row struct {
	ID  int64   `qdf:"id"`
	F32 float32 `qdf:"f32"`
	S   string  `qdf:"s"`
}

type f32NullRow struct {
	ID  int64    `qdf:"id"`
	F32 *float32 `qdf:"f32"`
	S   string   `qdf:"s"`
}

func f32(bits uint32) float32 { return math.Float32frombits(bits) }

// TestColumnar_Float32_NaNBitsPreserved pins exact-bit round-trip of float32
// columns under the columnar path (≥16 rows), across option bundles, for the
// typed decode path.
func TestColumnar_Float32_NaNBitsPreserved(t *testing.T) {
	const n = 24 // ≥16 ⇒ columnar transpose fires
	rows := make([]f32Row, n)
	for i := range rows {
		rows[i] = f32Row{ID: int64(i), F32: f32(f32NaNBits[i%len(f32NaNBits)]), S: "row"}
	}
	for _, opts := range []Options{OptBalanced, OptCompression, OptBalanced | OptColumnIndex, OptCompression | OptGorillaFloat} {
		buf, err := Marshal(rows, opts)
		if err != nil {
			t.Fatalf("opts=%d Marshal: %v", opts, err)
		}
		var out []f32Row
		if err := Unmarshal(buf, &out); err != nil {
			t.Fatalf("opts=%d Unmarshal: %v", opts, err)
		}
		if len(out) != n {
			t.Fatalf("opts=%d len=%d want %d", opts, len(out), n)
		}
		for i := range out {
			got := math.Float32bits(out[i].F32)
			want := math.Float32bits(rows[i].F32)
			if got != want {
				t.Fatalf("opts=%d row %d: f32 bits %#08x != %#08x", opts, i, got, want)
			}
			if out[i].ID != rows[i].ID || out[i].S != rows[i].S {
				t.Fatalf("opts=%d row %d: other fields corrupted: %+v", opts, i, out[i])
			}
		}
	}
}

// TestColumnar_Float32_NaNBits_AnyPath pins exact-bit round-trip through the
// schemaless any decode path (decode into []map[string]any).
func TestColumnar_Float32_NaNBits_AnyPath(t *testing.T) {
	const n = 20
	rows := make([]f32Row, n)
	for i := range rows {
		rows[i] = f32Row{ID: int64(i), F32: f32(f32NaNBits[i%len(f32NaNBits)]), S: "x"}
	}
	buf, err := Marshal(rows, OptBalanced)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var out []map[string]any
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatalf("Unmarshal any: %v", err)
	}
	for i := range out {
		v, ok := out[i]["f32"].(float32)
		if !ok {
			t.Fatalf("row %d: f32 not boxed as float32: %T", i, out[i]["f32"])
		}
		if got, want := math.Float32bits(v), math.Float32bits(rows[i].F32); got != want {
			t.Fatalf("any row %d: f32 bits %#08x != %#08x", i, got, want)
		}
	}
}

// TestColumnar_Float32_Nullable pins exact-bit round-trip for an optional
// (*float32) column mixing nil, NaN payloads, and normal values.
func TestColumnar_Float32_Nullable(t *testing.T) {
	const n = 24
	rows := make([]f32NullRow, n)
	for i := range rows {
		rows[i] = f32NullRow{ID: int64(i), S: "n"}
		if i%3 != 0 { // leave every third nil
			v := f32(f32NaNBits[i%len(f32NaNBits)])
			rows[i].F32 = &v
		}
	}
	for _, opts := range []Options{OptBalanced, OptCompression} {
		buf, err := Marshal(rows, opts)
		if err != nil {
			t.Fatalf("opts=%d Marshal: %v", opts, err)
		}
		var out []f32NullRow
		if err := Unmarshal(buf, &out); err != nil {
			t.Fatalf("opts=%d Unmarshal: %v", opts, err)
		}
		for i := range out {
			if (rows[i].F32 == nil) != (out[i].F32 == nil) {
				t.Fatalf("opts=%d row %d: nil mismatch want nil=%v got nil=%v", opts, i, rows[i].F32 == nil, out[i].F32 == nil)
			}
			if rows[i].F32 != nil {
				if got, want := math.Float32bits(*out[i].F32), math.Float32bits(*rows[i].F32); got != want {
					t.Fatalf("opts=%d row %d: *f32 bits %#08x != %#08x", opts, i, got, want)
				}
			}
		}
	}
}

// TestColumnar_Float32_Query pins that predicate pushdown still works on a
// float32 column after the codec change.
func TestColumnar_Float32_Query(t *testing.T) {
	const n = 32
	rows := make([]f32Row, n)
	for i := range rows {
		rows[i] = f32Row{ID: int64(i), F32: float32(i) + 0.5, S: "q"}
	}
	buf, err := Marshal(rows, OptBalanced|OptColumnIndex)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var out []f32Row
	if err := Unmarshal(buf, &out, Where("f32", func(v float32) bool { return v > 10 })); err != nil {
		t.Fatalf("Unmarshal query: %v", err)
	}
	for i := range out {
		if out[i].F32 <= 10 {
			t.Fatalf("query returned row with f32=%v (<=10)", out[i].F32)
		}
	}
	// rows with i+0.5 > 10 ⇒ i >= 10 ⇒ 22 rows
	if len(out) != 22 {
		t.Fatalf("query matched %d rows, want 22", len(out))
	}
}
