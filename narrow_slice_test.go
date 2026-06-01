package qdf

import "testing"

type level uint8

func TestNarrowSlices_Matrix(t *testing.T) {
	type rec struct {
		I8  []int8
		I16 []int16
		U16 []uint16
		F32 []float32
		Lv  []level // named uint8 enum slice
	}
	v := rec{
		I8:  []int8{-128, 0, 127, -1, 1},
		I16: []int16{-32768, 0, 32767},
		U16: []uint16{0, 65535, 1234},
		F32: []float32{1.5, -0.5, 3.4e38},
		Lv:  []level{0, 1, 2, 1, 0, 3},
	}
	roundtripBundles(t, v)
}
