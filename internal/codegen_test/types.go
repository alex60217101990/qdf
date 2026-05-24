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
