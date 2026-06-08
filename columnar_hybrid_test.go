package qdf

import (
	"reflect"
	"testing"
)

// Phase 1: buildColumnarPlan three-way classification.
//   - every field eligible  → pure columnar (residual == nil)
//   - some eligible, some not → hybrid (residual != nil, full ordered shape)
//   - no eligible field       → nil (full row-major)
func TestBuildColumnarPlanHybrid(t *testing.T) {
	type allEligible struct {
		A int64
		B string
		C bool
	}
	type mixed struct {
		ID    string            // eligible
		N     int64             // eligible
		Tags  []string          // residual (non-[]byte slice)
		Attrs map[string]string // residual (map)
	}
	type allResidual struct {
		M map[string]int
		S []string
	}

	plan := func(v any) *columnarPlan {
		td, err := descOf(reflect.TypeOf(v))
		if err != nil {
			t.Fatal(err)
		}
		return buildColumnarPlan(td)
	}

	// All eligible → pure columnar, no residual.
	if p := plan(allEligible{}); p == nil || len(p.cols) != 3 || p.residual != nil {
		t.Fatalf("allEligible: want pure-columnar 3 cols + nil residual, got %#v", p)
	}

	// Mixed → hybrid.
	p := plan(mixed{})
	if p == nil {
		t.Fatal("mixed: got nil plan, want hybrid")
	}
	if len(p.cols) != 2 {
		t.Fatalf("mixed: want 2 eligible cols (ID,N), got %d", len(p.cols))
	}
	if len(p.residual) != 2 {
		t.Fatalf("mixed: want 2 residual fields (Tags,Attrs), got %d", len(p.residual))
	}
	if len(p.hybridNames) != 4 || len(p.hybridKinds) != 4 {
		t.Fatalf("mixed: hybrid shape must list ALL 4 fields, got names=%d kinds=%d",
			len(p.hybridNames), len(p.hybridKinds))
	}
	// Field declaration order preserved; residual fields marked with residualKind.
	wantResidual := map[string]bool{"Tags": true, "Attrs": true}
	for i, name := range p.hybridNames {
		gotResidual := p.hybridKinds[i] == residualKind
		if gotResidual != wantResidual[name] {
			t.Fatalf("mixed: field %q residual=%v, want %v", name, gotResidual, wantResidual[name])
		}
	}
	// Residual descriptors carry the field's own codec (non-nil desc).
	for _, rf := range p.residual {
		if rf.desc == nil {
			t.Fatalf("mixed: residual field %q has nil desc", rf.name)
		}
	}

	// No eligible field → nil plan (nothing to transpose).
	if p := plan(allResidual{}); p != nil {
		t.Fatalf("allResidual: want nil plan, got %#v", p)
	}
}

// Phase 2: the hybrid shape table is a separate ID space from the pure-columnar
// table — a stream interleaving tagColStruct and tagHybridColStruct payloads
// must not alias shape IDs.
func TestHybridShapeIDIndependence(t *testing.T) {
	names := []string{"a", "b", "c"}
	colKinds := []colKind{colKindInt, colKindString, colKindBool}
	hybKinds := []colKind{colKindInt, residualKind, colKindBool}

	e := newEncState()
	// Declaring in each table starts independently at ID 1.
	if id := e.colShapeDeclare(names, colKinds); id != 1 {
		t.Fatalf("colShapeDeclare first id = %d, want 1", id)
	}
	if id := e.hybridShapeDeclare(names, hybKinds); id != 1 {
		t.Fatalf("hybridShapeDeclare first id = %d, want 1", id)
	}
	// Lookups stay within their own table.
	if got := e.hybridShapeFor(names, hybKinds); got != 1 {
		t.Fatalf("hybridShapeFor reuse = %d, want 1", got)
	}
	// A hybrid kinds set must NOT match a columnar shape with the same names
	// (different kinds — residualKind sentinel differs).
	if got := e.colShapeFor(names, hybKinds); got != 0 {
		t.Fatalf("colShapeFor must not match hybrid kinds, got %d", got)
	}
	if got := e.hybridShapeFor(names, colKinds); got != 0 {
		t.Fatalf("hybridShapeFor must not match columnar kinds, got %d", got)
	}

	d := newDecState()
	dc := d.colShapeDeclareDec(names, colKinds)
	dh := d.hybridShapeDeclareDec(names, hybKinds)
	if dc == nil || dh == nil {
		t.Fatal("decoder declare returned nil")
	}
	// Same wire ID (1) resolves to the correct, independent table entry.
	if got := d.colShapeLookup(1); got == nil || got.kinds[1] != colKindString {
		t.Fatalf("colShapeLookup(1) wrong: %+v", got)
	}
	if got := d.hybridShapeLookup(1); got == nil || got.kinds[1] != residualKind {
		t.Fatalf("hybridShapeLookup(1) wrong (want residualKind at [1]): %+v", got)
	}
}
