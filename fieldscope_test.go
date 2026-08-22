package qdf

import (
	"fmt"
	"testing"
)

// Without a bound scope WriteStringField must be exactly WriteString. That is
// what makes a standalone generated struct encode identically to a reflect
// struct outside a repeated slice, where the delta does not apply either.
func TestWriteStringFieldWithoutScopeIsPlain(t *testing.T) {
	a := NewEncoderWith(OptBalanced)
	a.EnsureHeader()
	b := NewEncoderWith(OptBalanced)
	b.EnsureHeader()
	for i := range 8 {
		s := fmt.Sprintf("com.acme.%04d.worker.service", i*7919%10000)
		a.WriteString(s)
		b.WriteStringField(3, s)
	}
	if string(a.Bytes()) != string(b.Bytes()) {
		t.Fatalf("unbound WriteStringField diverged from WriteString: %d vs %d bytes",
			len(a.Bytes()), len(b.Bytes()))
	}
}

// With a scope bound the delta must fire, and the wire must shrink.
func TestWriteStringFieldWithScopeEmitsTheDelta(t *testing.T) {
	var tok byte
	plain := NewEncoderWith(OptBalanced)
	plain.EnsureHeader()
	scoped := NewEncoderWith(OptBalanced)
	scoped.EnsureHeader()
	sc := scoped.PushFieldScope(&tok, 4)
	before := strDeltaEmitted.Load()
	for i := range 64 {
		s := fmt.Sprintf("com.acme.worker.service.shard.%04d", i)
		plain.WriteString(s)
		scoped.WriteStringField(3, s)
	}
	scoped.PopFieldScope(sc)
	if fired := strDeltaEmitted.Load() - before; fired == 0 {
		t.Fatal("a bound scope emitted no delta")
	}
	if len(scoped.Bytes()) >= len(plain.Bytes()) {
		t.Fatalf("scoped wire %d is not smaller than plain %d",
			len(scoped.Bytes()), len(plain.Bytes()))
	}
}

// An index outside the scope must fall back rather than panic or write against
// a neighboring field: WriteStringField is exported and takes an arbitrary int.
func TestWriteStringFieldRejectsAnOutOfRangeIndex(t *testing.T) {
	var tok byte
	e := NewEncoderWith(OptBalanced)
	e.EnsureHeader()
	sc := e.PushFieldScope(&tok, 2)
	defer e.PopFieldScope(sc)
	for _, i := range []int{-1, 2, 1 << 20} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("index %d panicked: %v", i, r)
				}
			}()
			e.WriteStringField(i, "value")
		}()
	}
}

// Scopes nest: a child's binding must not survive back into the parent, which
// is what a nested EncodeQDF between two of the parent's own fields would do.
func TestFieldScopeNests(t *testing.T) {
	var parent, child byte
	e := NewEncoderWith(OptBalanced)
	e.EnsureHeader()
	ps := e.PushFieldScope(&parent, 3)
	outer := e.curFields
	if outer == nil {
		t.Fatal("the parent scope bound nothing")
	}
	cs := e.PushFieldScope(&child, 2)
	if &e.curFields[0] == &outer[0] {
		t.Fatal("the child scope did not replace the parent's binding")
	}
	e.PopFieldScope(cs)
	if &e.curFields[0] != &outer[0] {
		t.Fatal("popping the child did not restore the parent's binding")
	}
	e.PopFieldScope(ps)
	if e.curFields != nil {
		t.Fatal("popping the outermost scope left a binding behind")
	}
}

// Reset must drop the binding, or a pooled encoder carries one type's field
// state into the next message.
func TestResetDropsTheFieldScope(t *testing.T) {
	var tok byte
	e := NewEncoderWith(OptBalanced)
	e.EnsureHeader()
	e.PushFieldScope(&tok, 3)
	e.Reset()
	if e.curFields != nil {
		t.Fatal("Reset left a field scope bound")
	}
}

// A suspended state is a never-larger trial whose bytes may be discarded.
// Binding there would leave the losing candidate's field state visible to
// whatever runs next — the same leak StructShape refuses to bind a token
// during.
func TestPushFieldScopeDeclinesWhileSuspended(t *testing.T) {
	var tok byte
	e := NewEncoderWith(OptBalanced)
	e.EnsureHeader()
	e.stateSuspended = true
	sc := e.PushFieldScope(&tok, 3)
	if e.curFields != nil {
		t.Fatal("a suspended encoder bound a field scope")
	}
	e.PopFieldScope(sc)
	e.stateSuspended = false
}

// Without Dense there is no state table for the codecs to live on, so the scope
// must decline and every value must take the plain path.
func TestPushFieldScopeDeclinesWithoutDense(t *testing.T) {
	var tok byte
	e := NewEncoderWith(OptSpeed)
	e.EnsureHeader()
	sc := e.PushFieldScope(&tok, 3)
	if e.curFields != nil {
		t.Fatal("a non-Dense encoder bound a field scope")
	}
	e.PopFieldScope(sc)
}
