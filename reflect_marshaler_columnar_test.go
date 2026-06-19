package qdf

import (
	"reflect"
	"sync/atomic"
	"testing"
)

var marshalerColCalls int64

// onlyMarshaler implements ONLY Marshaler (pointer receiver), with all-scalar
// fields so it would otherwise be columnar-eligible.
type onlyMarshaler struct {
	A int64
	B int64
}

func (m *onlyMarshaler) MarshalQDF(dst []byte) ([]byte, error) {
	atomic.AddInt64(&marshalerColCalls, 1)
	e := NewEncoder(Fast)
	e.WriteInt(m.A)
	return append(dst, e.Bytes()...), nil
}

// onlyUnmarshaler implements ONLY Unmarshaler (pointer receiver), all-scalar.
type onlyUnmarshaler struct {
	A int64
	B int64
}

func (m *onlyUnmarshaler) UnmarshalQDF(src []byte) (int, error) { return len(src), nil }

// plainScalarStruct has the same shape but no custom codec — it MUST stay
// columnar-eligible so the guard added for the marshaler cases is narrow.
type plainScalarStruct struct {
	A int64
	B int64
}

// A struct that implements a custom codec in EITHER direction must not have its
// []T columnar-transposed — the transpose replays the structural field layout
// and bypasses MarshalQDF/UnmarshalQDF. A type implementing BOTH directions
// returns early in fillDesc with empty fields (already nil colPlan); these
// single-direction cases are the ones the early return misses.
func TestMarshalerSliceColumnarBypass(t *testing.T) {
	colPlanOf := func(v any) *columnarPlan {
		td, err := descOf(reflect.TypeOf(v))
		if err != nil {
			t.Fatalf("descOf: %v", err)
		}
		return td.colPlan
	}

	if cp := colPlanOf([]onlyMarshaler(nil)); cp != nil {
		t.Errorf("[]onlyMarshaler got a columnar plan — transpose would skip MarshalQDF")
	}
	if cp := colPlanOf([]onlyUnmarshaler(nil)); cp != nil {
		t.Errorf("[]onlyUnmarshaler got a columnar plan — transpose would skip UnmarshalQDF")
	}
	// Guard is narrow: a plain all-scalar struct slice still columnar-transposes.
	if cp := colPlanOf([]plainScalarStruct(nil)); cp == nil {
		t.Errorf("[]plainScalarStruct lost its columnar plan — the marshaler guard is too broad")
	}

	// Behavioral check: a >=16 []onlyMarshaler under OptBalanced invokes
	// MarshalQDF per element instead of columnar-transposing.
	atomic.StoreInt64(&marshalerColCalls, 0)
	vals := make([]onlyMarshaler, 20)
	for i := range vals {
		vals[i] = onlyMarshaler{A: int64(i), B: int64(i * 2)}
	}
	if _, err := Marshal(vals, OptBalanced); err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got := atomic.LoadInt64(&marshalerColCalls); got != 20 {
		t.Fatalf("MarshalQDF called %d times, want 20 (columnar transpose bypassed the custom marshaler)", got)
	}
}
