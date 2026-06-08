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
