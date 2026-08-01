package qdf

import (
	"math"
	"reflect"

	"github.com/alex60217101990/qdf/internal/bitflag"
)

// Bound-carrying predicates. Unlike Where(func) — whose closure is opaque and so
// cannot drive zone-skip — WhereCmp and WhereRange carry the predicate's
// comparison bounds, so a zone-chunked column (OptZoneMap) can skip zones whose
// [min,max] cannot intersect the bounds without decoding them. They evaluate
// identically to the equivalent Where(func) per row; the bounds are an extra used
// only by the zone-skip fast path. v1 carries bounds for int/uint columns;
// float/string get the per-row predicate only (no skip yet) — still correct.
//
// Strict comparators (GT/LT) record a CONSERVATIVE inclusive bound (e.g. GT(b)
// uses [b, +inf]) so a zone is skipped only when provably empty; the exact strict
// test still runs per row, so the result is exact, just occasionally decoding a
// zone that contributes no rows.

// CmpOp is a comparison operator for WhereCmp, built from three primitive bits —
// "accept less-than", "accept equal", "accept greater-than" — so the named
// operators compose and the per-row test and zone bounds derive directly from the
// bits (no per-operator switch). Combine bits for custom relations, e.g. NE.
type CmpOp uint8

const (
	cmpLT CmpOp = 1 << iota // value <  bound accepted
	cmpEQ                   // value == bound accepted
	cmpGT                   // value >  bound accepted
)

const (
	LT CmpOp = cmpLT         // field <  val
	LE CmpOp = cmpLT | cmpEQ // field <= val
	EQ CmpOp = cmpEQ         // field == val
	GE CmpOp = cmpGT | cmpEQ // field >= val
	GT CmpOp = cmpGT         // field >  val
	NE CmpOp = cmpLT | cmpGT // field != val
)

// Ordered is Queryable minus bool — the kinds that support <,<=,>,>=.
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 | ~string
}

// boundKind classifies a Queryable value into its column kind and the value in
// the widened form the eval path uses. Boxing happens once, at construction. ok
// is false only for a value whose kind is none of int/uint/float/string — which
// the Ordered constraint forbids, so it is a defensive guard, not a normal path.
//
// The concrete type switch is the fast path; a NAMED type (e.g. `type Status
// int32`) does not match it and is resolved by the reflect fallback. Without that
// fallback a named signed-int silently classified as colKindInt(0) with value 0,
// turning WhereCmp/WhereRange into a `field == 0` predicate with no error.
func boundKind(x any) (kind colKind, i int64, u uint64, f float64, s string, ok bool) {
	switch v := x.(type) {
	case int:
		return colKindInt, int64(v), 0, 0, "", true
	case int8:
		return colKindInt, int64(v), 0, 0, "", true
	case int16:
		return colKindInt, int64(v), 0, 0, "", true
	case int32:
		return colKindInt, int64(v), 0, 0, "", true
	case int64:
		return colKindInt, v, 0, 0, "", true
	case uint:
		return colKindUint, 0, uint64(v), 0, "", true
	case uint8:
		return colKindUint, 0, uint64(v), 0, "", true
	case uint16:
		return colKindUint, 0, uint64(v), 0, "", true
	case uint32:
		return colKindUint, 0, uint64(v), 0, "", true
	case uint64:
		return colKindUint, 0, v, 0, "", true
	case uintptr:
		return colKindUint, 0, uint64(v), 0, "", true
	case float32:
		return colKindFloat32, 0, 0, float64(v), "", true
	case float64:
		return colKindFloat, 0, 0, v, "", true
	case string:
		return colKindString, 0, 0, 0, v, true
	}
	// Named type: resolve the underlying kind via reflect.
	switch rv := reflect.ValueOf(x); rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return colKindInt, rv.Int(), 0, 0, "", true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return colKindUint, 0, rv.Uint(), 0, "", true
	case reflect.Float32:
		return colKindFloat32, 0, 0, rv.Float(), "", true
	case reflect.Float64:
		return colKindFloat, 0, 0, rv.Float(), "", true
	case reflect.String:
		return colKindString, 0, 0, 0, rv.String(), true
	}
	return 0, 0, 0, 0, "", false
}

func leaf(t *predTerm) QueryOption { return QueryOption{node: &condNode{op: condLeaf, term: t}} }

