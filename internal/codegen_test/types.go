// Package cgsample contains the user-defined types used to exercise the
// qdfgen code generator. The actual *_qdf.go file is produced by the
// TestGenerate test which invokes the in-process Generate() entry point.
package cgsample

import "time"

type Inner struct {
	X int     `qdf:"x"`
	Y float64 `qdf:"y"`
}

type Sample struct {
	Name   string            `qdf:"name"`
	Age    int               `qdf:"age"`
	Active bool              `qdf:"active"`
	Score  float64           `qdf:"score"`
	Tags   []string          `qdf:"tags"`
	Meta   map[string]string `qdf:"meta"`
	Inner  Inner             `qdf:"inner"`
	When   time.Time         `qdf:"when"`
	Buf    []byte            `qdf:"buf"`
	OptPtr *Inner            `qdf:"opt"`
	Counts [3]int32          `qdf:"counts"`
}

// Label is a defined string type, used both as a map key (the intern fast
// path) and as a pointed-to named scalar — two spots the generator must emit
// an explicit type conversion for.
type Label string

// EmbeddedBase is embedded by value into Edge; its exported fields must be
// flattened into Edge's wire layout, matching the reflect path.
type EmbeddedBase struct {
	BaseA int    `qdf:"base_a"`
	BaseB string `qdf:"base_b"`
}

// Edge exercises three generator paths the reflect path already handles but the
// codegen previously mis-emitted: anonymous embedded value-struct flattening,
// a map keyed by a defined string type, and a pointer to a defined scalar.
type Edge struct {
	EmbeddedBase
	Labels map[Label]int `qdf:"labels"`
	Ptr    *Label        `qdf:"ptr"`
	Tail   int           `qdf:"tail"`
}
