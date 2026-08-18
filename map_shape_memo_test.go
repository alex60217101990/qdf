package qdf

import "testing"

// The map-shape memo has three parts — lastMapShapeID, lastMapShapeKeys and
// lastMapShapeIdx — written by TWO paths: mapStringShapeOrder (the generated
// map fast path) and encodeMapStringShape (reflect). The reflect memo indexes
// mapShapes with the Idx while putting the ID on the wire, so if one writer
// updates two parts and not the third, a stale index can pair one shape's id
// with another shape's slot ordering.
//
// mapStringShapeOrder used to leave Idx behind. The invariant is asserted here
// directly rather than through a payload: the desync depends on which paths a
// message happens to interleave, and a payload test would only cover the pairs
// that payload produces.
func TestMapShapeMemoPartsStayInStep(t *testing.T) {
	e := NewEncoderWith(OptBalanced)
	e.EnsureHeader()
	if e.state == nil {
		e.state = newEncState()
	}
	st := e.state

	// Two distinct key sets, driven through the generated path.
	for _, m := range []map[string]string{
		{"aa": "1", "bb": "2"},
		{"xx": "1", "yy": "2"},
		{"aa": "1", "bb": "2"},
	} {
		mapStringShapeOrder(e, m)
		if st.lastMapShapeID == 0 {
			t.Fatal("no shape memoised")
		}
		if st.lastMapShapeIdx >= len(st.mapShapes) {
			t.Fatalf("lastMapShapeIdx %d out of range (%d shapes)",
				st.lastMapShapeIdx, len(st.mapShapes))
		}
		if got := st.mapShapes[st.lastMapShapeIdx].id; got != st.lastMapShapeID {
			t.Fatalf("memo desync: index %d holds shape id %d, but lastMapShapeID is %d — "+
				"the reflect memo would emit the latter while ordering values by the former",
				st.lastMapShapeIdx, got, st.lastMapShapeID)
		}
	}
}
