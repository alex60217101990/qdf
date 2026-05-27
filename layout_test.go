package qdf

import (
	"fmt"
	"reflect"
	"testing"
	"unsafe"
)

func TestEncStateLayout(t *testing.T) {
	rt := reflect.TypeFor[encState]()
	fmt.Printf("encState: size=%d align=%d\n", rt.Size(), rt.Align())
	for f := range rt.Fields() {
		f := f
		fmt.Printf("  +%4d  %-20s %s (size %d)\n", f.Offset, f.Name, f.Type, f.Type.Size())
	}
	fmt.Printf("Encoder size: %d\n", unsafe.Sizeof(Encoder{}))
}

func TestDecStateLayout(t *testing.T) {
	rt := reflect.TypeFor[decState]()
	fmt.Printf("decState: size=%d align=%d\n", rt.Size(), rt.Align())
	for f := range rt.Fields() {
		f := f
		fmt.Printf("  +%4d  %-20s %s (size %d)\n", f.Offset, f.Name, f.Type, f.Type.Size())
	}
	fmt.Printf("Decoder size: %d\n", unsafe.Sizeof(Decoder{}))
}