// WhereCmp keeps rows where (field op val): one of GE / GT / LE / LT / EQ. It is
// the bound-carrying form of a single comparison, enabling zone-skip on
// OptZoneMap int/uint columns. For a two-sided range use WhereRange.
func WhereCmp[T Ordered](field string, op CmpOp, val T) QueryOption {
	k, vI, vU, vF, vS, ok := boundKind(any(val))
	if !ok {
		return QueryOption{node: &condNode{op: condLeaf, err: &QueryError{Op: "WhereCmp", Field: field, Err: ErrTypeMismatch}}}
	}
	t := &predTerm{field: field, want: k}
	switch k {
	case colKindInt:
		t.pI64 = func(v int64) bool {
			if v < vI {
				return bitflag.Has(op, cmpLT)
			}
			if v > vI {
				return bitflag.Has(op, cmpGT)
			}
			return bitflag.Has(op, cmpEQ)
		}
		// Conservative inclusive bounds: open the low side only if "<" is accepted,
		// the high side only if ">" is accepted; otherwise clamp to val.
		t.loI64, t.hiI64, t.hasBounds = math.MinInt64, math.MaxInt64, true
		if !bitflag.Has(op, cmpLT) {
			t.loI64 = vI
		}
		if !bitflag.Has(op, cmpGT) {
			t.hiI64 = vI
		}
	case colKindUint:
		t.pU64 = func(v uint64) bool {
			if v < vU {
				return bitflag.Has(op, cmpLT)
			}
			if v > vU {
				return bitflag.Has(op, cmpGT)
			}
			return bitflag.Has(op, cmpEQ)
		}
		t.loU64, t.hiU64, t.hasBounds = 0, math.MaxUint64, true
		if !bitflag.Has(op, cmpLT) {
			t.loU64 = vU
		}
		if !bitflag.Has(op, cmpGT) {
			t.hiU64 = vU
		}
	case colKindFloat, colKindFloat32:
		if vF != vF { // NaN query value matches nothing; no zone bounds
			t.pF64 = func(float64) bool { return false }
			break
		}
		t.pF64 = func(v float64) bool {
			if v != v { // NaN matches nothing
				return false
			}
			if v < vF {
				return bitflag.Has(op, cmpLT)
			}
			if v > vF {
				return bitflag.Has(op, cmpGT)
			}
			return bitflag.Has(op, cmpEQ)
		}
		// Float bounds enable zone-skip on float64 zone-chunked columns. An all-NaN
		// zone stores an empty [+Inf,-Inf] interval, so it is skipped (NaN never
		// matches), and a partly-NaN zone is governed by its finite min/max.
		t.loF64, t.hiF64, t.hasBounds = math.Inf(-1), math.Inf(1), true
		if !bitflag.Has(op, cmpLT) {
			t.loF64 = vF
		}
		if !bitflag.Has(op, cmpGT) {
			t.hiF64 = vF
		}
	case colKindString:
		t.pStr = func(v string) bool {
			if v < vS {
				return bitflag.Has(op, cmpLT)
			}
			if v > vS {
				return bitflag.Has(op, cmpGT)
			}
			return bitflag.Has(op, cmpEQ)
		}
	}
	return leaf(t)
}

// WhereRange keeps rows where lo <= field <= hi (inclusive). Bound-carrying:
// enables zone-skip on OptZoneMap int/uint columns.
func WhereRange[T Ordered](field string, lo, hi T) QueryOption {
	k, loI, loU, loF, loS, ok := boundKind(any(lo))
	_, hiI, hiU, hiF, hiS, _ := boundKind(any(hi))
	if !ok {
		return QueryOption{node: &condNode{op: condLeaf, err: &QueryError{Op: "WhereRange", Field: field, Err: ErrTypeMismatch}}}
	}
	t := &predTerm{field: field, want: k}
	switch k {
	case colKindInt:
		t.pI64 = func(v int64) bool { return v >= loI && v <= hiI }
		t.loI64, t.hiI64, t.hasBounds = loI, hiI, true
	case colKindUint:
		t.pU64 = func(v uint64) bool { return v >= loU && v <= hiU }
		t.loU64, t.hiU64, t.hasBounds = loU, hiU, true
	case colKindFloat, colKindFloat32:
		t.pF64 = func(v float64) bool { return v >= loF && v <= hiF } // NaN → false
		t.loF64, t.hiF64, t.hasBounds = loF, hiF, true
	case colKindString:
		t.pStr = func(v string) bool { return v >= loS && v <= hiS }
	}
	return leaf(t)
}
