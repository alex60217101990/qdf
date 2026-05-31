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

// QueryOption configures a filtering/projecting Unmarshal. Construct with Where,
// And, Or, Not (predicate tree) or Select (projection). Exactly one of node /
// selectFields is set.
type QueryOption struct {
	node         *condNode
	selectFields []string
}

// condOp is the kind of a predicate-tree node.
type condOp uint8

const (
	condLeaf condOp = iota // single typed predicate (term set)
	condAnd                // all kids true
	condOr                 // any kid true
	condNot                // single kid negated
)

// condNode is one node of the boolean predicate tree. A leaf carries exactly
// one typed predTerm; And/Or carry n kids; Not carries one. err records a
// construction misuse (e.g. a Select passed into a combinator) surfaced at
// buildQueryPlan time.
type condNode struct {
	op   condOp
	term *predTerm
	kids []*condNode
	err  error
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
	return QueryOption{node: &condNode{op: condLeaf, term: t}}
}

// Select restricts decoding to the named columns. Top-level only; nesting a
// Select inside And/Or/Not is reported as ErrUnsupported.
func Select(fields ...string) QueryOption {
	return QueryOption{selectFields: append([]string(nil), fields...)}
}

// combine builds an And/Or node from option kids, flagging any non-predicate
// (Select) kid as an ErrUnsupported misuse.
func combine(op condOp, opts []QueryOption) QueryOption {
	n := &condNode{op: op}
	for _, o := range opts {
		if o.node == nil {
			n.err = &QueryError{Op: "predicate combinator", Err: ErrUnsupported}
			continue
		}
		n.kids = append(n.kids, o.node)
	}
	return QueryOption{node: n}
}

// And keeps rows where every sub-predicate is true. Multiple top-level Where
// options are an implicit And; And is the explicit, nestable form.
func And(opts ...QueryOption) QueryOption { return combine(condAnd, opts) }

// Or keeps rows where at least one sub-predicate is true.
func Or(opts ...QueryOption) QueryOption { return combine(condOr, opts) }

// Not keeps rows where the sub-predicate is not true. Under three-valued NULL
// semantics a nil (absent) nullable row is UNKNOWN, so Not(pred) excludes nil
// rows just as the bare predicate does.
func Not(opt QueryOption) QueryOption {
	n := &condNode{op: condNot}
	if opt.node == nil {
		n.err = &QueryError{Op: "predicate combinator", Err: ErrUnsupported}
	} else {
		n.kids = []*condNode{opt.node}
	}
	return QueryOption{node: n}
}

// queryPlan is the resolved set of options for one decode: a boolean predicate
// tree (root, nil = no filter) and an optional projection.
type queryPlan struct {
	root         *condNode
	selectFields []string
}

// firstCondErr returns the first construction error in the tree, or nil.
func firstCondErr(n *condNode) error {
	if n == nil {
		return nil
	}
	if n.err != nil {
		return n.err
	}
	for _, k := range n.kids {
		if e := firstCondErr(k); e != nil {
			return e
		}
	}
	return nil
}

// collectLeaves appends every leaf node of the tree to dst in pre-order.
func collectLeaves(n *condNode, dst []*condNode) []*condNode {
	if n == nil {
		return dst
	}
	if n.op == condLeaf {
		return append(dst, n)
	}
	for _, k := range n.kids {
		dst = collectLeaves(k, dst)
	}
	return dst
}

func buildQueryPlan(opts []QueryOption) (*queryPlan, error) {
	qp := &queryPlan{}
	var nodes []*condNode
	for _, o := range opts {
		switch {
		case o.node != nil:
			nodes = append(nodes, o.node)
		case o.selectFields != nil:
			qp.selectFields = append(qp.selectFields, o.selectFields...)
		}
	}
	switch len(nodes) {
	case 0:
		// no filter
	case 1:
		qp.root = nodes[0]
	default:
		qp.root = &condNode{op: condAnd, kids: nodes}
	}
	if err := firstCondErr(qp.root); err != nil {
		return nil, err
	}
	return qp, nil
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

// bitsetOr sets a |= b (a and b same length).
func bitsetOr(a, b []uint64) {
	for i := range a {
		a[i] |= b[i]
	}
}

// bitsetAndNot sets a &^= b (a and b same length).
func bitsetAndNot(a, b []uint64) {
	for i := range a {
		a[i] &^= b[i]
	}
}

// notMask returns a fresh complement of m over n rows, with bits >= n cleared
// so popcount and whole-word ops stay meaningful.
func notMask(m []uint64, n int) []uint64 {
	out := make([]uint64, len(m))
	for i := range m {
		out[i] = ^m[i]
	}
	if r := n & 63; r != 0 && len(out) > 0 {
		out[len(out)-1] &= (uint64(1) << uint(r)) - 1
	}
	return out
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

// computeUnknown marks every node whose subtree contains a nullable leaf (a
// source of SQL UNKNOWN). Only such subtrees carry an explicit F mask.
func computeUnknown(n *condNode, cvOf map[*condNode]*colVals) map[*condNode]bool {
	unk := make(map[*condNode]bool)
	var walk func(*condNode) bool
	walk = func(nd *condNode) bool {
		if nd == nil {
			return false
		}
		u := false
		if nd.op == condLeaf {
			u = cvOf[nd] != nil && cvOf[nd].present != nil
		} else {
			for _, k := range nd.kids {
				if walk(k) {
					u = true
				}
			}
		}
		unk[nd] = u
		return u
	}
	walk(n)
	return unk
}

// evalCond returns the T mask (rows where node is TRUE) and, when the subtree
// can produce UNKNOWN (unk[node]), the F mask (rows where node is FALSE); a nil
// F means "no unknowns, F == complement of T". Leaf and And are handled here;
// Or and Not are added in a later task.
func evalCond(node *condNode, n int, cvOf map[*condNode]*colVals, unk map[*condNode]bool) (t, f []uint64) {
	switch node.op {
	case condLeaf:
		return cvOf[node].evalMasks(node.term, n)
	case condAnd:
		return evalAnd(node, n, cvOf, unk)
	default:
		panic("qdf: evalCond: unsupported op")
	}
}

// evalAnd combines kids: T = AND of kid T; F = OR of kid F (or ~kidT when a kid
// has no unknowns). F stays nil when the whole subtree has no unknowns.
func evalAnd(node *condNode, n int, cvOf map[*condNode]*colVals, unk map[*condNode]bool) (t, f []uint64) {
	for i, k := range node.kids {
		kt, kf := evalCond(k, n, cvOf, unk)
		if i == 0 {
			t = append([]uint64(nil), kt...)
		} else {
			bitsetAnd(t, kt)
		}
		if unk[node] {
			if kf == nil {
				kf = notMask(kt, n)
			}
			if f == nil {
				f = append([]uint64(nil), kf...)
			} else {
				bitsetOr(f, kf)
			}
		}
	}
	return t, f
}
