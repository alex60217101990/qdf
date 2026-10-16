package qdf

import (
	"strconv"
	"testing"
	"unsafe"
)

// TestFSSTCacheIsPerColumn gates the cache key. Before it, the encoder kept one
// trained table and handed it to every later FSST column of the same message,
// so a column's block was compressed with a table trained on a different
// column's text: +123% wire on the access corpus under OptCompression, and
// nothing failed, because the table is always serialized and decodes fine.
//
// The check is the property that bug broke: a column measures the same whether
// or not another column was measured on the same encoder first.
func TestFSSTCacheIsPerColumn(t *testing.T) {
	urls := make([]string, 512)
	logs := make([]string, 512)
	for i := range urls {
		urls[i] = "https://cdn.example.com/static/assets/img/" + strconv.Itoa(i*7919%100003) + ".webp?v=2"
		logs[i] = "level=warn component=scheduler msg=\"queue backlog\" depth=" + strconv.Itoa(i*31%977)
	}
	var keyA, keyB colColumn

	fresh := NewEncoderWith(OptBalanced | OptFSST)
	fresh.fsstKey = unsafe.Pointer(&keyB)
	want, ok := fresh.measureStringColumnFSST(logs)
	if !ok {
		t.Fatal("fixture: the log column must FSST-code, or this test checks nothing")
	}

	e := NewEncoderWith(OptBalanced | OptFSST)
	e.fsstKey = unsafe.Pointer(&keyA)
	if _, ok := e.measureStringColumnFSST(urls); !ok {
		t.Fatal("fixture: the URL column must FSST-code, or it trains no table to leak")
	}
	e.fsstKey = unsafe.Pointer(&keyB)
	got, ok := e.measureStringColumnFSST(logs)
	if !ok || got != want {
		t.Fatalf("log column after a URL column: %d B (ok=%v), alone: %d B — it was coded with another column's table", got, ok, want)
	}

	// And the cache still does its job for the SAME column on the next batch:
	// the table is reused rather than retrained.
	if len(e.sc().fsstCache) != 2 {
		t.Fatalf("want one cache slot per column, got %d", len(e.sc().fsstCache))
	}
	before := e.sc().fsstCache[0].uses
	e.fsstKey = unsafe.Pointer(&keyA)
	e.measureStringColumnFSST(urls)
	if e.sc().fsstCache[0].uses != before+1 {
		t.Fatalf("second batch of the URL column retrained instead of reusing (uses %d -> %d)", before, e.sc().fsstCache[0].uses)
	}
}

// TestFSSTCacheKeyStableAcrossDerivedPlans gates the identity the cache keys on.
// A derived (hybrid) plan is rebuilt in encoder scratch every batch, so the
// address of its column entry names whichever field landed in that slot this
// time. colIdentity must name the cached plan's entry instead.
func TestFSSTCacheKeyStableAcrossDerivedPlans(t *testing.T) {
	plan := &columnarPlan{cols: []colColumn{
		{name: "a", kind: colKindString},
		{name: "b", kind: colKindString},
		{name: "c", kind: colKindInt},
	}}
	plan.colNames = []string{"a", "b", "c"}
	plan.colKinds = []colKind{colKindString, colKindString, colKindInt}
	e := NewEncoderWith(OptBalanced | OptFSST)

	// Batch 1 keeps a and b; batch 2 keeps only b. In batch 2, b sits in the slot
	// a held in batch 1.
	d1 := e.derivePartialPlan(plan, []bool{true, true, false})
	idA := d1.colIdentity(0)
	e.deriveDepth = 0
	d2 := e.derivePartialPlan(plan, []bool{false, true, false})
	e.deriveDepth = 0
	if &d1.cols[0] != &d2.cols[0] {
		t.Skip("derived plans no longer share scratch; the hazard this guards is gone")
	}
	if d2.colIdentity(0) == idA {
		t.Fatal("column b in batch 2 has column a's identity from batch 1: the FSST cache would hand it a's table")
	}
	if d2.colIdentity(0) != unsafe.Pointer(&plan.cols[1]) {
		t.Fatal("identity must be the cached plan's entry for the field")
	}
}
