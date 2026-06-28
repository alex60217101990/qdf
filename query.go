package qdf

import (
	"math/bits"
	"slices"
)

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
	pI64  func(int64) bool
	pU64  func(uint64) bool
	pF64  func(float64) bool
	pStr  func(string) bool
	pBool func(bool) bool
	field string
	// Optional comparison bounds, set by the WhereRange/GE/GT/LE/LT/Eq
	// constructors so a zone-chunked column can skip zones whose [min,max] cannot
	// intersect [loI64,hiI64] (int) / [loU64,hiU64] (uint). hasBounds is false for
	// the opaque Where(func) path (no zone-skip) and for float/string in v1.
	loI64, hiI64 int64
	loU64, hiU64 uint64
	loF64, hiF64 float64
	hasBounds    bool
	want         colKind // 1-byte tail: kept last to avoid padding before the pointers
}

// QueryOption configures a filtering/projecting Unmarshal. Construct with Where,
// And, Or, Not (predicate tree) or Select (projection). Exactly one of node /
// selectFields is set.
type QueryOption struct {
	node         *condNode
	arena        *Arena
	selectFields []string
	noCopy       bool
}

// WithArena makes the decode pack copied inline string values into a, instead
// of allocating one string per field — near-zero allocations across an epoch of
// decodes (see Arena). The decoded strings alias a's memory and follow Arena's
// lifetime contract. Ignored together with WithNoCopy (aliasing already skips
// the copy). Composes with predicates / Select / WithNoCopy.
func WithArena(a *Arena) QueryOption { return QueryOption{arena: a} }

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
	term *predTerm
	err  error
	kids []*condNode
	op   condOp // 1-byte tail: kept last to avoid padding before the pointer fields
}

// Where keeps only rows whose field column satisfies pred. T must match the
// column's native kind (any integer width, float32/64, string, or bool); a
// mismatch is reported as ErrTypeMismatch at decode time. If field is absent
// from the wire the call returns ErrFieldNotFound. Multiple Where options are
// AND-ed. The predicate is called once per row with the native value — no
// interface boxing per value.
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
		// float32 columns carry colKindFloat32 (raw bits); the eval path widens
		// each f32 back to float64 before this predicate narrows it again.
		t.want, t.pF64 = colKindFloat32, func(v float64) bool { return p(float32(v)) }
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

// WithNoCopy makes the decode return string and []byte values that ALIAS the
// input buffer instead of copying them. On string-heavy payloads this is ~2x
// faster with near-zero allocations.
//
// DANGER — lifetime contract: the returned values are valid ONLY while the
// input buffer passed to Unmarshal stays alive and is never modified or reused.
// Do NOT use WithNoCopy when the input may be recycled, mutated, or freed before
// you finish reading the decoded values — e.g. a pooled HTTP request body. The
// decoded strings would silently become garbage. This is a manual-memory
// use-after-free, not a data race: the race detector will NOT catch it.
//
// Safe for caller-owned, immutable input such as an mmap or a file read fully
// into memory.
//
// Also safe — and the common zero-copy high-throughput pattern — when the input
// buffer is allocated FRESH per message and never pooled or overwritten: e.g.
// io.ReadAll(body), or a make([]byte, n) you read one message into and do not
// reuse. You do NOT have to track the buffer's lifetime manually: the aliasing
// string/[]byte headers in the decoded value keep the backing buffer alive
// through the garbage collector (Go marks the whole allocation from an interior
// pointer), so the buffer lives exactly as long as the values that point into
// it. The hazard is exclusively buffer REUSE (a sync.Pool buffer returned after
// the handler, a scratch slice you overwrite on the next read) or mutation —
// not lifetime. If every decode reads into its own fresh slice, WithNoCopy is
// safe and gives the full ~2x / near-zero-alloc win for free.
//
// Scope: WithNoCopy affects the reflect decode path. A type that implements
// Unmarshaler (including codegen-generated UnmarshalQDF) decodes through its own
// decoder via the byte-only UnmarshalQDF(data) interface, which cannot inherit
// this flag, so such types still copy. Use SetNoCopy on a Decoder/StreamDecoder
// you drive directly if you need zero-copy there.
func WithNoCopy() QueryOption { return QueryOption{noCopy: true} }

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
	arena        *Arena
	selectFields []string
	noCopy       bool
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

