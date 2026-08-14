package qdf

import "testing"

// A shape published into the encoder's table must not alias the caller's slice.
//
// derivePartialPlan builds its kinds in e.derivedPlan.hybridKinds — an encoder
// scratch that the NEXT derive rewrites in place. If the shape table retained
// that slice by reference, a second derive would mutate an entry the wire has
// already referred to by id, and hybridShapeFor would then match the new kinds
// against the mutated entry and hand back an id declared for a different layout.
//
// Asserted on the table directly rather than through a payload: the aliasing is
// a property of hybridShapeDeclare, and a payload-level test would only prove it
// for whichever shapes that payload happens to derive.
func TestHybridShapeDeclareCopiesKinds(t *testing.T) {
	st := newEncState()
	scratch := []colKind{colKindInt, residualKind, residualKind, colKindInt}
	names := []string{"a", "b", "c", "d"}

	id := st.hybridShapeDeclare(names, scratch)
	before := append([]colKind(nil), st.hybridShapeKinds[id-1]...)

	// What the next derive does to the same backing array.
	copy(scratch, []colKind{colKindInt, colKindInt, residualKind, residualKind})

	after := st.hybridShapeKinds[id-1]
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("shape id %d was declared as %v and now reads %v — the table "+
				"aliases the derive scratch, so a later derive rewrites a published entry",
				id, before, after)
		}
	}
}
