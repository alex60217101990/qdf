package qdf

import "testing"

// TestPForCostNeverWorseFloor pins the picker fix: when PFOR wins the codec
// race, pickI64Codec / pickU64Codec must report PFOR's cost as bestCost so the
// narrow-int never-worse floor (encodeSliceInt32 / encodeSliceUint32 vs native
// 32-bit raw) picks the smaller PFOR form. Before the fix bestCost kept the
// pre-PFOR value, so the floor fell back to native raw and bloated the column.
//
// The trigger: a narrow-int column whose bulk packs into a few bits but whose
// min/max span the full 32-bit range, forcing plain FOR as wide as raw while
// PFOR (small base width + a couple of patched exceptions) stays tiny.
func TestPForCostNeverWorseFloor(t *testing.T) {
	const n = 256

	t.Run("uint32", func(t *testing.T) {
		// Bulk in [0,128) plus two MaxUint32-ish outliers → 32-bit range.
		s := make([]uint32, n)
		for i := range s {
			s[i] = uint32(i % 128)
		}
		s[10] = 0xFFFFFFFF
		s[200] = 0xFFFFFF00
		type row struct {
			A []uint32 `qdf:"a"`
		}
		buf, err := Marshal(row{A: s}, OptQPack)
		if err != nil {
			t.Fatal(err)
		}
		// Native uint32-raw would be ~4*n bytes; PFOR must beat it decisively.
		if len(buf) >= 4*n {
			t.Fatalf("wire=%d, native-raw≈%d: PFOR not chosen (stale bestCost floor)", len(buf), 4*n)
		}
		var out row
		if err := Unmarshal(buf, &out); err != nil {
			t.Fatal(err)
		}
		for i := range s {
			if out.A[i] != s[i] {
				t.Fatalf("uint32 [%d] = %d, want %d", i, out.A[i], s[i])
			}
		}
	})

	t.Run("int32", func(t *testing.T) {
		// Bulk near MinInt32 plus a positive outlier → full 32-bit range.
		s := make([]int32, n)
		for i := range s {
			s[i] = int32(-2147483648 + i%128)
		}
		s[10] = 2147483600
		type row struct {
			A []int32 `qdf:"a"`
		}
		buf, err := Marshal(row{A: s}, OptQPack)
		if err != nil {
			t.Fatal(err)
		}
		if len(buf) >= 4*n {
			t.Fatalf("wire=%d, native-raw≈%d: PFOR not chosen (stale bestCost floor)", len(buf), 4*n)
		}
		var out row
		if err := Unmarshal(buf, &out); err != nil {
			t.Fatal(err)
		}
		for i := range s {
			if out.A[i] != s[i] {
				t.Fatalf("int32 [%d] = %d, want %d", i, out.A[i], s[i])
			}
		}
	})
}