func buildQueryPlan(opts []QueryOption) (*queryPlan, error) {
	qp := &queryPlan{}
	var nodes []*condNode
	for _, o := range opts {
		if o.noCopy {
			qp.noCopy = true
		}
		if o.arena != nil {
			qp.arena = o.arena
		}
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
	qp.root = simplifyCond(qp.root)
	if err := firstCondErr(qp.root); err != nil {
		return nil, err
	}
	return qp, nil
}

// simplifyCond rewrites the tree into an equivalent smaller form: it folds
// double negation, flattens nested same-op And/Or, unwraps single-child
// And/Or, and dedups identical sibling leaves (same field + same predicate
// closure, keyed by predTerm pointer). Purely an optimisation — semantics are
// unchanged.
func simplifyCond(n *condNode) *condNode {
	if n == nil || n.op == condLeaf {
		return n
	}
	if n.op == condNot {
		// An err-bearing Not built from a non-predicate option (e.g. Not(Select))
		// carries no kid; keep it verbatim so firstCondErr surfaces its error.
		// Indexing kids[0] here would panic.
		if len(n.kids) == 0 {
			return n
		}
		c := simplifyCond(n.kids[0])
		if c.op == condNot && len(c.kids) > 0 { // Not(Not(x)) -> x
			return c.kids[0]
		}
		return &condNode{op: condNot, kids: []*condNode{c}, err: n.err}
	}
	// And / Or: simplify kids, flatten same-op, dedup sibling leaves.
	var kids []*condNode
	err := n.err
	for _, k := range n.kids {
		k = simplifyCond(k)
		if k.op == n.op { // flatten associative same-op
			if k.err != nil && err == nil {
				err = k.err // flattening drops k itself, so carry its error up
			}
			kids = append(kids, k.kids...)
			continue
		}
		kids = append(kids, k)
	}
	seen := make(map[*predTerm]bool)
	out := kids[:0]
	for _, k := range kids {
		if k.op == condLeaf {
			if seen[k.term] {
				continue
			}
			seen[k.term] = true
		}
		out = append(out, k)
	}
	if len(out) == 1 && err == nil { // single-child And/Or -> the child (only when no error on this node)
		return out[0]
	}
	return &condNode{op: n.op, kids: out, err: err}
}

// --- bitset: row-match masks. LSB-first within each uint64 word. ---

func newBitset(n int) []uint64 { return make([]uint64, (n+63)>>6) }

// fullBitset returns an n-row bitset with every valid bit set (the "match all"
// / AND identity), bits >= n cleared. Word-fill, not a per-row setBit loop.
func fullBitset(n int) []uint64 {
	b := newBitset(n)
	for i := range b {
		b[i] = ^uint64(0)
	}
	if r := n & 63; r != 0 && len(b) > 0 {
		b[len(b)-1] = (uint64(1) << uint(r)) - 1
	}
	return b
}

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

// matchedIndices appends the set-bit indices (< n) of b to dst in order. It
// enumerates per word via bits.TrailingZeros64, skipping whole zero words, so a
// sparse (selective) result mask costs ~one append per match instead of n
// per-bit tests — the common case for a selective columnar-query predicate.
// Bit layout is LSB-first (see getBit), so the lowest set bit is the lowest
// index and the emitted order is ascending, identical to the per-bit scan.
func matchedIndices(b []uint64, n int, dst []int) []int {
	full := n >> 6 // count of words wholly in range [0, n)
	for w := range full {
		x := b[w]
		base := w << 6
		for x != 0 {
			dst = append(dst, base+bits.TrailingZeros64(x))
			x &= x - 1 // clear lowest set bit
		}
	}
	// Tail word: keep only the bits below n.
	if r := n & 63; r != 0 && full < len(b) {
		x := b[full] & ((uint64(1) << uint(r)) - 1)
		base := full << 6
		for x != 0 {
			dst = append(dst, base+bits.TrailingZeros64(x))
			x &= x - 1
		}
	}
	return dst
}

// cnode is a flattened predicate-tree node with a dense id. The flat form is
// built per decode (local — it never mutates the shared QueryOption tree, so
// the same tree stays safe to reuse across concurrent Unmarshal calls) and is
// indexed by int, avoiding the per-node allocations and pointer hashing a
// map[*condNode]… would cost on a small tree.
type cnode struct {
	term     *predTerm // leaf only
	cv       *colVals  // leaf only: decoded column values (nil until bound)
	precompT []uint64  // leaf only: precomputed T mask from zone-skip (nil = eval normally)
	kids     []int     // child ids (And/Or: n; Not: 1)
	col      int       // leaf only: resolved wire column index (-1 until resolved)
	op       condOp    // 1-byte tails kept last to avoid padding before the pointers above
	unk      bool      // subtree can produce SQL UNKNOWN (contains a nullable leaf)
}

// flattenCond linearises the tree in pre-order (a parent always precedes its
// descendants), returning the flat slice; the root is index 0 when non-empty.
func flattenCond(root *condNode) []cnode {
	if root == nil {
		return nil
	}
	var flat []cnode
	var walk func(*condNode) int
	walk = func(n *condNode) int {
		id := len(flat)
		flat = append(flat, cnode{op: n.op, term: n.term, col: -1})
		if len(n.kids) > 0 {
			kids := make([]int, len(n.kids))
			for i, k := range n.kids {
				kids[i] = walk(k)
			}
			flat[id].kids = kids
		}
		return id
	}
	walk(root)
	return flat
}

// singleBoundedLeafForCol returns the lone leaf resolved to wire column c when
// EXACTLY ONE leaf references it and that leaf carries comparison bounds — the
// case zone-skip handles. Multiple leaves on one column are declined (nil): each
// leaf's bounds are decoded independently (union), so two complementary
// half-ranges — e.g. WhereGE+WhereLE — would union to the whole domain and skip
// nothing; expressing a range as a single WhereRange keeps the skip. Returns nil
// for zero / multiple / unbounded leaves.
func singleBoundedLeafForCol(flat []cnode, c int) *cnode {
	var found *cnode
	for i := range flat {
		if flat[i].op == condLeaf && flat[i].col == c {
			if found != nil || !flat[i].term.hasBounds {
				return nil
			}
			found = &flat[i]
		}
	}
	return found
}

// markUnknown sets unk on every node whose subtree contains a nullable leaf (a
// source of SQL UNKNOWN); only such subtrees carry an explicit F mask. The
// pre-order ids let a single reverse pass finish every child before its parent.
func markUnknown(flat []cnode) {
	// NOTE: do not "modernize" this to slices.Backward — the body mutates
	// flat[i].unk through the index, and `for _, f := range slices.Backward`
	// binds f to a COPY of the value element ([]cnode), so the writes would be
	// lost (every node's unk stays false, silently breaking 3VL NULL handling).
	// Covered by TestMarkUnknownNullableProp.
	for i := len(flat) - 1; i >= 0; i-- {
		if flat[i].op == condLeaf {
			flat[i].unk = flat[i].cv != nil && flat[i].cv.present != nil
			continue
		}
		for _, k := range flat[i].kids {
			if flat[k].unk {
				flat[i].unk = true
				break
			}
		}
	}
}

// evalCond returns the T mask (rows where node id is TRUE) and, when the subtree
// can produce UNKNOWN (unk), the F mask (rows where it is FALSE); a nil F means
// "no unknowns, F == complement of T".
func evalCond(flat []cnode, id, n int) (t, f []uint64) {
	nd := &flat[id]
	switch nd.op {
	case condLeaf:
		if nd.precompT != nil {
			// Zone-skip produced this leaf's T mask directly (skipped zones are
			// FALSE). The column is non-nullable (zone-chunk), so F is implicit.
			return nd.precompT, nil
		}
		return nd.cv.evalMasks(nd.term, n)
	case condAnd:
		return evalAnd(flat, id, n)
	case condOr:
		return evalOr(flat, id, n)
	case condNot:
		ct, cf := evalCond(flat, nd.kids[0], n)
		if cf == nil {
			cf = notMask(ct, n) // child had no unknowns: its F is the exact complement
		}
		if !nd.unk {
			return cf, nil // no unknowns: keep F implicit (nil) per the protocol
		}
		// NOT swaps TRUE/FALSE; UNKNOWN rows stay in neither mask.
		return cf, ct
	default:
		panic("qdf: evalCond: unknown op")
	}
}

// evalOr combines kids: T = OR of kid T; F = AND of kid F (or ~kidT when a kid
// has no unknowns). F stays nil when the whole subtree has no unknowns. With no
// kids the OR identity (no rows) is returned.
func evalOr(flat []cnode, id, n int) (t, f []uint64) {
	nd := &flat[id]
	t = newBitset(n)
	first := true
	for _, k := range nd.kids {
		kt, kf := evalCond(flat, k, n)
		bitsetOr(t, kt)
		if nd.unk {
			if kf == nil {
				kf = notMask(kt, n)
			}
			if first {
				f = slices.Clone(kf)
			} else {
				bitsetAnd(f, kf)
			}
		}
		first = false
	}
	return t, f
}

// evalAnd combines kids: T = AND of kid T; F = OR of kid F (or ~kidT when a kid
// has no unknowns). F stays nil when the whole subtree has no unknowns. With no
// kids the AND identity (all rows true) is returned.
func evalAnd(flat []cnode, id, n int) (t, f []uint64) {
	nd := &flat[id]
	t = fullBitset(n)
	for _, k := range nd.kids {
		kt, kf := evalCond(flat, k, n)
		bitsetAnd(t, kt)
		if nd.unk {
			if kf == nil {
				kf = notMask(kt, n)
			}
			if f == nil {
				f = slices.Clone(kf)
			} else {
				bitsetOr(f, kf)
			}
		}
	}
	return t, f
}
