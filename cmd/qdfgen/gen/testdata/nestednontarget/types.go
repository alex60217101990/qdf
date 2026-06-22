// Package nestednontarget is a generator fixture: a target struct that
// references a nested struct which is neither a -type target nor a custom-codec
// type. qdfgen must reject this at generation time with a clear diagnostic
// instead of emitting an EncodeNested/DecodeNested call that fails to compile.
package nestednontarget

// Inner has no hand-written codec and is not passed as a -type target.
type Inner struct {
	X int64
}

// Parent is the generated target; its Inner field forces the nested-struct path.
type Parent struct {
	Name  string
	Inner Inner
}

// PtrParent exercises the pointer-to-nested-struct path.
type PtrParent struct {
	Name  string
	Inner *Inner
}
