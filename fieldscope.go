package qdf

import "unsafe"

// FieldScope is the previous field binding, returned by PushFieldScope and
// handed back to PopFieldScope. Opaque by design: its only correct use is to
// restore what was there before.
type FieldScope struct {
	prev []strFieldState
}

// FieldScoper lets a generated type name its field-state identity, so a caller
// looping over a slice of it — reflect's Marshaler slice path, or another
// generated encoder — can bind the scope once for the whole slice instead of
// once per element.
//
// The token is the same package-level var StructShape keys the shape table
// with, so a type has exactly one identity on the encoder no matter which
// mechanism asks for it.
type FieldScoper interface {
	QDFFieldScope() (token *byte, nFields int)
}

// PushFieldScope binds the per-field codec state for the struct type named by
// token and returns the previous binding for PopFieldScope.
//
// Call it ONCE around a loop over a slice of that type, not once per element.
// The lookup is a token compare against a two-slot cache; hoisting it out of
// the loop turns one lookup per element into one per slice.
//
// A bound scope is also what tells WriteStringField that the values it is given
// are consecutive rows of the same field. The string delta codes a value
// against the previous row and is meaningless otherwise, which is why the
// reflect path enables it only inside a repeated slice — the same condition,
// expressed once rather than as a second flag to keep in step.
func (e *Encoder) PushFieldScope(token *byte, nFields int) FieldScope {
	prev := e.curFields
	// Dense carries the state table the codecs live on. A suspended state is a
	// never-larger trial whose bytes may be thrown away, and binding there would
	// leave the losing candidate's field state visible to whatever runs next —
	// the same leak StructShape refuses to bind a token during.
	if token == nil || nFields <= 0 || !e.denseOn || e.stateSuspended {
		e.curFields = nil
		return FieldScope{prev: prev}
	}
	if e.state == nil {
		e.state = newEncState()
	}
	e.curFields = e.state.strFieldStates(unsafe.Pointer(token), nFields)
	return FieldScope{prev: prev}
}

// PopFieldScope restores the binding PushFieldScope replaced.
//
// Explicitly rather than through defer: a defer on this path measured +7.3% on
// OTLP and +10.4% on RTB when the slice encoder used one.
func (e *Encoder) PopFieldScope(prev FieldScope) {
	e.curFields = prev.prev
}

// WriteStringField writes struct field i of the type whose scope is bound.
//
// The index is explicit rather than a cursor because generated code writes
// fields conditionally — a nil slice field emits WriteNil, which would not
// advance a cursor, and every later field would then code against the wrong
// base. Silently, because the types still line up and nothing errors.
//
// With no scope bound this is WriteString, which is what a standalone struct
// gets on the reflect path too.
func (e *Encoder) WriteStringField(i int, s string) {
	fs := e.curFields
	// Unsigned, so the negative case costs nothing extra: a caller outside this
	// package can pass any int, and an out-of-range index must fall back rather
	// than panic or write against a neighbouring field.
	if uint(i) >= uint(len(fs)) {
		e.WriteString(s)
		return
	}
	e.writeStringField(s, &fs[i])
}
