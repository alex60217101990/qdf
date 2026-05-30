package qdf

import "math/bits"

// Queryable is the set of column element types a predicate can match. Exact
// base types only: Where dispatches on the concrete func(T) bool type once at
// construction, which matches base types but not user-defined named types.
type Queryable interface {
	int | int8 | int16 | int32 | int64 |
		uint | uint8 | uint16 | uint32 | uint64 | uintptr |
		float32 | float64 | string | bool
}

// predTerm is one resolved predicate: a column name, the column kind its
// predicate expects, and exactly one native predicate (the field matching
// want is non-nil). Native predicates take the decoder's wide scratch type so
// evaluation is a direct typed call with zero per-value interface boxing.
type predTerm struct {
	field string
	want  colKind
	pI64  func(int64) bool
	pU64  func(uint64) bool
	pF64  func(float64) bool
	pStr  func(string) bool
	pBool func(bool) bool
}

// QueryOption configures a filtering/projecting Unmarshal. Construct with Where
// or Select. Exactly one of term / selectFields is set.
type QueryOption struct {
	term         *predTerm
	selectFields []string
}

// Where keeps only rows whose field column satisfies pred. T must match the
// column's native kind (any integer width, float32/64, string, or bool); a
// mismatch is reported as ErrTypeMismatch at decode time. Multiple Where
// options are AND-ed. The predicate is called once per row with the native
// value — no interface boxing per value.
func Where[T Queryable](field string, pred func(T) bool) QueryOption {
	t := &predTerm{field: field}
	switch p := any(pred).(type) {
	case func(int) bool:
		t.want, t.pI64 = colKindInt, func(v int64) bool { return p(int(v)) }
	case func(int8) bool:
		t.want, t.pI64 = colKindInt, func(v int64) bool { return p(int8(v)) }
	case func(int16) bool:
		t.want, t.pI64 = colKindInt, func(v int64) bool { return p(int16(v)) }
	case func(int32) bool:
		t.want, t.pI64 = colKindInt, func(v int64) bool { return p(int32(v)) }
	case func(int64) bool:
		t.want, t.pI64 = colKindInt, p
	case func(uint) bool:
		t.want, t.pU64 = colKindUint, func(v uint64) bool { return p(uint(v)) }
	case func(uint8) bool:
		t.want, t.pU64 = colKindUint, func(v uint64) bool { return p(uint8(v)) }
	case func(uint16) bool:
		t.want, t.pU64 = colKindUint, func(v uint64) bool { return p(uint16(v)) }
	case func(uint32) bool:
		t.want, t.pU64 = colKindUint, func(v uint64) bool { return p(uint32(v)) }
	case func(uint64) bool:
		t.want, t.pU64 = colKindUint, p
	case func(uintptr) bool:
		t.want, t.pU64 = colKindUint, func(v uint64) bool { return p(uintptr(v)) }
	case func(float32) bool:
		t.want, t.pF64 = colKindFloat, func(v float64) bool { return p(float32(v)) }
	case func(float64) bool:
		t.want, t.pF64 = colKindFloat, p
	case func(string) bool:
		t.want, t.pStr = colKindString, p
	case func(bool) bool:
		t.want, t.pBool = colKindBool, p
	}
	return QueryOption{term: t}
}

// Select restricts decoding to the named columns, like the fields argument of
// UnmarshalColumns. Without a Select, the output columns are the fields of the
// target struct, matched by name.
func Select(fields ...string) QueryOption {
	return QueryOption{selectFields: append([]string(nil), fields...)}
}

// queryPlan is the resolved set of options for one decode, built from the
// variadic QueryOptions. preds are AND-ed; selectFields (may be nil) projects.
type queryPlan struct {
	preds        []*predTerm
	selectFields []string
}

func buildQueryPlan(opts []QueryOption) *queryPlan {
	qp := &queryPlan{}
	for _, o := range opts {
		switch {
		case o.term != nil:
			qp.preds = append(qp.preds, o.term)
		case o.selectFields != nil:
			qp.selectFields = append(qp.selectFields, o.selectFields...)
		}
	}
	return qp
}

// --- bitset: row-match masks. LSB-first within each uint64 word. ---

func newBitset(n int) []uint64 { return make([]uint64, (n+63)>>6) }

func setBit(b []uint64, i int) { b[i>>6] |= 1 << (uint(i) & 63) }

func getBit(b []uint64, i int) bool { return b[i>>6]&(1<<(uint(i)&63)) != 0 }

// bitsetAnd sets a &= b (a and b same length).
func bitsetAnd(a, b []uint64) {
	for i := range a {
		a[i] &= b[i]
	}
}

func popcount(b []uint64) int {
	n := 0
	for _, w := range b {
		n += bits.OnesCount64(w)
	}
	return n
}

// matchedIndices appends the set-bit indices (< n) of b to dst in order.
func matchedIndices(b []uint64, n int, dst []int) []int {
	for i := range n {
		if getBit(b, i) {
			dst = append(dst, i)
		}
	}
	return dst
}
